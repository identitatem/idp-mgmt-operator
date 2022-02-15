// Copyright Red Hat

package placementdecision

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ghodss/yaml"
	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	giterrors "github.com/pkg/errors"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	"github.com/identitatem/idp-mgmt-operator/resources"
)

func (r *PlacementDecisionReconciler) syncDexClients(authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {

	if err := r.deleteObsoleteConfigs(authRealm, placement); err != nil {
		return ctrl.Result{}, err
	}

	if result, err := r.createConfigs(authRealm, placement); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
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
			if err := r.deleteConfig(authRealm, dexClient.Name, dexClient.GetLabels()[helpers.ClusterNameLabel]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PlacementDecisionReconciler) createConfigs(authRealm *identitatemv1alpha1.AuthRealm,
	placement *clusterv1alpha1.Placement) (ctrl.Result, error) {
	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}
	if err := r.Client.List(context.TODO(), placementDecisions, client.MatchingLabels{
		clusterv1alpha1.PlacementLabel: placement.Name,
	}, client.InNamespace(placement.Namespace)); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	for _, placementDecision := range placementDecisions.Items {
		strategy, err := r.GetStrategyFromPlacementDecision(&placementDecision)
		if err != nil {
			return ctrl.Result{}, err
		}

		for _, decision := range placementDecision.Status.Decisions {
			isHypershift, err := helpers.IsHypershiftCluster(r.Client, decision.ClusterName)
			if err != nil {
				return ctrl.Result{}, err
			}
			if (isHypershift && strategy.Spec.Type != identitatemv1alpha1.HypershiftStrategyType) ||
				(!isHypershift && strategy.Spec.Type != identitatemv1alpha1.BackplaneStrategyType) {
				r.Log.Info("Skip processing as the placement is not for the correct strategy", "cluster", decision.ClusterName, "authRealm", authRealm.Name)
				return ctrl.Result{}, nil
			}

			// 		for _, idp := range authRealm.Spec.IdentityProviders {
			//Create Secret
			clientSecret, err := r.createClientSecret(decision, authRealm)
			if err != nil {
				return ctrl.Result{}, err
			}
			//Create dexClient
			if result, err := r.createDexClient(authRealm, placementDecision, strategy, decision, clientSecret); err != nil {
				return result, err
			}
			//Create ClusterOAuth
			if err := r.createClusterOAuth(authRealm, strategy, decision, clientSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}
func (r *PlacementDecisionReconciler) createDexClient(authRealm *identitatemv1alpha1.AuthRealm,
	placementDecision clusterv1alpha1.PlacementDecision,
	strategy *identitatemv1alpha1.Strategy,
	decision clusterv1alpha1.ClusterDecision,
	clientSecret *corev1.Secret) (ctrl.Result, error) {
	r.Log.Info("create dexClient for", "cluster", decision.ClusterName, "authrealm", authRealm.Name)
	dexClientExists := true
	dexClient := &dexoperatorv1alpha1.DexClient{}
	if err := r.Client.Get(context.TODO(), helpers.DexClientObjectKey(authRealm, decision.ClusterName), dexClient); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		dexClientExists = false
		dexClient = &dexoperatorv1alpha1.DexClient{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.DexClientName(authRealm, decision.ClusterName),
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

	dexClient.Spec.ClientID = helpers.DexClientName(authRealm, decision.ClusterName)
	dexClient.Spec.ClientSecretRef = corev1.SecretReference{
		Name:      clientSecret.Name,
		Namespace: clientSecret.Namespace,
	}

	mc := &clusterv1.ManagedCluster{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: decision.ClusterName}, mc)
	if err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}

	if len(mc.Spec.ManagedClusterClientConfigs) == 0 ||
		mc.Spec.ManagedClusterClientConfigs[0].URL == "" {
		return ctrl.Result{}, giterrors.WithStack(fmt.Errorf("api url not found for cluster %s", decision.ClusterName))
	}

	//TODO: Remove this when label will be available in managedcluster to detect hypershift or not
	isHypershift, err := helpers.IsHypershiftCluster(r.Client, mc.Name)
	if err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	switch strategy.Spec.Type {
	case identitatemv1alpha1.HypershiftStrategyType:
		if isHypershift {
			cm, result, err := r.getOAuthOpenshift(decision.ClusterName)
			if err != nil {
				return result, giterrors.WithStack(err)
			}
			// cm := &corev1.ConfigMap{}
			// if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: "oauth-openshift", Namespace: "clusters-" + decision.ClusterName}, cm); err != nil {
			// 	return giterrors.WithStack(err)
			// }
			var ok bool
			var configYaml string
			if configYaml, ok = cm.Data["config.yaml"]; !ok {
				return ctrl.Result{}, giterrors.WithStack(fmt.Errorf("config.yaml not found for cluster %s", decision.ClusterName))
			}
			m := make(map[string]interface{})
			if err := yaml.Unmarshal([]byte(configYaml), &m); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			var ioauthConfig interface{}
			if ioauthConfig, ok = m["oauthConfig"]; !ok {
				return ctrl.Result{}, giterrors.WithStack(fmt.Errorf("config.yaml/oauthConfig not found for cluster %s", decision.ClusterName))
			}
			oauthConfig := ioauthConfig.(map[string]interface{})
			var imasterPublicURL interface{}
			if imasterPublicURL, ok = oauthConfig["masterPublicURL"]; !ok {
				return ctrl.Result{}, giterrors.WithStack(fmt.Errorf("config.yaml/oauthConfig not found for cluster %s", decision.ClusterName))
			}
			redirectURI := fmt.Sprintf("%s/oauth2callback/%s", imasterPublicURL.(string), authRealm.Name)

			dexClient.Spec.RedirectURIs = []string{redirectURI}
		}
	case identitatemv1alpha1.BackplaneStrategyType:
		if !isHypershift {
			u, err := url.Parse(mc.Spec.ManagedClusterClientConfigs[0].URL)
			if err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}

			host, _, err := net.SplitHostPort(u.Host)
			if err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}

			host = strings.Replace(host, "api", "apps", 1)

			redirectURI := fmt.Sprintf("%s://oauth-openshift.%s/oauth2callback/%s", u.Scheme, host, authRealm.Name)

			dexClient.Spec.RedirectURIs = []string{redirectURI}
		}
	default:
		return ctrl.Result{}, giterrors.WithStack(fmt.Errorf("unsupported strategy %s for cluster %s", strategy.Spec.Type, decision.ClusterName))
	}
	switch dexClientExists {
	case true:
		return ctrl.Result{}, giterrors.WithStack(r.Client.Update(context.TODO(), dexClient))
	case false:
		if err := r.Client.Create(context.Background(), dexClient); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
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
			return ctrl.Result{}, giterrors.WithStack(err)
		}

	}

	r.Log.Info("after update", "dexClient", dexClient)
	return ctrl.Result{}, nil
}

func (r *PlacementDecisionReconciler) getOAuthOpenshift(clusterName string) (cm *corev1.ConfigMap, result ctrl.Result, err error) {
	hypershiftServer := r.getHypershiftServer(clusterName)
	r.Log.Info("create managedclusterview", "name", helpers.ManagedClusterViewOAuthOpenshiftName(), "namespace", hypershiftServer)
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).Build()

	readerResources := resources.GetScenarioResourcesReader()

	file := "managedclusterview/managed-cluster-view-oauth-openshift.yaml"

	values := struct {
		Name              string
		Namespace         string
		ResourceNamespace string
	}{
		Name:              helpers.ManagedClusterViewOAuthOpenshiftName(),
		Namespace:         hypershiftServer,
		ResourceNamespace: helpers.ManagedClusterViewOAuthOpenshiftNamespace(clusterName),
	}
	if _, err := applier.ApplyCustomResources(readerResources, values, false, "", file); err != nil {
		return cm, ctrl.Result{}, err
	}

	mcvOAuthOpenshift := &viewv1beta1.ManagedClusterView{}
	r.Log.Info("check if managedclusterview exists", "name", helpers.ManagedClusterViewOAuthOpenshiftName(), "namespace", hypershiftServer)
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ManagedClusterViewOAuthOpenshiftName(), Namespace: hypershiftServer}, mcvOAuthOpenshift); err != nil {
		return cm, ctrl.Result{}, err
	}

	r.Log.Info("check if managedclusterview has results", "name", helpers.ManagedClusterViewOAuthOpenshiftName(), "namespace", hypershiftServer)
	condition := meta.FindStatusCondition(mcvOAuthOpenshift.Status.Conditions, viewv1beta1.ConditionViewProcessing)
	if condition.Status == metav1.ConditionFalse {
		return cm, ctrl.Result{RequeueAfter: 10 * time.Second}, fmt.Errorf("waiting for cluster %s oauth", hypershiftServer)
	}

	cm = &corev1.ConfigMap{}
	r.Log.Info("convert to configmap containing the OAuthOpenshift", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", hypershiftServer)
	if err := json.Unmarshal(mcvOAuthOpenshift.Status.Result.Raw, cm); err != nil {
		return cm, ctrl.Result{}, err
	}

	// cm = mcvOAuthOpenshift.Status.Result.(*corev1.ConfigMap)
	r.Log.Info("delete managedclusterview", "name", helpers.ManagedClusterViewOAuthOpenshiftName(), "namespace", hypershiftServer)
	if err := r.Client.Delete(context.TODO(), mcvOAuthOpenshift); err != nil && !errors.IsNotFound(err) {
		return cm, ctrl.Result{}, err
	}
	return cm, ctrl.Result{}, nil

}

func (r *PlacementDecisionReconciler) getHypershiftServer(clusterName string) string {
	return "local-cluster"
}

func (r *PlacementDecisionReconciler) deleteConfig(authRealm *identitatemv1alpha1.AuthRealm,
	dexClientName,
	clusterName string) error {
	r.Log.Info("delete configuration for cluster", "name", clusterName)
	//Delete DexClient
	r.Log.Info("get dexclient", "namespace", authRealm.Name, "name", dexClientName)
	dexClient := &dexoperatorv1alpha1.DexClient{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: dexClientName, Namespace: helpers.DexServerNamespace(authRealm)}, dexClient)
	switch {
	case err == nil:
		if err := r.Delete(context.TODO(), dexClient); err != nil {
			return giterrors.WithStack(err)
		}
	case !errors.IsNotFound(err):
		return giterrors.WithStack(err)
	}
	//Delete ClientSecret
	r.Log.Info("delete clientSecret", "namespace", clusterName, "name", authRealm.Name)
	clientSecret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ClientSecretName(authRealm), Namespace: clusterName}, clientSecret)
	switch {
	case err == nil:
		if err = r.Delete(context.TODO(), clientSecret); err != nil {
			return giterrors.WithStack(err)
		}
	case !errors.IsNotFound(err):
		return giterrors.WithStack(err)
	}
	//Delete clusterOAuth
	r.Log.Info("delete clusterOAuth", "Namespace", clusterName, "Name", authRealm.Name)
	clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.ClusterOAuthName(authRealm), Namespace: clusterName}, clusterOAuth)
	switch {
	case err == nil:
		if err = r.Delete(context.TODO(), clusterOAuth); err != nil {
			return giterrors.WithStack(err)
		}
	case !errors.IsNotFound(err):
		return giterrors.WithStack(err)
	}
	return nil
}
