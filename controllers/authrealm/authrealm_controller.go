// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	giterrors "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/equality"
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

	identitatemdexserverv1lapha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	idpoperatorconfig "github.com/identitatem/idp-client-api/config"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

// AuthRealmReconciler reconciles a AuthRealm object
type AuthRealmReconciler struct {
	client.Client
	KubeClient         kubernetes.Interface
	DynamicClient      dynamic.Interface
	APIExtensionClient apiextensionsclient.Interface
	Log                logr.Logger
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,authrealms/status,strategies},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexservers,dexservers/status,dexclients,dexclients/status},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources={namespaces,secrets,serviceaccounts,configmaps},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources={deployments},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterrolebindings,rolebindings},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterroles,roles},verbs=get;list;watch;create;update;patch;delete;escalate;bind
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups="coordination.k8s.io",resources={leases},verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups="";events.k8s.io,resources=events,verbs=create;update;patch

func (r *AuthRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("namespace", req.NamespacedName, "name", req.Name)

	// your logic here
	// Fetch the ManagedCluster instance
	instance := &identitatemv1alpha1.AuthRealm{}

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

	//if deletetimestamp then delete dex namespace
	if instance.DeletionTimestamp != nil {
		if err := r.deleteAuthRealmNamespace(instance); err != nil {
			return reconcile.Result{}, err
		}
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return reconcile.Result{}, nil
	}

	//Add finalizer, it will be removed once the ns is deleted
	controllerutil.AddFinalizer(instance, helpers.AuthrealmFinalizer)

	r.Log.Info("Process", "Name", instance.GetName(), "Namespace", instance.GetNamespace())

	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	//Synchronize Dex CR
	switch {
	case instance.Spec.Type == identitatemv1alpha1.AuthProxyDex ||
		instance.Spec.Type == "":
		if err := r.syncDexCRs(instance); err != nil {
			return ctrl.Result{}, err
		}
		// case instance.Spec.Type == identitatemv1alpha1.AuthProxyRHSSO:
		// 	if err := r.syncRHSSOCRs(instance); err != nil {
		// 		return ctrl.Result{}, err
		// 	}
	}

	//Create GRC strategy
	// if err := r.createStrategy(identitatemv1alpha1.GrcStrategyType, instance); err != nil {
	// 	return ctrl.Result{}, err
	// }

	//Create Backplane strategy
	if err := r.createStrategy(identitatemv1alpha1.BackplaneStrategyType, instance); err != nil {
		r.Log.Info("Update status create strategy failure",
			"type", identitatemv1alpha1.BackplaneStrategyType,
			"name", helpers.StrategyName(instance, identitatemv1alpha1.BackplaneStrategyType),
			"namespace", instance.Namespace,
			"error", err.Error())
		cond := metav1.Condition{
			Type:   identitatemv1alpha1.AuthRealmApplied,
			Status: metav1.ConditionFalse,
			Reason: "AuthRealmAppliedFailed",
			Message: fmt.Sprintf("failed to create strategy type: %s name: %s namespace: %s error: %s",
				identitatemv1alpha1.BackplaneStrategyType,
				helpers.StrategyName(instance, identitatemv1alpha1.BackplaneStrategyType),
				instance.Namespace,
				err.Error()),
		}
		if err := helpers.UpdateAuthRealmStatusConditions(r.Client, instance, cond); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("Update status Authrealm applied Succeeded",
		"name", instance.Name,
		"namespace", instance.Namespace)
	cond := metav1.Condition{
		Type:    identitatemv1alpha1.AuthRealmApplied,
		Status:  metav1.ConditionTrue,
		Reason:  "AuthRealmAppliedSucceeded",
		Message: "AuthRealm successfully applied",
	}
	if err := helpers.UpdateAuthRealmStatusConditions(r.Client, instance, cond); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//Install CRD
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerIDPMgmtOperator := idpoperatorconfig.GetScenarioResourcesReader()

	files := []string{"crd/bases/identityconfig.identitatem.io_authrealms.yaml",
		"crd/bases/identityconfig.identitatem.io_strategies.yaml"}
	if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", files...); err != nil {
		return giterrors.WithStack(err)
	}

	if err := identitatemv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}
	if err := identitatemdexserverv1lapha1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}
	if err := appsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return giterrors.WithStack(err)
	}
	if err := r.installDexOperatorCRDs(); err != nil {
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
		For(&identitatemv1alpha1.AuthRealm{},
			builder.WithPredicates(authRealmPredicate),
		).
		Owns(&identitatemv1alpha1.Strategy{}).
		Watches(&source.Kind{Type: &identitatemdexserverv1lapha1.DexServer{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			dexServer := o.(*identitatemdexserverv1lapha1.DexServer)
			req := make([]reconcile.Request, 0)
			for _, relatedObject := range dexServer.Status.RelatedObjects {
				if relatedObject.Kind == "AuthRealm" {
					req = append(req, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      relatedObject.Name,
							Namespace: relatedObject.Namespace,
						},
					})
				}
			}
			return req
		})).
		//TODO change to watch with mapping
		// Owns(&identitatemdexserverv1lapha1.DexServer{}).
		Complete(r)
}
