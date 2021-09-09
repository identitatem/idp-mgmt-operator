// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"

	ocinfrav1 "github.com/openshift/api/config/v1"
	// corev1 "k8s.io/api/core/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
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

const (
	placementDecisionFinalizer          string = "placelementdecision.identitatem.io/cleanup"
	placementDecisionBackplaneFinalizer string = "placelementdecision.identitatem.io/cleanup-backplane"
)

// +kubebuilder:rbac:groups="",resources={namespaces,secrets},verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,strategies},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=identityconfig.identitatem.io,resources=strategies/finalizers,verbs=update
//+kubebuilder:rbac:groups=auth.identitatem.io,resources={dexclients},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;list;create;update;patch;delete

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources={managedclusters,placements,placementdecisions},verbs=get;list;watch;create;update;patch;delete;watch
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources={manifestworks},verbs=get;list;watch;create;update;patch;delete;watch
//+kubebuilder:rbac:groups=config.openshift.io,resources={infrastructures},verbs=get;list;watch;create;update;patch;delete;watch

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
	instance := &clusterv1alpha1.PlacementDecision{}

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

	r.Log.Info("running Reconcile for PlacementDecision")

	strategy, err := GetStrategyFromPlacementDecision(r.Client, instance)
	if err != nil {
		if !errors.IsNotFound(err) {
			r.Log.Error(err, "Error while getting the strategy")
			return reconcile.Result{}, err
		}
		r.Log.Info("PlacementDecision not linked to a strategy", "Error:", err)
		//No further processing
		return reconcile.Result{}, nil
	}

	//if deletetimestamp then delete dex namespace
	if instance.DeletionTimestamp != nil {
		if err := r.deletePlacementDecision(instance); err != nil {
			return reconcile.Result{}, err
		}
		r.Log.Info("remove PlacementDecision finalizer", "Finalizer:", placementDecisionFinalizer)
		controllerutil.RemoveFinalizer(instance, placementDecisionFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	//Add finalizer
	r.Log.Info("add PlacementDecision finalizer", "Finalizer:", placementDecisionFinalizer)
	controllerutil.AddFinalizer(instance, placementDecisionFinalizer)
	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}

	//Search the placement corresponding to the placementDecision
	r.Log.Info("search Placement", " Namespace:", instance.GetNamespace(), "Finalizer:", placementDecisionFinalizer, "Name: ", instance.GetLabels()[clusterv1alpha1.PlacementLabel])
	placement := &clusterv1alpha1.Placement{}
	err = r.Get(context.TODO(),
		client.ObjectKey{
			Name:      instance.GetLabels()[clusterv1alpha1.PlacementLabel],
			Namespace: instance.GetNamespace(),
		}, placement)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Add finalizer to the placement, it will be removed once the ns is deleted
	r.Log.Info("add PlacementDecision finalizer on placement", " Namespace:", placement.GetNamespace(), "Name: ", placement.GetName(), "Finalizer:", placementDecisionBackplaneFinalizer)
	if err := r.AddPlacementDecisionFinalizer(strategy, placement); err != nil {
		return reconcile.Result{}, err
	}

	//Add finalizer to the strategy, it will be removed once the ns is deleted
	r.Log.Info("add PlacementDecision finalizer on strategy", " Namespace:", strategy.GetNamespace(), "Name: ", strategy.GetName(), "Finalizer:", placementDecisionBackplaneFinalizer)
	if err := r.AddPlacementDecisionFinalizer(strategy, strategy); err != nil {
		return reconcile.Result{}, err
	}

	authrealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Add finalizer to the authrealm, it will be removed once the ns is deleted
	r.Log.Info("add PlacementDecision finalizer on authrealm", " Namespace:", authrealm.GetNamespace(), "Name: ", authrealm.GetName(), "Finalizer:", placementDecisionBackplaneFinalizer)
	if err := r.AddPlacementDecisionFinalizer(strategy, authrealm); err != nil {
		return reconcile.Result{}, err
	}

	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		//check if dex server installed
		// r.Log.Info("check if dex server namespace exists", "Namespace:", helpers.DexServerNamespace(authrealm))
		// ns := &corev1.Namespace{}
		// if err := r.Get(context.TODO(), client.ObjectKey{Name: helpers.DexServerNamespace(authrealm)}, ns); err != nil {
		// 	return reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
		// }

		if err := r.backplaneStrategy(authrealm, instance); err != nil {
			return reconcile.Result{}, err
		}
	// case identitatemv1alpha1.GrcStrategyType:
	// 	if err := r.grcStrategy(placement, instance); err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	default:
		return reconcile.Result{}, fmt.Errorf("strategy type %s not supported", strategy.Spec.Type)
	}

	return ctrl.Result{}, nil
}

func (r *PlacementDecisionReconciler) AddPlacementDecisionFinalizer(strategy *identitatemv1alpha1.Strategy, obj client.Object) error {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		controllerutil.AddFinalizer(obj, placementDecisionBackplaneFinalizer)
		// case identitatemv1alpha1.GrcStrategyType:
		// controllerutil.AddFinalizer(obj, placementDecisionGRCFinalizer)
	default:
		return fmt.Errorf("strategy type %s not supported", strategy.Spec.Type)
	}

	return r.Client.Update(context.TODO(), obj)

}

func (r *PlacementDecisionReconciler) RemovePlacementDecisionFinalizer(strategy *identitatemv1alpha1.Strategy, obj client.Object) error {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		controllerutil.RemoveFinalizer(obj, placementDecisionBackplaneFinalizer)
		// case identitatemv1alpha1.GrcStrategyType:
		// controllerutil.RemoveFinalizer(obj, placementDecisionGRCFinalizer)
	default:
		return fmt.Errorf("strategy type %s not supported", strategy.Spec.Type)
	}

	return r.Client.Update(context.TODO(), obj)

}

func (r *PlacementDecisionReconciler) deletePlacementDecision(placementDecision *clusterv1alpha1.PlacementDecision) error {
	r.Log.Info("delete PlacementDecision")
	strategy, err := GetStrategyFromPlacementDecision(r.Client, placementDecision)
	if err != nil {
		r.Log.Error(err, "Error while getting the strategy")
		return err
	}

	authrealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return err
	}

	for _, decision := range placementDecision.Status.Decisions {
		for _, idp := range authrealm.Spec.IdentityProviders {
			//Delete DexClient
			dexClientName := helpers.DexClientName(decision, idp)
			r.Log.Info("delete dexclient", "namespace", authrealm.Name, "name", dexClientName)
			dexClient := &dexoperatorv1alpha1.DexClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dexClientName,
					Namespace: authrealm.Name,
				},
			}
			if err := r.Delete(context.TODO(), dexClient); err != nil && !errors.IsNotFound(err) {
				return err
			}
			//Delete ClientSecret
			r.Log.Info("delete clientSecret", "namespace", decision.ClusterName, "name", idp.Name)
			clientSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      idp.Name,
					Namespace: decision.ClusterName,
				},
			}
			if err := r.Delete(context.TODO(), clientSecret); err != nil && !errors.IsNotFound(err) {
				return err
			}
			//Delete clusterOAuth
			r.Log.Info("delete clusterOAuth", "Namespace", dexClient.GetLabels()["cluster"], "Name", idp.Name)
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{
				ObjectMeta: metav1.ObjectMeta{
					Name:      idp.Name,
					Namespace: decision.ClusterName,
				},
			}
			if err := r.Client.Delete(context.TODO(), clusterOAuth); err != nil && !errors.IsNotFound(err) {
				return err
			}
			//Delete Manifestwork
			manifestworkName := helpers.ManifestWorkName()
			r.Log.Info("delete manifestwork", "namespace", decision.ClusterName, "name", manifestworkName)
			manifestwork := &workv1.ManifestWork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      manifestworkName,
					Namespace: decision.ClusterName,
				},
			}
			if err := r.Delete(context.TODO(), manifestwork); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel],
	}, client.InNamespace(placementDecision.Namespace)); err != nil {
		return err
	}

	//Remove the finalizers when there is no other placementDecisions for that placement.
	if len(placementDecisions.Items) == 1 {
		//Search the placement corresponding to the placementDecision
		placement := &clusterv1alpha1.Placement{}
		err := r.Get(context.TODO(),
			client.ObjectKey{
				Name:      placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel],
				Namespace: placementDecision.GetNamespace(),
			}, placement)
		if err != nil {
			return err
		}

		//All resources are cleaned, finalizers on authrealm and strategy can be removed
		if err := r.RemovePlacementDecisionFinalizer(strategy, placement); err != nil {
			return err
		}
		if err := r.RemovePlacementDecisionFinalizer(strategy, strategy); err != nil {
			return err
		}
		if err := r.RemovePlacementDecisionFinalizer(strategy, authrealm); err != nil {
			return err
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.PlacementDecision{}).
		//TODO Watch clientSecret to regenerate dexclient/clusterOAuth
		Complete(r)
}
