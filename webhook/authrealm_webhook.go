package webhook

import (
	"encoding/json"
	"net/http"
	"sync"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
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

func (a *AuthRealmAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "identityconfig.identitatem.io",
			Version:  "v1alpha1",
			Resource: "authrealms",
		},
		"authrealm"
}

func (a *AuthRealmAdmissionHook) Validate(admissionSpec *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	status := &admissionv1.AdmissionResponse{}

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
	if len(authrealm.Spec.Type) == 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: "type is required",
		}
		return status
	}

	return status
}

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
