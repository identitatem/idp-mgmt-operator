// Copyright Red Hat

package clusteroauth

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
		"ClusterOAuth Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	readerIDP := idpconfig.GetScenarioResourcesReader()
	clusterOAuthCRD, err := getCRD(readerIDP, "crd/bases/identityconfig.identitatem.io_clusteroauths.yaml")
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDs: []client.Object{
			clusterOAuthCRD,
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

var _ = Describe("Process clusterOAuth for Strategy backplane: ", func() {
	AuthRealmName := "my-authrealm"
	StrategyName := AuthRealmName + "-backplane"
	ClusterOAuthName1 := StrategyName + "-1"
	ClusterOAuthName2 := StrategyName + "-2"
	ClusterName := "my-cluster"
	MyIDPName1 := "my-idp" + "-1"
	MyIDPName2 := "my-idp" + "-2"
	MyIDPName3 := "my-idp" + "-3"

	It("process a ClusterOAuth CR", func() {
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
					Name:      MyIDPName1,
					Namespace: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), secret1)
			Expect(err).To(BeNil())

		})

		By(fmt.Sprintf("creation of ClusterOAuth for mangaed cluster %s", ClusterName), func() {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{
				TypeMeta: metav1.TypeMeta{
					APIVersion: identitatemv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ClusterOAuth",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterOAuthName1,
					Namespace: ClusterName,
				},
				Spec: identitatemv1alpha1.ClusterOAuthSpec{
					OAuth: &openshiftconfigv1.OAuth{
						TypeMeta: metav1.TypeMeta{
							APIVersion: openshiftconfigv1.SchemeGroupVersion.String(),
							Kind:       "OAuth",
						},

						Spec: openshiftconfigv1.OAuthSpec{
							IdentityProviders: []openshiftconfigv1.IdentityProvider{
								{
									Name:          MyIDPName1,
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

		By("Calling reconcile", func() {
			r := &ClusterOAuthReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}
			req := ctrl.Request{}
			req.Name = ClusterOAuthName1
			req.Namespace = ClusterName
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})

		By("Checking manifestwork", func() {
			mw := &workv1.ManifestWork{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: helpers.ManifestWorkName(), Namespace: ClusterName}, mw)
			//var mw *workv1.ManifestWork
			//mw, err := clientSetWork.WorkV1().ManifestWorks(ClusterName).Get(context.TODO(), "idp-backplane", metav1.GetOptions{})
			Expect(err).To(BeNil())
			// should find manifest for OAuth and manifest for Secret
			Expect(len(mw.Spec.Workload.Manifests)).To(Equal(3))
			//manifest := mw.Spec.Workload.Manifests[0]
		})

		By("Calling reconcile 2nd time", func() {
			r := &ClusterOAuthReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}
			req := ctrl.Request{}
			req.Name = ClusterOAuthName1
			req.Namespace = ClusterName
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})

		var secret2 *corev1.Secret
		By(fmt.Sprintf("creation of IDP secret 2 in cluster namespace %s", ClusterName), func() {
			secret2 = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyIDPName2,
					Namespace: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), secret2)
			Expect(err).To(BeNil())

		})

		var secret3 *corev1.Secret
		By(fmt.Sprintf("creation of IDP secret 3 in cluster namespace %s", ClusterName), func() {
			secret3 = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyIDPName3,
					Namespace: ClusterName,
				},
			}
			err := k8sClient.Create(context.TODO(), secret3)
			Expect(err).To(BeNil())

		})

		By(fmt.Sprintf("creation of ClusterOAuth 2 for mangaed cluster %s", ClusterName), func() {
			clusterOAuth := &identitatemv1alpha1.ClusterOAuth{
				TypeMeta: metav1.TypeMeta{
					APIVersion: identitatemv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ClusterOAuth",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterOAuthName2,
					Namespace: ClusterName,
				},
				Spec: identitatemv1alpha1.ClusterOAuthSpec{
					OAuth: &openshiftconfigv1.OAuth{
						TypeMeta: metav1.TypeMeta{
							APIVersion: openshiftconfigv1.SchemeGroupVersion.String(),
							Kind:       "OAuth",
						},

						Spec: openshiftconfigv1.OAuthSpec{
							IdentityProviders: []openshiftconfigv1.IdentityProvider{
								{
									Name:          MyIDPName2,
									MappingMethod: openshiftconfigv1.MappingMethodClaim,
									IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
										Type: openshiftconfigv1.IdentityProviderTypeGitHub,
										OpenID: &openshiftconfigv1.OpenIDIdentityProvider{
											ClientID: "me2",
											ClientSecret: openshiftconfigv1.SecretNameReference{
												Name: secret2.Name,
											},
										},
									},
								},
								{
									Name:          MyIDPName3,
									MappingMethod: openshiftconfigv1.MappingMethodClaim,
									IdentityProviderConfig: openshiftconfigv1.IdentityProviderConfig{
										Type: openshiftconfigv1.IdentityProviderTypeGitHub,
										OpenID: &openshiftconfigv1.OpenIDIdentityProvider{
											ClientID: "me3",
											ClientSecret: openshiftconfigv1.SecretNameReference{
												Name: secret3.Name,
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

		By("Calling reconcile", func() {
			r := &ClusterOAuthReconciler{
				Client: k8sClient,
				Log:    logf.Log,
				Scheme: scheme.Scheme,
			}
			req := ctrl.Request{}
			req.Name = ClusterOAuthName2
			req.Namespace = ClusterName
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).To(BeNil())
		})

		By("Checking manifestwork", func() {
			mw := &workv1.ManifestWork{}
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: helpers.ManifestWorkName(), Namespace: ClusterName}, mw)
			//var mw *workv1.ManifestWork
			//mw, err := clientSetWork.WorkV1().ManifestWorks(ClusterName).Get(context.TODO(), "idp-backplane", metav1.GetOptions{})
			Expect(err).To(BeNil())

			// should find manifest for OAuth 1 and manifest for Secret 1
			// AND
			// should find manifest for OAuth 2 and manifest for Secret 2 and Secret 3 and an aggregated role
			Expect(len(mw.Spec.Workload.Manifests)).To(Equal(6))
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
