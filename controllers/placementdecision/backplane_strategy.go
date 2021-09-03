// Copyright Red Hat

package placementdecision

import (
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

const (
	BackplaneManifestWorkName string = "idp-backplane"
)

//DV
//backplaneStrategy generates resources for the Backplane strategy
func (r *PlacementDecisionReconciler) backplaneStrategy(
	authrealm *identitatemv1alpha1.AuthRealm,
	placementDecision *clusterv1alpha1.PlacementDecision) error {
	r.Log.Info("run backplane strategy")
	if err := r.syncDexClients(authrealm, placementDecision); err != nil {
		return err
	}
	return nil
}
