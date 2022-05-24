// Copyright Red Hat

package manifestwork

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	clientsetwork "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var clientSetMgmt *idpclientset.Clientset
var clientSetWork *clientsetwork.Clientset
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	// fetch the current config
	suiteConfig, reporterConfig := GinkgoConfiguration()
	// adjust it
	suiteConfig.SkipStrings = []string{"NEVER-RUN"}
	reporterConfig.FullTrace = true
	RunSpecs(t,
		"ClusterOAuth Controller Suite",
		reporterConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	readerIDP := idpconfig.GetScenarioResourcesReader()
	clusterOAuthCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_clusteroauths.yaml")
	Expect(err).Should(BeNil())
	authRealmCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDs: []*apiextensionsv1.CustomResourceDefinition{
			clusterOAuthCRD,
			authRealmCRD,
		},
		CRDDirectoryPaths: []string{
			//DV added this line and copyed the authrealms CRD
			filepath.Join("..", "..", "test", "config", "crd", "external"),
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
		ControlPlaneStartTimeout: 1 * time.Minute,
		ControlPlaneStopTimeout:  1 * time.Minute,
	}
	err = identitatemv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme.Scheme)
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

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	clientSetMgmt, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	clientSetWork, err = clientsetwork.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetWork).ToNot(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("Creating infra", func() {
		infraConfig := &openshiftconfigv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.InfrastructureSpec{},
			Status: openshiftconfigv1.InfrastructureStatus{
				APIServerURL: "http://api.my.company.com:6443",
			},
		}
		err := k8sClient.Create(context.TODO(), infraConfig)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Process manifestwork: ", func() {
	AuthRealmName := "my-authrealm"
	AuthRealmNamespace := "my-authrealm-ns"
	StrategyName := AuthRealmName + "-backplane"
	ClusterName := "my-cluster"

	It("process a ClusterOAuth CR", func() {
		By(fmt.Sprintf("creation of authrealm namespace %s", ClusterName), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNamespace,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})

		By(fmt.Sprintf("creation of cluster namespace %s", ClusterName), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})

		var secret1 *corev1.Secret
		By(fmt.Sprintf("creation of IDP secret 1 in cluster namespace %s", ClusterName), func() {
			secret1 = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), secret1)
			Expect(err).To(BeNil())

		})

		var authRealm *identitatemv1alpha1.AuthRealm
		By(fmt.Sprintf("creation of AuthRealm in cluster namespace %s", ClusterName), func() {
			authRealm = &identitatemv1alpha1.AuthRealm{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "AuthRealm",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNamespace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					RouteSubDomain: "myroute",
				},
			}
			err := k8sClient.Create(context.TODO(), authRealm)
			Expect(err).To(BeNil())
			patch := client.MergeFrom(authRealm.DeepCopy())
			authRealm.Status.Strategies = []identitatemv1alpha1.AuthRealmStrategyStatus{
				{
					Name: StrategyName,
					Clusters: []identitatemv1alpha1.AuthRealmClusterStatus{
						{
							Name: ClusterName,
						},
					},
				},
			}
			err = k8sClient.Status().Patch(context.TODO(), authRealm, patch)
			Expect(err).To(BeNil())
		})

		By(fmt.Sprintf("creation of ClusterOAuth for managed cluster %s", ClusterName), func() {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{
				TypeMeta: metav1.TypeMeta{
					APIVersion: identitatemv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ClusterOAuth",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: ClusterName,
				},
				Spec: identitatemv1alpha1.ClusterOAuthSpec{
					AuthRealmReference: identitatemv1alpha1.RelatedObjectReference{
						Kind:      "AuthRealm",
						Name:      AuthRealmName,
						Namespace: AuthRealmNamespace,
					},
					StrategyReference: identitatemv1alpha1.RelatedObjectReference{
						Kind:      "Strategy",
						Name:      StrategyName,
						Namespace: AuthRealmNamespace,
					},
					OAuth: &openshiftconfigv1.OAuth{
						TypeMeta: metav1.TypeMeta{
							APIVersion: openshiftconfigv1.SchemeGroupVersion.String(),
							Kind:       "OAuth",
						},

						Spec: openshiftconfigv1.OAuthSpec{
							IdentityProviders: []openshiftconfigv1.IdentityProvider{
								{
									Name:          secret1.Name,
									MappingMethod: openshiftconfigv1.MappingMethodClaim,
									IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
										Type: openshiftconfigv1.IdentityProviderTypeGitHub,
										OpenID: &openshiftconfigv1.OpenIDIdentityProvider{
											ClientID: "me",
											ClientSecret: openshiftconfigv1.SecretNameReference{
												Name: secret1.Name,
											},
										},
									},
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(context.TODO(), clusterOAuth)
			Expect(err).To(BeNil())
		})

		By("Creating managedCluster", func() {
			mc := &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), mc)
			Expect(err).To(BeNil())
		})
		By("Creating a manifestwork idp-oauth", func() {
			mw := &workv1.ManifestWork{
				TypeMeta: metav1.TypeMeta{
					APIVersion: workv1.SchemeGroupVersion.String(),
					Kind:       "ManifestWork",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      helpers.ManifestWorkOAuthName(),
					Namespace: ClusterName,
				},
				Spec: workv1.ManifestWorkSpec{},
			}
			_, err := clientSetWork.WorkV1().ManifestWorks(ClusterName).Create(context.TODO(), mw, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			patch := client.MergeFrom(mw.DeepCopy())
			mw.Status.Conditions = []metav1.Condition{
				{
					LastTransitionTime: metav1.Now(),
					Type:               "test-type-manifestwork",
					Status:             metav1.ConditionTrue,
					Reason:             "TestReasonManifestwork",
					Message:            "test-message-manifestwork",
				},
			}
			mw.Status.ResourceStatus.Manifests = []workv1.ManifestCondition{
				{
					ResourceMeta: workv1.ManifestResourceMeta{
						Group:     "",
						Kind:      "Secret",
						Name:      "authrealm-github-ldap-authrealm-github-ldap-ns",
						Namespace: "openshift-config",
						Ordinal:   1,
						Resource:  "secrets",
						Version:   "v1",
					},
					Conditions: []metav1.Condition{
						{
							LastTransitionTime: metav1.Now(),
							Type:               "test-type-manifest",
							Status:             metav1.ConditionTrue,
							Reason:             "TestReasonManifest",
							Message:            "test-message-manifest",
						},
					},
				},
			}
			err = k8sClient.Status().Patch(context.TODO(), mw, patch)
			Expect(err).To(BeNil())

		})
		By("Calling reconcile", func() {
			r := &ManifestWorkReconciler{
				Client:        k8sClient,
				KubeClient:    kubernetes.NewForConfigOrDie(cfg),
				DynamicClient: dynamic.NewForConfigOrDie(cfg),
				Log:           logf.Log,
				Scheme:        scheme.Scheme,
			}
			req := ctrl.Request{}
			req.Name = helpers.ManifestWorkOAuthName()
			req.Namespace = ClusterName

			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})

		By("Checking authreRealm", func() {
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: AuthRealmName, Namespace: AuthRealmNamespace}, authRealm)
			Expect(err).To(BeNil())
			Expect(len(authRealm.Status.Strategies)).To(Equal(1))
			strategyStatus := authRealm.Status.Strategies[0]
			Expect(len(strategyStatus.Clusters)).To(Equal(1))
			clusterStatus := strategyStatus.Clusters[0]
			Expect(len(clusterStatus.ManifestWork.Conditions)).To(Equal(1))
			manifestWorkCondition := clusterStatus.ManifestWork.Conditions[0]
			Expect(manifestWorkCondition.Type).To(Equal("test-type-manifestwork"))
			resourceStatus := clusterStatus.ManifestWork.ResourceStatus
			Expect(len(resourceStatus.Manifests)).To(Equal(1))
			manifestStatus := resourceStatus.Manifests[0]
			Expect(manifestStatus.ResourceMeta.Kind).To(Equal("Secret"))
			Expect(len(manifestStatus.Conditions)).To(Equal(1))
			manifestCondition := manifestStatus.Conditions[0]
			Expect(manifestCondition.Type).To(Equal("test-type-manifest"))
		})

	})
})

func getCRD(reader *clusteradmasset.ScenarioResourcesReader, file string) (*apiextensionsv1.CustomResourceDefinition, error) {
	b, err := reader.Asset(file)
	if err != nil {
		return nil, err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(b, crd); err != nil {
		return nil, err
	}
	return crd, nil
}
