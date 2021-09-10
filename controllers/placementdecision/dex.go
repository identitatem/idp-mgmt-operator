// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) syncDexClients(authrealm *identitatemv1alpha1.AuthRealm,
	placementDecision *clusterv1alpha1.PlacementDecision) error {

	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel],
	}, client.InNamespace(placementDecision.Namespace)); err != nil {
		return err
	}

	if err := r.deleteObsoleteConfigs(authrealm, placementDecisions); err != nil {
		return err
	}

	if err := r.createConfigs(authrealm, placementDecisions); err != nil {
		return err
	}

	return nil
}

func (r *PlacementDecisionReconciler) deleteObsoleteConfigs(authrealm *identitatemv1alpha1.AuthRealm,
	placementDecisions *clusterv1alpha1.PlacementDecisionList) error {
	r.Log.Info("delete obsolete config for Authrealm", "Namespace", authrealm.Namespace, "Name", authrealm.Name)
	dexClients := &identitatemdexv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(), dexClients, &client.ListOptions{Namespace: helpers.DexServerNamespace(authrealm)}); err != nil {
		return err
	}

	for i, dexClient := range dexClients.Items {
		r.Log.Info("for dexClient", "Namespace", dexClient.Namespace, "Name", dexClient.Name)
		for _, idp := range authrealm.Spec.IdentityProviders {
			r.Log.Info("for IDP", "Name", idp.Name)
			if !inPlacementDecision(dexClient.GetLabels()["cluster"], placementDecisions) &&
				dexClient.GetLabels()["idp"] == idp.Name {
				//Delete the dexClient
				r.Log.Info("delete dexclient", "Name", idp.Name)
				if err := r.Client.Delete(context.TODO(), &dexClients.Items[i]); err != nil && !errors.IsNotFound(err) {
					return err
				}
				//Delete the clientSecret
				r.Log.Info("delete clientSecret", "Namespace", dexClient.GetLabels()["cluster"], "Name", idp.Name)
				clientSecret := &corev1.Secret{}
				if err := r.Get(context.TODO(), client.ObjectKey{
					Name:      idp.Name,
					Namespace: dexClient.GetLabels()["cluster"]},
					clientSecret); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
				}
				if err := r.Client.Delete(context.TODO(), clientSecret); err != nil && !errors.IsNotFound(err) {
					return err
				}
				//Delete clusterOAuth
				r.Log.Info("delete clusterOAuth", "Namespace", dexClient.GetLabels()["cluster"], "Name", idp.Name)
				clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
				if err := r.Get(context.TODO(), client.ObjectKey{
					Name:      idp.Name,
					Namespace: dexClient.GetLabels()["cluster"]},
					clusterOAuth); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
				}
				if err := r.Client.Delete(context.TODO(), clusterOAuth); err != nil && !errors.IsNotFound(err) {
					return err
				}
				//Delete Manifestwork
				// r.Log.Info("delete manifestwork", "namespace", dexClient.GetLabels()["cluster"], "name", helpers.ManifestWorkName())
				// manifestwork := &workv1.ManifestWork{
				// 	ObjectMeta: metav1.ObjectMeta{
				// 		Name:      helpers.ManifestWorkName(),
				// 		Namespace: dexClient.GetLabels()["cluster"],
				// 	},
				// }
				// if err := r.Delete(context.TODO(), manifestwork); err != nil && !errors.IsNotFound(err) {
				// 	return err
				// }
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createConfigs(authrealm *identitatemv1alpha1.AuthRealm,
	placementDecisions *clusterv1alpha1.PlacementDecisionList) error {
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			for _, idp := range authrealm.Spec.IdentityProviders {
				//Create Secret
				clientSecret, err := r.createClientSecret(decision, idp)
				if err != nil {
					return err
				}
				//Create dexClient
				if err := r.createDexClient(authrealm, decision, idp, clientSecret); err != nil {
					return err
				}
				//Create ClusterOAuth
				if err := r.createClusterOAuth(authrealm, decision, idp, clientSecret); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createDexClient(authrealm *identitatemv1alpha1.AuthRealm,
	decision clusterv1alpha1.ClusterDecision,
	idp openshiftconfigv1.IdentityProvider,
	clientSecret *corev1.Secret) error {
	r.Log.Info("create dexClient for", "cluster", decision.ClusterName, "identityProvider", idp.Name)
	dexClientExists := true
	dexClient := &identitatemdexv1alpha1.DexClient{}
	if err := r.Client.Get(context.TODO(), helpers.DexClientObjectKey(authrealm, decision, idp), dexClient); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dexClientExists = false
		dexClient = &identitatemdexv1alpha1.DexClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.DexClientName(decision, idp),
				Namespace: helpers.DexServerNamespace(authrealm),
				Labels: map[string]string{
					"cluster": decision.ClusterName,
					"idp":     idp.Name,
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
