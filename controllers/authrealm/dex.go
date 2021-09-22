// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"
	"os"

	identitatemdexserverv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/deploy"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dexOperatorImageEnvName string = "RELATED_IMAGE_DEX_OPERATOR"
)

func (r *AuthRealmReconciler) syncDexCRs(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("syncDexCRs", "AuthRealm.Name", authRealm.Name, "AuthRealm.Namespace", authRealm.Namespace)
	//TODO this test should maybe be in a webhook to avoid the creation of an invalid CR
	if len(authRealm.Spec.IdentityProviders) != 1 {
		return fmt.Errorf("the identityproviders array of the authrealm %s can have one and only one element", authRealm.Name)
	}

	// Create namespace and Install the dex-operator
	if err := r.installDexOperator(authRealm); err != nil {
		return err
	}
	//Create DexServer CR
	if err := r.createDexServer(authRealm); err != nil {
		return err
	}
	return nil
}

func (r *AuthRealmReconciler) installDexOperator(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("installDexOperator", "Name", authRealm.Name, "Namespace", authRealm.Name)

	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.
		WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
		WithTemplateFuncMap(FuncMap()).
		Build()

	dexOperatorImage := os.Getenv(dexOperatorImageEnvName)
	if len(dexOperatorImage) == 0 {
		return fmt.Errorf("EnvVar %s not provided", dexOperatorImageEnvName)
	}

	//Create the namespace
	readerDeploy := deploy.GetScenarioResourcesReader()
	readerDexOperator := dexoperatorconfig.GetScenarioResourcesReader()

	values := struct {
		Image              string
		Reader             *clusteradmasset.ScenarioResourcesReader
		File               string
		NewName            string
		FileLeader         string
		NewNameLeader      string
		NewNamespaceLeader string
	}{
		Image:              dexOperatorImage,
		Reader:             readerDexOperator,
		File:               "rbac/role.yaml",
		NewName:            "dex-operator-manager-role",
		FileLeader:         "rbac/leader_election_role.yaml",
		NewNameLeader:      "dex-operator-leader-election-role",
		NewNamespaceLeader: helpers.DexOperatorNamespace(),
	}

	files := []string{
		"dex-operator/namespace.yaml",
		"dex-operator/service_account.yaml",
		"dex-operator/role.yaml",
		"dex-operator/role_binding.yaml",
		"dex-operator/leader_election_role.yaml",
		"dex-operator/leader_election_role_binding.yaml",
	}

	_, err := applier.ApplyDirectly(readerDeploy, values, false, "", files...)
	if err != nil {
		return err
	}

	_, err = applier.ApplyDeployments(readerDeploy, values, false, "", "dex-operator/manager.yaml")
	if err != nil {
		return err
	}

	return nil
}
func (r *AuthRealmReconciler) deleteDexOperator(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("deleteDexOperator", "Name", authRealm.Name, "Namespace", authRealm.Name)
	authRealms := &identitatemv1alpha1.AuthRealmList{}
	if err := r.Client.List(context.TODO(), authRealms); err != nil {
		return err
	}
	nbFound := 0
	for _, authRealm := range authRealms.Items {
		if authRealm.Spec.Type == identitatemv1alpha1.AuthProxyDex {
			nbFound++
		}
	}
	if nbFound == 1 {
		//Delete dex-operator ns
		ns := &corev1.Namespace{}
		if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexOperatorNamespace()}, ns); err == nil {
			err := r.Client.Delete(context.TODO(), ns)
			if err != nil {
				return err
			}
		}
		//Delete clusterRoleBinding
		crb := &rbacv1.ClusterRoleBinding{}
		if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: "dex-operator-rolebinding"}, crb); err == nil {
			err := r.Client.Delete(context.TODO(), crb)
			if err != nil {
				return err
			}
		}
		//Delete clusterRole
		cr := &rbacv1.ClusterRole{}
		if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: "dex-operator-manager-role"}, cr); err == nil {
			err := r.Client.Delete(context.TODO(), cr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *AuthRealmReconciler) installDexOperatorCRDs() error {
	r.Log.Info("installDexCRDs")

	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.
		WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
		WithTemplateFuncMap(FuncMap()).
		Build()

	readerIDPStrategyOperator := dexoperatorconfig.GetScenarioResourcesReader()
	files := []string{"crd/bases/auth.identitatem.io_dexclients.yaml",
		"crd/bases/auth.identitatem.io_dexservers.yaml"}

	_, err := applier.ApplyDirectly(readerIDPStrategyOperator, nil, false, "", files...)
	if err != nil {
		return err
	}

	return nil
}

func (r *AuthRealmReconciler) createDexServer(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("createDexServer", "Name", helpers.DexServerName(), "Namespace", helpers.DexServerNamespace(authRealm))
	//Create namespace if not exists
	dexServerNamespace := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexServerNamespace(authRealm)}, dexServerNamespace); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dexServerNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: helpers.DexServerNamespace(authRealm),
			},
		}
		if err := r.Client.Create(context.TODO(), dexServerNamespace); err != nil {
			return err
		}
	}
	dexServerExists := true
	dexServer := &identitatemdexserverv1alpha1.DexServer{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexServerName(), Namespace: helpers.DexServerNamespace(authRealm)}, dexServer); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dexServerExists = false
		dexServer = &identitatemdexserverv1alpha1.DexServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      helpers.DexServerName(),
				Namespace: helpers.DexServerNamespace(authRealm),
			},
		}
	}

	if err := r.updateDexServer(authRealm, dexServer); err != nil {
		return err
	}

	switch dexServerExists {
	case true:
		r.Log.V(1).Info("createDexServer update dexServer", "Name", dexServer.Name, "Namespace", dexServer.Namespace)
		return r.Client.Update(context.TODO(), dexServer)
	case false:
		r.Log.V(1).Info("createDexServer create dexServer", "Name", dexServer.Name, "Namespace", dexServer.Namespace)
		if err := r.Client.Create(context.TODO(), dexServer); err != nil {
			return err
		}
		dexServer.Status.RelatedObjects =
			[]identitatemdexserverv1alpha1.RelatedObjectReference{
				{
					Kind:      "AuthRealm",
					Name:      authRealm.Name,
					Namespace: authRealm.Namespace,
				},
			}
		if err := r.Status().Update(context.TODO(), dexServer); err != nil {
			return err
		}
	}
	r.Log.Info("after update", "dexServer", dexServer)
	return nil
}

func (r *AuthRealmReconciler) updateDexServer(authRealm *identitatemv1alpha1.AuthRealm, dexServer *identitatemdexserverv1alpha1.DexServer) error {
	r.Log.Info("updateDexServer", "Name", dexServer.Name, "Namespace", dexServer.Namespace)
	uScheme, host, err := helpers.GetAppsURL(r.Client, false)
	if err != nil {
		return err
	}
	dexServer.Spec.Issuer = fmt.Sprintf("%s://%s.%s", uScheme, authRealm.Spec.RouteSubDomain, host)
	if len(authRealm.Spec.CertificatesSecretRef.Name) != 0 {
		certSecret := &corev1.Secret{}
		if err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: authRealm.Spec.CertificatesSecretRef.Name, Namespace: authRealm.Namespace},
			certSecret); err != nil {
			return err
		}
		dexServer.Spec.Web.TlsCert = string(certSecret.Data["tls.crt"])
		dexServer.Spec.Web.TlsKey = string(certSecret.Data["tls.key"])
	}
	cs, err := r.createDexConnectors(authRealm, dexServer)
	if err != nil {
		return err
	}
	dexServer.Spec.Connectors = cs
	return nil
}

func (r *AuthRealmReconciler) createDexConnectors(authRealm *identitatemv1alpha1.AuthRealm,
	dexServer *identitatemdexserverv1alpha1.DexServer) (cs []identitatemdexserverv1alpha1.ConnectorSpec, err error) {
	r.Log.Info("createDexConnectors", "Name", dexServer.Name, "Namespace", dexServer.Namespace)

	cs = make([]identitatemdexserverv1alpha1.ConnectorSpec, 0)
	for _, idp := range authRealm.Spec.IdentityProviders {
		switch idp.Type {
		case openshiftconfigv1.IdentityProviderTypeGitHub:
			c, err := r.createConnector(authRealm, identitatemdexserverv1alpha1.ConnectorTypeGitHub, idp.GitHub.ClientID, idp.GitHub.ClientSecret.Name)
			if err != nil {
				return nil, err
			}
			c.Config.RedirectURI = dexServer.Spec.Issuer + "/callback"
			cs = append(cs, *c)
		case openshiftconfigv1.IdentityProviderTypeLDAP:
			c, err := r.createConnector(authRealm, identitatemdexserverv1alpha1.ConnectorTypeLDAP, "", idp.LDAP.BindPassword.Name)
			if err != nil {
				return nil, err
			}
			cs = append(cs, *c)
		default:
			return nil, fmt.Errorf("unsupported provider type %s", idp.Type)
		}
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("no identityProvider defined in %s/%s", authRealm.Name, authRealm.Name)
	}
	return cs, err
}

func (r *AuthRealmReconciler) createConnector(authRealm *identitatemv1alpha1.AuthRealm,
	identityProviderType identitatemdexserverv1alpha1.ConnectorType, clientID, clientSecretName string) (c *identitatemdexserverv1alpha1.ConnectorSpec, err error) {

	c = &identitatemdexserverv1alpha1.ConnectorSpec{
		Type: identityProviderType,
		Name: authRealm.Name,
		Id:   authRealm.Name,
		Config: identitatemdexserverv1alpha1.ConfigSpec{
			ClientID: clientID,
			ClientSecretRef: corev1.SecretReference{
				Name:      clientSecretName,
				Namespace: authRealm.Namespace,
			},
		},
	}

	return c, nil
}

func (r *AuthRealmReconciler) deleteAuthRealmNamespace(authRealm *identitatemv1alpha1.AuthRealm) error {
	ns := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexServerNamespace(authRealm)}, ns); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.Client.Delete(context.TODO(), ns); err != nil {
		return err
	}
	return r.deleteDexOperator(authRealm)
}
