// Copyright Red Hat

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/identitatem/idp-mgmt-operator/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var TestOptions utils.TestOptionsType

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)

	// Set clients for Hub Cluster
	hubUrl := os.Getenv("CLUSTER_SERVER_URL")
	Expect(hubUrl).ToNot(BeNil())
	TestOptions.HubCluster.ClusterServerURL = hubUrl
	kubeConfig := os.Getenv("KUBECONFIG")
	TestOptions.HubCluster.KubeConfig = kubeConfig
	TestOptions.HubCluster.KubeClient = utils.NewKubeClient(
		TestOptions.HubCluster.ClusterServerURL,
		TestOptions.HubCluster.KubeConfig, "")	
	TestOptions.HubCluster.KubeClientDynamic = utils.NewKubeClientDynamic(
		TestOptions.HubCluster.ClusterServerURL,
		TestOptions.HubCluster.KubeConfig, "")
	TestOptions.HubCluster.ApiExtensionsClient = utils.NewKubeClientAPIExtension(
		TestOptions.HubCluster.ClusterServerURL,
		TestOptions.HubCluster.KubeConfig, "")

	// Set clients for Managed Cluster
	mcUrl := os.Getenv("MANAGED_CLUSTER_SERVER_URL")
	Expect(mcUrl).ToNot(BeNil())
	managedClusters := []utils.Cluster{
		{
			ClusterServerURL: mcUrl,
			KubeConfig: os.Getenv("MANAGED_CLUSTER_KUBECONFIG"),
			KubeContext: os.Getenv("MANAGED_CLUSTER_KUBECONTEXT"),
		},
	}	
	managedClusters[0].KubeClient = utils.NewKubeClient(
		managedClusters[0].ClusterServerURL,
		managedClusters[0].KubeConfig,
		managedClusters[0].KubeContext)	
	managedClusters[0].KubeClientDynamic = utils.NewKubeClientDynamic(
		managedClusters[0].ClusterServerURL,
		managedClusters[0].KubeConfig,
		managedClusters[0].KubeContext)
	managedClusters[0].ApiExtensionsClient = utils.NewKubeClientAPIExtension(
		managedClusters[0].ClusterServerURL,
		managedClusters[0].KubeConfig,
		managedClusters[0].KubeContext)	
	TestOptions.ManagedClusters = managedClusters

	By("Checking that the managed cluster is imported and the ManagedCluster resource has a url", func() {
		gvr, err := utils.GetGVRForResource("ManagedCluster")
		mcName := os.Getenv("MANAGED_CLUSTER_NAME")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
			List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logf.Log.Info("Error while reading ManagedCluster list", "Error", err)
				return err
			}
			Expect(len(list.Items)).Should(BeNumerically(">", 0))
			var specifiedManagedClusterExists bool
			for _, managedCluster := range list.Items {
				if (managedCluster.GetName() == mcName) {
					specifiedManagedClusterExists = true
					mcSlice, found, err := unstructured.NestedSlice(managedCluster.UnstructuredContent(), "spec", "managedClusterClientConfigs")
					mc0 := mcSlice[0].(map[string]interface{})
					Expect(err).NotTo(HaveOccurred())
					Expect(found).To(BeTrue())
					Expect(len(mc0["url"].(string))).To(BeNumerically(">", 0))
				}
			}
			if (!specifiedManagedClusterExists) {
				logf.Log.Info("Managed cluster was not found", "Error", err)
				return err				
			}			
			return nil
		}, 120, 1).Should(BeNil())	
	})

	// Verify installation of idp-mgmt-operator based on pods in the namespace idp-mgmt-config
	By("Checking operator installation", func() {
		Eventually(func() error {
			l, err := TestOptions.HubCluster.KubeClient.CoreV1().Pods("idp-mgmt-config").List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}
			// 1 pod for installer and 1 for operator
			if len(l.Items) < 2 {
				logf.Log.Info("operator pod not created yet", "name", "idp-mgmt-config")
				return fmt.Errorf("operator pod not created yet")
			}
			allReady := true
			for _, p := range l.Items {
				if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodSucceeded {
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
})

var _ = AfterSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter)))
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)

	// Remove labels from the managed cluster
	gvr, err := utils.GetGVRForResource("ManagedCluster")
	Expect(err).NotTo(HaveOccurred())
	mcName := os.Getenv("MANAGED_CLUSTER_NAME")
	managedCluster := &unstructured.Unstructured{}
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error {
		managedCluster, err = TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
		Get(context.TODO(), mcName, metav1.GetOptions{})
		if err != nil {
			logf.Log.Info("Error while reading ManagedCluster", "Error", err)
			return err
		}
		return nil
	}, 120, 1).Should(BeNil())

	labels := managedCluster.GetLabels()
	_, authDepLabelFound := labels["authdeployment"]
	if (authDepLabelFound) {
		delete(labels, "authdeployment")
	}
	_, clusterSetLabelFound := labels["cluster.open-cluster-management.io/clusterset"]
	if (clusterSetLabelFound) {
		delete(labels, "cluster.open-cluster-management.io/clusterset")
	}

	// Update labels
	managedCluster.SetLabels(labels)
	Eventually(func() error {
		_, err = TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
		Update(context.TODO(), managedCluster, metav1.UpdateOptions{})
		if err != nil {
			logf.Log.Info("Error while updating ManagedCluster", "Error", err)
			return err
		}
		return nil
	}, 120, 1).Should(BeNil())

	// TODO: Deleting idpConfig and confirming deletion of operator and associated cleanup
})

func TestIdpMgmtOperatorE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"IDP Management E2E Suite",
		[]Reporter{printer.NewlineReporter{}})
}
