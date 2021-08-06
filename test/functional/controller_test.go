// Copyright Contributors to the Open Cluster Management project

// +build functional

package functional

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	identitatemclientset "github.com/identitatem/idp-mgmt-operator/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
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

var authClientSet *identitatemclientset.Clientset
var kubeClientSet *kubernetes.Clientset
var cfg *rest.Config

var _ = Describe("AuthRealm", func() {
	AuthRealmName := "myauthrealm"
	AuthRealmNameSpace := "default"
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
		authClientSet, err = identitatemclientset.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(authClientSet).ToNot(BeNil())
		kubeClientSet, err = kubernetes.NewForConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeClientSet).ToNot(BeNil())
	})

	AfterEach(func() {
	})

	It("process a AuthRealm CR", func() {
		By("Create a AuthRealm", func() {
			authRealm := identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
			}
			_, err := authClientSet.IdentitatemV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), &authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Checking the AuthRealm for update on authrealm creation", func() {
			Eventually(func() error {
				authRealm, err := authClientSet.IdentitatemV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				if len(authRealm.Spec.MappingMethod) == 0 {
					logf.Log.Info("AuthRealm MappingMethod is still empty")
					return fmt.Errorf("AuthRealm %s/%s not processed", authRealm.Namespace, authRealm.Name)
				}
				logf.Log.Info("AuthRealm", "MappingMethod", authRealm.Spec.MappingMethod)
				return nil
			}, 30, 1).Should(BeNil())
		})
	})
	It("process a identityPRovider CR in another NS, the mappingMethod should not be updated", func() {
		IdentityProviderName := "myidentityprovider"
		IdentityProviderNamespace := "anotherns"
		By("Creating the other ns", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: IdentityProviderNamespace,
				},
			}
			_, err := kubeClientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Create a IdentityProvider", func() {
			identityProvider := identitatemv1alpha1.IdentityProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      IdentityProviderName,
					Namespace: IdentityProviderNamespace,
				},
			}
			_, err := authClientSet.IdentitatemV1alpha1().IdentityProviders(IdentityProviderNamespace).Create(context.TODO(), &identityProvider, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Checking the AuthRealm for update on identityprovider creation", func() {
			Consistently(func() error {
				authRealm, err := authClientSet.IdentitatemV1alpha1().AuthRealms(AuthRealmNameSpace).Get(context.TODO(), AuthRealmName, metav1.GetOptions{})
				if err != nil {
					logf.Log.Info("Error while reading authrealm", "Error", err)
					return err
				}
				if authRealm.Spec.MappingMethod == openshiftconfigv1.MappingMethodClaim {
					logf.Log.Info("AuthRealm MappingMethod is still Claim")
					return nil
				}
				logf.Log.Info("AuthRealm", "MappingMethod", authRealm.Spec.MappingMethod)
				return fmt.Errorf("AuthRealm %s/%s not changed and it is not Claim anymore", authRealm.Namespace, authRealm.Name)
			}, 30, 1).Should(BeNil())
		})
	})
})
