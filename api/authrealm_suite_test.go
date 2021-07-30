// Copyright Contributors to the Open Cluster Management project

package api

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	clientset "github.com/identitatem/idp-mgmt-operator/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var clientSet *clientset.Clientset
var testEnv *envtest.Environment
var dynamicClient dynamic.Interface
var sampleAuthRealm identitatemv1alpha1.AuthRealm

func TestReadFile(t *testing.T) {
	b, err := ioutil.ReadFile(filepath.Join("..", "config", "samples", "identitatem.io_v1alpha1_authrealm.yaml"))
	if err != nil {
		t.Error(err)
	}
	t.Errorf("%s", string(b))
}

func init() {
	b, err := ioutil.ReadFile(filepath.Join("..", "config", "samples", "identitatem.io_v1alpha1_authrealm.yaml"))
	if err != nil {
		panic(err)
	}
	sampleAuthRealm = identitatemv1alpha1.AuthRealm{}
	err = yaml.Unmarshal(b, &sampleAuthRealm)
	if err != nil {
		panic(err)
	}
}
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"AuthRealm Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	clientSet, err = clientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSet).ToNot(BeNil())

	dynamicClient, err = dynamic.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(dynamicClient).ToNot(BeNil())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Process AuthRealm: ", func() {
	// var sampleAuthRealm identitatemv1alpha1.AuthRealm
	// BeforeEach((func() {
	// 	b, err := ioutil.ReadFile(filepath.Join("..", "config", "samples", "identitatem.io_v1alpha1_authrealm.yaml"))
	// 	Expect(err).ToNot(HaveOccurred())
	// 	Expect(b).ShouldNot(BeNil())
	// 	sampleAuthRealm = identitatemv1alpha1.AuthRealm{}
	// 	err = yaml.Unmarshal(b, &sampleAuthRealm)
	// 	Expect(err).ToNot(HaveOccurred())
	// }))
	It("create a AuthRealm CR", func() {
		By("creating a AuthRealm CR", func() {
			authRealm := sampleAuthRealm.DeepCopy()
			_, err := clientSet.IdentitatemV1alpha1().AuthRealms("default").Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("reading a AuthRealm CR", func() {
			_, err := clientSet.IdentitatemV1alpha1().AuthRealms("default").Get(context.TODO(), sampleAuthRealm.Name, metav1.GetOptions{})
			Expect(err).Should(BeNil())
		})
	})
})
