// Copyright Red Hat

package helpers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
)

func GetAuthrealmFromStrategy(c client.Client, strategy *identitatemv1alpha1.Strategy) (*identitatemv1alpha1.AuthRealm, error) {
	authRealm := &identitatemv1alpha1.AuthRealm{}
	var ownerRef metav1.OwnerReference
	for _, or := range strategy.GetOwnerReferences() {
		if or.Kind == "AuthRealm" {
			ownerRef = or
			break
		}
	}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: ownerRef.Name, Namespace: strategy.Namespace}, authRealm); err != nil {
		return nil, err
	}
	return authRealm, nil
}
