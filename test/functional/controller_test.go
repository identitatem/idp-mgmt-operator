// Copyright Red Hat

// +build functional

package functional

import (
	"context"
	"fmt"

	dexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	manifestworkv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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

var _ = Describe("AuthRealm", func() {
	AuthRealmName := "my-authrealm-1"
	AuthRealmNameSpace := "my-authrealm-ns-1"
	RouteSubDomain := "myroute-1"
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
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					// CertificatesSecretRef: corev1.LocalObjectReference{
					// 	Name: CertificatesSecretRef,
					// },
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeGitHub,
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientSecret: openshiftconfigv1.SecretNameReference{
										Name: AuthRealmName + "-" + string(openshiftconfigv1.IdentityProviderTypeGitHub),
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
				_, err := authClientSet.
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
		// By("Checking the strategy grc creation", func() {
		// 	Eventually(func() error {
		// 		_, err := authClientSet.
		// 			IdentityconfigV1alpha1().
		// 			Strategies(AuthRealmNameSpace).
		// 			Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.GrcStrategyType), metav1.GetOptions{})
		// 		if err != nil {
		// 			logf.Log.Info("Error while reading strategy", "Error", err)
		// 			return err
		// 		}
		// 		return nil
		// 	}, 60, 1).Should(BeNil())
		// })
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
				_, err := authClientSet.
					IdentityconfigV1alpha1().
					Strategies(AuthRealmNameSpace).
					Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.BackplaneStrategyType), metav1.GetOptions{})
				return err
			}, 60, 1).ShouldNot(BeNil())
		})
		// By("Checking strategy GRC deleted", func() {
		// 	Eventually(func() error {
		// 		_, err := strategyClientSet.
		// 			IdentityconfigV1alpha1().
		// 			Strategies(AuthRealmNameSpace).
		// 			Get(context.TODO(), AuthRealmName+"-"+string(identitatemv1alpha1.GrcStrategyType), metav1.GetOptions{})
		// 		return err
		// 	}, 60, 1).ShouldNot(BeNil())
		// })
	})

})

var _ = Describe("Strategy", func() {
	AuthRealmName := "my-authrealm"
	AuthRealmNameSpace := "my-authrealmns"
	// CertificatesSecretRef := "my-certs"
	StrategyName := AuthRealmName + "-backplane"
	PlacementStrategyName := StrategyName
	ClusterName := "my-cluster"
	MyIDPName := "my-idp"
	MyGithubAppClientID := "my-github-app-client-id"
	RouteSubDomain := "myroute"
	var authRealm *identitatemv1alpha1.AuthRealm
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
		By("Creating placement", func() {
			placement = &clusterv1alpha1.Placement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
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
		By("creating a AuthRealm CR", func() {
			var err error
			authRealm = &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					// CertificatesSecretRef: corev1.LocalObjectReference{
					// 	Name: CertificatesSecretRef,
					// },
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							Name:          MyIDPName,
							MappingMethod: openshiftconfigv1.MappingMethodClaim,
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeGitHub,
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientID: MyGithubAppClientID,
								},
							},
						},
					},
					PlacementRef: corev1.LocalObjectReference{
						Name: placement.Name,
					},
				},
			}
			//DV reassign  to authRealm to get the extra info that kube set (ie:uuid as needed to set ownerref)
			authRealm, err = identitatemClientSet.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Create a Backplane Strategy", func() {
			strategy := &identitatemv1alpha1.Strategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      StrategyName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.StrategySpec{
					Type: identitatemv1alpha1.BackplaneStrategyType,
				},
			}
			controllerutil.SetOwnerReference(authRealm, strategy, scheme.Scheme)
			_, err := identitatemClientSet.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Create(context.TODO(), strategy, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("creation cluster namespace", func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		By(fmt.Sprintf("Checking creation of strategy placement %s", PlacementStrategyName), func() {
			Eventually(func() error {
				_, err := clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).Get(context.TODO(), PlacementStrategyName, metav1.GetOptions{})
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					logf.Log.Info("Placement", "Name", PlacementStrategyName, "Namespace", AuthRealmNameSpace)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
			By("Checking strategy", func() {
				var err error
				var strategy *identitatemv1alpha1.Strategy
				Eventually(func() error {
					strategy, err = identitatemClientSet.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), StrategyName, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if strategy.Spec.PlacementRef.Name != PlacementStrategyName {
						return fmt.Errorf("Expect PlacementStrategyName = %s but got strategy.Spec.PlacementRef.Name= %s",
							PlacementStrategyName,
							strategy.Spec.PlacementRef.Name)
					}
					return nil
				}, 30, 1).Should(BeNil())
			})
			By("Checking placement strategy", func() {
				_, err := clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
					Get(context.TODO(), PlacementStrategyName, metav1.GetOptions{})
				Expect(err).To(BeNil())
				Expect(len(placement.Spec.Predicates)).Should(Equal(1))
			})
		})
	})
	It("process a PlacementDecision", func() {
		var placementStrategy *clusterv1alpha1.Placement
		By("Checking placement strategy", func() {
			var err error
			placementStrategy, err = clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
				Get(context.TODO(), PlacementStrategyName, metav1.GetOptions{})
			Expect(err).To(BeNil())
		})

		var placementDecision *clusterv1alpha1.PlacementDecision
		By("Create Placement Decision CR", func() {
			placementDecision = &clusterv1alpha1.PlacementDecision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      StrategyName,
					Namespace: AuthRealmNameSpace,
					Labels: map[string]string{
						clusterv1alpha1.PlacementLabel: placementStrategy.Name,
					},
				},
			}
			controllerutil.SetOwnerReference(placementStrategy, placementDecision, scheme.Scheme)
			var err error
			placementDecision, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).
				Create(context.TODO(), placementDecision, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			Eventually(func() error {
				placementDecision, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).Get(context.TODO(), StrategyName, metav1.GetOptions{})
				Expect(err).To(BeNil())

				placementDecision.Status.Decisions = []clusterv1alpha1.ClusterDecision{
					{
						ClusterName: ClusterName,
					},
				}
				_, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).
					UpdateStatus(context.TODO(), placementDecision, metav1.UpdateOptions{})
				return err
			}, 30, 1).Should(BeNil())
		})

		By(fmt.Sprintf("Checking client secret %s", MyIDPName), func() {
			Eventually(func() error {
				clientSecret := &corev1.Secret{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clientSecret)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					logf.Log.Info("ClientSecret", "Name", MyIDPName, "Namespace", ClusterName)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking clusterOAuth %s", MyIDPName), func() {
			Eventually(func() error {
				clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clusterOAuth)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					logf.Log.Info("clusterOAuth", "Name", MyIDPName, "Namespace", ClusterName)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking Manifestwork %s", helpers.ManifestWorkName()), func() {
			Eventually(func() error {
				mw := &manifestworkv1.ManifestWork{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ManifestWorkName(), Namespace: ClusterName}, mw)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					logf.Log.Info("Manifestwork", "Name", helpers.ManifestWorkName(), "Namespace", ClusterName)
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking DexClient %s", ClusterName), func() {
			Eventually(func() error {
				dexClient := &dexv1alpha1.DexClient{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: ClusterName, Namespace: helpers.DexServerNamespace(authRealm)}, dexClient)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					logf.Log.Info("DexClient", "Name", ClusterName, "Namespace", helpers.DexServerNamespace(authRealm))
					return err
				}
				return nil
			}, 30, 1).Should(BeNil())
		})
		By("Deleting the placementdecision", func() {
			err := clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).Delete(context.TODO(), StrategyName, metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		})
		By(fmt.Sprintf("Checking client secret deletion %s", MyIDPName), func() {
			Eventually(func() error {
				clientSecret := &corev1.Secret{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clientSecret)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					return nil
				}
				return fmt.Errorf("clientSecret %s still exist", MyIDPName)
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking clusteroauth deletion %s", MyIDPName), func() {
			Eventually(func() error {
				clientSecret := &identitatemv1alpha1.ClusterOAuth{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clientSecret)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					return nil
				}
				return fmt.Errorf("clusteroauth %s still exist", MyIDPName)
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking manifestwork deletion %s", helpers.ManifestWorkName()), func() {
			Eventually(func() error {
				//TODO read manifest work and not clusterOauth
				clientSecret := &identitatemv1alpha1.ClusterOAuth{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: helpers.ManifestWorkName(), Namespace: ClusterName}, clientSecret)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					return nil
				}
				return fmt.Errorf("manifestwork %s still exist", MyIDPName)
			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking DexClient deletion %s", ClusterName), func() {
			Eventually(func() error {
				dexClient := &dexv1alpha1.DexClient{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: ClusterName, Namespace: AuthRealmName}, dexClient)
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					return nil
				}
				return fmt.Errorf("DexClient %s still exist", ClusterName)

			}, 30, 1).Should(BeNil())
		})
		By(fmt.Sprintf("Checking PlacementDecision deletion %s", StrategyName), func() {
			Eventually(func() error {
				_, err := clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).Get(context.TODO(), StrategyName, metav1.GetOptions{})
				if err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					return nil
				}
				return fmt.Errorf("PlacementDecision %s still exist", StrategyName)

			}, 30, 1).Should(BeNil())
		})
	})
})
