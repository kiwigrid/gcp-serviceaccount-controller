package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GcpServiceAccountSpec defines the desired state of GcpServiceAccount
type GcpServiceAccountSpec struct {
	GcpRoleBindings           []GcpRoleBindings `json:"bindings"`
	ServiceAccountIdentifier  string            `json:"serviceAccountIdentifier"`
	ServiceAccountDescription string            `json:"serviceAccountDescription,omitempty"`
	SecretName                string            `json:"secretName"`
	SecretKey                 string            `json:"secretKey,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// GcpRoleBindings defines the desired role bindings of GcpServiceAccount
type GcpRoleBindings struct {
	Resource string   `json:"resource"`
	Roles    []string `json:"roles"`
}

// GcpServiceAccountStatus defines the observed state of GcpServiceAccount
type GcpServiceAccountStatus struct {
	ServiceAccountPath     string            `json:"serviceAccountPath,omitempty"`
	ServiceAccountMail     string            `json:"serviceAccountMail,omitempty"`
	CredentialKey          string            `json:"credentialKey,omitempty"`
	AppliedGcpRoleBindings []GcpRoleBindings `json:"appliedBindings,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// GcpServiceAccount is the Schema for the gcpserviceaccounts API
// +k8s:openapi-gen=true
type GcpServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GcpServiceAccountSpec   `json:"spec,omitempty"`
	Status GcpServiceAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// GcpServiceAccountList contains a list of GcpServiceAccount
type GcpServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GcpServiceAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GcpServiceAccount{}, &GcpServiceAccountList{})
}
