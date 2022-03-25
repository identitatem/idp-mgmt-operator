// Copyright Red Hat

package manifestwork

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	manifestworkv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManifestWorkReconciler) updateAuthRealmStatusManifestWorkConditions(
	authRealm *identitatemv1alpha1.AuthRealm,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth,
	manifestWork *manifestworkv1.ManifestWork,
	delete bool) error {
	patch := client.MergeFrom(authRealm.DeepCopy())
	strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, clusterOAuth.Spec.StrategyReference.Name)
	if strategyIndex == -1 {
		return fmt.Errorf("strategy %s not found", clusterOAuth.Spec.StrategyReference.Name)
	}

	clusterStatusIndex := helpers.GetClusterStatusIndex(r.Log, &authRealm.Status.Strategies[strategyIndex], clusterOAuth.Namespace)
	r.Log.Info("UpdateAuthRealmStatusManifestWorkConditions/getClusterStatusIndex", "clusterName", clusterOAuth.Namespace, "clusterStatusIndex", clusterStatusIndex)
	if clusterStatusIndex == -1 {
		return fmt.Errorf("cluster %s not found", clusterOAuth.Namespace)
	}

	if delete {
		authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ManifestWork = identitatemv1alpha1.AuthRealmManifestWorkStatus{}
	} else {
		authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ManifestWork.Name = manifestWork.Name
		authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ManifestWork.Conditions = manifestWork.Status.Conditions
		authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ManifestWork.ResourceStatus = manifestWork.Status.ResourceStatus
	}
	r.Log.Info("Update AuthRealm status Manifestwork",
		"authrealm-name", authRealm.Name,
		"authrealm-ManifestWork", authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ManifestWork,
		"strategy-index", strategyIndex)
	return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
}
