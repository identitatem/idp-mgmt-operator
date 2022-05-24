// Copyright Red Hat

package helpers

import (
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func IsHostedCluster(mc *clusterv1.ManagedCluster) bool {
	_, ok := mc.GetAnnotations()[HostingClusterAnnotation]
	return ok
}
