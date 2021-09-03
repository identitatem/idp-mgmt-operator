// Copyright Red Hat

package strategy

import (
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func (r *StrategyReconciler) grcPlacementStrategy(strategy *identitatemv1alpha1.Strategy,
	authrealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement,
	placementStrategy *clusterv1alpha1.Placement) error {
	// Append any additional predicates the AuthRealm already had on it's Placement
	placementStrategy.Spec.Predicates = []clusterv1alpha1.ClusterPredicate{
		{
			RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"feature.open-cluster-management.io/addon-policy-controller": "available",
					},
				},
			},
		},
	}

	placementStrategy.Spec.Predicates = append(placementStrategy.Spec.Predicates, placement.Spec.Predicates...)

	return nil
}
