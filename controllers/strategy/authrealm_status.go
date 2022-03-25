// Copyright Red Hat

package strategy

import (
	"context"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *StrategyReconciler) UpdateStrategyStatusConditions(
	strategy *identitatemv1alpha1.Strategy, newConditions ...metav1.Condition) error {
	patch := client.MergeFrom(strategy.DeepCopy())
	strategy.Status.Conditions = helpers.MergeStatusConditions(strategy.Status.Conditions, newConditions...)
	if err := giterrors.WithStack(r.Client.Status().Patch(context.TODO(), strategy, patch)); err != nil {
		return err
	}
	authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return err
	}
	return r.UpdateAuthRealmStatusStrategyConditions(authRealm, strategy)
}

func (r *StrategyReconciler) UpdateAuthRealmStatusStrategyConditions(
	authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy) error {
	patch := client.MergeFrom(authRealm.DeepCopy())
	strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, strategy.Name)
	if strategyIndex == -1 {
		strategyIndex = len(authRealm.Status.Strategies)
		authRealm.Status.Strategies = append(authRealm.Status.Strategies, identitatemv1alpha1.AuthRealmStrategyStatus{
			Name: strategy.Name,
			StrategyStatus: identitatemv1alpha1.StrategyStatus{
				Conditions: make([]metav1.Condition, 0),
			},
		})
	}

	authRealm.Status.Strategies[strategyIndex].Conditions = strategy.Status.Conditions

	r.Log.Info("Update AuthRealm status Strategy",
		"authrealm-name", authRealm.Name,
		"authrealm-status", authRealm.Status,
		"strategy-index", strategyIndex)
	return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
}
