// Copyright Contributors to the Open Cluster Management project

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	identitatemdexserverv1lapha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemmgmtv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	identitatemstrategyv1alpha1 "github.com/identitatem/idp-strategy-operator/api/identitatem/v1alpha1"
)

const (
	authrealmFinalizer string = "authrealm.identitatem.io/cleanup"
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

// +kubebuilder:rbac:groups=identityconfig.identitatem.io,resources={authrealms,strategies},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.identitatem.io,resources={dexservers,dexclients},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources={namespaces,secrets,serviceaccounts},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources={deployments},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterrolebindings},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources={clusterroles},verbs=get;list;watch;create;update;patch;delete;escalate;bind
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources={customresourcedefinitions},verbs=get;list;create;update;patch;delete

func (r *AuthRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("authrealm", req.NamespacedName)

	// your logic here
	// Fetch the ManagedCluster instance
	instance := &identitatemmgmtv1alpha1.AuthRealm{}

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

	//if deletetimestamp then delete dex namespace
	if instance.DeletionTimestamp != nil {
		if err := r.deleteAuthRealmNamespace(instance); err != nil {
			return reconcile.Result{}, err
		}
		controllerutil.RemoveFinalizer(instance, authrealmFinalizer)
		if err := r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	//Add finalizer, it will be removed once the ns is deleted
	controllerutil.AddFinalizer(instance, authrealmFinalizer)

	r.Log.Info("Process", "Name", instance.GetName(), "Namespace", instance.GetNamespace())

	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, err
	}

	//Synchronize Dex CR
	switch {
	case instance.Spec.Type == identitatemmgmtv1alpha1.AuthProxyDex ||
		instance.Spec.Type == "":
		if err := r.syncDexCRs(instance); err != nil {
			return ctrl.Result{}, err
		}
	case instance.Spec.Type == identitatemmgmtv1alpha1.AuthProxyRHSSO:
		if err := r.syncRHSSOCRs(instance); err != nil {
			return ctrl.Result{}, err
		}
	default:
	}

	//Create GRC strategy
	if err := r.createStrategy(identitatemstrategyv1alpha1.GrcStrategyType, instance); err != nil {
		return ctrl.Result{}, err
	}

	//Create Backplane strategy
	if err := r.createStrategy(identitatemstrategyv1alpha1.BackplaneStrategyType, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := identitatemstrategyv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := identitatemdexserverv1lapha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := appsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemmgmtv1alpha1.AuthRealm{}).
		Owns(&identitatemstrategyv1alpha1.Strategy{}).
		Owns(&identitatemdexserverv1lapha1.DexServer{}).
		Watches(&source.Kind{Type: &identitatemdexserverv1lapha1.DexClient{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      o.GetName(),
						Namespace: o.GetNamespace(),
					},
				},
			}
		})).
		Complete(r)
}
