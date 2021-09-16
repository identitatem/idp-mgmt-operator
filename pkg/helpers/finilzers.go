// Copyright Red Hat

package helpers

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AuthrealmFinalizer string = "authrealm.identitatem.io/cleanup"

	ClusterOAuthFinalizer string = "clusteroauth.identitatem.io/cleanup"

	PlacementDecisionFinalizer          string = "placelementdecision.identitatem.io/cleanup"
	PlacementDecisionBackplaneFinalizer string = "placelementdecision.identitatem.io/cleanup-backplane"
)

func RemovePlacementDecisionFinalizer(c client.Client, strategy *identitatemv1alpha1.Strategy, obj client.Object) error {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		controllerutil.RemoveFinalizer(obj, PlacementDecisionBackplaneFinalizer)
		// case identitatemv1alpha1.GrcStrategyType:
		// controllerutil.RemoveFinalizer(obj, placementDecisionGRCFinalizer)
	default:
		return fmt.Errorf("strategy type %s not supported", strategy.Spec.Type)
	}

	return c.Update(context.TODO(), obj)

}

func ContainsPlacementDecisionFinalizer(strategy *identitatemv1alpha1.Strategy, obj client.Object) bool {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		return controllerutil.ContainsFinalizer(obj, PlacementDecisionBackplaneFinalizer)
	default:
		return false
	}
}