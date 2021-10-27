// Copyright Red Hat

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config

var clientSetMgmt *idpclientset.Clientset

var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Strategy Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	readerIDP := idpconfig.GetScenarioResourcesReader()

	authRealmCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDs: []client.Object{
			authRealmCRD,
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
		ControlPlaneStartTimeout: 1 * time.Minute,
		ControlPlaneStopTimeout:  1 * time.Minute,
	}
	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	clientSetMgmt, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Run webhook tests: ", func() {
	AuthRealmName1 := "my-authrealm-1"
	AuthRealmNameSpace := "my-authrealmns"
	RouteSubDomain := "abcdefghijklmnopqrstuvwxyz-0123456789"
	stopCh := make(chan struct{})
	var authRealmAdmissionHook *AuthRealmAdmissionHook
	It("Validate re-use of subRouteDomain", func() {
		By(fmt.Sprintf("creation of User namespace %s", AuthRealmNameSpace), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNameSpace,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		var authRealm *identitatemv1alpha1.AuthRealm
		By("creating a AuthRealm CR", func() {
			var err error
			authRealm = &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName1,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					Type:           identitatemv1alpha1.AuthProxyDex,
					RouteSubDomain: RouteSubDomain,
				},
			}
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Calling Initialize", func() {
			authRealmAdmissionHook = &AuthRealmAdmissionHook{}
			Expect(authRealmAdmissionHook.Initialize(cfg, stopCh)).To(BeNil())
		})
		//Allowed because no ns exist yet for the DexServer
		By("Calling Validate", func() {
			r := &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			}
			r.Object.Raw, _ = json.Marshal(authRealm)
			response := authRealmAdmissionHook.Validate(r)
			Expect(response.Allowed).To(BeTrue())
		})
		By("creating the dexserver namespace", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: helpers.DexServerNamespace(authRealm),
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		//Denied because a dexserver namespace exists
		By("Calling Validate", func() {
			r := &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			}
			r.Object.Raw, _ = json.Marshal(authRealm)
			response := authRealmAdmissionHook.Validate(r)
			Expect(response.Allowed).To(BeFalse())
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
}
