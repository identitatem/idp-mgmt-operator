// Copyright Red Hat

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
	clientsetmgmt "github.com/identitatem/idp-mgmt-operator/api/client/clientset/versioned"
	identitatemmgmtv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	clientsetstrategy "github.com/identitatem/idp-strategy-operator/api/client/clientset/versioned"
	identitatemstrategyv1alpha1 "github.com/identitatem/idp-strategy-operator/api/identitatem/v1alpha1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var clientSetMgmt *clientsetmgmt.Clientset
var clientSetStrategy *clientsetstrategy.Clientset
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
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	By("bootstrapping test environment")
	err := os.Setenv(dexOperatorImageEnvName, "dex_operator_inage")
	Expect(err).NotTo(HaveOccurred())
	err = identitatemmgmtv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = identitatemstrategyv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	err = identitatemdexserverv1lapha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())
	testEnv = &envtest.Environment{
		Scheme: scheme.Scheme,
		CRDs: []client.Object{
			&appsv1.Deployment{},
		},
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "config", "crd", "external"),
		},
	}

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	clientSetMgmt, err = clientsetmgmt.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	clientSetStrategy, err = clientsetstrategy.NewForConfig(cfg)
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

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Process AuthRealm: ", func() {
	AuthRealmName := "test-authrealm"
	AuthRealmNameSpace := "test"
	CertificatesSecretRef := "test-certs"
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
		By("creating a AuthRealm CR type dex", func() {
			authRealm := &identitatemmgmtv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemmgmtv1alpha1.AuthRealmSpec{
					Type: identitatemmgmtv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []identitatemmgmtv1alpha1.IdentityProvider{
						{
							GitHub: &openshiftconfigv1.GitHubIdentityProvider{},
						},
					},
				},
			}
			_, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
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
			_, err := clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		By("Checking Backplane Strategy", func() {
			_, err := clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName+"-backplane", metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		By("Checking GRC Strategy", func() {
			_, err := clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName+"-grc", metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
		By("Checking Dex Namespace", func() {
			ns := &corev1.Namespace{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName}, ns)
			Expect(err).Should(BeNil())
		})
		By("Checking Dex Deployment", func() {
			ns := &appsv1.Deployment{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: "dex-operator", Namespace: AuthRealmName}, ns)
			Expect(err).Should(BeNil())
		})
		By("Checking DexServer", func() {
			dexServer := &identitatemdexserverv1lapha1.DexServer{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: AuthRealmName}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].Config.ClientID).To(Equal(AuthRealmName))
			Expect(dexServer.Spec.Connectors[0].Config.ClientSecretRef).To(Equal(AuthRealmName + "-github"))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal("github"))
			Expect(dexServer.Spec.Web.TlsCert).To(Equal("tls.mycrt"))
			Expect(dexServer.Spec.Web.TlsKey).To(Equal("tls.mykey"))
			//TODO CA missing in Web
		})
		// By("Checking DexClient", func() {
		// 	dexClient := &identitatemdexserverv1lapha1.DexClient{}
		// 	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: AuthRealmName}, dexClient)
		// 	Expect(err).Should(BeNil())
		// })
	})
	It("process a AuthRealm CR again", func() {
		By("Run reconcile again", func() {
			req := ctrl.Request{}
			req.Name = AuthRealmName
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).Should(BeNil())
		})
		By("Checking DexServer", func() {
			dexServer := &identitatemdexserverv1lapha1.DexServer{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: AuthRealmName}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].Config.ClientID).To(Equal(AuthRealmName))
			Expect(dexServer.Spec.Connectors[0].Config.ClientSecretRef).To(Equal(AuthRealmName + "-github"))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal("github"))
			Expect(dexServer.Spec.Web.TlsCert).To(Equal("tls.mycrt"))
			Expect(dexServer.Spec.Web.TlsKey).To(Equal("tls.mykey"))
		})
	})
	It("process an updated AuthRealm CR", func() {
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
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: AuthRealmName, Namespace: AuthRealmName}, dexServer)
			Expect(err).Should(BeNil())
			Expect(len(dexServer.Spec.Connectors)).To(Equal(1))
			Expect(dexServer.Spec.Connectors[0].Config.ClientID).To(Equal(AuthRealmName))
			Expect(dexServer.Spec.Connectors[0].Config.ClientSecretRef).To(Equal(AuthRealmName + "-github"))
			Expect(dexServer.Spec.Connectors[0].Type).To(Equal("github"))
			Expect(dexServer.Spec.Web.TlsCert).To(Equal("tls.newcrt"))
			Expect(dexServer.Spec.Web.TlsKey).To(Equal("tls.mykey"))
		})
	})
})
