// Copyright Contributors to the Open Cluster Management project

package api

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	identitatemv1alpha1 "github.com/identitatem/idp-mgmt-operator/api/identitatem/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("Process IdentityProvider: ", func() {
	var sampleIdentityProvider identitatemv1alpha1.IdentityProvider
	BeforeEach(func() {
		b, err := ioutil.ReadFile(filepath.Join("..", "config", "samples", "identitatem.io_v1alpha1_identityprovider.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(b).ShouldNot(BeNil())
		sampleIdentityProvider = identitatemv1alpha1.IdentityProvider{}
		err = yaml.Unmarshal(b, &sampleIdentityProvider)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		cr := sampleIdentityProvider.DeepCopy()
		dynamicClient.Resource(identitatemv1alpha1.SchemeGroupVersion.WithResource("identityproviders")).
			Namespace(cr.Namespace).Delete(context.TODO(), cr.Name, metav1.DeleteOptions{})
	})
	It("create a IdentityProvider CR", func() {
		cr := sampleIdentityProvider.DeepCopy()
		createdCR, err := clientSet.IdentitatemV1alpha1().IdentityProviders(cr.Namespace).Create(context.TODO(), cr, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		cu, err := dynamicClient.Resource(identitatemv1alpha1.SchemeGroupVersion.WithResource("identityproviders")).
			Namespace(cr.Namespace).
			Get(context.TODO(), cr.Name, metav1.GetOptions{})
		Expect(err).To(BeNil())
		c := &identitatemv1alpha1.IdentityProvider{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(cu.UnstructuredContent(), c)
		Expect(err).To(BeNil())
		Expect(reflect.DeepEqual(createdCR.Spec, c.Spec)).To(BeTrue())
		Expect(reflect.DeepEqual(createdCR.ObjectMeta, c.ObjectMeta)).To(BeTrue())
	})
	It("read a IdentityProvider CR", func() {
		cr := sampleIdentityProvider.DeepCopy()
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
		cu := &unstructured.Unstructured{
			Object: content,
		}
		Expect(err).To(BeNil())
		_, err = dynamicClient.Resource(identitatemv1alpha1.SchemeGroupVersion.WithResource("identityproviders")).
			Namespace(cr.Namespace).Create(context.TODO(), cu, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		c, err := clientSet.IdentitatemV1alpha1().IdentityProviders(cr.Namespace).Get(context.TODO(), sampleIdentityProvider.Name, metav1.GetOptions{})
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(cr.Spec, c.Spec)).To(BeTrue())
		Expect(reflect.DeepEqual(cr.ObjectMeta.Name, c.ObjectMeta.Name)).To(BeTrue())
		Expect(reflect.DeepEqual(cr.ObjectMeta.Namespace, c.ObjectMeta.Namespace)).To(BeTrue())
	})
	It("delete a IdentityProvider CR", func() {
		cr := sampleIdentityProvider.DeepCopy()
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
		cu := &unstructured.Unstructured{
			Object: content,
		}
		Expect(err).To(BeNil())
		_, err = dynamicClient.Resource(identitatemv1alpha1.SchemeGroupVersion.WithResource("identityproviders")).
			Namespace(cr.Namespace).Create(context.TODO(), cu, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		err = clientSet.IdentitatemV1alpha1().IdentityProviders(cr.Namespace).Delete(context.TODO(), sampleIdentityProvider.Name, metav1.DeleteOptions{})
		Expect(err).Should(BeNil())
		err = dynamicClient.Resource(identitatemv1alpha1.SchemeGroupVersion.WithResource("identityproviders")).
			Namespace(cr.Namespace).Delete(context.TODO(), cr.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(BeNil())
	})
})
