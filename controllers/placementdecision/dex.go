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
	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) syncDexClients(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecision *clusterv1alpha1.PlacementDecision) error {

	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel],
	}, client.InNamespace(placementDecision.Namespace)); err != nil {
		return err
	}

	if err := r.deleteObsoleteConfigs(authRealm, placementDecisions); err != nil {
		return err
	}

	if err := r.createConfigs(authRealm, placementDecisions); err != nil {
		return err
	}

	return nil
}

func (r *PlacementDecisionReconciler) deleteObsoleteConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecisions *clusterv1alpha1.PlacementDecisionList) error {
	r.Log.Info("delete obsolete config for Authrealm", "Namespace", authRealm.Namespace, "Name", authRealm.Name)
	dexClients := &identitatemdexv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(), dexClients, &client.ListOptions{Namespace: helpers.DexServerNamespace(authRealm)}); err != nil {
		return err
	}

	for _, dexClient := range dexClients.Items {
		r.Log.Info("for dexClient", "Namespace", dexClient.Namespace, "Name", dexClient.Name)
		for _, idp := range authRealm.Spec.IdentityProviders {
			r.Log.Info("for IDP", "Name", idp.Name)
			if !inPlacementDecision(dexClient.GetLabels()[helpers.ClusterNameLabel], placementDecisions) &&
				dexClient.GetLabels()[helpers.IdentityProviderNameLabel] == idp.Name {
				if err := r.deleteConfig(authRealm, dexClient.Name, dexClient.GetLabels()[helpers.ClusterNameLabel], idp); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecisions *clusterv1alpha1.PlacementDecisionList) error {
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			for _, idp := range authRealm.Spec.IdentityProviders {
				//Create Secret
				clientSecret, err := r.createClientSecret(decision, idp)
				if err != nil {
					return err
				}
				//Create dexClient
				if err := r.createDexClient(authRealm, placementDecision, decision, idp, clientSecret); err != nil {
					return err
				}
				//Create ClusterOAuth
				if err := r.createClusterOAuth(authRealm, decision, idp, clientSecret); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createDexClient(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecision clusterv1alpha1.PlacementDecision,
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
	clientSecret *corev1.Secret) error {
	r.Log.Info("create dexClient for", "cluster", decision.ClusterName, "identityProvider", idp.Name)
	dexClientExists := true
	dexClient := &identitatemdexv1alpha1.DexClient{}
	if err := r.Client.Get(context.TODO(), helpers.DexClientObjectKey(authRealm, decision, idp), dexClient); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dexClientExists = false
		dexClient = &identitatemdexv1alpha1.DexClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.DexClientName(decision, idp),
				Namespace: helpers.DexServerNamespace(authRealm),
				Labels: map[string]string{
					helpers.ClusterNameLabel:                decision.ClusterName,
					helpers.IdentityProviderNameLabel:       idp.Name,
					helpers.PlacementDecisionNameLabel:      placementDecision.Name,
					helpers.PlacementDecisionNamespaceLabel: placementDecision.Namespace,
				},
			},
			Spec: identitatemdexv1alpha1.DexClientSpec{
				Public: false,
			},
		}
	}

	dexClient.Spec.ClientID = decision.ClusterName
	dexClient.Spec.ClientSecret = string(clientSecret.Data["clientSecret"])

	urlScheme, host, err := helpers.GetAppsURL(r.Client, false)
	if err != nil {
		return err
	}
	redirectURI := fmt.Sprintf("%s://oauth-openshift.%s/oauth2callback/%s", urlScheme, host, decision.ClusterName)
	dexClient.Spec.RedirectURIs = []string{redirectURI}
	switch dexClientExists {
	case true:
		return r.Client.Update(context.TODO(), dexClient)
	case false:
		return r.Client.Create(context.Background(), dexClient)
	}
	return nil
}

func (r *PlacementDecisionReconciler) deleteConfig(authRealm *identitatemv1alpha1.AuthRealm,
	dexClientName, clusterName string,
	idp openshiftconfigv1.IdentityProvider) error {
	//Delete DexClient
	r.Log.Info("delete dexclient", "namespace", authRealm.Name, "name", dexClientName)
	dexClient := &dexoperatorv1alpha1.DexClient{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: dexClientName, Namespace: helpers.DexServerNamespace(authRealm)}, dexClient); err == nil {
		if err := r.Delete(context.TODO(), dexClient); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	//Delete ClientSecret
	r.Log.Info("delete clientSecret", "namespace", clusterName, "name", idp.Name)
	clientSecret := &corev1.Secret{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: idp.Name, Namespace: clusterName}, clientSecret); err == nil {
		if err := r.Delete(context.TODO(), clientSecret); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	//Delete clusterOAuth
	r.Log.Info("delete clusterOAuth", "Namespace", dexClient.GetLabels()[helpers.ClusterNameLabel], "Name", idp.Name)
	clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: idp.Name, Namespace: clusterName}, clusterOAuth); err == nil {
		if err := r.Delete(context.TODO(), clusterOAuth); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
