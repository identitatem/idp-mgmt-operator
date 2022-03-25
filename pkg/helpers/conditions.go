// Copyright Red Hat

package helpers

import (
	"github.com/go-logr/logr"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MergeStatusConditions returns a new status condition array with merged status conditions. It is based on newConditions,
// and merges the corresponding existing conditions if exists.
func MergeStatusConditions(conditions []metav1.Condition, newConditions ...metav1.Condition) []metav1.Condition {
	merged := []metav1.Condition{}

	merged = append(merged, conditions...)

	for _, condition := range newConditions {
		// merge two conditions if necessary
		meta.SetStatusCondition(&merged, condition)
	}

	return merged
}

func GetStrategyStatusIndex(
	log logr.Logger,
	authRealm *identitatemv1alpha1.AuthRealm,
	strategyName string) int {
	status := authRealm.Status
	strategyIndex := -1
	for i, s := range status.Strategies {
		if s.Name == strategyName {
			strategyIndex = i
			break
		}
	}
	return strategyIndex
}

func GetClusterStatusIndex(
	log logr.Logger,
	strategyStatus *identitatemv1alpha1.AuthRealmStrategyStatus,
	clusterName string) int {
	clusterStatusIndex := -1
	for i, clusterStatus := range strategyStatus.Clusters {
		if clusterStatus.Name == clusterName {
			clusterStatusIndex = i
			break
		}
	}
	return clusterStatusIndex
}
