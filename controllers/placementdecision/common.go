// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	clientSecret := &corev1.Secret{}
	if err := r.Get(context.TODO(), client.ObjectKey{Name: helpers.ClientSecretName(authRealm), Namespace: decision.ClusterName}, clientSecret); err != nil {
		if !errors.IsNotFound(err) {
			return nil, giterrors.WithStack(err)
		}
		clientSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.ClientSecretName(authRealm),
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
	decision clusterv1alpha1.ClusterDecision,
	clientSecret *corev1.Secret) error {
	r.Log.Info("create clusterOAuth for", "cluster", decision.ClusterName, "authRealm", authRealm.Name)
	clusterOAuthExists := true
	clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ClusterOAuthName(authRealm), Namespace: decision.ClusterName}, clusterOAuth); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		clusterOAuthExists = false
		clusterOAuth = &identitatemv1alpha1.ClusterOAuth{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.ClusterOAuthName(authRealm),
				Namespace: decision.ClusterName,
			},
			Spec: identitatemv1alpha1.ClusterOAuthSpec{
				OAuth: &openshiftconfigv1.OAuth{
					ObjectMeta: metav1.ObjectMeta{
						Name:      helpers.ClusterOAuthName(authRealm),
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
				ClientID: helpers.DexClientName(authRealm, decision.ClusterName),
				ClientSecret: openshiftconfigv1.SecretNameReference{
					Name: clientSecret.Name,
				},
				// Was not working when tested
				// ExtraAuthorizeParameters: map[string]string{
				// 	"include_granted_scopes": "true",
				// },
				ExtraScopes: []string{
					//TEST	"email",
					//TEST "profile",
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

func (r *PlacementDecisionReconciler) GetStrategyFromPlacementDecision(placementDecision *clusterv1alpha1.PlacementDecision) (*identitatemv1alpha1.Strategy, error) {
	r.Log.Info("GetStrategyFromPlacementDecision placementDecision:", "name", placementDecision.Name, "namespace", placementDecision.Namespace)
	if placementName, ok := placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel]; ok {
		return r.GetStrategyFromPlacement(placementName, placementDecision.Namespace)
	}
	return nil, giterrors.WithStack(fmt.Errorf("placementDecision %s has no label %s", placementDecision.Name, clusterv1alpha1.PlacementLabel))
}

func (r *PlacementDecisionReconciler) GetStrategyFromPlacement(placementName, placementNamespace string) (*identitatemv1alpha1.Strategy, error) {
	r.Log.Info("GetStrategyFromPlacement", "placementName", placementName, "placementNamespace", placementNamespace)
	strategies := &identitatemv1alpha1.StrategyList{}
	if err := r.List(context.TODO(), strategies, &client.ListOptions{Namespace: placementNamespace}); err != nil {
		return nil, giterrors.WithStack(err)
	}
	for _, strategy := range strategies.Items {
		if strategy.Spec.PlacementRef.Name == placementName {
			return &strategy, nil
		}
	}
	return nil, giterrors.WithStack(errors.NewNotFound(identitatemv1alpha1.Resource("strategies"), placementName))
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
