// Copyright Contributors to the Open Cluster Management project

package v1alpha1

import (
	policyv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	placementv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthRealmSpec defines the desired state of AuthRealm
type AuthRealmSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Placement defines a rule to select a set of ManagedClusters from the ManagedClusterSets bound
	// to the placement namespace.
	Placement *Placement `json:"placement,omitempty"`
	// mappingMethod determines how identities from this provider are mapped to users
	// Defaults to "claim"
	// +optional
	MappingMethod openshiftconfigv1.MappingMethodType `json:"mappingMethod,omitempty"`

	//RemediateAction defines the remediation action to apply to the idp policy
	// +kubebuilder:validation:Enum=Enforce;inform
	// +required
	RemediateAction policyv1.RemediationAction `json:"remediateAction,omitempty"`

	// +kubebuilder:validation:Enum=dex;rhsso
	// +required
	Type AuthProxyType `json:"type,omitempty"`
	//Host defines the url of the proxy
	// +required
	Host string `json:"host,omitempty"`
	//Certificates references a secret containing `ca.crt`, `tls.crt`, and `tls.key`
	CertificatesSecretRef corev1.LocalObjectReference `json:"certificatesSecretRef,omitempty"`
	// IdentityProviders reference an identity provider
	// +required
	IdentityProviders []IdentityProvider `json:"identityProviders,omitempty"`
}

//Placement defines the placement.
type Placement struct {
	Name string                          `json:"name,omitempty"`
	Spec placementv1alpha1.PlacementSpec `json:"spec,omitempty"`
}

type AuthProxyType string

const (
	AuthProxyDex   AuthProxyType = "dex"
	AuthProxyRHSSO AuthProxyType = "rhsso"
)

const (
	LocalHost string = "local"
)

type IdentityProvider struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make generate-clients" to regenerate code after modifying this file
	// github enables user authentication using GitHub credentials
	// +optional
	GitHub *openshiftconfigv1.GitHubIdentityProvider `json:"github,omitempty"`

	// google enables user authentication using Google credentials
	// +optional
	Google *openshiftconfigv1.GoogleIdentityProvider `json:"google,omitempty"`

	// htpasswd enables user authentication using an HTPasswd file to validate credentials
	// +optional
	HTPasswd *openshiftconfigv1.HTPasswdIdentityProvider `json:"htpasswd,omitempty"`

	// ldap enables user authentication using LDAP credentials
	// +optional
	LDAP *openshiftconfigv1.LDAPIdentityProvider `json:"ldap,omitempty"`

	// openID enables user authentication using OpenID credentials
	// +optional
	OpenID *openshiftconfigv1.OpenIDIdentityProvider `json:"openID,omitempty"`
}

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
