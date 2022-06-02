// Copyright Red Hat

package placementdecision

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	giterrors "github.com/pkg/errors"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

func (r *PlacementDecisionReconciler) syncDexClients(authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {

	if result, err := r.deleteObsoleteConfigs(authRealm, strategy, placement); err != nil || result.Requeue {
		return result, err
	}

	if err := r.createConfigs(authRealm, strategy, placement); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PlacementDecisionReconciler) deleteObsoleteConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {
	r.Log.Info("delete obsolete config for Authrealm", "Namespace", authRealm.Namespace, "Name", authRealm.Name)
	dexClients := &dexoperatorv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(), dexClients,
		&client.ListOptions{Namespace: helpers.DexServerNamespace(authRealm)},
		client.MatchingLabels{
			helpers.StrategyTypeLabel: string(strategy.Spec.Type),
		}); err != nil {
		return ctrl.Result{}, err
	}

	dexClientsToBeDeleted := make([]dexoperatorv1alpha1.DexClient, 0)
	for _, dexClient := range dexClients.Items {
		r.Log.Info("for dexClient", "Namespace", dexClient.Namespace, "Name", dexClient.Name)
		ok, err := r.inPlacementDecision(dexClient.GetLabels()[helpers.ClusterNameLabel], placement)
		if err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("inPlacementDecision", "result", ok, "placement name", placement.Name)
		if !ok {
			dexClientsToBeDeleted = append(dexClientsToBeDeleted, dexClient)
			if result, err := r.deleteConfig(dexClient.GetLabels()[helpers.ClusterNameLabel],
				placement); err != nil || result.Requeue {
				return result, err
			}
		}
	}

	for _, dexClientToBeDeleted := range dexClientsToBeDeleted {
		//Delete DexClient
		r.Log.Info("delete dexclient", "namespace", dexClientToBeDeleted.Namespace, "name", dexClientToBeDeleted.Name)
		dexClient := &dexoperatorv1alpha1.DexClient{}
		err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: dexClientToBeDeleted.Name, Namespace: dexClientToBeDeleted.Namespace},
			dexClient)
		switch {
		case err == nil:
			if err := r.Delete(context.TODO(), dexClient); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}

	//Wait dexclient deleted
	for _, dexClientToBeDeleted := range dexClientsToBeDeleted {
		dexClient := &dexoperatorv1alpha1.DexClient{}
		r.Log.Info("check dexclient deletion", "namespace", dexClientToBeDeleted.Namespace, "name", dexClientToBeDeleted.Name)
		if err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: dexClientToBeDeleted.Name, Namespace: dexClientToBeDeleted.Namespace},
			dexClient); err == nil {
			r.Log.Info("waiting dexclient deletion",
				"namespace", dexClientToBeDeleted.Namespace,
				"name", dexClientToBeDeleted.Name)
			return ctrl.Result{Requeue: true,
				RequeueAfter: 1 * time.Second}, nil
		}

	}

	return ctrl.Result{}, nil
}

func (r *PlacementDecisionReconciler) createConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	strategy *identitatemv1alpha1.Strategy,
	placement *clusterv1alpha1.Placement) error {
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placement.Name,
	}, client.InNamespace(placement.Namespace)); err != nil {
		return giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		for _, decision := range placementDecision.Status.Decisions {
			//Create Secret
			clientSecret, err := r.createClientSecret(decision, authRealm)
			if err != nil {
				return err
			}
			//Create dexClient
			var dexClient *dexoperatorv1alpha1.DexClient
			if dexClient, err = r.createDexClient(authRealm, placementDecision, strategy, decision, clientSecret); err != nil {
				return err
			}
			//Create ClusterOAuth
			if err := r.createClusterOAuth(authRealm, strategy, placementDecision, decision, dexClient); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createDexClient(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecision clusterv1alpha1.PlacementDecision,
	strategy *identitatemv1alpha1.Strategy,
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
					helpers.ClusterNameLabel:  decision.ClusterName,
					helpers.StrategyTypeLabel: string(strategy.Spec.Type),
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

	var redirectURI string
	//Search oauthredirecturis in clusterclaims
	for _, cc := range mc.Status.ClusterClaims {
		if cc.Name == helpers.OAuthRedirectURIsClusterClaimName {
			redirectURI = cc.Value
			break
		}
	}
	//If found build the redirectURI
	if redirectURI != "" {
		redirectURIs := strings.Split(redirectURI, ",")
		u, err := url.Parse(redirectURIs[0])
		if err != nil {
			return dexClient, giterrors.WithStack(err)
		}

		redirectURI = fmt.Sprintf("%s://%s/oauth2callback/%s", u.Scheme, u.Host, authRealm.Name)
	} else {
		// if not found and hosted cluster then raise an error as we can not find the redirectURI
		if helpers.IsHostedCluster(mc) {
			return dexClient, fmt.Errorf("unable to find the redirectURI for hypershift cluster %s", mc.Name)
		}
		// Keep the old way to build the redirectURI as the above method is available only for >= 2.5
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

		redirectURI = fmt.Sprintf("%s://oauth-openshift.%s/oauth2callback/%s", u.Scheme, host, authRealm.Name)
	}
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
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {
	clusterOAuthList := &identitatemv1alpha1.ClusterOAuthList{}
	if err := r.List(context.TODO(), clusterOAuthList, &client.ListOptions{Namespace: clusterName}); err != nil {
		return ctrl.Result{}, err
	}
	clusterOAuthsToBeDeleted := make([]identitatemv1alpha1.ClusterOAuth, 0)
	for _, clusterOAuth := range clusterOAuthList.Items {
		authRealmObjectKey := client.ObjectKey{
			Name:      clusterOAuth.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuth.Spec.AuthRealmReference.Namespace,
		}
		toBeDeleted := false
		for _, ownerRef := range placement.GetOwnerReferences() {
			if ownerRef.Kind == "Strategy" &&
				ownerRef.Name == clusterOAuth.Spec.StrategyReference.Name {
				toBeDeleted = true
				break
			}
		}
		if !toBeDeleted {
			continue
		}
		clusterOAuthsToBeDeleted = append(clusterOAuthsToBeDeleted, clusterOAuth)
		r.Log.Info("delete configuration for cluster", "name", clusterName)
		//Delete ClientSecret
		r.Log.Info("delete clientSecret", "namespace", clusterName, "name", authRealmObjectKey.Name)
		clientSecret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: helpers.ClientSecretName(authRealmObjectKey), Namespace: clusterName},
			clientSecret)
		switch {
		case err == nil:
			if err = r.Delete(context.TODO(), clientSecret); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}

	for _, clusterOAuthToBeDeleted := range clusterOAuthsToBeDeleted {
		//Delete clusterOAuth
		r.Log.Info("delete clusterOAuth",
			"namespace", clusterOAuthToBeDeleted.Namespace,
			"name", clusterOAuthToBeDeleted.Name)
		clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
		err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: clusterOAuthToBeDeleted.Name,
				Namespace: clusterOAuthToBeDeleted.Namespace}, clusterOAuth)
		switch {
		case err == nil:
			if err = r.Delete(context.TODO(), clusterOAuth); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}

	//Wait ClusterOAuth delted
	for _, clusterOAuthToBeDeleted := range clusterOAuthsToBeDeleted {
		authRealmObjectKey := client.ObjectKey{
			Name:      clusterOAuthToBeDeleted.Spec.AuthRealmReference.Name,
			Namespace: clusterOAuthToBeDeleted.Spec.AuthRealmReference.Namespace,
		}
		clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
		r.Log.Info("check clusterOauth deletion",
			"namespace", clusterOAuthToBeDeleted.Namespace,
			"name", clusterOAuthToBeDeleted.Name)
		if err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: helpers.ClusterOAuthName(authRealmObjectKey),
				Namespace: clusterName}, clusterOAuth); err == nil {
			r.Log.Info("waiting clusterOauth deletion",
				"namespace", clusterOAuthToBeDeleted.Namespace,
				"name", clusterOAuthToBeDeleted.Name)
			return ctrl.Result{Requeue: true,
				RequeueAfter: 1 * time.Second}, nil
		}
	}
	return ctrl.Result{}, nil
}
