// Copyright Red Hat

package clusteroauth

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ClusterOAuthReconciler) updateClusterOAuthStatusConditions(
	clusterOAuth *identitatemv1alpha1.ClusterOAuth, newConditions ...metav1.Condition) error {
	patch := client.MergeFrom(clusterOAuth.DeepCopy())
	clusterOAuth.Status.Conditions = helpers.MergeStatusConditions(clusterOAuth.Status.Conditions, newConditions...)
	if err := giterrors.WithStack(r.Client.Status().Patch(context.TODO(), clusterOAuth, patch)); err != nil {
		return err
	}
	authRealm := &identitatemv1alpha1.AuthRealm{}
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{
			Name:      clusterOAuth.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace,
		}, authRealm); err != nil {
		return err
	}
	return r.updateAuthRealmStatusClusterOAuthConditions(authRealm, clusterOAuth)
}

func (r *ClusterOAuthReconciler) updateAuthRealmStatusClusterOAuthConditions(
	authRealm *identitatemv1alpha1.AuthRealm,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth) error {
	patch := client.MergeFrom(authRealm.DeepCopy())
	strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, clusterOAuth.Spec.StrategyReference.Name)
	if strategyIndex == -1 {
		return fmt.Errorf("strategy %s not found", clusterOAuth.Spec.StrategyReference.Name)
	}

	clusterStatusIndex := helpers.GetClusterStatusIndex(r.Log, &authRealm.Status.Strategies[strategyIndex], clusterOAuth.Namespace)
	r.Log.Info("UpdateAuthRealmStatusClusterOAuthConditions/getClusterStatusIndex", "clusterName", clusterOAuth.Namespace, "clusterStatusIndex", clusterStatusIndex)
	if clusterStatusIndex == -1 {
		clusterStatusIndex = len(authRealm.Status.Strategies[strategyIndex].Clusters)
		authRealm.Status.Strategies[strategyIndex].Clusters = append(authRealm.Status.Strategies[strategyIndex].Clusters, identitatemv1alpha1.AuthRealmClusterStatus{
			Name: clusterOAuth.Namespace,
		})
	}

	authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ClusterOAuth.Conditions = helpers.MergeStatusConditions(
		authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ClusterOAuth.Conditions,
		clusterOAuth.Status.Conditions...)
	authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex].ClusterOAuth.Name = clusterOAuth.Name
	r.Log.Info("Update AuthRealm status Cluster",
		"authrealm-name", authRealm.Name,
		"authrealm-status", authRealm.Status,
		"strategy-index", strategyIndex)
	return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
}

func (r *ClusterOAuthReconciler) deleteAuthRealmStatusClusterOAuthConditions(
	clusterOAuth *identitatemv1alpha1.ClusterOAuth) error {
	authRealm := &identitatemv1alpha1.AuthRealm{}
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{
			Name:      clusterOAuth.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace,
		}, authRealm); err != nil {
		return err
	}
	patch := client.MergeFrom(authRealm.DeepCopy())
	if strategyIndex := helpers.GetStrategyStatusIndex(r.Log, authRealm, clusterOAuth.Spec.StrategyReference.Name); strategyIndex != -1 {
		if clusterStatusIndex := helpers.GetClusterStatusIndex(r.Log, &authRealm.Status.Strategies[strategyIndex], clusterOAuth.Namespace); clusterStatusIndex != -1 {
			authRealm.Status.Strategies[strategyIndex].Clusters = append(authRealm.Status.Strategies[strategyIndex].Clusters[:clusterStatusIndex],
				authRealm.Status.Strategies[strategyIndex].Clusters[clusterStatusIndex+1:]...)
			return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
		}
	}
	return nil
}
