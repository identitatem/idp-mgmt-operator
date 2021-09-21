// Copyright Red Hat

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	RouteSubDomainErrorMessage string = "RouteSubDomain \"%s\" is invalid: a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example', regex used for validation is \"^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$\""
)

var authrealmSchema = metav1.GroupVersionResource{
	Group:    "admission.identityconfig.identitatem.io",
	Version:  "v1alpha1",
	Resource: "authrealms",
}

func TestAuthRealmValidate(t *testing.T) {
	cases := []struct {
		title            string
		name             string
		namespace        string
		proxytype        identitatemv1alpha1.AuthProxyType
		routeSubDomain   string
		request          *admissionv1beta1.AdmissionRequest
		expectedResponse *admissionv1beta1.AdmissionResponse
	}{
		{
			title:          "validate basic request",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcd",
			request: &admissionv1beta1.AdmissionRequest{
				Resource: metav1.GroupVersionResource{
					Group:    "test.identityconfig.identitatem.io",
					Version:  "v1alpha1",
					Resource: "tests",
				},
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			title:          "validate deleting operation",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcd",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Delete,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			title:          "validate creating AuthRealm Dex",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcdefghijklmnopqrstuvwxyz-0123456789",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: true,
			},
		},

		//TODO - Add when RHSSO is valid
		// {
		// 	title:          "validate creating AuthRealm RHSSO",
		// 	name:           "authrealm-test",
		// 	namespace:      "authrealm-test-ns",
		// 	proxytype:      identitatemv1alpha1.AuthProxyRHSSO
		// 	routeSubDomain: "abcd",
		// 	request: &admissionv1beta1.AdmissionRequest{
		// 		Resource:  authrealmSchema,
		// 		Operation: admissionv1beta1.Create,
		// 	},
		// 	expectedResponse: &admissionv1beta1.AdmissionResponse{
		// 		Allowed: true,
		// 	},
		// },
		{
			title:          "validate minimum RouteSubDomain length",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "a",
			request: &admissionv1beta1.AdmissionRequest{
				Resource: metav1.GroupVersionResource{
					Group:    "test.identityconfig.identitatem.io",
					Version:  "v1alpha1",
					Resource: "tests",
				},
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			title:          "validate maximum RouteSubDomain length",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcdefghijklmnopqrstuvwxyz0123456789-abcdefghijklmnopqrstuvwxyz",
			request: &admissionv1beta1.AdmissionRequest{
				Resource: metav1.GroupVersionResource{
					Group:    "test.identityconfig.identitatem.io",
					Version:  "v1alpha1",
					Resource: "tests",
				},
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: true,
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain is too long",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcdefghijklmnopqrstuvwxyz0123456789-abcdefghijklmnopqrstuvwxyz0",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain blank",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: "routeSubDomain is required",
				},
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain has capital letters",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "ABCD",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: fmt.Sprintf(RouteSubDomainErrorMessage, "ABCD"),
				},
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain has spaces",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "a b c d",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: fmt.Sprintf(RouteSubDomainErrorMessage, "a b c d"),
				},
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain has dash at front",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "-abcd",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: fmt.Sprintf(RouteSubDomainErrorMessage, "-abcd"),
				},
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain has dash at end",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcd-",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: fmt.Sprintf(RouteSubDomainErrorMessage, "abcd-"),
				},
			},
		},
		{
			title:          "invalidate creating AuthRealm routeSubDomain has invalid special characters",
			name:           "authrealm-test",
			namespace:      "authrealm-test-ns",
			proxytype:      identitatemv1alpha1.AuthProxyDex,
			routeSubDomain: "abcd!@",
			request: &admissionv1beta1.AdmissionRequest{
				Resource:  authrealmSchema,
				Operation: admissionv1beta1.Create,
			},
			expectedResponse: &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
					Message: fmt.Sprintf(RouteSubDomainErrorMessage, "abcd!@"),
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			authrealm, _ := NewAuthRealm(c.name, c.namespace, c.proxytype, c.routeSubDomain)
			c.request.Object.Raw, _ = json.Marshal(authrealm)
			admissionHook := &AuthRealmAdmissionHook{}
			actualResponse := admissionHook.Validate(c.request)
			if actualResponse != nil && c.expectedResponse != nil && actualResponse.Result != nil && c.expectedResponse.Result != nil && !reflect.DeepEqual(actualResponse, c.expectedResponse) {
				t.Errorf("expected %#v but got: %#v", c.expectedResponse.Result, actualResponse.Result)
			}
		})
	}
}

func NewUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func NewAuthRealm(name string, namespace string, proxytype identitatemv1alpha1.AuthProxyType, routeSubDomain string) (*identitatemv1alpha1.AuthRealm, string) {
	authrealm := &identitatemv1alpha1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: identitatemv1alpha1.AuthRealmSpec{
			Type:           proxytype,
			RouteSubDomain: routeSubDomain,
		},
	}

	return authrealm, authrealm.Name
}
