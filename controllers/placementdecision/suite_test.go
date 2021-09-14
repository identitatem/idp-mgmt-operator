// Copyright Red Hat

package placementdecision

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

	dexv1alpha1 "github.com/identitatem/dex-operator/api/v1alpha1"
	dexoperatorconfig "github.com/identitatem/dex-operator/config"
	idpclientset "github.com/identitatem/idp-client-api/api/client/clientset/versioned"
	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	idpconfig "github.com/identitatem/idp-client-api/config"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	clientsetcluster "open-cluster-management.io/api/client/cluster/clientset/versioned"
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
var clientSetStrategy *idpclientset.Clientset
var clientSetCluster *clientsetcluster.Clientset
var clientSetWork *clientsetwork.Clientset
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	readerIDP := idpconfig.GetScenarioResourcesReader()
	strategyCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_strategies.yaml")
	Expect(err).Should(BeNil())

	authRealmCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_authrealms.yaml")
	Expect(err).Should(BeNil())

	clusterOAuthCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_clusteroauths.yaml")
	Expect(err).Should(BeNil())

	readerDex := dexoperatorconfig.GetScenarioResourcesReader()
	dexClientCRD, err := getCRD(readerDex, "crd/bases/auth.identitatem.io_dexclients.yaml")
	Expect(err).Should(BeNil())

	dexServerCRD, err := getCRD(readerDex, "crd/bases/auth.identitatem.io_dexservers.yaml")
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDs: []client.Object{
			strategyCRD,
			authRealmCRD,
			clusterOAuthCRD,
			dexClientCRD,
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

	err = dexv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).Should(BeNil())

	err = clusterv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = workv1.AddToScheme(scheme.Scheme)
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
	ClusterName := "my-cluster"
	MyIDPName := "my-idp"
	RouteSubDomain := "myroute"
	It("Process the creation of a placementDecision", func() {
		By(fmt.Sprintf("creation of User namespace %s", AuthRealmNameSpace), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: AuthRealmNameSpace,
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		By(fmt.Sprintf("creation of Dex Operator namespace %s", AuthRealmName), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: helpers.DexOperatorNamespace(),
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		var placement *clusterv1alpha1.Placement
		By("Creating the placement", func() {
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
					RouteSubDomain: RouteSubDomain,
					Type:           identitatemv1alpha1.AuthProxyDex,
					CertificatesSecretRef: corev1.LocalObjectReference{
						Name: CertificatesSecretRef,
					},
					IdentityProviders: []openshiftconfigv1.IdentityProvider{
						{
							Name:          MyIDPName,
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
			//DV reassign  to authRealm to get the extra info that kube set (ie:uuid as needed to set ownerref)
			authRealm, err = clientSetMgmt.IdentityconfigV1alpha1().AuthRealms(AuthRealmNameSpace).Create(context.TODO(), authRealm, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		By(fmt.Sprintf("creation of Dex server namespace %s", AuthRealmName), func() {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: helpers.DexServerNamespace(authRealm),
				},
			}
			err := k8sClient.Create(context.TODO(), ns)
			Expect(err).To(BeNil())
		})
		var strategy *identitatemv1alpha1.Strategy
		By("creating a Strategy CR", func() {
			strategy = &identitatemv1alpha1.Strategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      StrategyName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: identitatemv1alpha1.StrategySpec{
					Type: identitatemv1alpha1.BackplaneStrategyType,
					PlacementRef: corev1.LocalObjectReference{
						Name: PlacementStrategyName,
					},
				},
			}
			controllerutil.SetOwnerReference(authRealm, strategy, scheme.Scheme)
			var err error
			strategy, err = clientSetStrategy.IdentityconfigV1alpha1().Strategies(AuthRealmNameSpace).Create(context.TODO(), strategy, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})
		var placementStrategy *clusterv1alpha1.Placement
		By("Creating the placement strategy", func() {
			placementStrategy = &clusterv1alpha1.Placement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlacementStrategyName,
					Namespace: AuthRealmNameSpace,
				},
				Spec: placement.Spec,
			}
			controllerutil.SetOwnerReference(strategy, placementStrategy, scheme.Scheme)
			var err error
			placementStrategy, err = clientSetCluster.ClusterV1alpha1().Placements(AuthRealmNameSpace).
				Create(context.TODO(), placementStrategy, metav1.CreateOptions{})
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
		By("Create Placement Decision CR", func() {
			placementDecision := &clusterv1alpha1.PlacementDecision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlacementStrategyName,
					Namespace: AuthRealmNameSpace,
					Labels: map[string]string{
						clusterv1alpha1.PlacementLabel: placementStrategy.Name,
					},
				},
			}
			controllerutil.SetOwnerReference(placementStrategy, placementDecision, scheme.Scheme)
			placementDecision, err := clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).
				Create(context.TODO(), placementDecision, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			placementDecision.Status.Decisions = []clusterv1alpha1.ClusterDecision{
				{
					ClusterName: ClusterName,
				},
			}
			_, err = clientSetCluster.ClusterV1alpha1().PlacementDecisions(AuthRealmNameSpace).
				UpdateStatus(context.TODO(), placementDecision, metav1.UpdateOptions{})
			Expect(err).To(BeNil())
		})
		By("Calling reconcile", func() {
			r := &PlacementDecisionReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}
			req := ctrl.Request{}
			req.Name = strategy.Spec.PlacementRef.Name
			req.Namespace = AuthRealmNameSpace
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})
		dexClientName := ClusterName
		clientSecret := &corev1.Secret{}
		By(fmt.Sprintf("Checking client secret %s", MyIDPName), func() {
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clientSecret)
			Expect(err).To(BeNil())
		})
		By(fmt.Sprintf("Checking DexClient %s", dexClientName), func() {
			dexClient := &dexv1alpha1.DexClient{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: dexClientName, Namespace: helpers.DexServerNamespace(authRealm)}, dexClient)
			Expect(err).To(BeNil())
			Expect(dexClient.Spec.ClientID).To(Equal(ClusterName))
			Expect(dexClient.Spec.ClientSecret).To(Equal(string(clientSecret.Data["clientSecret"])))
			Expect(len(dexClient.Status.RelatedObjects)).To(Equal(1))
			Expect(dexClient.Status.RelatedObjects[0].Kind).To(Equal("PlacementDecision"))
		})
		By(fmt.Sprintf("Checking ClusterOAuth %s", dexClientName), func() {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{}
			err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: MyIDPName, Namespace: ClusterName}, clusterOAuth)
			Expect(err).To(BeNil())
			Expect(clusterOAuth.Spec.OAuth.Spec.IdentityProviders[0].OpenID.ClientID).To(Equal(ClusterName))
			Expect(clusterOAuth.Spec.OAuth.Spec.IdentityProviders[0].OpenID.ClientSecret.Name).To(Equal(clientSecret.Name))
		})
	})
	It("Process the deletion of a placementDecision", func() {

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
