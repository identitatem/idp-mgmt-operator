// Copyright Contributors to the Open Cluster Management project

package v1alpha1

import (
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	placementrulev1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthRealmSpec defines the desired state of AuthRealm
type AuthRealmSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Placement defines a rule to select a set of ManagedClusters from the ManagedClusterSets bound
	// to the placement namespace.
	Placement *placementrulev1alpha1.Placement `json:"placement,omitempty"`
	// mappingMethod determines how identities from this provider are mapped to users
	// Defaults to "claim"
	// +optional
	MappingMethod openshiftconfigv1.MappingMethodType `json:"mappingMethod,omitempty"`

	//AuthProxy defines the list of authproxy to setup in each cluster defined by the placement.
	AuthProxy []AuthProxy `json:"authProxy,omitempty"`
}

type AuthProxyType string

const (
	DexAuthProxy   AuthProxyType = "dex"
	RHSSOAuthProxy AuthProxyType = "rhsso"
)

//AuthProxy defines the auth proxy (dex, rhsso...)
type AuthProxy struct {
	//AuthProxyType the proxy type (dex, rhsso...)
	Type AuthProxyType `json:"type,omitempty"`
	//Host defines the url of the proxy
	Host string `json:"host,omitempty"`
	// IdentityProviderRef reference an identity provider
	IdentityProviderRefs []corev1.LocalObjectReference `json:"identityProviderRefs,omitempty"`
}

const (
	LocalHost string = "local"
)

// AuthRealmStatus defines the observed state of AuthRealm
type AuthRealmStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make generate-clients" to regenerate code after modifying this file

	// Conditions contains the different condition statuses for this AuthRealm.
	// +optional
	Conditions []metav1.Condition `json:"conditions"`
}

const (
	AuthRealmSucceed string = "succeed"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AuthRealm is the Schema for the authrealms API
type AuthRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthRealmSpec   `json:"spec,omitempty"`
	Status AuthRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuthRealmList contains a list of AuthRealm
type AuthRealmList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of AuthRealm.
	// +listType=set
	Items []AuthRealm `json:"items"`
}
