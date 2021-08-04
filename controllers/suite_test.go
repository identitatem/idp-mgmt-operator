// Copyright Contributors to the Open Cluster Management project

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
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

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Process AuthRealm: ", func() {
	It("process a AuthRealm CR", func() {
		By("creating a AuthRealm CR", func() {
			authRealm := identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myauthrealm",
					Namespace: "default",
				},
			}
			_, err := clientSet.IdentitatemV1alpha1().AuthRealms("default").Create(context.TODO(), &authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Processing a first time", func() {
			Eventually(func() error {
				r := AuthRealmReconciler{
					Client: k8sClient,
					Log:    logf.Log,
					Scheme: scheme.Scheme,
				}

				req := ctrl.Request{}
				req.Name = "myauthrealm"
				req.Namespace = "default"
				_, err := r.Reconcile(context.TODO(), req)
				if err != nil {
					return err
				}
				authRealm, err := clientSet.IdentitatemV1alpha1().AuthRealms("default").Get(context.TODO(), "myauthrealm", metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				if len(authRealm.Spec.MappingMethod) == 0 {
					logf.Log.Info("AuthRealm MappingMethod is still empty")
					return fmt.Errorf("AuthRealm %s/%s not processed", authRealm.Namespace, authRealm.Name)
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
		By("Processing a second time", func() {
			Eventually(func() error {
				r := AuthRealmReconciler{
					Client: k8sClient,
					Log:    logf.Log,
					Scheme: scheme.Scheme,
				}

				req := ctrl.Request{}
				req.Name = "myauthrealm"
				req.Namespace = "default"
				_, err := r.Reconcile(context.TODO(), req)
				if err != nil {
					return err
				}
				authRealm, err := clientSet.IdentitatemV1alpha1().AuthRealms("default").Get(context.TODO(), "myauthrealm", metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				if authRealm.Spec.MappingMethod != openshiftconfigv1.MappingMethodAdd {
					logf.Log.Info("AuthRealm MappingMethod is still not add")
					return fmt.Errorf("AuthRealm %s/%s not processed", authRealm.Namespace, authRealm.Name)
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
	})
})
