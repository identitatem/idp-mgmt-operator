// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"

	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) syncDexClients(authrealm *identitatemv1alpha1.AuthRealm, placementDecision *clusterv1alpha1.PlacementDecision) error {

	dexClients := &identitatemdexv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(), dexClients, &client.ListOptions{Namespace: authrealm.Name}); err != nil {
		return err
	}

	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel],
	}, client.InNamespace(placementDecision.Namespace)); err != nil {
		return err
	}

	for i, dexClient := range dexClients.Items {
		for _, idp := range authrealm.Spec.IdentityProviders {
			if !inPlacementDecision(dexClient.GetLabels()["cluster"], placementDecisions) &&
				dexClient.GetLabels()["idp"] == idp.Name {
				//Delete the dexClient
				if err := r.Client.Delete(context.TODO(), &dexClients.Items[i]); err != nil {
					return err
				}
				//Delete the clientSecret
				clientSecret := &corev1.Secret{}
				if err := r.Get(context.TODO(), client.ObjectKey{
					Name:      idp.Name,
					Namespace: dexClient.GetLabels()["cluster"]},
					clientSecret); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
				}
				if err := r.Client.Delete(context.TODO(), clientSecret); err != nil {
					return err
				}
			}
		}
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			for _, idp := range authrealm.Spec.IdentityProviders {
				//Create Secret
				clusterName := decision.ClusterName
				clientSecret := &corev1.Secret{}
				if err := r.Get(context.TODO(), client.ObjectKey{Name: idp.Name, Namespace: clusterName}, clientSecret); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					clientSecret = &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      idp.Name,
							Namespace: clusterName,
						},
						Data: map[string][]byte{
							"client-id":     []byte(clusterName),
							"client-secret": []byte(helpers.RandStringRunes(32)),
						},
					}
					if err := r.Create(context.TODO(), clientSecret); err != nil {
						return err
					}
				}
				//Create dexClient
				dexClientExists := true
				dexClient := &identitatemdexv1alpha1.DexClient{}
				if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: fmt.Sprintf("%s-%s", clusterName, idp.Name), Namespace: authrealm.Name}, dexClient); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					dexClientExists = false
					dexClient = &identitatemdexv1alpha1.DexClient{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", clusterName, idp.Name),
							Namespace: authrealm.Name,
							Labels: map[string]string{
								"cluster": clusterName,
								"idp":     idp.Name,
							},
						},
					}
				}

				dexClient.Spec.ClientID = string(clientSecret.Data["client-id"])
				dexClient.Spec.ClientSecret = string(clientSecret.Data["client-secret"])

				apiServerURL, err := helpers.GetKubeAPIServerAddress(r.Client)
				if err != nil {
					return err
				}
				u, err := url.Parse(apiServerURL)
				if err != nil {
					return err
				}

				host, _, err := net.SplitHostPort(u.Host)
				if err != nil {
					return err
				}

				host = strings.Replace(host, "api", "apps", 1)

				redirectURI := fmt.Sprintf("%s://%s/oauth2callback/idpserver", u.Scheme, host)
				dexClient.Spec.RedirectURIs = []string{redirectURI}
				switch dexClientExists {
				case true:
					return r.Client.Update(context.TODO(), dexClient)
				case false:
					return r.Client.Create(context.Background(), dexClient)
				}
			}
		}
	}
	return nil
}

func GetStrategyFromPlacementDecision(c client.Client, placementDecision *clusterv1alpha1.PlacementDecision) (*identitatemv1alpha1.Strategy, error) {
	if placementName, ok := placementDecision.GetLabels()[clusterv1alpha1.PlacementLabel]; ok {
		return GetStrategyFromPlacement(c, placementName, placementDecision.Namespace)
	}
	return nil, fmt.Errorf("placementDecision %s has no label %s", placementDecision.Name, clusterv1alpha1.PlacementLabel)
}

func GetStrategyFromPlacement(c client.Client, placementName, placementNamespace string) (*identitatemv1alpha1.Strategy, error) {
	strategies := &identitatemv1alpha1.StrategyList{}
	if err := c.List(context.TODO(), strategies, &client.ListOptions{Namespace: placementNamespace}); err != nil {
		return nil, err
	}
	for _, strategy := range strategies.Items {
		if strategy.Spec.PlacementRef.Name == placementName {
			return &strategy, nil
		}
	}
	return nil, errors.NewNotFound(identitatemv1alpha1.Resource("strategies"), placementName)
}

func inPlacementDecision(clusterName string, placementDecisions *clusterv1alpha1.PlacementDecisionList) bool {
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			if decision.ClusterName == clusterName {
				return true
			}
		}
	}
	return false
}
