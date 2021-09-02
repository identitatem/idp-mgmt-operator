// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"
	"os"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	idpstrategyoperatorconfig "github.com/identitatem/idp-mgmt-operator/config"
	"github.com/identitatem/idp-mgmt-operator/deploy"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
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

func (r *AuthRealmReconciler) installIDPStrategyCRDs() error {
	r.Log.Info("installIDPStrategyCRDs")

	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.
		WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
		WithTemplateFuncMap(FuncMap()).
		Build()

	readerIDPStrategyOperator := idpconfig.GetScenarioResourcesReader()
	files := []string{"crd/bases/identityconfig.identitatem.io_strategies.yaml"}

	_, err := applier.ApplyDirectly(readerIDPStrategyOperator, nil, false, "", files...)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRealmReconciler) installIDPStrategyOperator() error {
	r.Log.Info("installIDPStrategyOperator")

	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.
		WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
		WithTemplateFuncMap(FuncMap()).
		Build()

	idpStrategyOperatorInage := os.Getenv(idpStrategyOperatorImageEnvName)
	if len(idpStrategyOperatorInage) == 0 {
		return fmt.Errorf("EnvVar %s not provided", idpStrategyOperatorImageEnvName)
	}

	podNamespace := os.Getenv(podNamespaceEnvName)
	if len(podNamespace) == 0 {
		return fmt.Errorf("EnvVar %s not provided", podNamespaceEnvName)
	}

	readerDeploy := deploy.GetScenarioResourcesReader()
	readerIDPStrategyOperator := idpstrategyoperatorconfig.GetScenarioResourcesReader()

	values := struct {
		Image     string
		Namespace string
		Reader    *clusteradmasset.ScenarioResourcesReader
		File      string
		NewName   string
	}{
		Image:     idpStrategyOperatorInage,
		Namespace: podNamespace,
		Reader:    readerIDPStrategyOperator,
		File:      "rbac/role.yaml",
		NewName:   "idp-strategy-operator-role",
	}
	//Create the namespace

	files := []string{
		"idp-strategy-operator/service_account.yaml",
		"idp-strategy-operator/role.yaml",
		"idp-strategy-operator/role_binding.yaml",
	}

	_, err := applier.ApplyDirectly(readerDeploy, values, false, "", files...)
	if err != nil {
		return err
	}

	_, err = applier.ApplyDeployments(readerDeploy, values, false, "", "idp-strategy-operator/manager.yaml")
	if err != nil {
		return err
	}

	return nil
}
