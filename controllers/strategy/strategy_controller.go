// Copyright Red Hat

package strategy

import (
	"context"
	"fmt"
	"time"

	// "time"

	ocinfrav1 "github.com/openshift/api/config/v1"
	giterrors "github.com/pkg/errors"
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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	// identitatemdexserverv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	idpconfig "github.com/identitatem/idp-client-api/config"

	//+kubebuilder:scaffold:imports

	// clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
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

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={strategies},verbs=get;list;watch;update
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={strategies/finalizers},verbs=create;delete;patch;update
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={strategies/status},verbs=patch

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placements},verbs=get;list;watch;create;update;delete

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
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

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

	r.Log.Info("Running Reconcile for Strategy.", "Name: ", instance.GetName(), " Namespace:", instance.GetNamespace())

	if instance.DeletionTimestamp != nil {
		if result, err := r.processStrategyDeletion(instance); err != nil || result.Requeue {
			return result, err
		}
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return reconcile.Result{}, nil
	}

	if err := r.processStrategyUpdate(instance); err != nil {
		cond := metav1.Condition{
			Type:    identitatemv1alpha1.StrategyApplied,
			Status:  metav1.ConditionFalse,
			Reason:  "StrategyAppliedFailed",
			Message: fmt.Sprintf("Strategy apply failed: %s", err.Error()),
		}
		if err := r.UpdateStrategyStatusConditions(instance, cond); err != nil {
			return ctrl.Result{}, err
		}

		return reconcile.Result{}, err
	}

	cond := metav1.Condition{
		Type:    identitatemv1alpha1.StrategyApplied,
		Status:  metav1.ConditionTrue,
		Reason:  "StrategyAppliedSucceeded",
		Message: "Strategy successfully applied",
	}

	if err := r.UpdateStrategyStatusConditions(instance, cond); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *StrategyReconciler) processStrategyUpdate(strategy *identitatemv1alpha1.Strategy) error {
	controllerutil.AddFinalizer(strategy, helpers.AuthrealmFinalizer)

	r.Log.Info("Process", "Name", strategy.GetName(), "Namespace", strategy.GetNamespace())

	if err := r.Client.Update(context.TODO(), strategy); err != nil {
		return giterrors.WithStack(err)
	}
	// Get the AuthRealm Placement bits we need to help create a new Placement

	r.Log.Info("Searching for AuthRealm in ownerRefs", "strategy", strategy.Name)
	authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return err
	}

	if authRealm.DeletionTimestamp != nil {
		return nil
	}

	placement := &clusterv1alpha1.Placement{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authRealm.Spec.PlacementRef.Name, Namespace: strategy.Namespace}, placement); err != nil {
		if errors.IsNotFound(err) {
			return r.deleteStrategyPlacement(authRealm.Spec.PlacementRef.Name, strategy.Namespace)
		}
		return err
	}

	//Get placementStrategy
	placementStrategy, placementStrategyExists, err := r.getStrategyPlacement(strategy, authRealm, placement)
	if err != nil {
		return err
	}

	//Enrich placementStrategy
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		r.backplanePlacementStrategy(strategy, authRealm, placement, placementStrategy)
	case identitatemv1alpha1.HypershiftStrategyType:
		r.hypershiftPlacementStrategy(strategy, authRealm, placement, placementStrategy)
	// case identitatemv1alpha1.GrcStrategyType:
	// 	if err := r.grcPlacementStrategy(instance, authRealm, placement, placementStrategy); err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	default:
		return fmt.Errorf("strategy type %s not supported", strategy.Spec.Type)
	}

	// update the Placement ref
	strategy.Spec.PlacementRef.Name = placementStrategy.Name
	if err := r.Client.Update(context.TODO(), strategy); err != nil {
		return err
	}

	//Create or update placementStrategy
	switch placementStrategyExists {
	case true:
		if err := r.Client.Update(context.TODO(), placementStrategy); err != nil {
			return err
		}
	case false:
		if err := r.Client.Create(context.TODO(), placementStrategy); err != nil {
			return err
		}
	}

	r.Log.Info("placementstrategy after update", "placementStrategy", placementStrategy)

	return nil
}

//deleteStrategyPlacement deletes strategy placement related to a placement
func (r *StrategyReconciler) deleteStrategyPlacement(name, namespace string) error {
	//Could be a user placement referenced in one or more strategy placement and so we have to delete the strategy placement
	strategyPlacements := &clusterv1alpha1.PlacementList{}
	err := r.Client.List(context.TODO(), strategyPlacements,
		client.InNamespace(namespace),
		client.MatchingLabels{
			helpers.PlacementStrategyLabel: name,
		})
	if err != nil {
		return err
	}
	for _, sp := range strategyPlacements.Items {
		err := r.Client.Delete(context.TODO(), sp.DeepCopy(), &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *StrategyReconciler) getStrategyPlacement(strategy *identitatemv1alpha1.Strategy,
	authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) (*clusterv1alpha1.Placement, bool, error) {
	placementStrategy := &clusterv1alpha1.Placement{}
	placementStrategyExists := true
	placementStrategyName := helpers.PlacementStrategyName(strategy, authRealm)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: placementStrategyName, Namespace: strategy.Namespace}, placementStrategy); err != nil {
		if !errors.IsNotFound(err) {
			return nil, false, err
		}
		placementStrategyExists = false
		// Not Found! Create
		placementStrategy = &clusterv1alpha1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					helpers.PlacementStrategyLabel: placement.Name,
				},
				Namespace: strategy.Namespace,
				Name:      placementStrategyName,
			},
			Spec: placement.Spec,
		}
	}
	// Set owner reference for cleanup
	err := controllerutil.SetOwnerReference(strategy, placementStrategy, r.Scheme)
	if err != nil {
		return nil, placementStrategyExists, err
	}
	return placementStrategy, placementStrategyExists, nil
}

func (r *StrategyReconciler) processStrategyDeletion(strategy *identitatemv1alpha1.Strategy) (ctrl.Result, error) {
	//Delete strategyPlacement
	r.Log.Info("process strategy deletion:", "name", strategy.Name, "namespace", strategy.Namespace)
	authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	pl := &clusterv1alpha1.Placement{}
	err = r.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.PlacementStrategyName(strategy, authRealm), Namespace: strategy.Namespace},
		pl)
	switch {
	case err == nil:
		if isLastOwnerReference(pl, strategy) {
			if err := r.Client.Delete(context.TODO(), pl); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			r.Log.Info("waiting strategy placement to be deleted",
				"name", helpers.PlacementStrategyName(strategy, authRealm),
				"namespace", strategy.Namespace)
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
		}
		removeOwnerReference(pl, strategy)
		if err := r.Client.Update(context.TODO(), pl); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	case errors.IsNotFound(err):
		r.Log.Info("nothing to do as placement is already deleted",
			"name", helpers.PlacementStrategyName(strategy, authRealm),
			"namespace", strategy.Namespace)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, err
}

func isLastOwnerReference(pl *clusterv1alpha1.Placement, strategy *identitatemv1alpha1.Strategy) bool {
	refs := pl.GetOwnerReferences()
	return len(refs) <= 1 && refs[0].UID == strategy.UID
}

func removeOwnerReference(pl *clusterv1alpha1.Placement, strategy *identitatemv1alpha1.Strategy) []metav1.OwnerReference {
	refs := pl.GetOwnerReferences()
	index := -1
	for i, ref := range refs {
		if ref.UID == strategy.UID {
			index = i
		}
	}
	if index != -1 {
		refs[index] = refs[len(refs)-1]
		return refs[:len(refs)-1]
	}
	return refs
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
		//Watch placement and reconcile the strategy link to the placement
		Watches(&source.Kind{Type: &clusterv1alpha1.Placement{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			pl := o.(*clusterv1alpha1.Placement)
			req := make([]reconcile.Request, 0)
			strategies := &identitatemv1alpha1.StrategyList{}
			if err := r.List(context.TODO(), strategies, client.InNamespace(pl.Namespace)); err != nil {
				r.Log.Error(err, "Error while getting list")
			}
			for _, strategy := range strategies.Items {
				r.Log.Info("Check if need to be reconcile",
					"placementRefName", strategy.Spec.PlacementRef.Name,
					"helpers.PlacementStrategyNameFromPlacementRefName(string(strategy.Spec.Type), pl.Name)", helpers.PlacementStrategyNameFromPlacementRefName(string(strategy.Spec.Type), pl.Name))
				if strategy.Spec.PlacementRef.Name == helpers.PlacementStrategyNameFromPlacementRefName(string(strategy.Spec.Type), pl.Name) {
					r.Log.Info("Add strategy to reconcile",
						"name", strategy.Name,
						"namespace", strategy.Namespace)
					req = append(req, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      strategy.Name,
							Namespace: strategy.Namespace,
						},
					})
				}
			}
			return req
		})).
		Complete(r)
}
