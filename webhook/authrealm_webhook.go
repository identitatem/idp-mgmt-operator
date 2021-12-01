// Copyright Red Hat

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	identitatemv1alpha1 "github.com/identitatem/idp-client-api/api/identitatem/v1alpha1"
	"github.com/identitatem/idp-mgmt-operator/pkg/helpers"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	GROUP_SUFFIX = "identityconfig.identitatem.io"
)

type AuthRealmAdmissionHook struct {
	Client      dynamic.ResourceInterface
	KubeClient  kubernetes.Interface
	lock        sync.RWMutex
	initialized bool
}

// ValidatingResource is called by generic-admission-server on startup to register the returned REST resource through which the
// webhook is accessed by the kube apiserver.
func (a *AuthRealmAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission." + GROUP_SUFFIX,
			Version:  "v1alpha1",
			Resource: "authrealms",
		},
		"authrealm"
}

// Validate is called by generic-admission-server when the registered REST resource above is called with an admission request.
func (a *AuthRealmAdmissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}
	klog.V(4).Infof("AuthRealm Validate %q operation for object %q, group: %s, resource: %s", admissionSpec.Operation, admissionSpec.Object, admissionSpec.Resource.Group, admissionSpec.Resource.Resource)

	// only validate the request for authrealm
	if !strings.HasSuffix(admissionSpec.Resource.Group, GROUP_SUFFIX) ||
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

	klog.V(4).Infof("Validate webhook for AuthRealm name: %s, type: %s, routeSubDomain: %s", authrealm.Name, authrealm.Spec.Type, authrealm.Spec.RouteSubDomain)
	switch admissionSpec.Operation {
	case admissionv1beta1.Create:
		klog.V(4).Info("Validate AuthRealm create")

		if len(authrealm.Spec.Type) == 0 {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "type is required",
			}
			return status
		}

		for _, idp := range authrealm.Spec.IdentityProviders {
			if idp.Type == openshiftconfigv1.IdentityProviderTypeGitHub {
				if len(idp.GitHub.Teams) > 0 {
					for _, team := range idp.GitHub.Teams {
						if len(strings.Split(team, "/")) != 2 {
							status.Allowed = false
							status.Result = &metav1.Status{
								Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
								Message: "team should be in formatt <org>/<team>",
							}
							return status
						}
					}
				}
			}
		}

		// This is the same regex used by kubernetes for ensuring a CR name is valid
		domainRegex, _ := regexp.Compile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`) // DNS-1123 subdomain
		switch {
		case len(authrealm.Spec.RouteSubDomain) == 0:
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "routeSubDomain is required",
			}
			return status
		case len(authrealm.Spec.RouteSubDomain) > 54:
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: "routeSubDomain is too long (max 54 characters)",
			}
			return status
		case !domainRegex.MatchString(authrealm.Spec.RouteSubDomain):
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
		// Now check to see if namespace dex server is going to go is already in use
		// In case we are running unit tests where this is not possible
		_, err = a.KubeClient.CoreV1().Namespaces().Get(context.TODO(), helpers.DexServerNamespace(authrealm), metav1.GetOptions{})
		switch {
		case err == nil:
			status.Allowed = false
			message := fmt.Sprintf("RouteSubDomain \"%s\" already in use",
				authrealm.Spec.RouteSubDomain)

			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: message,
			}
			return status
		case !errors.IsNotFound(err):
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: err.Error(),
			}
			return status
		}
	case admissionv1beta1.Update:
		klog.V(4).Info("Validate AuthRealm update")

		oldauthrealm := &identitatemv1alpha1.AuthRealm{}
		err := json.Unmarshal(admissionSpec.OldObject.Raw, oldauthrealm)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}

		klog.V(4).Info("Compare RouteSubDomain", " old value:", oldauthrealm.Spec.RouteSubDomain, " new value:", authrealm.Spec.RouteSubDomain)

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

	klog.V(0).Infof("Initialize admission webhook for AuthRealm")

	a.initialized = true

	shallowClientConfigCopy := *kubeClientConfig
	shallowClientConfigCopy.GroupVersion = &schema.GroupVersion{
		Group:   GROUP_SUFFIX,
		Version: "v1alpha1",
	}
	shallowClientConfigCopy.APIPath = "/apis"
	kubeClient, err := kubernetes.NewForConfig(&shallowClientConfigCopy)
	if err != nil {
		return err
	}

	a.KubeClient = kubeClient
	dynamicClient, err := dynamic.NewForConfig(&shallowClientConfigCopy)
	if err != nil {
		return err
	}
	a.Client = dynamicClient.Resource(schema.GroupVersionResource{
		Group:   GROUP_SUFFIX,
		Version: "v1alpha1",
		// kind is the kind for the resource (e.g. 'Foo' is the kind for a resource 'foo')
		Resource: "authrealms",
	})

	return nil
}
