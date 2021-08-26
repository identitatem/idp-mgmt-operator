// Copyright Red Hat

// +build functional

package functional

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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

var authClientSet *idpclientset.Clientset
var strategyClientSet *idpclientset.Clientset

var cfg *rest.Config
var kubeClient *kubernetes.Clientset
var apiExtensionsClient *apiextensionsclient.Clientset

var _ = Describe("AuthRealm", func() {
	AuthRealmName := "my-authrealm"
	AuthRealmNameSpace := "my-authrealm-ns"
	BeforeEach(func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
		SetDefaultEventuallyTimeout(20 * time.Second)
		SetDefaultEventuallyPollingInterval(1 * time.Second)

		kubeConfigFile := os.Getenv("KUBECONFIG")
		if len(kubeConfigFile) == 0 {
			home := homedir.HomeDir()
			kubeConfigFile = filepath.Join(home, ".kube", "config")
		}
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())
		authClientSet, err = idpclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
		strategyClientSet, err = idpclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
		kubeClient, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeClient).ToNot(BeNil())
		apiExtensionsClient, err = apiextensionsclient.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeClient).ToNot(BeNil())
	})

	AfterEach(func() {
	})

	It("process a AuthRealm CR", func() {
		By("checking CRD", func() {
			Eventually(func() error {
				_, err := apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "authrealms.identityconfig.identitatem.io", metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealms crd", "Error", err)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())

		})
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
			authRealm := identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					Type: identitatemv1alpha1.AuthProxyDex,
					// CertificatesSecretRef: corev1.LocalObjectReference{
					// 	Name: CertificatesSecretRef,
					// },
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientSecret: openshiftconfigv1.SecretNameReference{
										Name: AuthRealmName + "-github",
									},
								},
							},
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
		By("Checking the dex-operator deployment creation", func() {
			Eventually(func() error {
				_, err := kubeClient.AppsV1().Deployments(AuthRealmName).
					Get(context.TODO(), "dex-operator", metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading deployments", "Error", err)
					return err
				}
				return nil
			}, 60, 1).Should(BeNil())
		})
		By("Checking the strategy backplane creation", func() {
			Eventually(func() error {
				_, err := strategyClientSet.
					IdentityconfigV1alpha1().
					Strategies(AuthRealmNameSpace).
					Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading strategy", "Error", err)
					return err
				}
				return nil
			}, 60, 1).Should(BeNil())
		})
		By("Checking the strategy grc creation", func() {
			Eventually(func() error {
				_, err := strategyClientSet.
					IdentityconfigV1alpha1().
					Strategies(AuthRealmNameSpace).
					Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.GrcStrategyType), metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading strategy", "Error", err)
					return err
				}
				return nil
			}, 60, 1).Should(BeNil())
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
			}, 60, 1).Should(BeNil())
		})
		By("Checking authrealm ns deleted", func() {
			Eventually(func() bool {
				_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 60, 1).Should(BeTrue())
		})
		By("Checking strategy Backplane deleted", func() {
			Eventually(func() error {
				_, err := strategyClientSet.
					IdentityconfigV1alpha1().
					Strategies(AuthRealmNameSpace).
					Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
				return err
			}, 60, 1).ShouldNot(BeNil())
		})
		By("Checking strategy GRC deleted", func() {
			Eventually(func() error {
				_, err := strategyClientSet.
					IdentityconfigV1alpha1().
					Strategies(AuthRealmNameSpace).
					Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.GrcStrategyType), metav1.GetOptions{})
				return err
			}, 60, 1).ShouldNot(BeNil())
		})
	})
})
