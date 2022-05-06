// Copyright Red Hat

package authrealm

import (
	"context"

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

// +kubebuilder:rbac:groups="",resources={namespaces},verbs=get;create;delete;list;watch
// +kubebuilder:rbac:groups="",resources={secrets,serviceaccounts,configmaps},verbs=get;create;update;list;watch

// +kubebuilder:rbac:groups="apps",resources={deployments},verbs=get;create;update;delete

// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;create;update;delete

// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterroles},verbs=escalate;get;create;update;delete;bind;list
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterrolebindings},verbs=get;create;update;delete;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={roles},verbs=get;create;update;delete;escalate;bind
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={rolebindings},verbs=get;create;update;delete

// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms},verbs=get;update;watch
// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms/finalizers},verbs=create;delete;update;patch
// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms/status},verbs=update
// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={strategies},verbs=get;create;delete

// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexservers},verbs=get;create;update;watch;list;delete
// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexservers/status},verbs=update
// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexclients},verbs=get;create;update;watch;list;delete

// +kubebuilder:rbac:groups="coordination.k8s.io",resources={leases},verbs=get;list;create;update;patch;delete;watch
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
		if r, err := r.processAuthRealmDeletion(instance); err != nil || r.Requeue {
			return r, err
		}
		controllerutil.RemoveFinalizer(instance, helpers.AuthrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		return reconcile.Result{}, nil
	}

	if cond, err := r.processAuthRealmUpdate(instance); err != nil {
		if cond == nil {
			cond = &metav1.Condition{
				Type:    identitatemv1alpha1.AuthRealmApplied,
				Status:  metav1.ConditionFalse,
				Reason:  "AuthRealmAppliedFailed",
				Message: err.Error(),
			}
		}
		if err := r.updateAuthRealmStatusConditions(instance, *cond); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}
	cond := &metav1.Condition{
		Type:    identitatemv1alpha1.AuthRealmApplied,
		Status:  metav1.ConditionTrue,
		Reason:  "AuthRealmAppliedSucceeded",
		Message: "AuthRealm successfully applied",
	}
	if err := r.updateAuthRealmStatusConditions(instance, *cond); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *AuthRealmReconciler) processAuthRealmUpdate(authRealm *identitatemv1alpha1.AuthRealm) (*metav1.Condition, error) {
	//Add finalizer, it will be removed once the ns is deleted
	controllerutil.AddFinalizer(authRealm, helpers.AuthrealmFinalizer)

	r.Log.Info("Process", "Name", authRealm.GetName(), "Namespace", authRealm.GetNamespace())

	if err := r.Client.Update(context.TODO(), authRealm); err != nil {
		return nil, giterrors.WithStack(err)
	}

	//Synchronize Dex CR
	switch {
	case authRealm.Spec.Type == identitatemv1alpha1.AuthProxyDex ||
		authRealm.Spec.Type == "":
		if cond, err := r.syncDexCRs(authRealm); err != nil {
			return cond, err
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

	if cond, err := r.createStrategies(authRealm); err != nil {
		return cond, err
	}

	r.Log.Info("Update status Authrealm applied Succeeded",
		"name", authRealm.Name,
		"namespace", authRealm.Namespace)

	return nil, nil
}

func (r *AuthRealmReconciler) processAuthRealmDeletion(authRealm *identitatemv1alpha1.AuthRealm) (ctrl.Result, error) {
	if result, err := r.processDexServerDeletion(authRealm); err != nil || result.Requeue {
		return result, err
	}
	if result, err := r.deleteStrategies(authRealm); err != nil || result.Requeue {
		return result, err
	}
	r.Log.Info("delete DexOperator")
	return r.deleteDexOperator(authRealm)
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// //Install CRD
	// applierBuilder := &clusteradmapply.ApplierBuilder{}
	// applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	// readerIDPMgmtOperator := idpoperatorconfig.GetScenarioResourcesReader()

	// files := []string{"crd/bases/identityconfig.identitatem.io_authrealms.yaml",
	// 	"crd/bases/identityconfig.identitatem.io_strategies.yaml"}
	// if _, err := applier.ApplyDirectly(readerIDPMgmtOperator, nil, false, "", files...); err != nil {
	// 	return giterrors.WithStack(err)
	// }

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
				if dexServer.DeletionTimestamp.IsZero() && relatedObject.Kind == "AuthRealm" {
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
		Owns(&identitatemdexserverv1lapha1.DexServer{}).
		Complete(r)
}
