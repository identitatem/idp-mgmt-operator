// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"

	giterrors "github.com/pkg/errors"

	ocinfrav1 "github.com/openshift/api/config/v1"
	// corev1 "k8s.io/api/core/v1"

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

	//identitatemdexserverv1lapha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpoperatorconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	//+kubebuilder:scaffold:imports
)

// PlacementDecisionReconciler reconciles a Strategy object
type PlacementDecisionReconciler struct {
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
func (r *PlacementDecisionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, giterrors.WithStack(err)
	}

	r.Log.Info("running Reconcile for Placement")

	//if deletetimestamp then delete dex namespace
	if instance.DeletionTimestamp != nil {
		if err := r.processPlacementDecisionDeletion(instance, false); err != nil {
			return reconcile.Result{}, err
		}
		r.Log.Info("remove finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", instance.Name, "namespace", instance.Namespace)
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return reconcile.Result{}, nil
	}

	if err := r.processPlacementDecisionUpdate(instance); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil

}

func (r *PlacementDecisionReconciler) processPlacementDecisionUpdate(placement *clusterv1alpha1.Placement) error {
	//Add finalizer
	r.Log.Info("add finalizer", "Finalizer:", helpers.AuthrealmFinalizer, "name", placement.Name, "namespace", placement.Namespace)
	controllerutil.AddFinalizer(placement, helpers.AuthrealmFinalizer)
	if err := r.Client.Update(context.TODO(), placement); err != nil {
		return giterrors.WithStack(err)
	}

	//Check if the placementDecision is linked to a strategy
	ok, err := r.isLinkedToStrategy(placement)
	if !ok {
		return nil
	}
	if err != nil {
		return err
	}

	strategies, err := r.GetStrategiesFromPlacement(placement)
	if err != nil {
		return err
	}

	for i, strategy := range strategies.Items {
		authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, &strategies.Items[i])
		if err != nil {
			return err
		}

		if err := r.processPlacementDecision(authRealm, &strategy, placement); err != nil {
			return err
		}
		if err := r.updateAuthRealmStatusPlacementStatus(&strategies.Items[i], placement); err != nil {
			return err
		}
	}
	if err := r.processPlacementDecisionDeletion(placement, true); err != nil {
		return err
	}
	return nil
}

//processPlacementDecision generates resources for the Backplane strategy
func (r *PlacementDecisionReconciler) processPlacementDecision(
	authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) error {
	r.Log.Info("run backplane strategy")
	if err := r.syncDexClients(authRealm, strategy, placement); err != nil {
		return err
	}
	return nil
}

func (r *PlacementDecisionReconciler) isLinkedToStrategy(placement *clusterv1alpha1.Placement) (bool, error) {
	_, err := r.GetStrategiesFromPlacement(placement)
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, nil
		}
		r.Log.Info("PlacementDecision not linked to a strategy", "Error:", err)
		//No further processing
		return false, nil
	}
	return true, giterrors.WithStack(err)
}

func (r *PlacementDecisionReconciler) processPlacementDecisionDeletion(placement *clusterv1alpha1.Placement, onlyIfAuthRealmDeleted bool) error {
	ok, err := r.isLinkedToStrategy(placement)
	if !ok {
		return nil
	}
	if err != nil {
		return err
	}

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
		return giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			if err := r.deleteConfig(decision.ClusterName, onlyIfAuthRealmDeleted); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlacementDecisionReconciler) SetupWithManager(mgr ctrl.Manager) error {

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

	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Placement{}).
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
			})).
		Complete(r)
}
