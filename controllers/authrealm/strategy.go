// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AuthRealmReconciler) createStrategy(t identitatemv1alpha1.StrategyType, authRealm *identitatemv1alpha1.AuthRealm) (err error) {
	strategy := &identitatemv1alpha1.Strategy{}
	name := helpers.StrategyName(authRealm, t)
	if err = r.Client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: authRealm.Namespace}, strategy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		strategy := &identitatemv1alpha1.Strategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: authRealm.Namespace,
			},
			Spec: identitatemv1alpha1.StrategySpec{
				Type: t,
			},
		}

		if err = controllerutil.SetControllerReference(authRealm, strategy, r.Scheme); err == nil {
			if err = r.Client.Create(context.TODO(), strategy); err != nil {
				return err
			}
		}
		if err != nil {
			r.Log.Info("Update status create strategy failure",
				"type", identitatemv1alpha1.BackplaneStrategyType,
				"name", helpers.StrategyName(authRealm, t),
				"namespace", authRealm.Namespace,
				"error", err.Error())
			cond := metav1.Condition{
				Type:   identitatemv1alpha1.AuthRealmApplied,
				Status: metav1.ConditionFalse,
				Reason: "AuthRealmAppliedFailed",
				Message: fmt.Sprintf("failed to create strategy type: %s name: %s namespace: %s error: %s",
					identitatemv1alpha1.BackplaneStrategyType,
					helpers.StrategyName(authRealm, identitatemv1alpha1.BackplaneStrategyType),
					authRealm.Namespace,
					err.Error()),
			}
			if err := helpers.UpdateAuthRealmStatusConditions(r.Client, authRealm, cond); err != nil {
				return err
			}
		}
		return err
	}
	return nil
}
