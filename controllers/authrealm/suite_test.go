// Copyright Red Hat

package authrealm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	identitatemdexserverv1lapha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/deploy"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var clientSetMgmt *idpclientset.Clientset
var clientSetStrategy *idpclientset.Clientset
var k8sClient client.Client
var testEnv *envtest.Environment
var r AuthRealmReconciler

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	err := os.Setenv(dexOperatorImageEnvName, "dex_operator_inage")
	Expect(err).NotTo(HaveOccurred())
	err = os.Setenv(dexServerImageEnvName, "dex_server_inage")
	Expect(err).NotTo(HaveOccurred())
	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	err = identitatemdexserverv1lapha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	err = openshiftconfigv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	readerIDP := idpconfig.GetScenarioResourcesReader()
	strategyCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_strategies.yaml")
	Expect(err).Should(BeNil())

	authRealmCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
	Expect(err).Should(BeNil())

	readerDex := dexoperatorconfig.GetScenarioResourcesReader()
	dexClientCRD, err := getCRD(readerDex, "crd/bases/auth.identitatem.io_dexclients.yaml")
	Expect(err).Should(BeNil())

	dexServerCRD, err := getCRD(readerDex, "crd/bases/auth.identitatem.io_dexservers.yaml")
	Expect(err).Should(BeNil())

	testEnv = &envtest.Environment{
		Scheme: scheme.Scheme,
		CRDs: []client.Object{
			strategyCRD,
			authRealmCRD,
			dexClientCRD,
			dexServerCRD,
		},
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "test", "config", "crd", "external"),
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
		ControlPlaneStartTimeout: 1 * time.Minute,
		ControlPlaneStopTimeout:  1 * time.Minute,
	}

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	clientSetMgmt, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	clientSetStrategy, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	By("Init the reconciler")
	r = AuthRealmReconciler{
		Client:             k8sClient,
		KubeClient:         kubernetes.NewForConfigOrDie(cfg),
		DynamicClient:      dynamic.NewForConfigOrDie(cfg),
		APIExtensionClient: apiextensionsclient.NewForConfigOrDie(cfg),
		Log:                logf.Log,
		Scheme:             scheme.Scheme,
	}

	By("Creating infra", func() {
		infraConfig := &openshiftconfigv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.InfrastructureSpec{},
			Status: openshiftconfigv1.InfrastructureStatus{
				APIServerURL: "http://api.my.company.com:6443",
			},
		}
		err := k8sClient.Create(context.TODO(), infraConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Process deployment: ", func() {
	It("Check if roles are correctly created", func() {
		readerDeploy := deploy.GetScenarioResourcesReader()
		readerDexOperator := dexoperatorconfig.GetScenarioResourcesReader()
		applierBuilder := &clusteradmapply.ApplierBuilder{}
		applier := applierBuilder.
			WithClient(r.KubeClient, r.APIExtensionClient, r.DynamicClient).
			WithTemplateFuncMap(FuncMap()).
			Build()
		files := []string{"dex-operator/leader_election_role.yaml", "dex-operator/role.yaml"}
		values := struct {
			AuthRealm          *identitatemv1alpha1.AuthRealm
			Reader             *clusteradmasset.ScenarioResourcesReader
			File               string
			NewName            string
			FileLeader         string
			NewNameLeader      string
			NewNamespaceLeader string
		}{
			AuthRealm:          &identitatemv1alpha1.AuthRealm{ObjectMeta: metav1.ObjectMeta{Name: "my-authrealm"}},
			Reader:             readerDexOperator,
			File:               "rbac/role.yaml",
			NewName:            "dex-operator-manager-role",
			FileLeader:         "rbac/leader_election_role.yaml",
			NewNameLeader:      "dex-operator-leader-election-role",
			NewNamespaceLeader: "hello",
		}
		output, err := applier.ApplyDirectly(readerDeploy, values, true, "", files...)
		Expect(err).To(BeNil())
		role := &rbacv1.Role{}
		Expect(yaml.Unmarshal([]byte(output[0]), role)).To(BeNil())
		Expect(role.Name).To(Equal(values.NewNameLeader))
		Expect(role.Namespace).To(Equal(values.NewNamespaceLeader))
	})
})
var _ = Describe("Process AuthRealm: ", func() {
	AuthRealmName := "my-authrealm"
	AuthRealmNameSpace := "my-authrealm-ns"
	RouteSubDomain := "myroute"
	MyGithubAppClientID := "my-github-app-client-id"
	CertificatesSecretRef := "my-certs"
	It("Check CRDs availability", func() {
		By("Checking authrealms CRD", func() {
			readerStrategy := idpconfig.GetScenarioResourcesReader()
			_, err := getCRD(readerStrategy, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
			Expect(err).Should(BeNil())
		})
	})
	It("process a AuthRealm CR", func() {
		By("creation test namespace", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNameSpace,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		By("creating the certificate secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      CertificatesSecretRef,
					Namespace: AuthRealmNameSpace,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("tls.mycrt"),
					"tls.key": []byte("tls.mykey"),
					"ca.crt":  []byte("ca.crt"),
				},
			}
			err := k8sClient.Create(context.TODO(), secret)
			Expect(err).To(BeNil())

		})
		var authRealm *identitatemv1alpha1.AuthRealm
		By("creating a AuthRealm CR type dex", func() {
			authRealm = &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							Name: "my-github",
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeGitHub,
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientID: MyGithubAppClientID,
									ClientSecret: openshiftconfigv1.SecretNameReference{
										Name: AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
									},
								},
							},
						},
					},
				},
			}
			var err error
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Run reconcile", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).Should(BeNil())
		})
		By("Checking AuthRealm", func() {
			authRealm, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
			Expect(err).Should(BeNil())
			status := meta.FindStatusCondition(authRealm.Status.Conditions, identitatemv1alpha1.AuthRealmApplied)
			Expect(status).NotTo(BeNil())
			Expect(meta.IsStatusConditionTrue(authRealm.Status.Conditions, identitatemv1alpha1.AuthRealmApplied)).To(BeTrue())
		})
		By("Checking Backplane Strategy", func() {
			_, err := clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName+"-backplane", metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		// By("Checking GRC Strategy", func() {
		// 	_, err := clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName+"-grc", metav1.GetOptions{})
		// 	Expect(err).Should(BeNil())
		// })
		By("Checking Dex Operator Namespace", func() {
			ns := &corev1.Namespace{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.DexOperatorNamespace()}, ns)
			Expect(err).Should(BeNil())
		})
		By("Checking Dex Deployment", func() {
			ns := &appsv1.Deployment{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: "dex-operator", Namespace: helpers.DexOperatorNamespace()}, ns)
			Expect(err).Should(BeNil())
		})
		By("Checking DexServer", func() {
			dexServer := &identitatemdexserverv1lapha1.DexServer{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: helpers.DexServerNamespace(authRealm)}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientID).To(Equal(MyGithubAppClientID))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientSecretRef.Name).To(Equal(AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub)))
			// Expect(dexServer.Spec.Connectors[0].Config.ClientSecretRef.Namespace).To(Equal(dexServerName))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal(identitatemdexserverv1lapha1.ConnectorTypeGitHub))
			Expect(dexServer.Spec.IngressCertificateRef.Name).To(Equal(authRealm.Spec.CertificatesSecretRef.Name))
			Expect(len(dexServer.Status.RelatedObjects)).To(Equal(1))
			Expect(dexServer.Status.RelatedObjects[0].Kind).To(Equal("AuthRealm"))
			//TODO CA missing in Web
		})
	})
	It("process a AuthRealm CR again", func() {
		var authRealm *identitatemv1alpha1.AuthRealm
		By("Retrieving the Authrealm", func() {
			var err error
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).Should(BeNil())
		})
		By("Checking DexServer", func() {
			dexServer := &identitatemdexserverv1lapha1.DexServer{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: helpers.DexServerNamespace(authRealm)}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientID).To(Equal(MyGithubAppClientID))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientSecretRef.Name).To(Equal(AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub)))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal(identitatemdexserverv1lapha1.ConnectorTypeGitHub))
			Expect(dexServer.Spec.IngressCertificateRef.Name).To(Equal(authRealm.Spec.CertificatesSecretRef.Name))
		})
	})
	It("process an updated AuthRealm CR", func() {
		var authRealm *identitatemv1alpha1.AuthRealm
		By("Retrieving the Authrealm", func() {
			var err error
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		By("Changing the cert", func() {
			secret := &corev1.Secret{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: CertificatesSecretRef, Namespace: AuthRealmNameSpace}, secret)
			Expect(err).Should(BeNil())
			secret.Data["tls.crt"] = []byte("tls.newcrt")
			err = k8sClient.Update(context.TODO(), secret)
			Expect(err).Should(BeNil())
		})
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).Should(BeNil())
		})
		By("Checking DexServer", func() {
			dexServer := &identitatemdexserverv1lapha1.DexServer{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: helpers.DexServerNamespace(authRealm)}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientID).To(Equal(MyGithubAppClientID))
			Expect(dexServer.Spec.Connectors[0].GitHub.ClientSecretRef.Name).To(Equal(AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub)))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal(identitatemdexserverv1lapha1.ConnectorTypeGitHub))
			Expect(dexServer.Spec.IngressCertificateRef.Name).To(Equal(authRealm.Spec.CertificatesSecretRef.Name))
		})
	})
	It("process AuthRealm CR with 2 identityProviders", func() {
		By("creating a AuthRealm CR type dex", func() {
			authRealm := &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName + "-1",
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							Name: "my-github",
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeGitHub,
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientSecret: openshiftconfigv1.SecretNameReference{
										Name: AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
									},
								},
							},
						},
						{
							Name: "my-ldap",
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeLDAP,
								LDAP: &openshiftconfigv1.LDAPIdentityProvider{
									BindPassword: openshiftconfigv1.SecretNameReference{
										Name: AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeLDAP),
									},
									Attributes: openshiftconfigv1.LDAPAttributeMapping{
										ID: []string{},
									},
								},
							},
						}},
				},
			}
			_, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName + "-1"
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(BeNil())
		})
	})
	It("process AuthRealm CR without identityProviders", func() {
		By("creating a AuthRealm CR type dex", func() {
			authRealm := &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName + "-2",
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []openshiftconfigv1.IdentityProvider{},
				},
			}
			_, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName + "-2"
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(BeNil())
		})
	})
	It("process AuthRealm CR with identityProviders nil", func() {
		By("creating a AuthRealm CR type dex", func() {
			authRealm := &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName + "-3",
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
				},
			}
			_, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName + "-3"
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(BeNil())
		})
	})

})

func getCRD(reader *clusteradmasset.ScenarioResourcesReader, file string) (*apiextensionsv1.CustomResourceDefinition, error) {
	b, err := reader.Asset(file)
	if err != nil {
		return nil, err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(b, crd); err != nil {
		return nil, err
	}
	return crd, nil
	// apiClient.ApiextensionsV1().CustomResourceDefinitions().Get()
}
