// Copyright Red Hat

package helpers

import (
	"context"

	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsHypershiftCluster(c client.Client, clusterName string) (bool, error) {
	hc := &hypershiftv1alpha1.HostedCluster{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: clusterName, Namespace: "clusters"}, hc); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
