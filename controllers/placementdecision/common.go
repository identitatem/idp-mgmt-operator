// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	giterrors "github.com/pkg/errors"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) createClientSecret(
	decision clusterv1alpha1.ClusterDecision,
	authRealm *identitatemv1alpha1.AuthRealm) (*corev1.Secret, error) {
	r.Log.Info("create clientSecret for", "cluster", decision.ClusterName, "identityProvider", authRealm.Name)
	authRealmObjectKey := client.ObjectKey{
		Name:      authRealm.Name,
		Namespace: authRealm.Namespace,
	}
	clientSecret := &corev1.Secret{}
	if err := r.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ClientSecretName(authRealmObjectKey), Namespace: decision.ClusterName},
		clientSecret); err != nil {
		if !errors.IsNotFound(err) {
			return nil, giterrors.WithStack(err)
		}
		clientSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.ClientSecretName(authRealmObjectKey),
				Namespace: decision.ClusterName,
			},
			Data: map[string][]byte{
				"clientSecret": []byte(helpers.RandomString(32, helpers.RandomTypePassword)),
			},
		}
		if err := r.Create(context.TODO(), clientSecret); err != nil {
			return nil, giterrors.WithStack(err)
		}
	}
	return clientSecret, nil
}

func (r *PlacementDecisionReconciler) createClusterOAuth(authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placementDecision clusterv1alpha1.PlacementDecision,
	decision clusterv1alpha1.ClusterDecision,
	dexClient *dexoperatorv1alpha1.DexClient) error {
	r.Log.Info("create clusterOAuth for", "cluster", decision.ClusterName, "authRealm", authRealm.Name)
	authRealmObjectKey := client.ObjectKey{
		Name:      authRealm.Name,
		Namespace: authRealm.Namespace,
	}
	clusterOAuthExists := true
	clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ClusterOAuthName(authRealmObjectKey), Namespace: decision.ClusterName},
		clusterOAuth); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		clusterOAuthExists = false
		clusterOAuth = &identitatemv1alpha1.ClusterOAuth{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.ClusterOAuthName(authRealmObjectKey),
				Namespace: decision.ClusterName,
			},
			Spec: identitatemv1alpha1.ClusterOAuthSpec{
				OAuth: &openshiftconfigv1.OAuth{
					ObjectMeta: metav1.ObjectMeta{
						Name:      helpers.ClusterOAuthName(authRealmObjectKey),
						Namespace: decision.ClusterName,
					},
					Spec: openshiftconfigv1.OAuthSpec{
						IdentityProviders: make([]openshiftconfigv1.IdentityProvider, 1),
					},
				},
			},
		}
	}

	uScheme, host, err := helpers.GetAppsURL(r.Client, false)
	if err != nil {
		return err
	}

	clusterOAuth.Spec.AuthRealmReference = identitatemv1alpha1.RelatedObjectReference{
		Kind:      "AuthRealm",
		Name:      authRealm.Name,
		Namespace: authRealm.Namespace,
	}
	clusterOAuth.Spec.StrategyReference = identitatemv1alpha1.RelatedObjectReference{
		Kind:      "Strategy",
		Name:      strategy.Name,
		Namespace: strategy.Namespace,
	}
	clusterOAuth.Spec.DexClientReference = identitatemv1alpha1.RelatedObjectReference{
		Kind:      "DexClient",
		Name:      dexClient.Name,
		Namespace: dexClient.Namespace,
	}
	clusterOAuth.Spec.OAuth.Spec.IdentityProviders[0] = openshiftconfigv1.IdentityProvider{
		Name:          authRealm.Name,
		MappingMethod: authRealm.Spec.IdentityProviders[0].MappingMethod,
		IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
			Type: openshiftconfigv1.IdentityProviderTypeOpenID,
			OpenID: &openshiftconfigv1.OpenIDIdentityProvider{
				Claims: openshiftconfigv1.OpenIDClaims{
					Email: []string{
						"email",
					},
					Name: []string{
						"name",
					},
					PreferredUsername: []string{
						"preferred_username",
						"email",
						"name",
					},
					Groups: []openshiftconfigv1.OpenIDClaim{
						"groups",
					},
				},
				ClientID: helpers.DexClientName(authRealmObjectKey, decision.ClusterName),
				ClientSecret: openshiftconfigv1.SecretNameReference{
					Name: dexClient.Spec.ClientSecretRef.Name,
				},
				// Was not working when tested
				// ExtraAuthorizeParameters: map[string]string{
				// 	"include_granted_scopes": "true",
				// },
				ExtraScopes: []string{
					"email",
					"profile",
					"groups",
					"federated:id",
					"offline_access",
				},
				Issuer: fmt.Sprintf("%s://%s.%s", uScheme, authRealm.Spec.RouteSubDomain, host),
			},
		},
	}

	switch clusterOAuthExists {
	case true:
		return giterrors.WithStack(r.Client.Update(context.TODO(), clusterOAuth))
	case false:
		return giterrors.WithStack(r.Client.Create(context.Background(), clusterOAuth))
	}
	return nil
}

func (r *PlacementDecisionReconciler) GetStrategyFromPlacementDecision(placementDecision *clusterv1alpha1.PlacementDecision) (*identitatemv1alpha1.StrategyList, error) {
	r.Log.Info("GetStrategyFromPlacementDecision placementDecision:", "name", placementDecision.Name, "namespace", placementDecision.Namespace)
	if placementName, ok := placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel]; ok {
		placement := &clusterv1alpha1.Placement{}
		if err := r.Get(context.TODO(), client.ObjectKey{Name: placementName, Namespace: placementDecision.Namespace}, placement); err != nil {
			return nil, err
		}
		return r.GetStrategiesFromPlacement(placement)
	}
	return nil, giterrors.WithStack(fmt.Errorf("placementDecision %s has no label %s", placementDecision.Name, clusterv1alpha1.PlacementLabel))
}

func (r *PlacementDecisionReconciler) GetStrategiesFromPlacement(placement *clusterv1alpha1.Placement) (*identitatemv1alpha1.StrategyList, error) {
	r.Log.Info("GetStrategiesFromPlacement", "placementName", placement.Name, "placementNamespace", placement.Namespace)
	strategies := &identitatemv1alpha1.StrategyList{}
	if err := r.List(context.TODO(), strategies, &client.ListOptions{Namespace: placement.Namespace}); err != nil {
		return nil, giterrors.WithStack(err)
	}
	foundStrategies := &identitatemv1alpha1.StrategyList{}
	foundStrategies.Items = make([]identitatemv1alpha1.Strategy, 0)
	for _, strategy := range strategies.Items {
		if strategy.Spec.PlacementRef.Name == placement.Name {
			foundStrategies.Items = append(foundStrategies.Items, strategy)
		}
	}
	if len(foundStrategies.Items) == 0 {
		return nil, giterrors.WithStack(errors.NewNotFound(identitatemv1alpha1.Resource("strategies"), placement.Name))
	}
	return foundStrategies, nil
}

func (r *PlacementDecisionReconciler) inPlacementDecision(clusterName string, placement *clusterv1alpha1.Placement) (bool, error) {
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placement.Name,
	}, client.InNamespace(placement.Namespace)); err != nil {
		return false, giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			if decision.ClusterName == clusterName {
				return true, nil
			}
		}
	}
	return false, nil
}
