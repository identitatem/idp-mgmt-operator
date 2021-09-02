// Copyright Red Hat

package strategy

import (
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func (r *StrategyReconciler) backplanePlacementStrategy(strategy *identitatemv1alpha1.Strategy,
	authrealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement,
	placementStrategy *clusterv1alpha1.Placement) error {
	// Append any additional predicates the AuthRealm already had on it's Placement
	placementStrategy.Spec.Predicates = placement.Spec.Predicates

	return nil
}
