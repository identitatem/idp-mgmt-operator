// Copyright Red Hat

package hypershiftdeployment

import (
	"context"

	giterrors "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	// clusterv1 "open-cluster-management.io/api/cluster/v1"
	// clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	hypershiftdeploymentv1alpha1 "github.com/stolostron/hypershift-deployment-controller/api/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	//+kubebuilder:scaffold:imports
)

// HypershiftDeploymentReconciler reconciles a Strategy object
type HypershiftDeploymentReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources={configmaps},verbs=get;create;update;list;watch;delete

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms},verbs=get;list;watch;get
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms/status},verbs=patch

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={hypershiftdeployments},verbs=get;list;watch;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Strategy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HypershiftDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

	// your logic here
	// Fetch the ClusterOAuth instance
	instance := &hypershiftdeploymentv1alpha1.HypershiftDeployment{}

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

	r.Log.Info("Running Reconcile for HypershiftDeployment.", "Name: ", instance.GetName(), " Namespace:", instance.GetNamespace())

	//TODO add test if managedclsuter exists for that ns/cluster
	if instance.DeletionTimestamp != nil {
		if result, err := r.processHypershiftDeployment(instance, true); err != nil {
			return result, err
		}
		return reconcile.Result{}, nil
	}

	if result, err := r.processHypershiftDeployment(instance, false); err != nil {
		return result, err
	}

	return reconcile.Result{}, nil
}

func (r *HypershiftDeploymentReconciler) processHypershiftDeployment(hd *hypershiftdeploymentv1alpha1.HypershiftDeployment, delete bool) (reconcile.Result, error) {
	clusterOAuths := &identitatemv1alpha1.ClusterOAuthList{}
	if err := r.Client.List(context.TODO(), clusterOAuths, client.InNamespace(hd.Spec.InfraID)); err != nil {
		return ctrl.Result{}, err
	}
	for i, clusterOAuth := range clusterOAuths.Items {
		r.Log.Info("process hypershiftDeployment", "hypershiftDeploymentName", hd.Name, "clusterOAuthName", clusterOAuth.Name)
		authRealm := &identitatemv1alpha1.AuthRealm{}
		if err := r.Client.Get(context.TODO(),
			client.ObjectKey{
				Name:      clusterOAuth.Spec.AuthRealmReference.Name,
				Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace},
			authRealm); err != nil {
			return ctrl.Result{}, err
		}
		mc := &clusterv1.ManagedCluster{}
		err := r.Client.Get(context.TODO(), client.ObjectKey{Name: clusterOAuth.Namespace}, mc)
		if err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		if helpers.IsHostedCluster(mc) {
			r.Log.Info("Update AuthRealm hypershift status")
			if err := r.updateAuthRealmStatusHypershiftDeploymentConditions(
				authRealm,
				&clusterOAuths.Items[i],
				hd,
				delete); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HypershiftDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := corev1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	if err := hypershiftdeploymentv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&hypershiftdeploymentv1alpha1.HypershiftDeployment{},
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					mwOld := e.ObjectOld.(*hypershiftdeploymentv1alpha1.HypershiftDeployment)
					mwNew := e.ObjectNew.(*hypershiftdeploymentv1alpha1.HypershiftDeployment)
					return !equality.Semantic.DeepEqual(mwOld.Status, mwNew.Status) ||
						mwOld.DeletionTimestamp != mwNew.DeletionTimestamp
				}},
			)).Complete(r)
}
