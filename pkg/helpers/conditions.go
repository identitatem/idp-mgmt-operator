// Copyright Red Hat

package helpers

import (
	"context"

	giterrors "github.com/pkg/errors"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MergeStatusConditions returns a new status condition array with merged status conditions. It is based on newConditions,
// and merges the corresponding existing conditions if exists.
func mergeStatusConditions(conditions []metav1.Condition, newConditions ...metav1.Condition) []metav1.Condition {
	merged := []metav1.Condition{}

	merged = append(merged, conditions...)

	for _, condition := range newConditions {
		// merge two conditions if necessary
		meta.SetStatusCondition(&merged, condition)
	}

	return merged
}

func UpdateAuthRealmStatusConditions(c client.Client, authRealm *identitatemv1alpha1.AuthRealm, newConditions ...metav1.Condition) error {
	authRealm.Status.Conditions = mergeStatusConditions(authRealm.Status.Conditions, newConditions...)
	return giterrors.WithStack(c.Status().Update(context.TODO(), authRealm))
}

func UpdateClusterOAuthStatusConditions(
	c client.Client,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth, newConditions ...metav1.Condition) error {
	clusterOAuth.Status.Conditions = mergeStatusConditions(clusterOAuth.Status.Conditions, newConditions...)
	if err := giterrors.WithStack(c.Status().Update(context.TODO(), clusterOAuth)); err != nil {
		return err
	}
	authRealm := &identitatemv1alpha1.AuthRealm{}
	if err := c.Get(context.TODO(),
		client.ObjectKey{
			Name:      clusterOAuth.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace,
		}, authRealm); err != nil {
		return err
	}
	return UpdateAuthRealmStatusClusterOAuthConditions(c, authRealm, clusterOAuth)
}

func UpdateAuthRealmStatusClusterOAuthConditions(
	c client.Client,
	authRealm *identitatemv1alpha1.AuthRealm,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth) error {
	authRealm, strategyIndex := getAuthRealmStatusStrategy(authRealm, clusterOAuth.Spec.StrategyReference.Name)
	strategyStatus, clusterStatusIndex := getAuthRealmStatusCluster(&authRealm.Status.Strategies[strategyIndex], clusterOAuth)

	if authRealm.Status.Conditions == nil {
		authRealm.Status.Conditions = make([]metav1.Condition, 0)
	}
	strategyStatus.Clusters[clusterStatusIndex].ClusterOAuth.ClusterOAuthStatus.Conditions = mergeStatusConditions(
		strategyStatus.Clusters[clusterStatusIndex].ClusterOAuth.ClusterOAuthStatus.Conditions,
		clusterOAuth.Status.Conditions...)

	return giterrors.WithStack(c.Status().Update(context.TODO(), authRealm))
}

func getAuthRealmStatusStrategy(
	authRealm *identitatemv1alpha1.AuthRealm,
	strategyName string) (*identitatemv1alpha1.AuthRealm, int) {
	status := &authRealm.Status
	strategyIndex := -1
	for i, s := range status.Strategies {
		if s.Name == strategyName {
			strategyIndex = i
			break
		}
	}
	if strategyIndex == -1 {
		strategyIndex = len(status.Strategies)
		status.Strategies = append(status.Strategies, identitatemv1alpha1.AuthRealmStrategyStatus{
			Name:       strategyName,
			Placements: make([]clusterv1alpha1.PlacementStatus, 0),
		})
	}
	return authRealm, strategyIndex
}

func getAuthRealmStatusCluster(
	strategyStatus *identitatemv1alpha1.AuthRealmStrategyStatus,
	clusterOAuth *identitatemv1alpha1.ClusterOAuth) (*identitatemv1alpha1.AuthRealmStrategyStatus, int) {
	clusterStatusIndex := -1
	for i, clusterStatus := range strategyStatus.Clusters {
		if clusterStatus.Name == clusterOAuth.Name {
			clusterStatusIndex = i
			break
		}
	}
	if clusterStatusIndex == -1 {
		clusterStatusIndex = len(strategyStatus.Clusters)
		strategyStatus.Clusters = append(strategyStatus.Clusters, identitatemv1alpha1.AuthRealmClusterStatus{
			Name: clusterOAuth.Name,
		})
	}
	return strategyStatus, clusterStatusIndex
}
