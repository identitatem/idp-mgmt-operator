// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"
	"time"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *AuthRealmReconciler) createStrategies(authRealm *identitatemv1alpha1.AuthRealm) (*metav1.Condition, error) {
	//Create Backplane strategy
	if cond, err := r.createStrategy(authRealm, identitatemv1alpha1.BackplaneStrategyType); err != nil {
		return cond, err
	}
	//Create Hypershift strategy
	if cond, err := r.createStrategy(authRealm, identitatemv1alpha1.HypershiftStrategyType); err != nil {
		return cond, err
	}
	return nil, nil

}

func (r *AuthRealmReconciler) createStrategy(authRealm *identitatemv1alpha1.AuthRealm,
	t identitatemv1alpha1.StrategyType) (*metav1.Condition, error) {
	if err := helpers.CreateStrategy(r.Client, r.Scheme, t, authRealm); err != nil {
		r.Log.Info("Update status create strategy failure",
			"type", t,
			"name", helpers.StrategyName(authRealm, t),
			"namespace", authRealm.Namespace,
			"error", err.Error())
		cond := &metav1.Condition{
			Type:   identitatemv1alpha1.AuthRealmApplied,
			Status: metav1.ConditionFalse,
			Reason: "AuthRealmAppliedFailed",
			Message: fmt.Sprintf("failed to create strategy type: %s name: %s namespace: %s error: %s",
				t,
				helpers.StrategyName(authRealm, t),
				authRealm.Namespace,
				err.Error()),
		}
		return cond, err
	}
	return nil, nil
}

func (r *AuthRealmReconciler) deleteStrategies(authRealm *identitatemv1alpha1.AuthRealm) (ctrl.Result, error) {
	if result, err := r.deleteStrategy(authRealm,
		identitatemv1alpha1.BackplaneStrategyType); err != nil || result.Requeue {
		return result, err
	}
	if result, err := r.deleteStrategy(authRealm,
		identitatemv1alpha1.HypershiftStrategyType); err != nil || result.Requeue {
		return result, err
	}
	return ctrl.Result{}, nil
}

func (r *AuthRealmReconciler) deleteStrategy(authRealm *identitatemv1alpha1.AuthRealm,
	t identitatemv1alpha1.StrategyType) (ctrl.Result, error) {
	st := &identitatemv1alpha1.Strategy{}
	err := r.Client.Get(context.TODO(),
		client.ObjectKey{
			Name:      helpers.StrategyName(authRealm, t),
			Namespace: authRealm.Namespace},
		st)
	switch {
	case err == nil:
		r.Log.Info("delete Strategy", "name", helpers.StrategyName(authRealm, t))
		if err := r.Client.Delete(context.TODO(), st); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		r.Log.Info("waiting strategy to be deleted",
			"name", helpers.StrategyName(authRealm, t),
			"namespace", authRealm.Namespace)
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	case !errors.IsNotFound(err):
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	return ctrl.Result{}, nil
}
