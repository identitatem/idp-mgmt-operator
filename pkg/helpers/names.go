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
	manifestWorkOAuthName    string = "idp-oauth"
	manifestWorkSecretName   string = "idp-secret"
	managedClusterViewOAuth  string = "oauth-view"
	configMapOriginalOAuth   string = "oauth-original"
	dexServerName            string = "dex-server"
	dexOperatorNamespace     string = "idp-mgmt-dex"
	dexServerNamespacePrefix string = "idp-mgmt"
)

const (
	ClusterNameLabel          string = "cluster.identitatem.io/name"
	IdentityProviderNameLabel string = "identityprovider.identitatem.io/name"
)

func ManifestWorkOAuthName() string {
	return manifestWorkOAuthName
}

func ManifestWorkSecretName() string {
	return manifestWorkSecretName
}

func ManagedClusterViewOAuthName() string {
	return managedClusterViewOAuth
}

func ConfigMapOriginalOAuthName() string {
	return configMapOriginalOAuth
}

func DexOperatorNamespace() string {
	return dexOperatorNamespace
}

func DexServerName() string {
	return dexServerName
}

func DexServerNamespace(authRealm *identitatemv1alpha1.AuthRealm) string {
	return fmt.Sprintf("%s-%s", dexServerNamespacePrefix, authRealm.Spec.RouteSubDomain)
}

func DexClientName(
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
) string {
	return decision.ClusterName
}

func DexClientObjectKey(
	authRealm *identitatemv1alpha1.AuthRealm,
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
) client.ObjectKey {
	return client.ObjectKey{
		Name:      DexClientName(decision, idp),
		Namespace: DexServerNamespace(authRealm),
	}
}

func StrategyName(authRealm *identitatemv1alpha1.AuthRealm, t identitatemv1alpha1.StrategyType) string {
	return fmt.Sprintf("%s-%s", authRealm.Name, string(t))
}
