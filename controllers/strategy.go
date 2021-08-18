// Copyright Red Hat

package controllers

import (
	"context"
	"fmt"

	identitatemmgmtv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	identitatemstrategyv1alpha1 "github.com/identitatem/idp-strategy-operator/api/identitatem/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AuthRealmReconciler) createStrategy(t identitatemstrategyv1alpha1.StrategyType, authrealm *identitatemmgmtv1alpha1.AuthRealm) error {
	strategy := &identitatemstrategyv1alpha1.Strategy{}
	name := GenerateStrategyName(t, authrealm)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: authrealm.Namespace}, strategy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		strategy := &identitatemstrategyv1alpha1.Strategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: authrealm.Namespace,
			},
			Spec: identitatemstrategyv1alpha1.StrategySpec{
				StrategyType: t,
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

func GenerateStrategyName(t identitatemstrategyv1alpha1.StrategyType, authrealm *identitatemmgmtv1alpha1.AuthRealm) string {
	return fmt.Sprintf("%s-%s", authrealm.Name, string(t))
}
