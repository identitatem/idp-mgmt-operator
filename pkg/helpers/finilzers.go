// Copyright Red Hat

package helpers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	giterrors "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AuthrealmFinalizer string = "authrealm.identitatem.io/cleanup"

	ClusterOAuthFinalizer string = "clusteroauth.identitatem.io/cleanup"

	PlacementDecisionFinalizer          string = "placelementdecision.identitatem.io/cleanup"
	PlacementDecisionBackplaneFinalizer string = "placelementdecision.identitatem.io/cleanup-backplane"
)

func RemovePlacementDecisionFinalizer(c client.Client, log logr.Logger, strategy *identitatemv1alpha1.Strategy, obj client.Object) error {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		log.Info("Remove finalizer from",
			"finalizer", PlacementDecisionBackplaneFinalizer,
			"kind", obj.GetObjectKind(),
			"namespace", obj.GetNamespace(),
			"name", obj.GetName())
		controllerutil.RemoveFinalizer(obj, PlacementDecisionBackplaneFinalizer)
		// case identitatemv1alpha1.GrcStrategyType:
		// controllerutil.RemoveFinalizer(obj, placementDecisionGRCFinalizer)
	default:
		return giterrors.WithStack(fmt.Errorf("strategy type %s not supported", strategy.Spec.Type))
	}

	return giterrors.WithStack(c.Update(context.TODO(), obj))

}

func ContainsPlacementDecisionFinalizer(strategy *identitatemv1alpha1.Strategy, obj client.Object) bool {
	switch strategy.Spec.Type {
	case identitatemv1alpha1.BackplaneStrategyType:
		return controllerutil.ContainsFinalizer(obj, PlacementDecisionBackplaneFinalizer)
	default:
		return false
	}
}
