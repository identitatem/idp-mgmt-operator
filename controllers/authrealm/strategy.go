// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	idpStrategyOperatorImageEnvName string = "IDP_STRATEGY_OPERATOR_IMAGE"
	podNamespaceEnvName             string = "POD_NAMESPACE"
)

func (r *AuthRealmReconciler) createStrategy(t identitatemv1alpha1.StrategyType, authrealm *identitatemv1alpha1.AuthRealm) error {
	strategy := &identitatemv1alpha1.Strategy{}
	name := GenerateStrategyName(t, authrealm)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: authrealm.Namespace}, strategy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		strategy := &identitatemv1alpha1.Strategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: authrealm.Namespace,
			},
			Spec: identitatemv1alpha1.StrategySpec{
				Type: t,
			},
		}
		if err := controllerutil.SetOwnerReference(authrealm, strategy, r.Scheme); err != nil {
			return err
		}
		if err := r.Client.Create(context.TODO(), strategy); err != nil {
			return err
		}
	}
	return nil
}

func GenerateStrategyName(t identitatemv1alpha1.StrategyType, authrealm *identitatemv1alpha1.AuthRealm) string {
	return fmt.Sprintf("%s-%s", authrealm.Name, string(t))
}
