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

	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	giterrors "github.com/pkg/errors"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) syncDexClients(authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) error {

	if err := r.deleteObsoleteConfigs(authRealm, placement); err != nil {
		return err
	}

	if err := r.createConfigs(authRealm, placement); err != nil {
		return err
	}

	return nil
}

func (r *PlacementDecisionReconciler) deleteObsoleteConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) error {
	r.Log.Info("delete obsolete config for Authrealm", "Namespace", authRealm.Namespace, "Name", authRealm.Name)
	dexClients := &dexoperatorv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(), dexClients, &client.ListOptions{Namespace: helpers.DexServerNamespace(authRealm)}); err != nil {
		return err
	}

	for _, dexClient := range dexClients.Items {
		r.Log.Info("for dexClient", "Namespace", dexClient.Namespace, "Name", dexClient.Name)
		ok, err := r.inPlacementDecision(dexClient.GetLabels()[helpers.ClusterNameLabel], placement)
		if err != nil {
			return err
		}
		if !ok {
			if err := r.deleteConfig(dexClient.GetLabels()[helpers.ClusterNameLabel], false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) error {
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placement.Name,
	}, client.InNamespace(placement.Namespace)); err != nil {
		return giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			// 		for _, idp := range authRealm.Spec.IdentityProviders {
			//Create Secret
			clientSecret, err := r.createClientSecret(decision, authRealm)
			if err != nil {
				return err
			}
			//Create dexClient
			var dexClient *dexoperatorv1alpha1.DexClient
			if dexClient, err = r.createDexClient(authRealm, placementDecision, decision, clientSecret); err != nil {
				return err
			}
			//Create ClusterOAuth
			if err = r.createClusterOAuth(authRealm, decision, dexClient); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createDexClient(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecision clusterv1alpha1.PlacementDecision,
	decision clusterv1alpha1.ClusterDecision,
	clientSecret *corev1.Secret) (*dexoperatorv1alpha1.DexClient, error) {
	r.Log.Info("create dexClient for", "cluster", decision.ClusterName, "authrealm", authRealm.Name)
	authRealmObjectKey := client.ObjectKey{
		Name:      authRealm.Name,
		Namespace: authRealm.Namespace,
	}
	dexClientExists := true
	dexClient := &dexoperatorv1alpha1.DexClient{}
	if err := r.Client.Get(context.TODO(), helpers.DexClientObjectKey(authRealm, decision.ClusterName), dexClient); err != nil {
		if !errors.IsNotFound(err) {
			return nil, giterrors.WithStack(err)
		}
		dexClientExists = false
		dexClient = &dexoperatorv1alpha1.DexClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.DexClientName(authRealmObjectKey, decision.ClusterName),
				Namespace: helpers.DexServerNamespace(authRealm),
				Labels: map[string]string{
					helpers.ClusterNameLabel: decision.ClusterName,
				},
			},
			Spec: dexoperatorv1alpha1.DexClientSpec{
				Public: false,
			},
		}
	}

	dexClient.Spec.ClientID = helpers.DexClientName(authRealmObjectKey, decision.ClusterName)
	dexClient.Spec.ClientSecretRef = corev1.SecretReference{
		Name:      clientSecret.Name,
		Namespace: clientSecret.Namespace,
	}

	mc := &clusterv1.ManagedCluster{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: decision.ClusterName}, mc)
	if err != nil {
		return dexClient, giterrors.WithStack(err)
	}

	if len(mc.Spec.ManagedClusterClientConfigs) == 0 ||
		mc.Spec.ManagedClusterClientConfigs[0].URL == "" {
		return dexClient, giterrors.WithStack(fmt.Errorf("api url not found for cluster %s", decision.ClusterName))
	}

	u, err := url.Parse(mc.Spec.ManagedClusterClientConfigs[0].URL)
	if err != nil {
		return dexClient, giterrors.WithStack(err)
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return dexClient, giterrors.WithStack(err)
	}

	host = strings.Replace(host, "api", "apps", 1)

	redirectURI := fmt.Sprintf("%s://oauth-openshift.%s/oauth2callback/%s", u.Scheme, host, authRealm.Name)

	dexClient.Spec.RedirectURIs = []string{redirectURI}
	switch dexClientExists {
	case true:
		return dexClient, giterrors.WithStack(r.Client.Update(context.TODO(), dexClient))
	case false:
		if err := r.Client.Create(context.Background(), dexClient); err != nil {
			return dexClient, giterrors.WithStack(err)
		}
		dexClient.Status.RelatedObjects =
			[]dexoperatorv1alpha1.RelatedObjectReference{
				{
					Kind:      "PlacementDecision",
					Name:      placementDecision.Name,
					Namespace: placementDecision.Namespace,
				},
			}
		if err := r.Status().Update(context.TODO(), dexClient); err != nil {
			return dexClient, giterrors.WithStack(err)
		}

	}

	r.Log.Info("after update", "dexClient", dexClient)
	return dexClient, nil
}

func (r *PlacementDecisionReconciler) deleteConfig(
	clusterName string,
	onlyIfAuthRealmDeleted bool) error {
	r.Log.Info("delete configuration for cluster", "name", clusterName, "onlyIfAuthRealmDeleted", onlyIfAuthRealmDeleted)
	clusterOAuthList := &identitatemv1alpha1.ClusterOAuthList{}
	if err := r.List(context.TODO(), clusterOAuthList, &client.ListOptions{Namespace: clusterName}); err != nil {
		return err
	}
	for _, clusterOAuth := range clusterOAuthList.Items {
		authRealmObjectKey := client.ObjectKey{
			Name:      clusterOAuth.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace,
		}
		if onlyIfAuthRealmDeleted {
			//Delete only if the authrealm is deleted or in deletion
			authRealm := &identitatemv1alpha1.AuthRealm{}
			err := r.Get(context.TODO(), authRealmObjectKey, authRealm)
			if err == nil {
				if authRealm.DeletionTimestamp == nil {
					continue
				}
			}
			if !errors.IsNotFound(err) {
				return err
			}
		}
		//Delete DexClient
		r.Log.Info("get dexclient", "namespace", clusterOAuth.Spec.DexClientReference.Namespace, "name", clusterOAuth.Spec.DexClientReference.Name)
		dexClient := &dexoperatorv1alpha1.DexClient{}
		err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: clusterOAuth.Spec.DexClientReference.Name, Namespace: clusterOAuth.Spec.DexClientReference.Namespace},
			dexClient)
		switch {
		case err == nil:
			if err := r.Delete(context.TODO(), dexClient); err != nil {
				return giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return giterrors.WithStack(err)
		}
		//Delete ClientSecret
		r.Log.Info("delete clientSecret", "namespace", clusterName, "name", authRealmObjectKey.Name)
		clientSecret := &corev1.Secret{}
		err = r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ClientSecretName(authRealmObjectKey), Namespace: clusterName}, clientSecret)
		switch {
		case err == nil:
			if err = r.Delete(context.TODO(), clientSecret); err != nil {
				return giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return giterrors.WithStack(err)
		}
		//Delete clusterOAuth
		r.Log.Info("delete clusterOAuth", "Namespace", clusterName, "Name", authRealmObjectKey.Name)
		clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
		err = r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ClusterOAuthName(authRealmObjectKey), Namespace: clusterName}, clusterOAuth)
		switch {
		case err == nil:
			if err = r.Delete(context.TODO(), clusterOAuth); err != nil {
				return giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return giterrors.WithStack(err)
		}
	}
	return nil
}
