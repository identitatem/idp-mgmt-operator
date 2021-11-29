// Copyright Red Hat

// +build functional

package functional

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	dexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	identitatemclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	clientsetcluster "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

var identitatemClientSet *identitatemclientset.Clientset
var clientSetCluster *clientsetcluster.Clientset
var k8sClient client.Client
var kubeClient *kubernetes.Clientset
var apiExtensionsClient *apiextensionsclient.Clientset
var dynamicClient dynamic.Interface
var authClientSet *idpclientset.Clientset

var cfg *rest.Config

var idpConfig *corev1.ConfigMap

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)

	kubeConfigFile := os.Getenv("KUBECONFIG")
	if len(kubeConfigFile) == 0 {
		home := homedir.HomeDir()
		kubeConfigFile = filepath.Join(home, ".kube", "config")
	}
	klog.Infof("KUBECONFIG=%s", kubeConfigFile)
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	kubeClient, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(kubeClient).ToNot(BeNil())

	apiExtensionsClient, err = apiextensionsclient.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(apiExtensionsClient).ToNot(BeNil())

	dynamicClient, err = dynamic.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(dynamicClient).ToNot(BeNil())

	authClientSet, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(authClientSet).ToNot(BeNil())

	identitatemClientSet, err = identitatemclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(identitatemClientSet).ToNot(BeNil())

	clientSetCluster, err = clientsetcluster.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetCluster).ToNot(BeNil())

	err = dexv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	readerIDP := idpconfig.GetScenarioResourcesReader()
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"crd/bases/identityconfig.identitatem.io_authrealms.yaml",
	}
	_, err = applier.ApplyDirectly(readerIDP, nil, false, "", files...)
	Expect(err).Should(BeNil())

	readerDex := dexoperatorconfig.GetScenarioResourcesReader()
	files = []string{
		"crd/bases/auth.identitatem.io_dexclients.yaml",
	}
	_, err = applier.ApplyDirectly(readerDex, nil, false, "", files...)
	Expect(err).Should(BeNil())

	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = workv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = openshiftconfigv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = viewv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("Creating infra", func() {
		infraConfig := &openshiftconfigv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.InfrastructureSpec{},
			Status: openshiftconfigv1.InfrastructureStatus{
				APIServerURL: "http://127.0.0.1:6443",
			},
		}
		err := k8sClient.Create(context.TODO(), infraConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	By("Creating idpconfig", func() {
		idpConfig = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "idpconfig",
				Namespace: "idp-mgmt-config",
				Labels: map[string]string{
					"auth.identitatem.io/installer-config": "",
				},
			},
		}
		_, err := kubeClient.CoreV1().ConfigMaps("idp-mgmt-config").Create(context.TODO(), idpConfig, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

})

var _ = AfterSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	By("Deleting idpconfig", func() {
		err := kubeClient.CoreV1().ConfigMaps("idp-mgmt-config").Delete(context.TODO(), "idpconfig", metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
	By("Checking idpconfig deletion", func() {
		Eventually(func() error {
			_, err := kubeClient.CoreV1().ConfigMaps("idp-mgmt-config").
				Get(context.TODO(), "idpconfig", metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				return nil
			}
			return fmt.Errorf("clientSecret %s in ns %s still exist", "idpconfig", "idp-mgmt-config")
		}, 60, 1).Should(BeNil())
	})

})

func TestRcmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "idp-mgmt-operator Suite")
}
