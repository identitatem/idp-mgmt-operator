/*
Copyright Contributors to the Open Cluster Management project
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	identitatemiov1 "github.com/identitatem/idp-mgmt-operator/api/v1"
)

// AuthRealmReconciler reconciles a AuthRealm object
type AuthRealmReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=identitatem.io.my.domain,resources=authrealms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=identitatem.io.my.domain,resources=authrealms/status,verbs=get;update;patch

func (r *AuthRealmReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("authrealm", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&identitatemiov1.AuthRealm{}).
		Complete(r)
}
