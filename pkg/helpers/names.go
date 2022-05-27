// Copyright Red Hat

package helpers

import (
	"fmt"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	manifestWorkOAuthName            string = "idp-oauth"
	manifestWorkOriginalOAuthName    string = "idp-oauth-original"
	manifestWorkSecretName           string = "idp-secret"
	managedClusterViewOAuth          string = "oauth-view"
	managedClusterViewOAuthOpenshift string = "oauth-openshift-view"
	configMapOriginalOAuth           string = "idp-oauth-original"
	dexServerName                    string = "dex-server"
	dexOperatorNamespace             string = "idp-mgmt-dex"
	dexServerNamespacePrefix         string = "idp-mgmt"
	idpConfigLabel                   string = "auth.identitatem.io/installer-config"
)

const (
	HypershiftDeploymentInfraIDLabel        string = "hypershift.openshift.io/infra-id"
	HypershiftDeploymentForceReconcileLabel string = "hypershift.openshift.io/force-reconcile"
	ClusterNameLabel                        string = "cluster.identitatem.io/name"
	IdentityProviderNameLabel               string = "identityprovider.identitatem.io/name"
	StrategyTypeLabel                       string = "identityprovider.identitatem.io/strategy-type"
	HostedClusterClusterClaim               string = "hostedcluster.hypershift.openshift.io"
	ConsoleURLClusterClaim                  string = "consoleurl.cluster.open-cluster-management.io"
	HostingClusterAnnotation                string = "import.open-cluster-management.io/hosting-cluster-name"
	PlacementStrategyAnnotation             string = "auth.identitatem.io/placement-strategy"
)

const (
	OAuthRedirectURIsClusterClaimName string = "oauthredirecturis.openshift.io"
)

func ManifestWorkOAuthName() string {
	return manifestWorkOAuthName
}

func ManifestWorkHostedOAuthName(clusterName string) string {
	return fmt.Sprintf("%s-%s", ManifestWorkOAuthName(), clusterName)
}

func ManifestWorkOriginalOAuthName() string {
	return manifestWorkOriginalOAuthName
}

func ManifestWorkSecretName() string {
	return manifestWorkSecretName
}

func ManagedClusterViewOAuthName() string {
	return managedClusterViewOAuth
}
func ManagedClusterViewOAuthOpenshiftName() string {
	return managedClusterViewOAuthOpenshift
}

func ManagedClusterViewOAuthOpenshiftNamespace(clusterName string) string {
	return "clusters-" + clusterName
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
	authRealmObjectKey client.ObjectKey,
	clusterName string,
) string {
	return fmt.Sprintf("%s-%s", clusterName, authRealmObjectKey.Name)
}

func ClientSecretName(
	authRealmObjectKey client.ObjectKey,
) string {
	return ClusterOAuthName(authRealmObjectKey)
}

func ClusterOAuthName(
	authRealmObjectKey client.ObjectKey,
) string {
	return fmt.Sprintf("%s-%s", authRealmObjectKey.Name, authRealmObjectKey.Namespace)
}

func DexClientObjectKey(
	authRealm *identitatemv1alpha1.AuthRealm,
	clusterName string,
) client.ObjectKey {
	return client.ObjectKey{
		Name:      DexClientName(client.ObjectKey{Name: authRealm.Name, Namespace: authRealm.Namespace}, clusterName),
		Namespace: DexServerNamespace(authRealm),
	}
}

func StrategyName(authRealm *identitatemv1alpha1.AuthRealm, t identitatemv1alpha1.StrategyType) string {
	return fmt.Sprintf("%s-%s", authRealm.Name, string(t))
}

func PlacementStrategyName(strategy *identitatemv1alpha1.Strategy,
	authRealm *identitatemv1alpha1.AuthRealm) string {
	return PlacementStrategyNameFromPlacementRefName(string(strategy.Spec.Type), authRealm.Spec.PlacementRef.Name)
}

func PlacementStrategyNameFromPlacementRefName(strategyType string,
	placementRefName string) string {
	return fmt.Sprintf("%s-%s", placementRefName, strategyType)

}

func IDPConfigLabel() string {
	return idpConfigLabel
}
