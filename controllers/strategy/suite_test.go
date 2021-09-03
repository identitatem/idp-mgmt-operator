// Copyright Red Hat

package strategy

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	identitatemdexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	clientsetcluster "open-cluster-management.io/api/client/cluster/clientset/versioned"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusteradmasset "open-cluster-management.io/clusteradm/pkg/helpers/asset"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var clientSetMgmt *idpclientset.Clientset
var clientSetStrategy *idpclientset.Clientset
var clientSetCluster *clientsetcluster.Clientset

var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Strategy Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	readerIDP := idpconfig.GetScenarioResourcesReader()
	strategyCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_strategies.yaml")
	Expect(err).Should(BeNil())

	authrealmCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
	Expect(err).Should(BeNil())

	readerDex := dexoperatorconfig.GetScenarioResourcesReader()

	dexServerCRD, err := getCRD(readerDex, "crd/bases/auth.identitatem.io_dexservers.yaml")
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDs: []client.Object{
			strategyCRD,
			authrealmCRD,
			dexServerCRD,
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

	err = identitatemdexv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())

	err = clusterv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = openshiftconfigv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	clientSetMgmt, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetMgmt).ToNot(BeNil())

	clientSetStrategy, err = idpclientset.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetStrategy).ToNot(BeNil())

	clientSetCluster, err = clientsetcluster.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSetCluster).ToNot(BeNil())

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
				APIServerURL: "http://127.0.0.1:6443",
			},
		}
		err := k8sClient.Create(context.TODO(), infraConfig)
		Expect(err).NotTo(HaveOccurred())
	})
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Process Strategy backplane: ", func() {
	AuthRealmName := "my-authrealm"
	AuthRealmNameSpace := "my-authrealmns"
	CertificatesSecretRef := "my-certs"
	StrategyName := AuthRealmName + "-backplane"
	PlacementStrategyName := StrategyName
	PlacementName := AuthRealmName

	It("process a Strategy backplane CR", func() {
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
		var authRealm *identitatemv1alpha1.AuthRealm
		By("creating a AuthRealm CR", func() {
			var err error
			authRealm = &identitatemv1alpha1.AuthRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AuthRealmName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.AuthRealmSpec{
					Type: identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							Name:          "my-idp",
							MappingMethod: openshiftconfigv1.MappingMethodClaim,
							IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
								Type: openshiftconfigv1.IdentityProviderTypeGitHub,
								GitHub: &openshiftconfigv1.GitHubIdentityProvider{
									ClientID: "me",
								},
							},
						},
					},
					PlacementRef: corev1.LocalObjectReference{
						Name: placement.Name,
					},
				},
			}
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("creating a Strategy CR", func() {
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

			_, err := clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Create(context.TODO(), strategy, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By("Calling reconcile", func() {
			r := StrategyReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}

			req := ctrl.Request{}
			req.Name = StrategyName
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})
		var strategy *identitatemv1alpha1.Strategy
		By("Checking strategy", func() {
			var err error
			strategy, err = clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Get(context.TODO(), StrategyName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(strategy.Spec.PlacementRef.Name).Should(Equal(PlacementStrategyName))
		})
		By("Checking placement strategy", func() {
			_, err := clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
				Get(context.TODO(), PlacementStrategyName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(len(placement.Spec.Predicates)).Should(Equal(1))
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
