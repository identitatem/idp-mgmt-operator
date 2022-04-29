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
	// placementStrategy.Spec.Predicates = placement.Spec.Predicates
	// Append any additional predicates the AuthRealm already had on it's Placement
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
		// {
		// 	RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
		// 		LabelSelector: metav1.LabelSelector{
		// 			MatchExpressions: []metav1.LabelSelectorRequirement{
		// 				notGRCRequirement(),
		// 			},
		// 		},
		// 	},
		// },
	}

	for _, cp := range placement.Spec.Predicates {
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.ClaimSelector.MatchExpressions =
			append(placementStrategy.Spec.Predicates[0].RequiredClusterSelector.ClaimSelector.MatchExpressions,
				cp.RequiredClusterSelector.ClaimSelector.MatchExpressions...)
	}

	for _, cp := range placement.Spec.Predicates {
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchExpressions =
			append(placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchExpressions,
				cp.RequiredClusterSelector.LabelSelector.MatchExpressions...)
		//We can overwrite the Matchlabel as the strategy check only on MatchExpression, in the future if the strategy
		//checks also on label, we will need to merge
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchLabels =
			cp.RequiredClusterSelector.LabelSelector.MatchLabels
	}

	r.Log.Info("placementhypershift after adding labelselector", "placementstrategy", placementStrategy)
}
