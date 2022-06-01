// Copyright Red Hat

package clusteroauth

import (
	"context"
	"fmt"

	giterrors "github.com/pkg/errors"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	// clusterv1 "open-cluster-management.io/api/cluster/v1"
	// clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	hypershiftdeploymentv1alpha1 "github.com/stolostron/hypershift-deployment-controller/api/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"

	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	idpconfig "github.com/identitatem/idp-client-api/config"
	openshiftconfigv1 "github.com/openshift/api/config/v1"

	//+kubebuilder:scaffold:imports

	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

const posthookAnnotation string = "managedcluster-import-controller.open-cluster-management.io/posthook-graceperiod"

const DEX_CLIENT_SECRET_LABEL = "auth.identitatem.io/dex-client-secret"

// ClusterOAuthReconciler reconciles a Strategy object
type ClusterOAuthReconciler struct {
	client.Client
	KubeClient                    kubernetes.Interface
	DynamicClient                 dynamic.Interface
	APIExtensionClient            apiextensionsclient.Interface
	HypershiftDeploymentInstalled bool
	Log                           logr.Logger
	Scheme                        *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources={configmaps},verbs=get;create;update;list;watch;delete

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={clusteroauths},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={clusteroauths/finalizers},verbs=create;delete;patch;update
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={clusteroauths/status},verbs=update;patch
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,strategies},verbs=get;list;watch
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms/status},verbs=update;patch

//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources={manifestworks},verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={hypershiftdeployments},verbs=get;list;watch;update

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
		r.Log.Info("delete clusterOAuth", "name", instance.Name, "namespace", instance.Namespace)

		if result, err := r.deleteClusterOAuth(instance); err != nil || result.Requeue {
			return result, err
		}
		r.Log.Info("remove finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", instance.Name, "namespace", instance.Namespace)
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	r.Log.Info("update clusterOAuth", "name", instance.Name, "namespace", instance.Namespace)
	if result, err := r.processClusterOAuth(instance); err != nil || result.Requeue {
		cond := metav1.Condition{
			Type:    identitatemv1alpha1.ClusterOAuthApplied,
			Status:  metav1.ConditionFalse,
			Reason:  "ClusterOAuthAppliedFailed",
			Message: fmt.Sprintf("ClusterOAuth %s failed to save original OAuth for cluster %s Error:, %s", instance.Name, instance.Namespace, err.Error()),
		}

		if err := r.setConditions(instance, cond); err != nil {
			return reconcile.Result{}, err
		}
		return result, err
	}

	cond := metav1.Condition{
		Type:    identitatemv1alpha1.ClusterOAuthApplied,
		Status:  metav1.ConditionTrue,
		Reason:  "ClusterOAuthAppliedSucceeded",
		Message: "ClusterOAuth successfully applied",
	}

	if err := r.setConditions(instance, cond); err != nil {
		return ctrl.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ClusterOAuthReconciler) processClusterOAuth(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (reconcile.Result, error) {

	controllerutil.AddFinalizer(clusterOAuth, helpers.AuthrealmFinalizer)

	r.Log.Info("Process", "Name", clusterOAuth.GetName(), "Namespace", clusterOAuth.GetNamespace())

	if err := r.Client.Update(context.TODO(), clusterOAuth); err != nil {
		return reconcile.Result{}, giterrors.WithStack(err)
	}

	strategy := identitatemv1alpha1.Strategy{}
	if err := r.Get(context.TODO(), client.ObjectKey{
		Name:      clusterOAuth.Spec.StrategyReference.Name,
		Namespace: clusterOAuth.Spec.StrategyReference.Namespace,
	}, &strategy); err != nil {
		return ctrl.Result{}, err
	}
	var mgr ClusterOAuthMgr
	switch strategy.Spec.Type {
	case identitatemv1alpha1.HypershiftStrategyType:
		mgr = &HypershiftMgr{r, clusterOAuth}
	case identitatemv1alpha1.BackplaneStrategyType:
		mgr = &BackplaneMgr{r, clusterOAuth}
	default:
		return ctrl.Result{}, giterrors.WithStack(
			fmt.Errorf("unsupported strategy %s for cluster %s",
				strategy.Spec.Type, clusterOAuth.Namespace))
	}

	r.Log.Info("save oauth for", "name", clusterOAuth.Name, "namespace", clusterOAuth.Namespace)
	if result, err := mgr.Save(); err != nil || result.Requeue {
		return result, err
	}
	r.Log.Info("push oauth for", "name", clusterOAuth.Name, "namespace", clusterOAuth.Namespace)
	if err := mgr.Push(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deleteClusterOAuth deletes a manifestwork
func (r *ClusterOAuthReconciler) deleteClusterOAuth(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (result ctrl.Result, err error) {
	r.Log.Info("deleteClusterOAuth", "clusterOAuth", clusterOAuth)
	if err := r.deleteAuthRealmConditions(clusterOAuth); err != nil {
		return ctrl.Result{}, err
	}
	mc := &clusterv1.ManagedCluster{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: clusterOAuth.Namespace}, mc)
	if err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	var mgr ClusterOAuthMgr
	if helpers.IsHostedCluster(mc) {
		mgr = &HypershiftMgr{r, clusterOAuth}
	} else {
		mgr = &BackplaneMgr{r, clusterOAuth}
	}
	return mgr.Unmanage()
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterOAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpconfig.GetScenarioResourcesReader()

	file := "crd/bases/identityconfig.identitatem.io_clusteroauths.yaml"
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", file); err != nil {
		return giterrors.WithStack(err)
	}

	if err := corev1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := manifestworkv1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := identitatemdexv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := openshiftconfigv1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := viewv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := hypershiftv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if r.HypershiftDeploymentInstalled {
		if err := hypershiftdeploymentv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return giterrors.WithStack(err)
		}
	}

	c := ctrl.NewControllerManagedBy(mgr).
		For(&identitatemv1alpha1.ClusterOAuth{}).
		Owns(&manifestworkv1.ManifestWork{}).
		Watches(&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					if _, ok := e.ObjectNew.GetLabels()[DEX_CLIENT_SECRET_LABEL]; ok {
						return ok
					}
					return false
				},
			}))
	if r.HypershiftDeploymentInstalled {
		c = c.Watches(&source.Kind{Type: &hypershiftdeploymentv1alpha1.HypershiftDeployment{}}, &handler.EnqueueRequestForObject{})
	}
	return c.Complete(r)
}
