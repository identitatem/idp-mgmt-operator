// Copyright Red Hat

package authrealm

import (
	"context"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	giterrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *AuthRealmReconciler) updateAuthRealmStatusConditions(authRealm *identitatemv1alpha1.AuthRealm, newConditions ...metav1.Condition) error {
	patch := client.MergeFrom(authRealm.DeepCopy())
	authRealm.Status.Conditions = helpers.MergeStatusConditions(authRealm.Status.Conditions, newConditions...)
	return giterrors.WithStack(r.Client.Status().Patch(context.TODO(), authRealm, patch))
}
