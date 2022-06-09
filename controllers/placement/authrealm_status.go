// Copyright Red Hat

package placement

import (
	"context"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *PlacementReconciler) updateAuthRealmStatusPlacementStatus(
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) error {

	authRealm, err := helpers.GetAuthrealmFromStrategy(r.Client, strategy)
	if err != nil {
		return err
	}
	patch := client.MergeFrom(authRealm.DeepCopy())
	strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, strategy.Name)
	r.Log.Info("UpdateAuthRealmStatusPlacementStatus/getStrategyStatusIndex",
		"strategyName", strategy.Name,
		"strategyIndex", strategyIndex,
		"placement.Status", placement.Status)
	if strategyIndex == -1 {
		strategyIndex = len(authRealm.Status.Strategies)
		authRealm.Status.Strategies = append(authRealm.Status.Strategies, identitatemv1alpha1.AuthRealmStrategyStatus{
			Name: strategy.Name,
			StrategyStatus: identitatemv1alpha1.StrategyStatus{
				Conditions: make([]metav1.Condition, 0),
			},
		})
	}

	r.Log.Info("UpdateAuthRealmStatusPlacementStatus",
		"placement.Status", placement.Status)
	authRealm.Status.Strategies[strategyIndex].Placement.Name = placement.Name
	// if placement.Status.Conditions != nil {
	authRealm.Status.Strategies[strategyIndex].Placement.NumberOfSelectedClusters = placement.Status.NumberOfSelectedClusters
	authRealm.Status.Strategies[strategyIndex].Placement.Conditions = placement.Status.Conditions
	// }
	r.Log.Info("Update AuthRealm status Placement",
		"authrealm-name", authRealm.Name,
		"authrealm-status", authRealm.Status,
		"strategy-index", strategyIndex)
	if err := giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch)); err != nil {
		return err
	}
	r.Log.Info("Update AuthRealm status Placement",
		"authrealm-name", authRealm.Name,
		"authrealm-status", authRealm.Status,
		"strategy-index", strategyIndex)
	return nil
}
