// Copyright Red Hat

//go:build functional
// +build functional

package functional

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	dexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	OAUTH string = `apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  annotations:
    include.release.openshift.io/ibm-cloud-managed: "true"
    include.release.openshift.io/self-managed-high-availability: "true"
    include.release.openshift.io/single-node-developer: "true"
    release.openshift.io/create-only: "true"
  creationTimestamp: "2021-09-30T15:39:20Z"
  name: cluster
  uid: 7417f691-dff7-4869-9813-d492e9b7cec9
spec: {}`
)

func init() {
	klog.SetOutput(GinkgoWriter)
	klog.InitFlags(nil)

}

// var _ = Describe("Webhook", func() {
// 	AuthRealmName := "my-authrealm-webhook"
// 	AuthRealmNameSpace := "my-authrealm-ns-webhook"
// 	By("creation test namespace", func() {
// 		ns := &corev1.Namespace{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: AuthRealmNameSpace,
// 			},
// 		}
// 		_, err := kubeClient.
// 			CoreV1().
// 			Namespaces().
// 			Create(context.TODO(), ns, metav1.CreateOptions{})
// 		Expect(err).To(BeNil())
// 	})
// 	By("Create a AuthRealm with no type", func() {
// 		authRealm := identitatemv1alpha1.AuthRealm{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      AuthRealmName,
// 				Namespace: AuthRealmNameSpace,
// 			},
// 			Spec: identitatemv1alpha1.AuthRealmSpec{},
// 		}
// 		_, err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), &authRealm, metav1.CreateOptions{})
// 		Expect(err).ToNot(BeNil())
// 	})
// 	By("Create a AuthRealm with type", func() {
// 		authRealm := identitatemv1alpha1.AuthRealm{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      AuthRealmName,
// 				Namespace: AuthRealmNameSpace,
// 			},
// 			Spec: identitatemv1alpha1.AuthRealmSpec{
// 				Type: identitatemv1alpha1.AuthProxyDex,
// 			},
// 		}
// 		_, err := authClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), &authRealm, metav1.CreateOptions{})
// 		Expect(err).To(BeNil())
// 	})

// })

var checkEnvironment = func() {
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
}

var createPlacement = func(testData TestData) {
	By(fmt.Sprintf("creation test namespace %s", testData.AuthRealm.Namespace), func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testData.AuthRealm.Namespace,
			},
		}
		_, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		Expect(err).To(BeNil())
	})
	By(fmt.Sprintf("Creating placement %s", testData.PlacementName), func() {
		placement := &clusterv1alpha1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testData.PlacementName,
				Namespace: testData.AuthRealm.Namespace,
			},
			Spec: clusterv1alpha1.PlacementSpec{
				Predicates: []clusterv1alpha1.ClusterPredicate{
					{
						RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"mylabel": "test",
								},
							},
						},
					},
				},
			},
		}
		var err error
		_, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			Create(context.TODO(), placement, metav1.CreateOptions{})
		Expect(err).To(BeNil())

	})
}

var updatePlacement = func(name, namespace string, predicates []clusterv1alpha1.ClusterPredicate) {
	var placement *clusterv1alpha1.Placement
	By("Getting the placement", func() {
		var err error
		placement, err = clientSetCluster.ClusterV1alpha1().Placements(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		Expect(err).To(BeNil())
	})

	By(fmt.Sprintf("Updating placement %s with new predicates", name), func() {
		placement.Spec.Predicates = predicates
		var err error
		_, err = clientSetCluster.ClusterV1alpha1().Placements(namespace).
			Update(context.TODO(), placement, metav1.UpdateOptions{})
		Expect(err).To(BeNil())
	})
}

var createManagedCluster = func(testData TestData) {
	By(fmt.Sprintf("creation cluster namespace %s", testData.ClusterName), func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testData.ClusterName,
			},
		}
		err := k8sClient.Create(context.TODO(), ns)
		Expect(err).To(BeNil())
	})
	By(fmt.Sprintf("Creating the managedcluster %s", testData.ClusterName), func() {
		mc := &clusterv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: testData.ClusterName,
			},
			Spec: clusterv1.ManagedClusterSpec{
				ManagedClusterClientConfigs: []clusterv1.ClientConfig{
					{
						URL: "https://api.my-cluster.com:6443",
					},
				},
			},
		}
		_, err := clientSetCluster.ClusterV1().ManagedClusters().Create(context.TODO(), mc, metav1.CreateOptions{})
		Expect(err).To(BeNil())
	})
}

var createAuthRealm = func(testData TestData) {
	By("Create a AuthRealm", func() {
		_, err := identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(testData.AuthRealm.Namespace).Create(context.TODO(), testData.AuthRealm, metav1.CreateOptions{})
		Expect(err).To(BeNil())
	})
	By("Checking the AuthRealm on authrealm creation", func() {
		Eventually(func() error {
			_, err := identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(testData.AuthRealm.Namespace).Get(context.TODO(), testData.AuthRealm.Name, metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading authrealm", "Error", err)
				return err
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
}

var checkGeneratedResources = func(testData TestData) {
	By("Checking the dex-operator deployment creation", func() {
		Eventually(func() error {
			_, err := kubeClient.AppsV1().Deployments(helpers.DexOperatorNamespace()).
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
			_, err := identitatemClientSet.
				IdentityconfigV1alpha1().
				Strategies(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.StrategyBackplaneName, metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading strategy",
					"namespace", testData.AuthRealm.Namespace,
					"name", testData.StrategyBackplaneName,
					"Error", err)
				return err
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
	By("Checking the strategy hypershift creation", func() {
		Eventually(func() error {
			_, err := identitatemClientSet.
				IdentityconfigV1alpha1().
				Strategies(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.StrategyHypershiftName, metav1.GetOptions{})
			if err != nil {
				logf.Log.Info("Error while reading strategy",
					"namespace", testData.AuthRealm.Namespace,
					"name", testData.StrategyBackplaneName,
					"Error", err)
				return err
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking creation of strategy placement %s", testData.PlacementStrategyBackplaneName), func() {
		var strategyPlacement *clusterv1alpha1.Placement
		Eventually(func() error {
			var err error
			strategyPlacement, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).Get(context.TODO(), testData.PlacementStrategyBackplaneName, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("Placement", "Name", testData.PlacementStrategyBackplaneName, "Namespace", testData.AuthRealm.Namespace)
				return err
			}
			placement, err := clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).Get(context.TODO(), testData.PlacementName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			found := false
			for _, p := range strategyPlacement.Spec.Predicates {
				logf.Log.Info("StrategyPlacement", "Namespace", strategyPlacement.Namespace, "Name", strategyPlacement.Name, "LabelSelector", p.RequiredClusterSelector.LabelSelector)
				logf.Log.Info("Placement", "Namespace", placement.Namespace, "Name", placement.Name, "LabelSelector", placement.Spec.Predicates[0].RequiredClusterSelector.LabelSelector)
				if equality.Semantic.DeepEqual(p.RequiredClusterSelector.LabelSelector, placement.Spec.Predicates[0].RequiredClusterSelector.LabelSelector) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Placement strategy %s/%s doesn't contain the placement labelselector", strategyPlacement.Namespace, strategyPlacement.Name)
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking creation of strategy placement %s", testData.PlacementStrategyHypershiftName), func() {
		var strategyPlacement *clusterv1alpha1.Placement
		Eventually(func() error {
			var err error
			strategyPlacement, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).Get(context.TODO(), testData.PlacementStrategyHypershiftName, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("Placement", "Name", testData.PlacementStrategyHypershiftName, "Namespace", testData.AuthRealm.Namespace)
				return err
			}
			placement, err := clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).Get(context.TODO(), testData.PlacementName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			found := false
			for _, p := range strategyPlacement.Spec.Predicates {
				logf.Log.Info("StrategyPlacement", "Namespace", strategyPlacement.Namespace, "Name", strategyPlacement.Name, "LabelSelector", p.RequiredClusterSelector.LabelSelector)
				logf.Log.Info("Placement", "Namespace", placement.Namespace, "Name", placement.Name, "LabelSelector", placement.Spec.Predicates[0].RequiredClusterSelector.LabelSelector)
				if equality.Semantic.DeepEqual(p.RequiredClusterSelector.LabelSelector, placement.Spec.Predicates[0].RequiredClusterSelector.LabelSelector) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Placement strategy %s/%s doesn't contain the placement labelselector", strategyPlacement.Namespace, strategyPlacement.Name)
			}
			return nil
		}, 60, 1).Should(BeNil())

	})
}

var emulateAddClusterPlacementController = func(testData TestData) {
	var placementStrategy *clusterv1alpha1.Placement
	By("Checking placement strategy", func() {
		var err error
		placementStrategy, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			Get(context.TODO(), testData.PlacementStrategyBackplaneName, metav1.GetOptions{})
		Expect(err).To(BeNil())
	})

	var placementDecision *clusterv1alpha1.PlacementDecision
	By("Create Placement Decision CR with the cluster in it", func() {
		placementDecision = &clusterv1alpha1.PlacementDecision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testData.StrategyBackplaneName,
				Namespace: testData.AuthRealm.Namespace,
				Labels: map[string]string{
					clusterv1alpha1.PlacementLabel: placementStrategy.Name,
				},
			},
		}
		controllerutil.SetOwnerReference(placementStrategy, placementDecision, scheme.Scheme)
		var err error
		placementDecision, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(testData.AuthRealm.Namespace).
			Create(context.TODO(), placementDecision, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		Eventually(func() error {
			placementDecision, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(testData.AuthRealm.Namespace).Get(context.TODO(), testData.StrategyBackplaneName, metav1.GetOptions{})
			Expect(err).To(BeNil())

			placementDecision.Status.Decisions = []clusterv1alpha1.ClusterDecision{
				{
					ClusterName: testData.ClusterName,
				},
			}
			_, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(testData.AuthRealm.Namespace).
				UpdateStatus(context.TODO(), placementDecision, metav1.UpdateOptions{})
			return err
		}, 30, 1).Should(BeNil())
	})

	By("Update Placement CR", func() {
		placement, err := clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			Get(context.TODO(), testData.PlacementStrategyBackplaneName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		Expect(len(placement.Spec.Predicates)).Should(Equal(1))

		placement.Status.NumberOfSelectedClusters = 1
		_, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			UpdateStatus(context.TODO(), placement, metav1.UpdateOptions{})
	})
}

var checkManagedCluster = func(testData TestData) {
	authRealmObjectKey := client.ObjectKey{Name: testData.AuthRealm.Name, Namespace: testData.AuthRealm.Namespace}
	var authRealm *identitatemv1alpha1.AuthRealm
	By("Getting the authrealm", func() {
		var err error
		authRealm, err = identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(testData.AuthRealm.Namespace).Get(context.TODO(), testData.AuthRealm.Name, metav1.GetOptions{})
		Expect(err).To(BeNil())
	})
	By(fmt.Sprintf("Checking clusterOAuth %s", helpers.ClusterOAuthName(authRealmObjectKey)), func() {
		Eventually(func() error {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ClusterOAuthName(authRealmObjectKey), Namespace: testData.ClusterName}, clusterOAuth)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("clusterOAuth", "Name", helpers.ClusterOAuthName(authRealmObjectKey), "Namespace", testData.ClusterName)
				return err
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
	var errCM error
	By("Getting idp-oauth-original", func() {
		Eventually(func() error {
			cm := &corev1.ConfigMap{}
			errCM = k8sClient.Get(context.TODO(), client.ObjectKey{Name: "idp-oauth-original", Namespace: testData.ClusterName}, cm)
			logf.Log.Info("Getting idp-oauth-original", "error", errCM)

			if errCM != nil && !errors.IsNotFound(errCM) {
				return errCM
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
	if errors.IsNotFound(errCM) {
		By(fmt.Sprintf("Checking ManagedClusterView %s", helpers.ManagedClusterViewOAuthName()), func() {
			Eventually(func() error {
				mcv := &viewv1beta1.ManagedClusterView{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ManagedClusterViewOAuthName(), Namespace: testData.ClusterName}, mcv)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					//Only if no cm, the managedclusterview must be created
					if errors.IsNotFound(errCM) {
						logf.Log.Info("ManagedClusterView", "Name", helpers.ManagedClusterViewOAuthName(), "Namespace", testData.ClusterName)
						return err
					}
				}
				return nil
			}, 60, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Update status ManagedClusterView %s", helpers.ManagedClusterViewOAuthName()), func() {
			gvr := schema.GroupVersionResource{Group: "view.open-cluster-management.io", Version: "v1beta1", Resource: "managedclusterviews"}
			u, err := dynamicClient.Resource(gvr).Namespace(testData.ClusterName).Get(context.TODO(), helpers.ManagedClusterViewOAuthName(), metav1.GetOptions{})
			Expect(err).To(BeNil())
			mcv := &viewv1beta1.ManagedClusterView{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), mcv)
			Expect(err).To(BeNil())
			b, err := yaml.YAMLToJSON([]byte(OAUTH))
			Expect(err).To(BeNil())
			mcv.Status.Result.Raw = b
			uc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(mcv)
			Expect(err).To(BeNil())
			u.SetUnstructuredContent(uc)
			_, err = dynamicClient.Resource(gvr).Namespace(testData.ClusterName).UpdateStatus(context.TODO(), u, metav1.UpdateOptions{})
			Expect(err).To(BeNil())
		})
	}
	By(fmt.Sprintf("Checking Manifestwork %s", helpers.ManifestWorkOAuthName()), func() {
		Eventually(func() error {
			mw := &manifestworkv1.ManifestWork{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ManifestWorkOAuthName(), Namespace: testData.ClusterName}, mw)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("Manifestwork", "Name", helpers.ManifestWorkOAuthName(), "Namespace", testData.ClusterName)
				return err
			}
			return nil
		}, 120, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking client secret %s", helpers.ClientSecretName(authRealmObjectKey)), func() {
		Eventually(func() error {
			clientSecret := &corev1.Secret{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ClientSecretName(authRealmObjectKey), Namespace: testData.ClusterName}, clientSecret)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("ClientSecret", "Name", helpers.ClientSecretName(authRealmObjectKey), "Namespace", testData.ClusterName)
				return err
			}
			return nil
		}, 60, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking DexClient %s", helpers.DexClientName(authRealmObjectKey, testData.ClusterName)), func() {
		Eventually(func() error {
			dexClient := &dexv1alpha1.DexClient{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.DexClientName(authRealmObjectKey, testData.ClusterName), Namespace: helpers.DexServerNamespace(authRealm)}, dexClient)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				logf.Log.Info("DexClient", "Name", helpers.DexClientName(authRealmObjectKey, testData.ClusterName), "Namespace", helpers.DexServerNamespace(authRealm))
				return err
			}
			return nil
		}, 30, 1).Should(BeNil())
	})
}

var emulateRemoveClusterPlacementController = func(testData TestData) {
	By("Remove cluster from placementdecision", func() {
		placementDecision, err := clientSetCluster.ClusterV1alpha1().PlacementDecisions(testData.AuthRealm.Namespace).Get(context.TODO(), testData.StrategyBackplaneName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		placementDecision.Status.Decisions = []clusterv1alpha1.ClusterDecision{}
		_, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(testData.AuthRealm.Namespace).
			UpdateStatus(context.TODO(), placementDecision, metav1.UpdateOptions{})
		Expect(err).To(BeNil())
	})
	By("Remove the cluster from placement", func() {
		placement, err := clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			Get(context.TODO(), testData.PlacementStrategyBackplaneName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		Expect(len(placement.Spec.Predicates)).Should(Equal(1))

		placement.Status.NumberOfSelectedClusters = 0
		_, err = clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).
			UpdateStatus(context.TODO(), placement, metav1.UpdateOptions{})
	})
}

var checkManagedClusterClean = func(testData TestData) {
	authRealmObjectKey := client.ObjectKey{Name: testData.AuthRealm.Name, Namespace: testData.AuthRealm.Namespace}
	By(fmt.Sprintf("Checking client secret deletion %s", helpers.ClientSecretName(authRealmObjectKey)), func() {
		Eventually(func() error {
			clientSecret := &corev1.Secret{}
			err := k8sClient.Get(context.TODO(),
				client.ObjectKey{Name: helpers.ClientSecretName(authRealmObjectKey), Namespace: testData.ClusterName},
				clientSecret)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				return nil
			}
			return fmt.Errorf("clientSecret %s in ns %s still exist", helpers.ClusterOAuthName(authRealmObjectKey), testData.ClusterName)
		}, 60, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking clusteroauth deletion %s", testData.AuthRealm.Name), func() {
		Eventually(func() error {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
			err := k8sClient.Get(context.TODO(),
				client.ObjectKey{Name: helpers.ClusterOAuthName(authRealmObjectKey), Namespace: testData.ClusterName},
				clusterOAuth)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				return nil
			}
			return fmt.Errorf("clusteroauth %s still exist", helpers.ClusterOAuthName(authRealmObjectKey))
		}, 60, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking manifestwork deletion %s", helpers.ManifestWorkOAuthName()), func() {
		Eventually(func() error {
			mw := &clusterv1.ManagedCluster{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ManifestWorkOAuthName(), Namespace: testData.ClusterName}, mw)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				return nil
			}
			return fmt.Errorf("manifestwork %s still exist", helpers.ManifestWorkOAuthName())
		}, 30, 1).Should(BeNil())
	})
	By(fmt.Sprintf("Checking DexClient deletion %s", testData.AuthRealm.Name), func() {
		Eventually(func() error {
			dexClient := &dexv1alpha1.DexClient{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: testData.AuthRealm.Name, Namespace: testData.AuthRealm.Name}, dexClient)
			if err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				return nil
			}
			return fmt.Errorf("DexClient %s still exist", testData.AuthRealm.Name)

		}, 30, 1).Should(BeNil())
	})
}

var checkRestoreOAuth = func(testData TestData) {
	By("Setting restore oauth manifestwork to Applied", func() {
		Eventually(func() error {
			gvr := schema.GroupVersionResource{Group: "work.open-cluster-management.io", Version: "v1", Resource: "manifestworks"}
			u, err := dynamicClient.Resource(gvr).Namespace(testData.ClusterName).Get(context.TODO(), helpers.ManifestWorkOriginalOAuthName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			mcv := &manifestworkv1.ManifestWork{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), mcv)
			if err != nil {
				return err
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
				return err
			}
			u.SetUnstructuredContent(uc)
			_, err = dynamicClient.Resource(gvr).Namespace(testData.ClusterName).UpdateStatus(context.TODO(), u, metav1.UpdateOptions{})
			return err

		}, 60, 1).Should(BeNil())
	})
}

var deleteAuthRealm = func(testData TestData) {
	By("Deleting the authrealm", func() {
		err := identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(testData.AuthRealm.Namespace).Delete(context.TODO(), testData.AuthRealm.Name, metav1.DeleteOptions{})
		Expect(err).To(BeNil())
	})
}

var checkPlacementClean = func(testData TestData) {
	By("Checking placement Backplane deleted", func() {
		Eventually(func() error {
			_, err := clientSetCluster.
				ClusterV1alpha1().
				Placements(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.PlacementName+"-"+string(identitatemv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
			return err
		}, 120, 1).ShouldNot(BeNil())
	})
	By("Checking placement Hypershift deleted", func() {
		Eventually(func() error {
			_, err := clientSetCluster.
				ClusterV1alpha1().
				Placements(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.PlacementName+"-"+string(identitatemv1alpha1.HypershiftStrategyType), metav1.GetOptions{})
			return err
		}, 120, 1).ShouldNot(BeNil())
	})
}

var checkAuthrealmClean = func(testData TestData) {
	By("Checking placements", func() { checkPlacementClean(testData) })
	By("Checking strategy Backplane deleted", func() {
		Eventually(func() error {
			_, err := identitatemClientSet.
				IdentityconfigV1alpha1().
				Strategies(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.AuthRealm.Name+"-"+string(identitatemv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
			return err
		}, 120, 1).ShouldNot(BeNil())
	})
	By("Checking strategy Hypershift deleted", func() {
		Eventually(func() error {
			_, err := identitatemClientSet.
				IdentityconfigV1alpha1().
				Strategies(testData.AuthRealm.Namespace).
				Get(context.TODO(), testData.AuthRealm.Name+"-"+string(identitatemv1alpha1.HypershiftStrategyType), metav1.GetOptions{})
			return err
		}, 60, 1).ShouldNot(BeNil())
	})
	By("Authrealm is deleted", func() {
		Eventually(func() error {
			_, err := identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(testData.AuthRealm.Namespace).Get(context.TODO(), testData.AuthRealm.Name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				logf.Log.Info("Error while reading authrealm", "Error", err)
				return err
			}
			return fmt.Errorf("Authrealm still exists")
		}, 120, 1).Should(BeNil())
	})
	By("Checking authrealm ns deleted", func() {
		Eventually(func() bool {
			_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), testData.AuthRealm.Name, metav1.GetOptions{})
			return errors.IsNotFound(err)
		}, 60, 1).Should(BeTrue())
	})
}

type TestData struct {
	AuthRealm                       *identitatemv1alpha1.AuthRealm
	PlacementName                   string
	StrategyBackplaneName           string
	StrategyHypershiftName          string
	PlacementStrategyBackplaneName  string
	PlacementStrategyHypershiftName string
	ClusterName                     string
}

var _ = Describe("AuthRealmAddRemoveFromPlacement", func() {
	testData := TestData{
		ClusterName:   "my-cluster-0",
		PlacementName: "my-placement-0",
	}
	testData.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-0",
			Namespace: "my-authrealm-ns-0",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-0",
			Type:           identitatemv1alpha1.AuthProxyDex,
			// CertificatesSecretRef: corev1.LocalObjectReference{
			// 	Name: CertificatesSecretRef,
			// },
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-0",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-0" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-0",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData.PlacementName,
			},
		},
	}
	testData.StrategyBackplaneName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.StrategyHypershiftName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData.PlacementStrategyBackplaneName = testData.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.PlacementStrategyHypershiftName = testData.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	It("Check requirement", func() { checkEnvironment() })
	It(fmt.Sprintf("Create placement %s", testData.PlacementName), func() { createPlacement(testData) })
	It(fmt.Sprintf("create managedcluster %s", testData.ClusterName), func() { createManagedCluster(testData) })

	It(fmt.Sprintf("create AuthRealm %s", testData.AuthRealm.Name), func() { createAuthRealm(testData) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData.AuthRealm.Name), func() { checkGeneratedResources(testData) })

	It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData.PlacementName, testData.ClusterName), func() { emulateAddClusterPlacementController(testData) })
	It(fmt.Sprintf("Check managedcluster %s configuration", testData.ClusterName), func() { checkManagedCluster(testData) })

	It(fmt.Sprintf("emulate a Placement controller for placement %s by removing cluster %s", testData.PlacementName, testData.ClusterName), func() { emulateRemoveClusterPlacementController(testData) })
	It(fmt.Sprintf("check restore oauth %s", testData.AuthRealm.Name), func() { checkRestoreOAuth(testData) })
	It(fmt.Sprintf("check managedcluster %s cleaned", testData.ClusterName), func() { checkManagedClusterClean(testData) })

	It(fmt.Sprintf("delete the authrealm %s", testData.AuthRealm.Name), func() { deleteAuthRealm(testData) })
	It(fmt.Sprintf("check authrealm %s cleaned", testData.AuthRealm.Name), func() { checkAuthrealmClean(testData) })

})

var _ = Describe("CreateDelete", func() {
	testData := TestData{
		ClusterName:   "my-cluster-1",
		PlacementName: "my-placement-1",
	}
	testData.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-1",
			Namespace: "my-authrealm-ns-1",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-1",
			Type:           identitatemv1alpha1.AuthProxyDex,
			// CertificatesSecretRef: corev1.LocalObjectReference{
			// 	Name: CertificatesSecretRef,
			// },
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-1",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-1" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-1",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData.PlacementName,
			},
		},
	}
	testData.StrategyBackplaneName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.StrategyHypershiftName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData.PlacementStrategyBackplaneName = testData.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.PlacementStrategyHypershiftName = testData.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	It("Check requirement", func() { checkEnvironment() })
	It(fmt.Sprintf("Create placement %s", testData.PlacementName), func() { createPlacement(testData) })
	It(fmt.Sprintf("create managedcluster %s", testData.ClusterName), func() { createManagedCluster(testData) })

	It(fmt.Sprintf("create AuthRealm %s", testData.AuthRealm.Name), func() { createAuthRealm(testData) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData.AuthRealm.Name), func() { checkGeneratedResources(testData) })
	It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData.PlacementName, testData.ClusterName), func() { emulateAddClusterPlacementController(testData) })
	It(fmt.Sprintf("Check managedcluster %s configuration", testData.ClusterName), func() { checkManagedCluster(testData) })
	It(fmt.Sprintf("delete the authrealm %s", testData.AuthRealm.Name), func() { deleteAuthRealm(testData) })
	It(fmt.Sprintf("check restore oauth %s", testData.AuthRealm.Name), func() { checkRestoreOAuth(testData) })
	It(fmt.Sprintf("check managedcluster %s cleaned", testData.ClusterName), func() { checkManagedClusterClean(testData) })
	It(fmt.Sprintf("check authrealm %s cleaned", testData.AuthRealm.Name), func() { checkAuthrealmClean(testData) })
})

var _ = Describe("UpatePlacememt", func() {
	testData := TestData{
		ClusterName:   "my-cluster-pl",
		PlacementName: "my-placement-pl",
	}
	testData.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-pl",
			Namespace: "my-authrealm-ns-pl",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-pl",
			Type:           identitatemv1alpha1.AuthProxyDex,
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-pl",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-pl" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-pl",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData.PlacementName,
			},
		},
	}
	testData.StrategyBackplaneName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.StrategyHypershiftName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData.PlacementStrategyBackplaneName = testData.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.PlacementStrategyHypershiftName = testData.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	It("Check requirement", func() { checkEnvironment() })
	It(fmt.Sprintf("Create placement %s", testData.PlacementName), func() { createPlacement(testData) })

	It(fmt.Sprintf("create AuthRealm %s", testData.AuthRealm.Name), func() { createAuthRealm(testData) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData.AuthRealm.Name), func() { checkGeneratedResources(testData) })

	predicates := []clusterv1alpha1.ClusterPredicate{
		{
			RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"mylabel": "newlabel",
					},
				},
			},
		},
	}
	It(fmt.Sprintf("Update placement with new predicate %s", testData.PlacementName), func() { updatePlacement(testData.PlacementName, testData.AuthRealm.Namespace, predicates) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData.AuthRealm.Name), func() { checkGeneratedResources(testData) })
})

var _ = Describe("DeletePlacememt", func() {
	testData := TestData{
		ClusterName:   "my-cluster-dpl",
		PlacementName: "my-placement-dpl",
	}
	testData.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-dpl",
			Namespace: "my-authrealm-ns-dpl",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-dpl",
			Type:           identitatemv1alpha1.AuthProxyDex,
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-dpl",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-dpl" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-dpl",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData.PlacementName,
			},
		},
	}
	testData.StrategyBackplaneName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.StrategyHypershiftName = testData.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData.PlacementStrategyBackplaneName = testData.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData.PlacementStrategyHypershiftName = testData.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	It("Check requirement", func() { checkEnvironment() })
	It(fmt.Sprintf("Create placement %s", testData.PlacementName), func() { createPlacement(testData) })

	It(fmt.Sprintf("create AuthRealm %s", testData.AuthRealm.Name), func() { createAuthRealm(testData) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData.AuthRealm.Name), func() { checkGeneratedResources(testData) })

	It(fmt.Sprintf("Deleting placement %s", testData.PlacementName), func() {
		err := clientSetCluster.ClusterV1alpha1().Placements(testData.AuthRealm.Namespace).Delete(context.TODO(), testData.PlacementName, metav1.DeleteOptions{})
		Expect(err).Should(BeNil())
	})

	It(fmt.Sprintf("Check strategy placement deletion for placement %s", testData.PlacementName), func() { checkPlacementClean(testData) })
})

var _ = Describe("2AuthRealms-2Placements-1Cluster", func() {
	testData1 := TestData{
		//Use the same cluster
		ClusterName:   "my-cluster-22",
		PlacementName: "my-placement-22-1",
	}
	testData1.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-22-1",
			Namespace: "my-authrealm-ns-22-1",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-22-1",
			Type:           identitatemv1alpha1.AuthProxyDex,
			// CertificatesSecretRef: corev1.LocalObjectReference{
			// 	Name: CertificatesSecretRef,
			// },
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-22-1",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-22-1" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-22-1",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData1.PlacementName,
			},
		},
	}
	testData1.StrategyBackplaneName = testData1.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData1.StrategyHypershiftName = testData1.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData1.PlacementStrategyBackplaneName = testData1.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData1.PlacementStrategyHypershiftName = testData1.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)

	testData2 := TestData{
		//Use the same cluster
		ClusterName:   "my-cluster-22",
		PlacementName: "my-placement-22-2",
	}
	testData2.AuthRealm = &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-authrealm-22-2",
			Namespace: "my-authrealm-ns-22-2",
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			RouteSubDomain: "myroute-22-2",
			Type:           identitatemv1alpha1.AuthProxyDex,
			// CertificatesSecretRef: corev1.LocalObjectReference{
			// 	Name: CertificatesSecretRef,
			// },
			IdentityProviders: []openshiftconfigv1.IdentityProvider{
				{
					Name:          "my-idp-22-2",
					MappingMethod: openshiftconfigv1.MappingMethodClaim,
					IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
						Type: openshiftconfigv1.IdentityProviderTypeGitHub,
						GitHub: &openshiftconfigv1.GitHubIdentityProvider{
							ClientSecret: openshiftconfigv1.SecretNameReference{
								Name: "my-authrealm-22-2" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
							},
							ClientID: "my-github-app-client-id-22-2",
						},
					},
				},
			},
			PlacementRef: corev1.LocalObjectReference{
				Name: testData2.PlacementName,
			},
		},
	}
	testData2.StrategyBackplaneName = testData2.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData2.StrategyHypershiftName = testData2.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
	testData2.PlacementStrategyBackplaneName = testData2.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
	testData2.PlacementStrategyHypershiftName = testData2.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)

	It("Check requirement", func() { checkEnvironment() })
	It(fmt.Sprintf("create managedcluster %s", testData1.ClusterName), func() { createManagedCluster(testData1) })

	//Create and check testData1
	It(fmt.Sprintf("Create placement %s", testData1.PlacementName), func() { createPlacement(testData1) })
	It(fmt.Sprintf("create AuthRealm %s", testData1.AuthRealm.Name), func() { createAuthRealm(testData1) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData1.AuthRealm.Name), func() { checkGeneratedResources(testData1) })
	It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData1.PlacementName, testData1.ClusterName), func() { emulateAddClusterPlacementController(testData1) })
	It(fmt.Sprintf("Check managedcluster %s configuration", testData1.ClusterName), func() { checkManagedCluster(testData1) })

	//Create and check testData2
	It(fmt.Sprintf("Create placement %s", testData2.PlacementName), func() { createPlacement(testData2) })
	It(fmt.Sprintf("create AuthRealm %s", testData2.AuthRealm.Name), func() { createAuthRealm(testData2) })
	It(fmt.Sprintf("check generated resources for authrealm %s", testData2.AuthRealm.Name), func() { checkGeneratedResources(testData2) })
	It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData2.PlacementName, testData2.ClusterName), func() { emulateAddClusterPlacementController(testData2) })
	It(fmt.Sprintf("Check managedcluster %s configuration", testData2.ClusterName), func() { checkManagedCluster(testData2) })

	//Delete and check testData1
	It(fmt.Sprintf("delete the authrealm %s", testData1.AuthRealm.Name), func() { deleteAuthRealm(testData1) })
	It(fmt.Sprintf("check managedcluster %s cleaned", testData1.ClusterName), func() { checkManagedClusterClean(testData1) })
	It(fmt.Sprintf("check authrealm %s cleaned", testData1.AuthRealm.Name), func() { checkAuthrealmClean(testData1) })

	//Check testData2
	It(fmt.Sprintf("check generated resources for authrealm %s", testData2.AuthRealm.Name), func() { checkGeneratedResources(testData2) })
	It(fmt.Sprintf("Check managedcluster %s configuration", testData2.ClusterName), func() { checkManagedCluster(testData2) })

	//Delete and check testData2
	It(fmt.Sprintf("delete the authrealm %s", testData2.AuthRealm.Name), func() { deleteAuthRealm(testData2) })
	It(fmt.Sprintf("check restore oauth %s", testData2.AuthRealm.Name), func() { checkRestoreOAuth(testData2) })
	It(fmt.Sprintf("check managedcluster %s cleaned", testData2.ClusterName), func() { checkManagedClusterClean(testData2) })
	It(fmt.Sprintf("check authrealm %s cleaned", testData2.AuthRealm.Name), func() { checkAuthrealmClean(testData2) })

})

// var _ = Describe("2AuthRealms-1Placement-1Cluster", func() {
// 	Context("221", func() {
// 		testData1 := TestData{
// 			//Use the same cluster
// 			ClusterName:   "my-cluster-21",
// 			PlacementName: "my-placement-21",
// 		}
// 		testData1.AuthRealm = &identitatemv1alpha1.AuthRealm{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "my-authrealm-21-1",
// 				Namespace: "my-authrealm-ns-21",
// 			},
// 			Spec: identitatemv1alpha1.AuthRealmSpec{
// 				RouteSubDomain: "myroute-21-1",
// 				Type:           identitatemv1alpha1.AuthProxyDex,
// 				// CertificatesSecretRef: corev1.LocalObjectReference{
// 				// 	Name: CertificatesSecretRef,
// 				// },
// 				IdentityProviders: []openshiftconfigv1.IdentityProvider{
// 					{
// 						Name:          "my-idp-21-1",
// 						MappingMethod: openshiftconfigv1.MappingMethodClaim,
// 						IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
// 							Type: openshiftconfigv1.IdentityProviderTypeGitHub,
// 							GitHub: &openshiftconfigv1.GitHubIdentityProvider{
// 								ClientSecret: openshiftconfigv1.SecretNameReference{
// 									Name: "my-authrealm-21-1" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
// 								},
// 								ClientID: "my-github-app-client-id-21-1",
// 							},
// 						},
// 					},
// 				},
// 				PlacementRef: corev1.LocalObjectReference{
// 					Name: testData1.PlacementName,
// 				},
// 			},
// 		}
// 		testData1.StrategyBackplaneName = testData1.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
// 		testData1.StrategyHypershiftName = testData1.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
// 		testData1.PlacementStrategyBackplaneName = testData1.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
// 		testData1.PlacementStrategyHypershiftName = testData1.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)

// 		testData2 := TestData{
// 			//Use the same cluster
// 			ClusterName:   "my-cluster-21",
// 			PlacementName: "my-placement-21",
// 		}
// 		testData2.AuthRealm = &identitatemv1alpha1.AuthRealm{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "my-authrealm-21-2",
// 				Namespace: "my-authrealm-ns-21",
// 			},
// 			Spec: identitatemv1alpha1.AuthRealmSpec{
// 				RouteSubDomain: "myroute-21-2",
// 				Type:           identitatemv1alpha1.AuthProxyDex,
// 				// CertificatesSecretRef: corev1.LocalObjectReference{
// 				// 	Name: CertificatesSecretRef,
// 				// },
// 				IdentityProviders: []openshiftconfigv1.IdentityProvider{
// 					{
// 						Name:          "my-idp-21-2",
// 						MappingMethod: openshiftconfigv1.MappingMethodClaim,
// 						IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
// 							Type: openshiftconfigv1.IdentityProviderTypeGitHub,
// 							GitHub: &openshiftconfigv1.GitHubIdentityProvider{
// 								ClientSecret: openshiftconfigv1.SecretNameReference{
// 									Name: "my-authrealm-21-2" + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
// 								},
// 								ClientID: "my-github-app-client-id-21-2",
// 							},
// 						},
// 					},
// 				},
// 				PlacementRef: corev1.LocalObjectReference{
// 					Name: testData2.PlacementName,
// 				},
// 			},
// 		}
// 		testData2.StrategyBackplaneName = testData2.AuthRealm.Name + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
// 		testData2.StrategyHypershiftName = testData2.AuthRealm.Name + "-" + string(identitatemv1alpha1.HypershiftStrategyType)
// 		testData2.PlacementStrategyBackplaneName = testData2.PlacementName + "-" + string(identitatemv1alpha1.BackplaneStrategyType)
// 		testData2.PlacementStrategyHypershiftName = testData2.PlacementName + "-" + string(identitatemv1alpha1.HypershiftStrategyType)

// 		It("Check requirement", func() { checkEnvironment() })
// 		It(fmt.Sprintf("create managedcluster %s", testData1.ClusterName), func() { createManagedCluster(testData1) })

// 		//Create and check testData1
// 		It(fmt.Sprintf("Create placement %s", testData1.PlacementName), func() { createPlacement(testData1) })
// 		It(fmt.Sprintf("create AuthRealm %s", testData1.AuthRealm.Name), func() { createAuthRealm(testData1) })
// 		It(fmt.Sprintf("check generated resources for authrealm %s", testData1.AuthRealm.Name), func() { checkGeneratedResources(testData1) })
// 		It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData1.PlacementName, testData1.ClusterName), func() { emulateAddClusterPlacementController(testData1) })
// 		It(fmt.Sprintf("Check managedcluster %s configuration", testData1.ClusterName), func() { checkManagedCluster(testData1) })

// 		//Create and check testData2
// 		It(fmt.Sprintf("create AuthRealm %s", testData2.AuthRealm.Name), func() { createAuthRealm(testData2) })
// 		It(fmt.Sprintf("check generated resources for authrealm %s", testData2.AuthRealm.Name), func() { checkGeneratedResources(testData2) })
// 		It(fmt.Sprintf("emulate a Placement controller for placement %s by adding cluster %s", testData2.PlacementName, testData2.ClusterName), func() { emulateAddClusterPlacementController(testData2) })
// 		It(fmt.Sprintf("Check managedcluster %s configuration", testData2.ClusterName), func() { checkManagedCluster(testData2) })

// 		//Delete and check testData1
// 		It(fmt.Sprintf("delete the authrealm %s", testData1.AuthRealm.Name), func() { deleteAuthRealm(testData1) })
// 		It(fmt.Sprintf("check managedcluster %s cleaned", testData1.ClusterName), func() { checkManagedClusterClean(testData1) })
// 		It(fmt.Sprintf("check authrealm %s cleaned", testData1.AuthRealm.Name), func() { checkAuthrealmClean(testData1) })

// 		// Check testData2
// 		It(fmt.Sprintf("check generated resources for authrealm %s", testData2.AuthRealm.Name), func() { checkGeneratedResources(testData2) })
// 		It(fmt.Sprintf("Check managedcluster %s configuration", testData2.ClusterName), func() { checkManagedCluster(testData2) })

// 		// Delete and check testData2
// 		It(fmt.Sprintf("delete the authrealm %s", testData2.AuthRealm.Name), func() { deleteAuthRealm(testData2) })
// 		It(fmt.Sprintf("check restore oauth %s", testData2.AuthRealm.Name), func() { checkRestoreOAuth(testData2) })
// 		It(fmt.Sprintf("check managedcluster %s cleaned", testData2.ClusterName), func() { checkManagedClusterClean(testData2) })
// 		It(fmt.Sprintf("check authrealm %s cleaned", testData2.AuthRealm.Name), func() { checkAuthrealmClean(testData2) })

// 	})

// })

var _ = Describe("NonStrategyPlacement", func() {
	AuthRealmNameSpace := "my-authrealmns-nps"
	// CertificatesSecretRef := "my-certs"
	PlacementName := "non-strategy-placement"
	It("process a Strategy", func() {
		By(fmt.Sprintf("creation of User namespace %s", AuthRealmNameSpace), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNameSpace,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		var placement *clusterv1alpha1.Placement
		By("Creating a non strategy placement", func() {
			placement = &clusterv1alpha1.Placement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlacementName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: clusterv1alpha1.PlacementSpec{
					Predicates: []clusterv1alpha1.ClusterPredicate{
						{
							RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
								LabelSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{
										"mylabel": "test",
									},
								},
							},
						},
					},
				},
			}
			var err error
			placement, err = clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
				Create(context.TODO(), placement, metav1.CreateOptions{})
			Expect(err).To(BeNil())

		})
		By("delete non strategy placement", func() {
			err := clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
				Delete(context.TODO(), PlacementName, metav1.DeleteOptions{})
			Expect(err).To(BeNil())
			Eventually(func() error {
				_, err := clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).Get(context.TODO(), PlacementName, metav1.GetOptions{})
				if err == nil {
					return fmt.Errorf("non strategy placement %s/%s still exists", AuthRealmNameSpace, PlacementName)
				}
				return nil
			}, 60, 1).Should(BeNil())

		})
	})
})
