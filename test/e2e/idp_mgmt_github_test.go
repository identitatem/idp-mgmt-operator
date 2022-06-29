// Copyright Red Hat

//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/identitatem/idp-mgmt-operator/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	klog.SetOutput(GinkgoWriter)
	klog.InitFlags(nil)
}

var _ = Describe("Verify that CRDs exist", func() {
	It("Check for AuthRealm CRD", func() {
		Eventually(func() error {
			_, err := TestOptions.HubCluster.ApiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "authrealms.identityconfig.identitatem.io", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading authrealms crd", "Error", err)
				return err
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
	It("Check for Strategy CRD", func() {
		Eventually(func() error {
			_, err := TestOptions.HubCluster.ApiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "strategies.identityconfig.identitatem.io", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading authrealms crd", "Error", err)
				return err
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
	It("Check for ClusterOauth CRD", func() {
		Eventually(func() error {
			_, err := TestOptions.HubCluster.ApiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), "clusteroauths.identityconfig.identitatem.io", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading authrealms crd", "Error", err)
				return err
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
})

var _ = Describe("Process AuthRealm configured for GitHub IDP", func() {
	It("Hub: AuthRealm is successfully created", func() {
		By("Reading the AuthRealm related resources from test data file")
		// The following file includes the yamls for resources needed to create the AuthRealm
		// such as namespace, managedclusterset, placement, managedclustersetbinding and secret
		pathToAuthRealmTestData := "testdata/github-authrealm.yaml"
		yamlB, err := ioutil.ReadFile(pathToAuthRealmTestData)
		Expect(err).NotTo(HaveOccurred())
		By("Applying the resources from test data file")
		err = utils.Apply(
			TestOptions.HubCluster.KubeClient,
			TestOptions.HubCluster.KubeClientDynamic,
			yamlB,
		)
		Expect(err).NotTo(HaveOccurred())
		By("Checking that the AuthRealm is created")
		gvr, err := utils.GetGVRForResource("AuthRealm")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("my-authrealm").
				Get(context.TODO(), "authrealm-sample", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading AuthRealm", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Hub: Strategy resource is created", func() {
		gvr, err := utils.GetGVRForResource("Strategy")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("my-authrealm").
				Get(context.TODO(), "authrealm-sample-backplane", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading Strategy", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Hub: DexServer is created", func() {
		gvr, err := utils.GetGVRForResource("DexServer")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("idp-mgmt-testdomain").
				Get(context.TODO(), "dex-server", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading DexServer", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Hub: There is no DexClient as yet", func() {
		gvr, err := utils.GetGVRForResource("DexClient")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("idp-mgmt-testdomain").
				List(context.TODO(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			return len(list.Items)
		}, 300, 1).Should(Equal(0))
	})
	It("Managed Cluster: Verify that the OAuth does not contain IDP config", func() {
		gvr, err := utils.GetGVRForResource("OAuth")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			oauth, err := TestOptions.ManagedClusters[0].KubeClientDynamic.Resource(gvr).
				Get(context.TODO(), "cluster", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading OAuth in managed cluster", "Error", err)
				return err
			}
			_, found, err := unstructured.NestedSlice(oauth.UnstructuredContent(), "spec", "identityProviders")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Hub: PlacementDecision is created", func() {
		By("Applying labels associated with the placement to the managed cluster")
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
		}, 300, 1).Should(BeNil())
		// Check that labels don't already exist
		labels := managedCluster.GetLabels()
		_, authDepLabelFound := labels["authdeployment"]
		//Starting with ACM 2.5, this is set to "default" and is no longer null, so ignore the test
		//_, clusterSetLabelFound := labels["cluster.open-cluster-management.io/clusterset"]
		Expect(authDepLabelFound).ToNot(BeTrue())
		//Expect(clusterSetLabelFound).ToNot(BeTrue())
		// Add labels to include managed cluster in the placement for the IDP config
		labels["authdeployment"] = "east"
		labels["cluster.open-cluster-management.io/clusterset"] = "clusterset-sample"
		managedCluster.SetLabels(labels)
		Eventually(func() error {
			_, err = TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Update(context.TODO(), managedCluster, metav1.UpdateOptions{})
			if err != nil {
				logf.Log.Info("Error while updating ManagedCluster", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
		// Check that PlacementDecision is created
		gvr, err = utils.GetGVRForResource("PlacementDecision")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("my-authrealm").
				List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logf.Log.Info("Error while reading PlacementDecisions in hub cluster", "Error", err)
				return 0
			}
			return len(list.Items)
		}, 300, 1).Should(BeNumerically(">", 0))
	})
	It("Hub: Dexclient is created for the managed cluster", func() {
		gvr, err := utils.GetGVRForResource("DexClient")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("idp-mgmt-testdomain").
				List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				logf.Log.Info("Error while reading Dexclient in hub cluster", "Error", err)
				return 0
			}
			return len(list.Items)
		}, 300, 1).Should(BeNumerically(">", 0))
	})
	It("Hub: ClusterOauth is created in the managed cluster ns", func() {
		gvr, err := utils.GetGVRForResource("ClusterOAuth")
		Expect(err).NotTo(HaveOccurred())
		mcName := os.Getenv("MANAGED_CLUSTER_NAME")
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace(mcName).
				Get(context.TODO(), "authrealm-sample-my-authrealm", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading ClusterOAuth in hub cluster", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Hub: ManifestWork is created", func() {
		gvr, err := utils.GetGVRForResource("ManifestWork")
		Expect(err).NotTo(HaveOccurred())
		mcName := os.Getenv("MANAGED_CLUSTER_NAME")
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace(mcName).
				Get(context.TODO(), "idp-oauth", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading ManifestWork in hub cluster", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(BeNil())
	})
	It("Managed Cluster: Verify that the OAuth is updated with the IDP", func() {
		gvr, err := utils.GetGVRForResource("OAuth")
		Expect(err).NotTo(HaveOccurred())
		var idps []interface{}
		Eventually(func() bool {
			oauth, err := TestOptions.ManagedClusters[0].KubeClientDynamic.Resource(gvr).
				Get(context.TODO(), "cluster", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading OAuth in managed cluster", "Error", err)
				return false
			}
			var found bool
			idps, found, err = unstructured.NestedSlice(oauth.UnstructuredContent(), "spec", "identityProviders")
			if err != nil {
				logf.Log.Info("Error while reading idps in managed cluster OAuth", "Error", err)
				return false
			}
			return found
		}, 600, 1).Should(BeTrue())
		Expect(len(idps)).To(BeNumerically(">", 0))
		idp0 := idps[0].(map[string]interface{})
		Expect(idp0["name"]).To(Equal("authrealm-sample"))
	})
	It("Hub: Deleting the AuthRealm should delete the DexClient, ClusterOauth and ManifestWork", func() {
		By("Deleting the AuthRealm")
		gvr, err := utils.GetGVRForResource("AuthRealm")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("my-authrealm").
				Delete(context.TODO(), "authrealm-sample", metav1.DeleteOptions{})
			if err != nil {
				logf.Log.Info("Error while deleting AuthRealm", "Error", err)
				return err
			}
			return nil
		}, 600, 1).Should(BeNil())
		By("Checking that the DexClient is deleted")
		gvr, err = utils.GetGVRForResource("DexClient")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace("idp-mgmt-testdomain").
				List(context.TODO(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			return len(list.Items)
		}, 300, 1).Should(Equal(0))
		By("Checking that the ClusterOAuth is deleted")
		gvr, err = utils.GetGVRForResource("ClusterOAuth")
		mcName := os.Getenv("MANAGED_CLUSTER_NAME")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			list, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace(mcName).
				List(context.TODO(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			return len(list.Items)
		}, 300, 1).Should(Equal(0))
		By("Checking that the ManifestWork is deleted")
		gvr, err = utils.GetGVRForResource("ManifestWork")
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClientDynamic.Resource(gvr).
				Namespace(mcName).
				Get(context.TODO(), "idp-oauth", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading ManifestWork in hub cluster", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(HaveOccurred())
	})
	It("Managed Cluster: Once AuthRealm is deleted on the hub, managed cluster Oauth should be restored to its original value", func() {
		gvr, err := utils.GetGVRForResource("OAuth")
		Expect(err).NotTo(HaveOccurred())
		var found bool
		var oauth *unstructured.Unstructured
		Eventually(func() bool {
			oauth, err = TestOptions.ManagedClusters[0].KubeClientDynamic.Resource(gvr).
				Get(context.TODO(), "cluster", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading OAuth in managed cluster", "Error", err)
				return true
			}
			_, found, err = unstructured.NestedSlice(oauth.UnstructuredContent(), "spec", "identityProviders")
			if err != nil {
				logf.Log.Info("Error while reading IDPs from OAuth in managed cluster", "Error", err)
				return true
			}
			return found
		}, 300, 1).Should(BeFalse())

		By("Wait for the idp-mgmt-dex namespace to be deleted")
		Eventually(func() error {
			_, err := TestOptions.HubCluster.KubeClient.CoreV1().
				Namespaces().
				Get(context.TODO(), "idp-mgmt-dex", metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error retrieving idp-mgmt-dex namespace in hub cluster", "Error", err)
				return err
			}
			return nil
		}, 300, 1).Should(HaveOccurred())
		By("Remove labels from the managed cluster")
		gvr, err = utils.GetGVRForResource("ManagedCluster")
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
		if authDepLabelFound {
			delete(labels, "authdeployment")
		}
		_, clusterSetLabelFound := labels["cluster.open-cluster-management.io/clusterset"]
		if clusterSetLabelFound {
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
	})
})
