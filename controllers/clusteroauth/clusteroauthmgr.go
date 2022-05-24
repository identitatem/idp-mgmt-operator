// Copyright Red Hat

package clusteroauth

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/ghodss/yaml"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	giterrors "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpmgmtconfig "github.com/identitatem/idp-mgmt-operator/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
)

type ClusterOAuthMgr interface {
	//Push the OAuth configuration to the targeted cluster
	Push() (err error)
	//Save the original OAuth configuration from the targeted cluster
	Save() (result ctrl.Result, err error)
	//unmanage removes the OAuth configuration and restore the old configuration
	Unmanage() (result ctrl.Result, err error)
}

type BackplaneMgr struct {
	Reconciler   *ClusterOAuthReconciler
	ClusterOAuth *identitatemv1alpha1.ClusterOAuth
}

type HypershiftMgr struct {
	Reconciler   *ClusterOAuthReconciler
	ClusterOAuth *identitatemv1alpha1.ClusterOAuth
}

var _ BackplaneMgr
var _ HypershiftMgr

func (r *ClusterOAuthReconciler) getOriginalOAuth(clusterOAuth *identitatemv1alpha1.ClusterOAuth) (*corev1.ConfigMap, error) {
	originalOAuth := &corev1.ConfigMap{}
	err := r.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: clusterOAuth.Namespace},
		originalOAuth)
	return originalOAuth, err
}

func (r *ClusterOAuthReconciler) deleteOriginalOAuth(ns string) error {
	cm := &corev1.ConfigMap{}
	r.Log.Info("check if configmap already exists", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", ns)
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.ConfigMapOriginalOAuthName(), Namespace: ns},
		cm); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		//nothing to do as already deleted
		return nil
	}

	if err := r.Delete(context.TODO(), cm); err != nil {
		return giterrors.WithStack(err)
	}
	r.Log.Info("configmap deleted", "name", helpers.ConfigMapOriginalOAuthName(), "namespace", ns)
	return nil
}

func (r *ClusterOAuthReconciler) aggregateClusterOAuths(clusterOAuths *identitatemv1alpha1.ClusterOAuthList,
	namespace string) *openshiftconfigv1.OAuth {
	//	singleOAuth := &openshiftconfigv1.OAuth{}
	singleOAuth := &openshiftconfigv1.OAuth{
		TypeMeta: metav1.TypeMeta{
			APIVersion: openshiftconfigv1.SchemeGroupVersion.String(),
			Kind:       "OAuth",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},

		Spec: openshiftconfigv1.OAuthSpec{},
	}

	for _, clusterOAuth := range clusterOAuths.Items {
		if clusterOAuth.DeletionTimestamp != nil {
			continue
		}
		//build OAuth and add to manifest work
		r.Log.Info(" build clusterOAuth",
			"name: ", clusterOAuth.GetName(),
			" namespace:", clusterOAuth.GetNamespace(),
			"identityProviders:", len(clusterOAuth.Spec.OAuth.Spec.IdentityProviders))

		for j, idp := range clusterOAuth.Spec.OAuth.Spec.IdentityProviders {

			r.Log.Info("process identityProvider", "identityProvider  ", j, " name:", idp.Name)

			idp.OpenID.ClientSecret.Name = clusterOAuth.GetNamespace() + "-" + idp.OpenID.ClientSecret.Name
			//build oauth by appending first clusterOAuth entry into single OAuth
			singleOAuth.Spec.IdentityProviders = append(singleOAuth.Spec.IdentityProviders, idp)
		}
	}
	return singleOAuth
}

func (r *ClusterOAuthReconciler) generateAggregatedRole() ([]byte, error) {
	r.Log.Info("add aggregated role")
	aggregatedRoleYaml, err := idpmgmtconfig.GetScenarioResourcesReader().Asset("rbac/role-aggregated-clusterrole.yaml")
	if err != nil {
		return nil, giterrors.WithStack(err)
	}

	aggregatedRoleJson, err := yaml.YAMLToJSON(aggregatedRoleYaml)
	if err != nil {
		return nil, giterrors.WithStack(err)
	}
	return aggregatedRoleJson, nil
}
