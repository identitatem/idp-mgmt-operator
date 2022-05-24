// Copyright Red Hat

package helpers

import (
	"context"
	"fmt"

	giterrors "github.com/pkg/errors"
	hypershiftdeploymentv1alpha1 "github.com/stolostron/hypershift-deployment-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetHypershiftDeployment(c client.Client,
	clusterName string) (*hypershiftdeploymentv1alpha1.HypershiftDeployment, error) {
	hypershiftDeployments := &hypershiftdeploymentv1alpha1.HypershiftDeploymentList{}
	if err := c.List(context.TODO(), hypershiftDeployments, client.MatchingLabels{
		HypershiftDeploymentInfraIDLabel: clusterName,
	}); err != nil {
		return nil, giterrors.WithStack(err)
	}
	switch len(hypershiftDeployments.Items) {
	case 0:
		return nil, giterrors.WithStack(
			fmt.Errorf("no hypershiftdeployment found with the infraID equal to %s", clusterName))
	case 1:
		return &hypershiftDeployments.Items[0], nil
	default:
		return nil, giterrors.WithStack(
			fmt.Errorf("more than one hypershiftdeployment have the infraID equal to %s", clusterName))
	}
}
