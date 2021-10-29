// Copyright Red Hat

package authrealm

import (
	"context"
	"fmt"
	"os"
	"time"

	dexoperatorv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/deploy"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	giterrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	dexOperatorImageEnvName string = "RELATED_IMAGE_DEX_OPERATOR"
	dexServerImageEnvName   string = "RELATED_IMAGE_DEX_SERVER"
)

func (r *AuthRealmReconciler) syncDexCRs(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("syncDexCRs", "AuthRealm.Name", authRealm.Name, "AuthRealm.Namespace", authRealm.Namespace)
	if len(authRealm.Spec.IdentityProviders) < 1 {
		return giterrors.WithStack(
			fmt.Errorf("the identityproviders array of the authrealm %s can have one and only one element",
				authRealm.Name))
	}

	// Create namespace and Install the dex-operator
	if err := r.installDexOperator(authRealm); err != nil {
		r.Log.Info("Update status create dexOperator failure",
			"name", "dex-operator",
			"namespace", helpers.DexOperatorNamespace(),
			"error", err.Error())
		cond := metav1.Condition{
			Type:    identitatemv1alpha1.AuthRealmApplied,
			Status:  metav1.ConditionFalse,
			Reason:  "AuthRealmAppliedFailed",
			Message: fmt.Sprintf("failed to install installDexOperator error: %s", err.Error()),
		}
		if err := helpers.UpdateAuthRealmStatusConditions(r.Client, authRealm, cond); err != nil {
			return err
		}
		return err
	}
	//Create DexServer CR
	if err := r.createDexServer(authRealm); err != nil {
		r.Log.Info("Update status create dexServer CR failure",
			"name", helpers.DexServerName(),
			"namespace", helpers.DexServerNamespace(authRealm),
			"error", err.Error())
		cond := metav1.Condition{
			Type:   identitatemv1alpha1.AuthRealmApplied,
			Status: metav1.ConditionFalse,
			Reason: "AuthRealmAppliedFailed",
			Message: fmt.Sprintf("failed to create dexServer name: %s namespace: %s error: %s",
				helpers.DexServerName(),
				helpers.DexServerNamespace(authRealm),
				err.Error()),
		}
		if err := helpers.UpdateAuthRealmStatusConditions(r.Client, authRealm, cond); err != nil {
			return err
		}
		return err
	}
	return nil
}

func (r *AuthRealmReconciler) installDexOperator(authRealm *identitatemv1alpha1.AuthRealm) error {
	r.Log.Info("installDexOperator", "Name", "dex-operator", "Namespace", helpers.DexOperatorNamespace())

	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.
		WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
		WithTemplateFuncMap(FuncMap()).
		Build()

	dexOperatorImage := os.Getenv(dexOperatorImageEnvName)
	if len(dexOperatorImage) == 0 {
		return giterrors.WithStack(fmt.Errorf("EnvVar %s not provided", dexOperatorImageEnvName))
	}

	dexServerImage := os.Getenv(dexServerImageEnvName)
	if len(dexServerImage) == 0 {
		return giterrors.WithStack(fmt.Errorf("EnvVar %s not provided", dexServerImageEnvName))
	}

	//Create the namespace
	readerDeploy := deploy.GetScenarioResourcesReader()
	readerDexOperator := dexoperatorconfig.GetScenarioResourcesReader()

	values := struct {
		Image              string
		DexServerImage     string
		Reader             *clusteradmasset.ScenarioResourcesReader
		File               string
		NewName            string
		FileLeader         string
		NewNameLeader      string
		NewNamespaceLeader string
	}{
		Image:              dexOperatorImage,
		DexServerImage:     dexServerImage,
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
		return giterrors.WithStack(err)
	}

	_, err = applier.ApplyDeployments(readerDeploy, values, false, "", "dex-operator/manager.yaml")
	if err != nil {
		return giterrors.WithStack(err)
	}

	return nil
}

func (r *AuthRealmReconciler) deleteDexOperator(authRealm *identitatemv1alpha1.AuthRealm) (ctrl.Result, error) {
	r.Log.Info("deleteDexOperator", "Name", authRealm.Name, "Namespace", authRealm.Name)
	authRealms := &identitatemv1alpha1.AuthRealmList{}
	if err := r.Client.List(context.TODO(), authRealms); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
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
		err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexOperatorNamespace()}, ns)
		switch {
		case err == nil:
			if err := r.Client.Delete(context.TODO(), ns); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			r.Log.Info("waiting dexoperator to be deleted", "name", helpers.DexOperatorNamespace(), "namespace", ns)
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}

		//Delete clusterRoleBinding
		crb := &rbacv1.ClusterRoleBinding{}
		err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "dex-operator-rolebinding"}, crb)
		switch {
		case err == nil:
			if err := r.Client.Delete(context.TODO(), crb); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			r.Log.Info("waiting clusterRoleBiding to be deleted", "name", "dex-operator-rolebinding")
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}

		//Delete clusterRole
		cr := &rbacv1.ClusterRole{}
		err = r.Client.Get(context.TODO(), client.ObjectKey{Name: "dex-operator-manager-role"}, cr)
		switch {
		case err == nil:
			if err := r.Client.Delete(context.TODO(), cr); err != nil {
				return ctrl.Result{}, giterrors.WithStack(err)
			}
			r.Log.Info("waiting clusterRole to be deleted", "name", "dex-operator-manager-role")
			return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
		case !errors.IsNotFound(err):
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}
	return ctrl.Result{}, nil
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
	r.Log.Info("createDexServer", "Name", authRealm.Name, "Namespace", helpers.DexServerNamespace(authRealm))
	//Create namespace if not exists
	dexServerNamespace := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.DexServerNamespace(authRealm)},
		dexServerNamespace); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		dexServerNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: helpers.DexServerNamespace(authRealm),
			},
		}
		if err := r.Client.Create(context.TODO(), dexServerNamespace); err != nil {
			return giterrors.WithStack(err)
		}
	}
	dexServerExists := true
	dexServer := &dexoperatorv1alpha1.DexServer{}
	if err := r.Client.Get(context.TODO(),
		client.ObjectKey{Name: helpers.DexServerName(), Namespace: helpers.DexServerNamespace(authRealm)},
		dexServer); err != nil {
		if !errors.IsNotFound(err) {
			return giterrors.WithStack(err)
		}
		dexServerExists = false
		dexServer = &dexoperatorv1alpha1.DexServer{
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
		return giterrors.WithStack(r.Client.Update(context.TODO(), dexServer))
	case false:
		r.Log.V(1).Info("createDexServer create dexServer", "Name", dexServer.Name, "Namespace", dexServer.Namespace)
		if err := r.Client.Create(context.TODO(), dexServer); err != nil {
			return giterrors.WithStack(err)
		}
		dexServer.Status.RelatedObjects =
			[]dexoperatorv1alpha1.RelatedObjectReference{
				{
					Kind:      "AuthRealm",
					Name:      authRealm.Name,
					Namespace: authRealm.Namespace,
				},
			}
		if err := r.Status().Update(context.TODO(), dexServer); err != nil {
			return giterrors.WithStack(err)
		}
	}
	r.Log.Info("after update", "dexServer", dexServer)
	return nil
}

func (r *AuthRealmReconciler) updateDexServer(authRealm *identitatemv1alpha1.AuthRealm, dexServer *dexoperatorv1alpha1.DexServer) error {
	r.Log.Info("updateDexServer", "Name", dexServer.Name, "Namespace", dexServer.Namespace)
	uScheme, host, err := helpers.GetAppsURL(r.Client, false)
	if err != nil {
		return err
	}
	dexServer.Spec.Issuer = fmt.Sprintf("%s://%s.%s", uScheme, authRealm.Name+"-"+authRealm.Spec.RouteSubDomain, host)
	if len(authRealm.Spec.CertificatesSecretRef.Name) != 0 {
		certSecret := &corev1.Secret{}
		if err := r.Client.Get(context.TODO(),
			client.ObjectKey{Name: authRealm.Spec.CertificatesSecretRef.Name, Namespace: authRealm.Namespace},
			certSecret); err != nil {
			return giterrors.WithStack(err)
		}
		//Copy the secret from the authrealm to the dexserver namespace
		dexServerCertSecret := &corev1.Secret{}
		if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: certSecret.Name, Namespace: dexServer.Namespace},
			dexServerCertSecret); err != nil {
			if !errors.IsNotFound(err) {
				return giterrors.WithStack(err)
			}
			dexServerCertSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      certSecret.Name,
					Namespace: dexServer.Namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: certSecret.Data,
			}
			if err := r.Client.Create(context.TODO(), dexServerCertSecret); err != nil {
				return giterrors.WithStack(err)
			}
		} else {
			dexServerCertSecret.Data = certSecret.Data
			if err := r.Client.Update(context.TODO(), dexServerCertSecret); err != nil {
				return giterrors.WithStack(err)
			}
		}
		dexServer.Spec.IngressCertificateRef = corev1.LocalObjectReference{
			Name: dexServerCertSecret.Name,
		}
	}
	cs, err := r.createDexConnectors(authRealm, dexServer)
	if err != nil {
		return err
	}
	dexServer.Spec.Connectors = cs
	r.Log.Info("Updated dexserver", "dexServer:", dexServer)
	return nil
}

func (r *AuthRealmReconciler) createDexConnectors(authRealm *identitatemv1alpha1.AuthRealm,
	dexServer *dexoperatorv1alpha1.DexServer) (cs []dexoperatorv1alpha1.ConnectorSpec, err error) {
	r.Log.Info("createDexConnectors", "Name", dexServer.Name, "Namespace", dexServer.Namespace)

	cs = make([]dexoperatorv1alpha1.ConnectorSpec, 0)
	for _, idp := range authRealm.Spec.IdentityProviders {
		switch idp.Type {
		case openshiftconfigv1.IdentityProviderTypeGitHub:
			r.Log.Info("create connector for GitHub")
			c := &dexoperatorv1alpha1.ConnectorSpec{
				Type: dexoperatorv1alpha1.ConnectorTypeGitHub,
				Name: idp.Name,
				Id:   idp.Name,
				GitHub: dexoperatorv1alpha1.GitHubConfigSpec{
					ClientID: idp.GitHub.ClientID,
					ClientSecretRef: corev1.SecretReference{
						Name:      idp.GitHub.ClientSecret.Name,
						Namespace: authRealm.Namespace,
					},
					RedirectURI: dexServer.Spec.Issuer + "/callback",
				},
			}
			c.GitHub.Orgs = make([]dexoperatorv1alpha1.Org, len(idp.GitHub.Organizations))
			for i, org := range idp.GitHub.Organizations {
				c.GitHub.Orgs[i].Name = org
			}
			r.Log.Info("generated connector", "c.GitHub", c.GitHub)
			cs = append(cs, *c)
			r.Log.Info("generated intermediate connectors", "cs", cs)
		case openshiftconfigv1.IdentityProviderTypeLDAP:
			r.Log.Info("create connector for LDAP")
			c := &dexoperatorv1alpha1.ConnectorSpec{
				Type: dexoperatorv1alpha1.ConnectorTypeLDAP,
				Name: idp.Name,
				Id:   idp.Name,
				LDAP: dexoperatorv1alpha1.LDAPConfigSpec{
					Host:   idp.LDAP.URL,
					BindDN: idp.LDAP.BindDN,
					BindPWRef: corev1.SecretReference{
						Name:      idp.LDAP.BindPassword.Name,
						Namespace: authRealm.Namespace,
					},
					InsecureNoSSL:      false,
					InsecureSkipVerify: idp.LDAP.Insecure,
					UsernamePrompt:     "Email Address",
					RootCARef: corev1.SecretReference{
						Name:      idp.LDAP.CA.Name,
						Namespace: authRealm.Namespace,
					},
					UserSearch: dexoperatorv1alpha1.UserSearchSpec{
						BaseDN:    authRealm.Spec.LDAPExtraConfigs[idp.Name].BaseDN,
						Filter:    authRealm.Spec.LDAPExtraConfigs[idp.Name].Filter,
						Username:  idp.LDAP.Attributes.PreferredUsername[0],
						IDAttr:    idp.LDAP.Attributes.ID[0],
						EmailAttr: idp.LDAP.Attributes.Email[0],
						NameAttr:  idp.LDAP.Attributes.Name[0],
					},
				},
			}
			r.Log.Info("generated connector", "c.LDAP", c.LDAP)
			cs = append(cs, *c)
			fmt.Println("cs: ", cs)
			r.Log.Info("generated intermediate connextors", "cs", cs)
		default:
			return nil, giterrors.WithStack(fmt.Errorf("unsupported provider type %s", idp.Type))
		}
	}
	if len(cs) == 0 {
		return nil, giterrors.WithStack(fmt.Errorf("no identityProvider defined in %s/%s", authRealm.Name, authRealm.Name))
	}
	r.Log.Info("generated connectors", "cs", cs)
	return cs, giterrors.WithStack(err)
}

func (r *AuthRealmReconciler) processAuthRealmDeletion(authRealm *identitatemv1alpha1.AuthRealm) (ctrl.Result, error) {
	if r, err := r.processDexServerDeletion(authRealm); err != nil || r.Requeue {
		return r, err
	}
	r.Log.Info("delete Strategy", "name", helpers.StrategyName(authRealm, identitatemv1alpha1.BackplaneStrategyType))
	st := &identitatemv1alpha1.Strategy{}
	err := r.Client.Get(context.TODO(),
		client.ObjectKey{
			Name:      helpers.StrategyName(authRealm, identitatemv1alpha1.BackplaneStrategyType),
			Namespace: authRealm.Namespace},
		st)
	switch {
	case err == nil:
		if err := r.Client.Delete(context.TODO(), st); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		r.Log.Info("waiting strategy to be deleted",
			"name", helpers.StrategyName(authRealm, identitatemv1alpha1.BackplaneStrategyType),
			"namespace", authRealm.Namespace)
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	case !errors.IsNotFound(err):
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	r.Log.Info("deleted Strategy", "name", helpers.StrategyName(authRealm, identitatemv1alpha1.BackplaneStrategyType))
	r.Log.Info("delete DexOperator")
	return r.deleteDexOperator(authRealm)
}

func (r *AuthRealmReconciler) processDexServerDeletion(authRealm *identitatemv1alpha1.AuthRealm) (ctrl.Result, error) {
	r.Log.Info("delete DexClients in ns", "namespace", helpers.DexServerNamespace(authRealm))
	ldc := &dexoperatorv1alpha1.DexClientList{}
	if err := r.Client.List(context.TODO(),
		ldc,
		&client.ListOptions{Namespace: helpers.DexServerNamespace(authRealm)}); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	for _, dc := range ldc.Items {
		dcc := dc.DeepCopy()
		if err := r.Client.Delete(context.TODO(), dcc); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}
	if len(ldc.Items) != 0 {
		r.Log.Info("waiting dexclients to be deleted")
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	}

	r.Log.Info("delete DexServers in ns", "namespace", helpers.DexServerNamespace(authRealm))
	lds := &dexoperatorv1alpha1.DexServerList{}
	if err := r.Client.List(context.TODO(), lds, &client.ListOptions{Namespace: helpers.DexServerNamespace(authRealm)}); err != nil {
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	for _, ds := range lds.Items {
		dsc := ds.DeepCopy()
		if err := r.Client.Delete(context.TODO(), dsc); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
	}
	if len(lds.Items) != 0 {
		r.Log.Info("waiting dexservers to be deleted")
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	}

	r.Log.Info("delete DexServer ns", "namespace", helpers.DexServerNamespace(authRealm))
	ns := &corev1.Namespace{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Name: helpers.DexServerNamespace(authRealm)}, ns)
	switch {
	case err == nil:
		if err := r.Client.Delete(context.TODO(), ns); err != nil {
			return ctrl.Result{}, giterrors.WithStack(err)
		}
		r.Log.Info("waiting dexserver ns to be deleted")
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Second}, nil
	case !errors.IsNotFound(err):
		return ctrl.Result{}, giterrors.WithStack(err)
	}
	return ctrl.Result{}, nil
}
