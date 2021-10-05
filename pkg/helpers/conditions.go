// Copyright Red Hat

package helpers

import (
	"context"

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
	return c.Status().Update(context.TODO(), authRealm)
}
