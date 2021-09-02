// Copyright Red Hat

package placementdecision

import (
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

//DV
//grcStrategy generates resources for the GRC strategy
func (r *PlacementDecisionReconciler) grcStrategy(
	placement *clusterv1alpha1.Placement,
	placementDecision *clusterv1alpha1.PlacementDecision) error {
	return nil
}
