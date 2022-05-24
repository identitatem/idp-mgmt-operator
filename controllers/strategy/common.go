// Copyright Red Hat

package strategy

import (
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
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

//addUserDefinedSelectors Append the user defined placement ClaimSelector, LabelSelector  and MatchLabels to the hypershift strategy
func addUserDefinedSelectors(placement, placementStrategy *clusterv1alpha1.Placement) {
	for _, cp := range placement.Spec.Predicates {
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.ClaimSelector.MatchExpressions =
			append(placementStrategy.Spec.Predicates[0].RequiredClusterSelector.ClaimSelector.MatchExpressions,
				cp.RequiredClusterSelector.ClaimSelector.MatchExpressions...)
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchExpressions =
			append(placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchExpressions,
				cp.RequiredClusterSelector.LabelSelector.MatchExpressions...)
		//We can overwrite the Matchlabel as the strategy check only on MatchExpression, in the future if the strategy
		//checks also on label, we will need to merge
		placementStrategy.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchLabels =
			cp.RequiredClusterSelector.LabelSelector.MatchLabels
	}
}
