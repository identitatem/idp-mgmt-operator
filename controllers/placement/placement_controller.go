// Copyright Red Hat

package placement

import (
	"context"
	"fmt"

	giterrors "github.com/pkg/errors"

	ocinfrav1 "github.com/openshift/api/config/v1"

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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpoperatorconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
	//+kubebuilder:scaffold:imports
)

// PlacementReconciler reconciles a Strategy object
type PlacementReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources={secrets},verbs=get;create;delete

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={clusteroauths},verbs=get;create;update;delete
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={strategies},verbs=list

// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexclients,dexclients/status},verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placementdecisions},verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placements},verbs=watch;list
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={placementdecisions/finalizer},verbs=create;delete;patch;update
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={managedclusters},verbs=get;list;watch

//+kubebuilder:rbac:groups=config.openshift.io,resources={infrastructures},verbs=get;watch;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Strategy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

	// your logic here
	// Fetch the ManagedCluster instance
	instance := &clusterv1alpha1.Placement{}

	if err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: req.Namespace, Name: req.Name},
		instance,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	r.Log.Info("running Reconcile for Placement", "name", instance.Name, "namespace", instance.Namespace)

	//if deletetimestamp then delete dex namespace
	if instance.DeletionTimestamp != nil {
		if result, err := r.processPlacementDeletion(instance, false); err != nil || result.Requeue {
			return result, err
		}
		r.Log.Info("remove finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", instance.Name, "namespace", instance.Namespace)
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return ctrl.Result{}, nil
	}

	if result, err := r.processPlacementUpdate(instance); err != nil || result.Requeue {
		return result, err
	}
	return ctrl.Result{}, nil

}

// Check if a placement is linked to a strategy
func isLinkedToStrategy(obj client.Object) bool {
	labels := obj.GetLabels()
	if len(labels) == 0 {
		return false
	}
	_, ok := labels[helpers.PlacementStrategyLabel]
	return ok
}

func (r *PlacementReconciler) processPlacementUpdate(placement *clusterv1alpha1.Placement) (ctrl.Result, error) {

	//Add finalizer
	r.Log.Info("add finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", placement.Name, "namespace", placement.Namespace)
	controllerutil.AddFinalizer(placement, helpers.AuthrealmFinalizer)
	if err := r.Client.Update(context.TODO(), placement); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	strategies, err := r.GetStrategyFromPlacement(placement)
	if err != nil {
		return ctrl.Result{}, err
	}

	for i := range strategies.Items {

		authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, &strategies.Items[i])
		if err != nil {
			return ctrl.Result{}, err
		}

		if result, err := r.processPlacement(authRealm, &strategies.Items[i], placement); err != nil || result.Requeue {
			return result, err
		}
		if err := r.updateAuthRealmStatusPlacementStatus(&strategies.Items[i], placement); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

//processPlacement generates resources for the Backplane strategy
func (r *PlacementReconciler) processPlacement(
	authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {
	r.Log.Info("run", "strategy", strategy.Name)
	if result, err := r.syncDexClients(authRealm, strategy, placement); err != nil || result.Requeue {
		return result, err
	}
	return ctrl.Result{}, nil
}

func (r *PlacementReconciler) processPlacementDeletion(placement *clusterv1alpha1.Placement,
	onlyIfAuthRealmDeleted bool) (ctrl.Result, error) {

	r.Log.Info("start deletion of Placement",
		"namespace", placement.Namespace,
		"name", placement.Name)

	r.Log.Info("search placementdecisions in ns and label",
		"namespace", placement.Namespace,
		"label", fmt.Sprintf("%s=%s", clusterv1alpha1.PlacementLabel, placement.Name))
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placement.Name,
	}, client.InNamespace(placement.Namespace)); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			if result, err := r.deleteClusterOAuthConfig(decision.ClusterName, placement); err != nil || result.Requeue {
				return result, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlacementReconciler) SetupWithManager(mgr ctrl.Manager) error {

	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpoperatorconfig.GetScenarioResourcesReader()

	files := []string{"crd/bases/identityconfig.identitatem.io_clusteroauths.yaml"}
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", files...); err != nil {
		return err
	}

	if err := dexoperatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
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

	if err := workv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := ocinfrav1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := hypershiftv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	authRealmPredicate := predicate.Predicate(predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			authRealmOld := e.ObjectOld.(*identitatemv1alpha1.AuthRealm)
			authRealmNew := e.ObjectNew.(*identitatemv1alpha1.AuthRealm)
			// only handle the Finalizer and Spec changes
			return !equality.Semantic.DeepEqual(e.ObjectOld.GetFinalizers(), e.ObjectNew.GetFinalizers()) ||
				!equality.Semantic.DeepEqual(authRealmOld.Spec, authRealmNew.Spec) ||
				authRealmOld.DeletionTimestamp != authRealmNew.DeletionTimestamp

		},
	})

	placementPredicate := predicate.Predicate(predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool { return false },
		DeleteFunc: func(e event.DeleteEvent) bool {
			// only handle the IDP generated placements
			process := isLinkedToStrategy(e.Object)
			r.Log.Info("placement event delete", "process", process)
			return process
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// only handle the IDP generated placements
			process := isLinkedToStrategy(e.Object)
			r.Log.Info("placement event create", "process", process)
			return process
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			placementOld := e.ObjectOld.(*clusterv1alpha1.Placement)
			placementNew := e.ObjectNew.(*clusterv1alpha1.Placement)
			process := isLinkedToStrategy(placementNew) || isLinkedToStrategy(placementOld)
			r.Log.Info("placement event delete", "process", process)
			return process
			// only handle the IDP generated placements
		},
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Placement{}, builder.WithPredicates(placementPredicate)).
		Watches(&source.Kind{Type: &identitatemv1alpha1.AuthRealm{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				authrealm := o.(*identitatemv1alpha1.AuthRealm)
				req := make([]reconcile.Request, 0)
				strategies := &identitatemv1alpha1.StrategyList{}
				if err := r.Client.List(context.TODO(),
					strategies,
					&client.ListOptions{Namespace: authrealm.Namespace}); err == nil {
					for _, strategy := range strategies.Items {
						placement := &clusterv1alpha1.Placement{}
						if err := r.Client.Get(context.TODO(),
							client.ObjectKey{
								Name:      strategy.Spec.PlacementRef.Name,
								Namespace: authrealm.Namespace},
							placement); err != nil {
							continue
						}
						r.Log.Info(fmt.Sprintf("Reconcile placement %s/%s because authrealm %s/%s changed",
							placement.Name,
							placement.Namespace,
							authrealm.Name,
							authrealm.Namespace))
						req = append(req, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      placement.Name,
								Namespace: authrealm.Namespace,
							},
						})
					}
				}
				return req
			}),
			builder.WithPredicates(authRealmPredicate)).
		Watches(&source.Kind{Type: &dexoperatorv1alpha1.DexClient{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				dexClient := o.(*dexoperatorv1alpha1.DexClient)
				req := make([]reconcile.Request, 0)
				for _, relatedObject := range dexClient.Status.RelatedObjects {
					if relatedObject.Kind == "PlacementDecision" {
						placementDecision := &clusterv1alpha1.PlacementDecision{}
						if err := r.Client.Get(context.TODO(),
							client.ObjectKey{Name: relatedObject.Name, Namespace: relatedObject.Namespace},
							placementDecision); err == nil {
							r.Log.Info(fmt.Sprintf("Reconcile placement %s/%s because dexclient %s/%s changed",
								placementDecision.GetLabels()["cluster.open-cluster-management.io/placement"],
								placementDecision.Namespace,
								dexClient.Name,
								dexClient.Namespace))
							req = append(req, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      placementDecision.GetLabels()["cluster.open-cluster-management.io/placement"],
									Namespace: placementDecision.Namespace,
								},
							})
						}
					}
				}
				return req
			})).Complete(r)
}
