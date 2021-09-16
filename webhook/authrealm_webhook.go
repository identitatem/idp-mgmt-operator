// Copyright Red Hat

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type AuthRealmAdmissionHook struct {
	Client      dynamic.ResourceInterface
	lock        sync.RWMutex
	initialized bool
}

// ValidatingResource is called by generic-admission-server on startup to register the returned REST resource through which the
// webhook is accessed by the kube apiserver.
func (a *AuthRealmAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.identityconfig.identitatem.io",
			Version:  "v1alpha1",
			Resource: "authrealms",
		},
		"authrealm"
}

// Validate is called by generic-admission-server when the registered REST resource above is called with an admission request.
func (a *AuthRealmAdmissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}

	// only validate the request for authrealm
	if admissionSpec.Resource.Group != "identityconfig.identitatem.io" ||
		admissionSpec.Resource.Resource != "authrealms" {
		status.Allowed = true
		return status
	}

	authrealm := &identitatemv1alpha1.AuthRealm{}

	err := json.Unmarshal(admissionSpec.Object.Raw, authrealm)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	switch admissionSpec.Operation {
	case admissionv1beta1.Create:
		if len(authrealm.Spec.Type) == 0 {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "type is required",
			}
			return status
		}

		// This is the same regex used by kubernetes for ensuring a CR name is valid
		domainRegex, _ := regexp.Compile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`) // DNS-1123 subdomain
		if len(authrealm.Spec.RouteSubDomain) == 0 {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "routeSubDomain is required",
			}
			return status

		} else if !domainRegex.MatchString(authrealm.Spec.RouteSubDomain) {
			status.Allowed = false
			message := fmt.Sprintf("RouteSubDomain \"%s\" is invalid: a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example', regex used for validation is \"%s\"",
				authrealm.Spec.RouteSubDomain,
				domainRegex.String())

			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: message,
			}
			return status

		}
	case admissionv1beta1.Update:
		oldauthrealm := &identitatemv1alpha1.AuthRealm{}
		err := json.Unmarshal(admissionSpec.OldObject.Raw, authrealm)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}

		if authrealm.Spec.RouteSubDomain != oldauthrealm.Spec.RouteSubDomain {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "RouteSubDomain is immutable and cannot be changed",
			}
			return status
		}

	default:
		status.Allowed = true
		return status
	}

	status.Allowed = true
	return status
}

// Initialize is called by generic-admission-server on startup to setup initialization that webhook needs.
func (a *AuthRealmAdmissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	shallowClientConfigCopy := *kubeClientConfig
	shallowClientConfigCopy.GroupVersion = &schema.GroupVersion{
		Group:   "identityconfig.identitatem.io",
		Version: "v1alpha1",
	}
	shallowClientConfigCopy.APIPath = "/apis"
	dynamicClient, err := dynamic.NewForConfig(&shallowClientConfigCopy)
	if err != nil {
		return err
	}
	a.Client = dynamicClient.Resource(schema.GroupVersionResource{
		Group:   "identityconfig.identitatem.io",
		Version: "v1alpha1",
		// kind is the kind for the resource (e.g. 'Foo' is the kind for a resource 'foo')
		Resource: "authrealms",
	})

	return nil
}
