// Copyright Contributors to the Open Cluster Management project

// +build functional

package functional

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	identitatemmgmtclientset "github.com/identitatem/idp-mgmt-operator/api/client/clientset/versioned"
	identitatemmgmtv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	identitatemstrategyclientset "github.com/identitatem/idp-strategy-operator/api/client/clientset/versioned"
	identitatemstrategyv1alpha1 "github.com/identitatem/idp-strategy-operator/api/identitatem/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

var authClientSet *identitatemmgmtclientset.Clientset
var strategyClientSet *identitatemstrategyclientset.Clientset

var cfg *rest.Config
var kubeClient *kubernetes.Clientset

var _ = Describe("AuthRealm", func() {
	AuthRealmName := "test-authrealm"
	AuthRealmNameSpace := "test"
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
		authClientSet, err = identitatemmgmtclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
		strategyClientSet, err = identitatemstrategyclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
		kubeClient, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeClient).ToNot(BeNil())
	})

	AfterEach(func() {
	})

	It("process a AuthRealm CR", func() {
		By("creation test namespace", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNameSpace,
				},
			}
			_, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Create a AuthRealm", func() {
			authRealm := identitatemmgmtv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemmgmtv1alpha1.AuthRealmSpec{
					Type: identitatemmgmtv1alpha1.AuthProxyDex,
					// CertificatesSecretRef: corev1.LocalObjectReference{
					// 	Name: CertificatesSecretRef,
					// },
					IdentityProviders: []identitatemmgmtv1alpha1.IdentityProvider{
						{
							GitHub: &openshiftconfigv1.GitHubIdentityProvider{},
						},
					},
				},
			}
			_, err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), &authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Checking the AuthRealm on authrealm creation", func() {
			Eventually(func() error {
				_, err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
	})
	//TODO check dexserver when update authrealm
	It("Delete the authrealm", func() {
		By("Deleting the authrealm", func() {
			err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Delete(context.TODO(), AuthRealmName, metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})
		By("Authrealm is deleted", func() {
			Eventually(func() error {
				_, err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return nil
					}
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				return fmt.Errorf("Authrealm still exists")
			}, 30, 1).Should(BeNil())
		})
		By("Checking strategy Backplane deleted", func() {
			_, err := strategyClientSet.
				IdentityconfigV1alpha1().
				Strategies(AuthRealmNameSpace).
				Get(context.TODO(), AuthRealmName+"-"+string(identitatemstrategyv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
		By("Checking strategy GRC deleted", func() {
			_, err := strategyClientSet.
				IdentityconfigV1alpha1().
				Strategies(AuthRealmNameSpace).
				Get(context.TODO(), AuthRealmName+"-"+string(identitatemstrategyv1alpha1.GrcStrategyType), metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
		By("Checking authrealm ns deleted", func() {
			Eventually(func() bool {
				_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 30, 1).Should(BeTrue())
		})
	})
})
