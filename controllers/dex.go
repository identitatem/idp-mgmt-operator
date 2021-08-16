// Copyright Contributors to the Open Cluster Management project

package controllers

import (
	"context"
	"fmt"

	identitatemdexserverv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemmgmtv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *AuthRealmReconciler) syncDexCRs(authRealm *identitatemmgmtv1alpha1.AuthRealm) error {
	r.Log.Info("syncDexCRs", "AuthRealm.Name", authRealm.Name, "AuthRealm.Namespace", authRealm.Namespace)
	//TODO this test should maybe be in a webhook to avoid the creation of an invalid CR
	if len(authRealm.Spec.IdentityProviders) != 1 {
		return fmt.Errorf("the identityproviders array of the authrealm %s can have only and only one element", authRealm.Name)
	}
	r.Log.Info("syncDexCRs manage dex namespace", "Name", authRealm.Name)
	ns := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authRealm.Name}, ns); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: authRealm.Name,
			},
		}
		r.Log.V(1).Info("syncDexCRs create namespace", "Name", authRealm.Name)
		if err := r.Client.Create(context.TODO(), ns); err != nil {
			return err
		}
	}

	r.Log.Info("syncDexCRs manage dexServer", "Name", authRealm.Name, "Namespace", authRealm.Name)
	dexServerExists := true
	dexServer := &identitatemdexserverv1alpha1.DexServer{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authRealm.Name, Namespace: authRealm.Name}, dexServer); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dexServerExists = false
		dexServer = &identitatemdexserverv1alpha1.DexServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      authRealm.Name,
				Namespace: authRealm.Name,
			},
		}
	}

	if err := r.updateDexServer(authRealm, dexServer); err != nil {
		return err
	}

	switch dexServerExists {
	case true:
		r.Log.V(1).Info("syncDexCRs update dexServer", "Name", authRealm.Name, "Namespace", authRealm.Name)
		if err := r.Client.Update(context.TODO(), dexServer); err != nil {
			return err
		}
	case false:
		r.Log.V(1).Info("syncDexCRs create dexServer", "Name", authRealm.Name, "Namespace", authRealm.Name)
		if err := r.Client.Create(context.TODO(), dexServer); err != nil {
			return err
		}
	}

	// r.Log.Info("syncDexCRs manage dexClient", "Name", authRealm.Name, "Namespace", authRealm.Name)
	// dexClientExists := true
	// dexClient := &identitatemdexserverv1alpha1.DexClient{}
	// if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authRealm.Name, Namespace: authRealm.Name}, dexClient); err != nil {
	// 	if !errors.IsNotFound(err) {
	// 		return err
	// 	}
	// 	dexClientExists = false
	// 	dexClient = &identitatemdexserverv1alpha1.DexClient{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name:      authRealm.Name,
	// 			Namespace: authRealm.Name,
	// 		},
	// 	}
	// }

	// if err := r.updateDexClient(authRealm); err != nil {
	// 	return err
	// }

	// switch dexClientExists {
	// case true:
	// 	r.Log.V(1).Info("syncDexCRs update dexClient", "Name", authRealm.Name, "Namespace", authRealm.Name)
	// 	if err := r.Client.Update(context.TODO(), dexClient); err != nil {
	// 		return err
	// 	}
	// case false:
	// 	r.Log.V(1).Info("syncDexCRs update dexClient", "Name", authRealm.Name, "Namespace", authRealm.Name)
	// 	if err := r.Client.Create(context.TODO(), dexClient); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

func (r *AuthRealmReconciler) updateDexServer(authRealm *identitatemmgmtv1alpha1.AuthRealm, dexServer *identitatemdexserverv1alpha1.DexServer) error {
	r.Log.Info("updateDexServer", "Name", authRealm.Name, "Namespace", authRealm.Name)
	dexServer.Spec.Issuer = authRealm.Spec.Host
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
	cs, err := r.createDexConnectors(authRealm)
	if err != nil {
		return err
	}
	dexServer.Spec.Connectors = cs
	return nil
}

func (r *AuthRealmReconciler) createDexConnectors(authRealm *identitatemmgmtv1alpha1.AuthRealm) (cs []identitatemdexserverv1alpha1.ConnectorSpec, err error) {
	r.Log.Info("createDexConnectors", "Name", authRealm.Name, "Namespace", authRealm.Name)
	idp := authRealm.Spec.IdentityProviders[0]
	cs = make([]identitatemdexserverv1alpha1.ConnectorSpec, 0)
	if idp.GitHub != nil {
		c, err := r.createConnector(authRealm, "github")
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	if idp.Google != nil {
		c, err := r.createConnector(authRealm, "google")
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	if idp.HTPasswd != nil {
		c, err := r.createConnector(authRealm, "htppasswd")
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	if idp.LDAP != nil {
		c, err := r.createConnector(authRealm, "ldap")
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	if idp.OpenID != nil {
		c, err := r.createConnector(authRealm, "oidc")
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	return cs, err
}

func (r *AuthRealmReconciler) createConnector(authRealm *identitatemmgmtv1alpha1.AuthRealm,
	identityProviderType string) (c *identitatemdexserverv1alpha1.ConnectorSpec, err error) {

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", authRealm.Name, identityProviderType),
			Namespace: authRealm.Name,
		},
		Data: map[string][]byte{
			"client-secret": []byte(helpers.RandStringRunes(32)),
		},
	}

	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: clientSecret.Name, Namespace: clientSecret.Namespace}, clientSecret); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := r.Client.Create(context.TODO(), clientSecret); err != nil {
			return nil, err
		}
	}

	c = &identitatemdexserverv1alpha1.ConnectorSpec{
		Type: identityProviderType,
		Name: authRealm.Name,
		Id:   authRealm.Name,
		Config: identitatemdexserverv1alpha1.ConfigSpec{
			ClientID:        authRealm.Name,
			ClientSecretRef: clientSecret.Name,
		},
	}

	return c, nil
}

func (r *AuthRealmReconciler) deleteAuthRealmNamespace(authRealm *identitatemmgmtv1alpha1.AuthRealm) error {
	ns := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), client.ObjectKey{Name: authRealm.Name}, ns); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.Client.Delete(context.TODO(), ns); err != nil {
		return err
	}
	return nil
}

// func (r *AuthRealmReconciler) updateDexClient(authRealm *identitatemmgmtv1alpha1.AuthRealm) error {
// 	return nil
// }
