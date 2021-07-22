/*
Copyright Contributors to the Open Cluster Management project
*/

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	identitatemiov1 "github.com/identitatem/idp-mgmt-operator/api/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = identitatemiov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

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
			authRealm := identitatemiov1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myauthrealm",
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(context.TODO(), &authRealm)).To(BeNil())
		})
		Eventually(func() error {
			r := AuthRealmReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}

			req := ctrl.Request{}
			req.Name = "myauthrealm"
			req.Namespace = "default"
			_, err := r.Reconcile(req)
			if err != nil {
				return err
			}
			authRealm := &identitatemiov1.AuthRealm{}
			if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: "myauthrealm", Namespace: "default"}, authRealm); err != nil {
				logf.Log.Logger.Info("Error while reading authrealm", "Error", err)
				return err
			}
			if len(authRealm.Spec.Foo) == 0 {
				logf.Log.Logger.Info("AuthRealm Foo is still empty")
				return fmt.Errorf("AuthRealm %s/%s not processed", authRealm.Namespace, authRealm.Name)
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
})
