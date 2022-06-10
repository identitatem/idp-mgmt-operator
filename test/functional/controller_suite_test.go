// Copyright Red Hat

//go:build functional
// +build functional

package functional

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	dexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	identitatemclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	clientsetcluster "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	clusteradmapply "open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

var identitatemClientSet *identitatemclientset.Clientset
var clientSetCluster *clientsetcluster.Clientset
var k8sClient client.Client
var kubeClient *kubernetes.Clientset
var apiExtensionsClient *apiextensionsclient.Clientset
var dynamicClient dynamic.Interface

var cfg *rest.Config

var idpConfig *identitatemv1alpha1.IDPConfig

var stopFakeManagedClusterControllers chan bool
var stopFakeDexControllers chan bool

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

	// readerIDP := idpconfig.GetScenarioResourcesReader()
	applierBuilder := &clusteradmapply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	// files := []string{
	// 	"crd/bases/identityconfig.identitatem.io_authrealms.yaml",
	// 	"crd/bases/identityconfig.identitatem.io_idpconfigs.yaml",
	// }
	// _, err = applier.ApplyDirectly(readerIDP, nil, false, "", files...)
	// Expect(err).Should(BeNil())

	readerDex := dexoperatorconfig.GetScenarioResourcesReader()
	files := []string{
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

	By("Creating MCE CR", func() {
		gvr := schema.GroupVersionResource{Group: "multicluster.openshift.io", Version: "v1alpha1", Resource: "multiclusterengines"}
		mceu := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": gvr.Group + "/" + gvr.Version,
				"kind":       "MultiClusterEngine",
				"metadata": map[string]interface{}{
					"name": "mce",
				},
			},
		}
		_, err := dynamicClient.Resource(gvr).Create(context.TODO(), mceu, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	By("Creating idpconfig", func() {
		Eventually(func() error {
			idpConfig = &identitatemv1alpha1.IDPConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "idpconfig",
					Namespace: "idp-mgmt-config",
				},
			}
			_, err := identitatemClientSet.IdentityconfigV1alpha1().IDPConfigs("idp-mgmt-config").Create(context.TODO(), idpConfig, metav1.CreateOptions{})
			return err
		}, 30, 1).Should(BeNil())
	})

	By("Checking operator installation", func() {
		Eventually(func() error {
			l, err := kubeClient.CoreV1().Pods("idp-mgmt-config").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}
			//1 pod for installer and 1 for operator
			if len(l.Items) < 2 {
				logf.Log.Info("operator pod not created yet", "name", "idp-mgmt-config")
				return fmt.Errorf("operator pod not created yet")
			}
			allReady := true
			for _, p := range l.Items {
				if p.Status.Phase != corev1.PodRunning {
					allReady = false
					break
				}
			}
			if !allReady {
				logf.Log.Info("some pods are not ready yet", "name", "idp-mgmt-config")
				return fmt.Errorf("some pods are not ready yet")
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
	stopFakeManagedClusterControllers = make(chan bool)
	go manifestworkFakeController(stopFakeManagedClusterControllers)
	stopFakeDexControllers = make(chan bool)
	go dexFakeController(stopFakeDexControllers)
})

var _ = AfterSuite(func() {
	defer func() {
		stopFakeManagedClusterControllers <- true
		stopFakeDexControllers <- true
	}()
	//Uncomment the skip bellow if you want to be able to see logs during debugging.
	// Skip("")
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	By("Deleting idpconfig", func() {
		err := identitatemClientSet.IdentityconfigV1alpha1().IDPConfigs("idp-mgmt-config").
			Delete(context.TODO(), "idpconfig", metav1.DeleteOptions{})
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
	By("Checking operator deletion", func() {
		Eventually(func() error {
			l, err := kubeClient.CoreV1().Pods("idp-mgmt-config").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}
			//1 pod for installer remain
			if len(l.Items) != 1 {
				logf.Log.Info("operator pod not deleted yet", "name", "idp-mgmt-config")
				return fmt.Errorf("operator pod not deleted yet")
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
})

func manifestworkFakeController(done chan bool) {
	gvr := schema.GroupVersionResource{Group: "work.open-cluster-management.io", Version: "v1", Resource: "manifestworks"}
	for {
		select {
		case <-done:
			logf.Log.Info("managedWork controller stopped")
			return
		default:
			logf.Log.Info("Run manifestwork controller")
			mwus, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logf.Log.Error(err, "error when retrieving mw")
			}
			for _, mwu := range mwus.Items {
				if mwu.GetName() == helpers.ManifestWorkOriginalOAuthName() {
					fmt.Printf("set %s/%s/%s to applied\n", mwu.GetKind(), mwu.GetNamespace(), mwu.GetName())
					mcv := &manifestworkv1.ManifestWork{}
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(mwu.UnstructuredContent(), mcv)
					if err != nil {
						logf.Log.Error(err, "error when converting mw", "name", mwu.GetName())
					}
					alreadySet := false
					for _, c := range mcv.Status.Conditions {
						if c.Type == "Applied" && c.Status == metav1.ConditionTrue {
							alreadySet = true
							break
						}
					}
					if alreadySet {
						logf.Log.Info("already set to applied", "kind", mwu.GetKind(), "namespace", mwu.GetNamespace(), "name", mwu.GetName())
						continue
					}
					mcv.Status.Conditions = []metav1.Condition{
						{
							Type:               "Applied",
							Status:             metav1.ConditionTrue,
							Reason:             "AppliedManifestComplete",
							Message:            "Apply manifest complete",
							LastTransitionTime: metav1.Now(),
						},
					}
					uc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(mcv)
					if err != nil {
						logf.Log.Error(err, "error when converting mw", "name", mwu.GetName())
					}
					mwu.SetUnstructuredContent(uc)
					mwuu := mwu.DeepCopy()
					mwuu, err = dynamicClient.Resource(gvr).Namespace(mwu.GetNamespace()).UpdateStatus(context.TODO(), mwuu, metav1.UpdateOptions{})
					if err != nil {
						logf.Log.Error(err, "error when updating status mw", "name", mwu.GetName())
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func dexFakeController(done chan bool) {
	gvr := schema.GroupVersionResource{Group: "auth.identitatem.io", Version: "v1alpha1", Resource: "dexclients"}
	for {
		select {
		case <-done:
			logf.Log.Info("dex controller stopped")
			return
		default:
			logf.Log.Info("Run dex controller")
			dexus, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logf.Log.Error(err, "error when retrieving dexcleint")
			}
			for _, dexu := range dexus.Items {
				dexuu := dexu.DeepCopy()
				if dexu.GetDeletionTimestamp() == nil {
					logf.Log.Info("Add finalizer", "Finalizer", helpers.AuthrealmFinalizer)
					controllerutil.AddFinalizer(dexuu, helpers.AuthrealmFinalizer)
				} else {
					logf.Log.Info("Remove finalizer", "Finalizer", helpers.AuthrealmFinalizer)
					controllerutil.RemoveFinalizer(dexuu, helpers.AuthrealmFinalizer)
				}
				dexuu, err = dynamicClient.Resource(gvr).Namespace(dexu.GetNamespace()).Update(context.TODO(), dexuu, metav1.UpdateOptions{})
				if err != nil {
					logf.Log.Error(err, "error when updating status mw", "name", dexu.GetName())
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

func TestRcmController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "idp-mgmt-operator Suite")
}
