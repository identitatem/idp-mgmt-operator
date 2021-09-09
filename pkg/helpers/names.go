// Copyright Red Hat

package helpers

import (
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	manifestWorkName string = "idp-oauth"
	dexServerName    string = "dex-server"
)

func ManifestWorkName() string {
	return manifestWorkName
}

func DexServerName() string {
	return dexServerName
}

func DexServerNamespace(authrealm *identitatemv1alpha1.AuthRealm) string {
	return fmt.Sprintf("%s-%s", authrealm.Namespace, authrealm.Name)
}

func DexClientName(
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
) string {
	return decision.ClusterName
}

func DexClientObjectKey(
	authrealm *identitatemv1alpha1.AuthRealm,
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
) client.ObjectKey {
	return client.ObjectKey{
		Name:      DexClientName(decision, idp),
		Namespace: DexServerNamespace(authrealm),
	}
}

func StrategyName(authrealm *identitatemv1alpha1.AuthRealm, t identitatemv1alpha1.StrategyType) string {
	return fmt.Sprintf("%s-%s", authrealm.Name, string(t))
}
