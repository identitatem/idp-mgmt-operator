// Copyright Red Hat

package hypershiftdeployment

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	hypershiftdeploymentv1alpha1 "github.com/stolostron/hypershift-deployment-controller/api/v1alpha1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *HypershiftDeploymentReconciler) updateAuthRealmStatusHypershiftDeploymentConditions(
	authRealm *identitatemv1alpha1.AuthRealm,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth,
	hd *hypershiftdeploymentv1alpha1.HypershiftDeployment,
	delete bool) error {
	patch := client.MergeFrom(authRealm.DeepCopy())
	strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, clusterOAuth.Spec.StrategyReference.Name)
	if strategyIndex == -1 {
		return fmt.Errorf("strategy %s not found", clusterOAuth.Spec.StrategyReference.Name)
	}

	clusterStatusIndex := helpers.GetClusterStatusIndex(r.Log, &authRealm.Status.Strategies[strategyIndex], clusterOAuth.Namespace)
	r.Log.Info("updateAuthRealmStatusHypershiftDeploymentConditions/getClusterStatusIndex", "clusterName", clusterOAuth.Namespace, "clusterStatusIndex", clusterStatusIndex)
	if clusterStatusIndex == -1 {
		return fmt.Errorf("cluster %s not found", clusterOAuth.Namespace)
	}

	clusterStatus := authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex]
	if delete {
		clusterStatus.HypershiftDeployment = identitatemv1alpha1.AuthRealmHypershiftDeploymentStatus{}
		clusterStatus.ManifestWork = identitatemv1alpha1.AuthRealmManifestWorkStatus{}
	} else {
		clusterStatus.HypershiftDeployment.Name = hd.Name
		clusterStatus.HypershiftDeployment.Conditions = hd.Status.Conditions
		clusterStatus.HypershiftDeployment.Phase = hd.Status.Phase
		mw := &manifestworkv1.ManifestWork{}
		err := r.Get(context.TODO(), client.ObjectKey{
			Name:      hd.Spec.InfraID,
			Namespace: hd.Namespace,
		}, mw)
		if err != nil {
			return err
		}
		clusterStatus.ManifestWork.Name = mw.Name
		clusterStatus.ManifestWork.ManifestWorkStatus.Conditions = mw.Status.Conditions
		clusterStatus.ManifestWork.ManifestWorkStatus.ResourceStatus = mw.Status.ResourceStatus
	}
	authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex] = clusterStatus
	r.Log.Info("Update AuthRealm status Hypershift",
		"authrealm-name", authRealm.Name,
		"authrealm-hypershiftDeployment", clusterStatus.HypershiftDeployment,
		"authrealm-manifestwork", clusterStatus.ManifestWork,
		"strategy-index", strategyIndex)
	return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
}
