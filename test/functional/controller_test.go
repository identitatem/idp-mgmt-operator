// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

// +build functional

package functional

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	authrealmv1 "github.com/identitatem/idp-mgmt-operator/api/authrealm/v1"
	authclientset "github.com/identitatem/idp-mgmt-operator/api/client/clientset/versioned"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func init() {
	klog.SetOutput(GinkgoWriter)
	klog.InitFlags(nil)

}

var authClientSet *authclientset.Clientset
var cfg *rest.Config

var _ = Describe("AuthRealm", func() {
	BeforeEach(func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
		SetDefaultEventuallyTimeout(20 * time.Second)
		SetDefaultEventuallyPollingInterval(1 * time.Second)

		var err error
		kubeConfigFile := os.Getenv("KUBECONFIG")
		if len(kubeConfigFile) == 0 {
			home := homedir.HomeDir()
			kubeConfigFile = filepath.Join(home, ".kube", "config")
		}
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())
		authClientSet, err = authclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
	})

	AfterEach(func() {
	})

	It("process a AuthRealm", func() {
		By("Create a AuthRealm", func() {
			authRealm := authrealmv1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myauthrealm",
					Namespace: "default",
				},
			}
			_, err := authClientSet.IdentitatemV1().AuthRealms("default").Create(context.TODO(), &authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		Eventually(func() error {
			authRealm, err := authClientSet.IdentitatemV1().AuthRealms("default").Get(context.TODO(), "myauthrealm", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading authrealm", "Error", err)
				return err
			}
			if len(authRealm.Spec.Foo) == 0 {
				logf.Log.Info("AuthRealm Foo is still empty")
				return fmt.Errorf("AuthRealm %s/%s not processed", authRealm.Namespace, authRealm.Name)
			}
			return nil
		}, 30, 1).Should(BeNil())

	})

})
