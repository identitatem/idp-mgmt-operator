// Copyright Red Hat

package strategy

import (
	"context"
	"fmt"

	// "time"

	ocinfrav1 "github.com/openshift/api/config/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	// "k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	// identitatemdexserverv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	//ocm "github.com/open-cluster-management-io/api/cluster/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	idpconfig "github.com/identitatem/idp-client-api/config"

	//+kubebuilder:scaffold:imports

	// clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/identitatem/idp-mgmt-operator/controllers/helpers"
)

// StrategyReconciler reconciles a Strategy object
type StrategyReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,strategies},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/finalizers,verbs=update
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;list;create;update;patch;delete

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placements},verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Strategy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *StrategyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("strategy", req.NamespacedName)

	// your logic here
	// Fetch the ManagedCluster instance
	instance := &identitatemv1alpha1.Strategy{}

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

	if instance.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	r.Log.Info("Instance", "instance", instance)
	r.Log.Info("Running Reconcile for Strategy.", "Name: ", instance.GetName(), " Namespace:", instance.GetNamespace())

	// Get the AuthRealm Placement bits we need to help create a new Placement

	r.Log.Info("Searching for AuthRealm in ownerRefs", "strategy", instance.Name)
	authrealm, err := helpers.GetAuthrealmFromStrategy(r.Client, instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	// get placement info from AuthRealm ownerRef

	//Make sure Placement is created and correct
	//TODO!!! Right now we will have to manullay add a label to managed clusters in order for the placementDecision
	//        to return a result cloudservices=grc|backplane
	//Check if there is a predicate to add, if not nothing to do

	placement := &clusterv1alpha1.Placement{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authrealm.Spec.PlacementRef.Name, Namespace: req.Namespace}, placement); err != nil {
		return reconcile.Result{}, err
	}

	//Get placementStrategy
	placementStrategy, placementStrategyExists, err := r.getStrategyPlacement(instance, authrealm, placement)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Enrich placementStrategy
	switch instance.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		if err := r.backplanePlacementStrategy(instance, authrealm, placement, placementStrategy); err != nil {
			return reconcile.Result{}, err
		}
	// case identitatemv1alpha1.GrcStrategyType:
	// 	if err := r.grcPlacementStrategy(instance, authrealm, placement, placementStrategy); err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	default:
		return reconcile.Result{}, fmt.Errorf("strategy type %s not supported", instance.Spec.Type)
	}

	//Create or update placementStrategy
	switch placementStrategyExists {
	case true:
		if err := r.Client.Update(context.TODO(), placementStrategy); err != nil {
			return reconcile.Result{}, err
		}
	case false:
		if err := r.Client.Create(context.Background(), placementStrategy); err != nil {
			return reconcile.Result{}, err
		}
	}

	// update the Placement ref
	instance.Spec.PlacementRef.Name = placementStrategy.Name
	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *StrategyReconciler) getStrategyPlacement(strategy *identitatemv1alpha1.Strategy,
	authrealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) (*clusterv1alpha1.Placement, bool, error) {
	placementStrategy := &clusterv1alpha1.Placement{}
	placementStrategyExists := true
	placementStrategyName := getPlacementStrategyName(strategy, authrealm)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: placementStrategyName, Namespace: strategy.Namespace}, placementStrategy); err != nil {
		if !errors.IsNotFound(err) {
			return nil, false, err
		}
		placementStrategyExists = false
		// Not Found! Create
		placementStrategy = &clusterv1alpha1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: strategy.Namespace,
				//DV The name is given by the authrealm as the user will define the binding with the clusterset
				//Name:      req.Name,
				Name: placementStrategyName,
			},
			//DV move below
			Spec: placement.Spec,
		}
		// Set owner reference for cleanup
		controllerutil.SetOwnerReference(strategy, placementStrategy, r.Scheme)
	}
	return placementStrategy, placementStrategyExists, nil
}

func getPlacementStrategyName(strategy *identitatemv1alpha1.Strategy,
	authrealm *identitatemv1alpha1.AuthRealm) string {
	return fmt.Sprintf("%s-%s", authrealm.Spec.PlacementRef.Name, strategy.Spec.Type)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StrategyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpconfig.GetScenarioResourcesReader()

	file := "crd/bases/identityconfig.identitatem.io_strategies.yaml"
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", file); err != nil {
		return err
	}

	// b := retry.DefaultBackoff
	// b.Duration = 10 * time.Second
	// if err := clusteradmhelpers.WaitCRDToBeReady(r.APIExtensionClient, "strategies.identityconfig.identiatem.io", b); err != nil {
	// 	return err
	// }
	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := clusterv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := workv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := identitatemdexv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := ocinfrav1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemv1alpha1.Strategy{}).
		Owns(&clusterv1alpha1.Placement{}).
		Complete(r)
}
