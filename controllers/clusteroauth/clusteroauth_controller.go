// Copyright Red Hat

package clusteroauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	//"fmt"

	//"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	"github.com/identitatem/idp-mgmt-operator/resources"

	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"

	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	idpconfig "github.com/identitatem/idp-client-api/config"
	idpmgmtconfig "github.com/identitatem/idp-mgmt-operator/config"
	openshiftconfigv1 "github.com/openshift/api/config/v1"

	//+kubebuilder:scaffold:imports

	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

const posthookAnnotation string = "managedcluster-import-controller.open-cluster-management.io/posthook-graceperiod"

// ClusterOAuthReconciler reconciles a Strategy object
type ClusterOAuthReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,strategies,clusteroauths},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/finalizers,verbs=update
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;list;create;update;patch;delete

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placements,placementdecisions},verbs=get;list;watch;create;update;patch;delete;watch
//+kubebuilder:rbac:groups=view.open-cluster-management.io,resources={managedclusterviews},verbs=get;list;watch;create;update;patch;delete;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Strategy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterOAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

	// your logic here
	// Fetch the ClusterOAuth instance
	instance := &identitatemv1alpha1.ClusterOAuth{}

	if err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: req.Namespace, Name: req.Name},
		instance,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	r.Log.Info("Instance", "instance", instance)
	r.Log.Info("Running Reconcile for ClusterOAuth.", "Name: ", instance.GetName(), " Namespace:", instance.GetNamespace())

	//TODO   - I think this only applies to backplane so no need to check
	//         If grc also uses this, we need to have a way of knowing what strategy type created
	//         the CR so we can build ManifestWork properly for that strategy.  For example, only secrets in
	//         GRC manifestwork
	//switch instance.Spec.Type {
	//case identitatemv1alpha1.BackplaneStrategyType:

	if instance.DeletionTimestamp != nil {
		if result, err := r.unmanagedCluster(instance); err != nil {
			return result, err
		}
		// r.Log.Info("remove PlacementDecision finalizer", "Finalizer:", helpers.ClusterOAuthFinalizer)
		// controllerutil.RemoveFinalizer(instance, helpers.ClusterOAuthFinalizer)
		// if err := r.Client.Update(context.TODO(), instance); err != nil {
		// 	return ctrl.Result{}, err
		// }
		return reconcile.Result{}, nil
	}

	r.Log.Info("generateManagedClusterViewForOAuth for", "name", instance.Name, "namespace", instance.Namespace)
	if result, err := r.generateManagedClusterViewForOAuth(instance); err != nil {
		return result, err
	}

	r.Log.Info("generateManifestWork for", "name", instance.Name, "namespace", instance.Namespace)
	if err := r.generateManifestWork(instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterOAuthReconciler) generateManagedClusterViewForOAuth(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (result ctrl.Result, err error) {
	cm := &corev1.ConfigMap{}

	//If already exist, do nothing
	r.Log.Info("check if configMap already exists", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", clusterOAuth.Namespace)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: clusterOAuth.Namespace}, cm); err == nil {
		return ctrl.Result{}, nil
	}

	r.Log.Info("create managedclusterview", "name", helpers.ManagedClusterViewOAuthName(), "namespace", clusterOAuth.Namespace)
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerResources := resources.GetScenarioResourcesReader()

	file := "managedclusterview/managed-cluster-view-oauth.yaml"

	values := struct {
		Name      string
		Namespace string
	}{
		Name:      helpers.ManagedClusterViewOAuthName(),
		Namespace: clusterOAuth.Namespace,
	}
	if _, err := applier.ApplyCustomResources(readerResources, values, false, "", file); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("saveManagedClusterViewForOAuthResult for", "name", clusterOAuth.Name, "namespace", clusterOAuth.Namespace)
	return r.saveManagedClusterViewForOAuthResult(clusterOAuth)
}

func (r *ClusterOAuthReconciler) saveManagedClusterViewForOAuthResult(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (result ctrl.Result, err error) {
	mcvOAuth := &viewv1beta1.ManagedClusterView{}
	r.Log.Info("check if managedclusterview exists", "name", helpers.ManagedClusterViewOAuthName(), "namespace", clusterOAuth.Namespace)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ManagedClusterViewOAuthName(), Namespace: clusterOAuth.Namespace}, mcvOAuth); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("check if managedclusterview has results", "name", helpers.ManagedClusterViewOAuthName(), "namespace", clusterOAuth.Namespace)
	if len(mcvOAuth.Status.Result.Raw) == 0 {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, fmt.Errorf("waiting for cluster %s oauth", clusterOAuth.Namespace)
	}

	r.Log.Info("create configmap containing the OAuth", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", clusterOAuth.Namespace)
	b, err := yaml.JSONToYAML(mcvOAuth.Status.Result.Raw)
	if err != nil {
		return ctrl.Result{}, err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ConfigMapOriginalOAuthName(),
			Namespace: clusterOAuth.Namespace,
		},
		Data: map[string]string{
			"json": string(mcvOAuth.Status.Result.Raw),
			"yaml": string(b),
		},
	}

	if err := r.Client.Create(context.TODO(), cm); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("delete managedclusterview", "name", helpers.ManagedClusterViewOAuthName(), "namespace", clusterOAuth.Namespace)
	if err := r.Client.Delete(context.TODO(), mcvOAuth); err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterOAuthReconciler) generateManifestWork(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (err error) {
	// Create empty manifest work
	r.Log.Info("prepare manifestwork", "name", helpers.ManifestWorkOAuthName(), "namespace", clusterOAuth.GetNamespace())
	manifestWorkOAuth := &manifestworkv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helpers.ManifestWorkOAuthName(),
			Namespace: clusterOAuth.GetNamespace(),
			Annotations: map[string]string{
				posthookAnnotation: "60",
			},
		},
		Spec: manifestworkv1.ManifestWorkSpec{
			Workload: manifestworkv1.ManifestsTemplate{
				Manifests: []manifestworkv1.Manifest{},
			},
		},
	}
	//Add Aggregated role
	r.Log.Info("add aggregated role")
	aggregatedRoleYaml, err := idpmgmtconfig.GetScenarioResourcesReader().Asset("rbac/role-aggregated-clusterrole.yaml")
	if err != nil {
		return err
	}

	aggregatedRoleJson, err := yaml.YAMLToJSON(aggregatedRoleYaml)
	if err != nil {
		return err
	}

	manifestAggregated := manifestworkv1.Manifest{
		RawExtension: runtime.RawExtension{Raw: aggregatedRoleJson},
	}

	manifestWorkOAuth.Spec.Workload.Manifests = append(manifestWorkOAuth.Spec.Workload.Manifests, manifestAggregated)

	// Get a list of all clusterOAuth
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	//	singleOAuth := &openshiftconfigv1.OAuth{}
	singleOAuth := &openshiftconfigv1.OAuth{
		TypeMeta: metav1.TypeMeta{
			APIVersion: openshiftconfigv1.SchemeGroupVersion.String(),
			Kind:       "OAuth",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},

		Spec: openshiftconfigv1.OAuthSpec{},
	}

	r.Log.Info("search the clusterOAuths in namepsace", "namespace", clusterOAuth.GetNamespace())
	if err := r.List(context.TODO(), clusterOAuths, &client.ListOptions{Namespace: clusterOAuth.GetNamespace()}); err != nil {
		// Error reading the object - requeue the request.
		return err
	}

	for _, clusterOAuth := range clusterOAuths.Items {
		//build OAuth and add to manifest work
		r.Log.Info(" build clusterOAuth", "name: ", clusterOAuth.GetName(), " namespace:", clusterOAuth.GetNamespace(), "identityProviders:", len(clusterOAuth.Spec.OAuth.Spec.IdentityProviders))

		for j, idp := range clusterOAuth.Spec.OAuth.Spec.IdentityProviders {

			r.Log.Info("process identityProvider", "identityProvider  ", j, " name:", idp.Name)

			//build oauth by appending first clusterOAuth entry into single OAuth
			singleOAuth.Spec.IdentityProviders = append(singleOAuth.Spec.IdentityProviders, idp)
		}
	}

	for _, clusterOAuth := range clusterOAuths.Items {
		//build OAuth and add to manifest work
		r.Log.Info(" build clusterOAuth", "name: ", clusterOAuth.GetName(), " namespace:", clusterOAuth.GetNamespace(), "identityProviders:", len(clusterOAuth.Spec.OAuth.Spec.IdentityProviders))

		for _, idp := range clusterOAuth.Spec.OAuth.Spec.IdentityProviders {

			//Look for secret for Identity Provider and if found, add to manifest work
			secret := &corev1.Secret{}

			r.Log.Info("retrieving client secret", "name", idp.OpenID.ClientSecret.Name, "namespace", clusterOAuth.Namespace)
			if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: clusterOAuth.Namespace, Name: idp.OpenID.ClientSecret.Name}, secret); err != nil {
				return err
			}
			//add secret to manifest

			//TODO TEMP PATCH
			newSecret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: "openshift-config",
				},
			}

			newSecret.Data = secret.Data
			newSecret.Type = secret.Type

			data, err := json.Marshal(newSecret)
			if err != nil {
				return err
			}

			manifest := manifestworkv1.Manifest{
				RawExtension: runtime.RawExtension{Raw: data},
			}

			//add manifest to manifest work
			r.Log.Info("append workload for secret in manifest for idp", "name", idp.Name)
			manifestWorkOAuth.Spec.Workload.Manifests = append(manifestWorkOAuth.Spec.Workload.Manifests, manifest)

		}
	}

	// create manifest for single OAuth
	data, err := json.Marshal(singleOAuth)
	if err != nil {
		return err
	}

	manifest := manifestworkv1.Manifest{
		RawExtension: runtime.RawExtension{Raw: data},
	}

	//add OAuth manifest to manifest work
	r.Log.Info("append workload for oauth in manifest")
	manifestWorkOAuth.Spec.Workload.Manifests = append(manifestWorkOAuth.Spec.Workload.Manifests, manifest)

	// create manifest work for managed cluster
	// (borrowed from https://github.com/open-cluster-management/endpoint-operator/blob/master/pkg/utils/utils.go)
	if err := r.CreateOrUpdateManifestWork(manifestWorkOAuth, r.Client, manifestWorkOAuth, r.Scheme); err != nil {
		r.Log.Error(err, "Failed to create manifest work for component")
		// Error reading the object - requeue the request.
		return err
	}
	return nil
}

// CreateOrUpdateManifestWork creates a new ManifestWork or update an existing ManifestWork
func (r *ClusterOAuthReconciler) CreateOrUpdateManifestWork(
	manifestwork *manifestworkv1.ManifestWork,
	client client.Client,
	owner metav1.Object,
	scheme *runtime.Scheme,
) error {

	oldManifestwork := &manifestworkv1.ManifestWork{}

	err := client.Get(
		context.TODO(),
		types.NamespacedName{Name: manifestwork.Name, Namespace: manifestwork.Namespace},
		oldManifestwork,
	)
	if err == nil {
		oldManifestwork.Spec.Workload = manifestwork.Spec.Workload
		if err := client.Update(context.TODO(), oldManifestwork); err != nil {
			r.Log.Error(err, "Fail to update manifestwork")
			return err
		}
		return nil
	}
	if errors.IsNotFound(err) {
		r.Log.Info("create manifestwork", "name", manifestwork.Name, "namespace", manifestwork.Namespace)
		if err := client.Create(context.TODO(), manifestwork); err != nil {
			r.Log.Error(err, "Fail to create manifestwork")
			return err
		}
		return nil
	}
	return nil
}

// unmanagedCluster deletes a manifestwork
func (r *ClusterOAuthReconciler) unmanagedCluster(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (result ctrl.Result, err error) {
	r.Log.Info("deleteManifestWork", "clusterOAuth", clusterOAuth)
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	if err := r.Client.List(context.TODO(), clusterOAuths, client.InNamespace(clusterOAuth.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	if len(clusterOAuths.Items) == 1 {
		r.Log.Info("delete for manifestwork", "name", helpers.ManifestWorkOAuthName(), "namespace", clusterOAuth.Namespace)
		if err := r.deleteManifestWork(clusterOAuth); err != nil {
			return ctrl.Result{}, err
		}
		// if result, err := r.restoreOriginalOAuth(clusterOAuth); err != nil {
		// 	return result, err
		// }
	}
	return ctrl.Result{}, nil
}

func (r *ClusterOAuthReconciler) restoreOriginalOAuth(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (result ctrl.Result, err error) {
	cm := &corev1.ConfigMap{}

	//If already exist, do nothing
	r.Log.Info("check if configMap exists", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", clusterOAuth.Namespace)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: clusterOAuth.Namespace}, cm); err != nil {
		return ctrl.Result{}, err
	}

	originalOAuth := &corev1.ConfigMap{}
	if err := r.Get(context.TODO(), client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: clusterOAuth.Namespace}, originalOAuth); err != nil {
		return ctrl.Result{}, err
	}

	jsonData, err := yaml.YAMLToJSON([]byte(originalOAuth.Data[helpers.ConfigMapOriginalOAuthName()]))
	if err != nil {
		return ctrl.Result{}, err
	}
	mw := &manifestworkv1.ManifestWork{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ManifestWorkOAuthName(), Namespace: clusterOAuth.Namespace}, mw); err != nil {
		return ctrl.Result{}, err
	}

	mw.Spec.Workload = manifestworkv1.ManifestsTemplate{
		Manifests: []manifestworkv1.Manifest{
			{RawExtension: runtime.RawExtension{Raw: jsonData}},
		},
	}

	if err := r.Update(context.TODO(), mw); err != nil {
		return ctrl.Result{}, err
	}

	//TODO create mananifestwork with orphan

	//TODO wait manifestwork applied

	//TODO delete manifestwork

	return ctrl.Result{}, nil
}

func (r *ClusterOAuthReconciler) deleteManifestWork(clusterOAuth *identitatemv1alpha1.ClusterOAuth) error {
	manifestWork := &manifestworkv1.ManifestWork{}
	if err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: helpers.ManifestWorkOAuthName(), Namespace: clusterOAuth.Namespace},
		manifestWork,
	); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if manifestWork.DeletionTimestamp.IsZero() {
			r.Log.Info("delete manifest", "name", manifestWork.Name, "namespace", manifestWork.Namespace)
			err := r.Client.Delete(context.TODO(), manifestWork)
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterOAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpconfig.GetScenarioResourcesReader()

	file := "crd/bases/identityconfig.identitatem.io_clusteroauths.yaml"
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", file); err != nil {
		return err
	}

	if err := corev1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := clusterv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := manifestworkv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := identitatemdexv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := openshiftconfigv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := viewv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemv1alpha1.ClusterOAuth{}).
		Owns(&manifestworkv1.ManifestWork{}).
		Complete(r)
}
