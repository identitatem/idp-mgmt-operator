// Copyright Red Hat

package strategy

import (
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func (r *StrategyReconciler) hypershiftPlacementStrategy(strategy *identitatemv1alpha1.Strategy,
	authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement,
	placementStrategy *clusterv1alpha1.Placement) {

	// Append the hypershift condition
	placementStrategy.Spec.Predicates = []clusterv1alpha1.ClusterPredicate{
		{
			RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
				ClaimSelector: clusterv1alpha1.ClusterClaimSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						hostedClusterRequirement(),
					},
				},
			},
		},
	}

	addUserDefinedSelectors(placement, placementStrategy)

	r.Log.Info("placementhypershift after adding labelselector", "placementstrategy", placementStrategy)

}
