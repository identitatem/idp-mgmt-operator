package strategy

import (
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func hypershiftRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      helpers.HypershiftClusterLabel,
		Operator: metav1.LabelSelectorOpExists,
		Values:   []string{},
	}
}

func notHypershiftRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      helpers.HypershiftClusterLabel,
		Operator: metav1.LabelSelectorOpDoesNotExist,
		Values:   []string{},
	}
}

func notGRCRequirement() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      "feature.open-cluster-management.io/addon-policy-controller",
		Operator: metav1.LabelSelectorOpDoesNotExist,
		Values:   []string{},
	}
}
