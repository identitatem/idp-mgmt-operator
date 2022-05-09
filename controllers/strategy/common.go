// Copyright Red Hat

package strategy

import (
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func hostedClusterRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      helpers.HostedClusterClusterClaim,
		Operator: metav1.LabelSelectorOpIn,
		Values:   []string{"true"},
	}
}

func notHostedClusterRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      helpers.HostedClusterClusterClaim,
		Operator: metav1.LabelSelectorOpNotIn,
		Values:   []string{"true"},
	}
}

func notGRCRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      "feature.open-cluster-management.io/addon-governance-policy-framework",
		Operator: metav1.LabelSelectorOpDoesNotExist,
		Values:   []string{},
	}
}
