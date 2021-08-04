// Copyright Contributors to the Open Cluster Management project

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	identitatemv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
)

// AuthRealmReconciler reconciles a AuthRealm object
type AuthRealmReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=identitatem.io,resources={authrealms,identityproviders},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=identitatem.io,resources={authrealms/status,identityproviders/status},verbs=get;update;patch

func (r *AuthRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("authrealm", req.NamespacedName)

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
		return reconcile.Result{}, err
	}

	switch instance.Spec.MappingMethod {
	case "":
		instance.Spec.MappingMethod = openshiftconfigv1.MappingMethodClaim
	case openshiftconfigv1.MappingMethodClaim:
		instance.Spec.MappingMethod = openshiftconfigv1.MappingMethodAdd
	}

	if err := r.Client.Update(context.TODO(), instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemv1alpha1.AuthRealm{}).
		Watches(
			&source.Kind{Type: &identitatemv1alpha1.IdentityProvider{}},
			&handler.EnqueueRequestForOwner{IsController: true, OwnerType: &identitatemv1alpha1.AuthRealm{}},
		).
		Complete(r)
}
