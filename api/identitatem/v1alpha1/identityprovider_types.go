// Copyright Contributors to the Open Cluster Management project

package v1alpha1

import (
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IdentityProviderSpec defines the desired state of IdentityProvider
type IdentityProviderSpec struct {
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

// IdentityProviderStatus defines the observed state of IdentityProvider
type IdentityProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make generate-clients" to regenerate code after modifying this file
	// Conditions contains the different condition statuses for this AuthRealm.
	// +optional
	Conditions []metav1.Condition `json:"conditions"`
}

const (
	IdentityProviderSucceed string = "succeed"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IdentityProvider is the Schema for the strategies API
type IdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IdentityProviderSpec   `json:"spec,omitempty"`
	Status IdentityProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IdentityProviderList contains a list of IdentityProvider
type IdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdentityProvider `json:"items"`
}
