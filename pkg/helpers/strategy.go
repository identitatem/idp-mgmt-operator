// Copyright Red Hat

package helpers

import (
	"context"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateStrategy(c client.Client, scheme *runtime.Scheme, t identitatemv1alpha1.StrategyType, authRealm *identitatemv1alpha1.AuthRealm) error {
	strategy := &identitatemv1alpha1.Strategy{}
	name := StrategyName(authRealm, t)
	if err := c.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: authRealm.Namespace}, strategy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		strategy := &identitatemv1alpha1.Strategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: authRealm.Namespace,
			},
			Spec: identitatemv1alpha1.StrategySpec{
				Type: t,
			},
		}

		if err := controllerutil.SetControllerReference(authRealm, strategy, scheme); err != nil {
			return err
		}
		if err := c.Create(context.TODO(), strategy); err != nil {
			return err
		}
	}
	return nil
}
