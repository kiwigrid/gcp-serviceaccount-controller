package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GcpNamespaceRestrictionSpec defines the desired state of GcpNamespaceRestriction
type GcpNamespaceRestrictionSpec struct {
	Namespace      string                      `json:"namespace"`
	Regex          bool                        `json:"regex"`
	GcpRestriction []GcpRestrictionRoleBinding `json:"restrictions,omitempty"`
}

// GcpRestrictionRoleBinding defines a restriction
// all string files can be regex
type GcpRestrictionRoleBinding struct {
	Resource string   `json:"resource"`
	Roles    []string `json:"roles"`
}

// GcpNamespaceRestrictionStatus defines the observed state of GcpNamespaceRestriction
type GcpNamespaceRestrictionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// GcpNamespaceRestriction is the Schema for the gcpnamespacerestrictions API
// +k8s:openapi-gen=true
type GcpNamespaceRestriction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GcpNamespaceRestrictionSpec   `json:"spec,omitempty"`
	Status GcpNamespaceRestrictionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// GcpNamespaceRestrictionList contains a list of GcpNamespaceRestriction
type GcpNamespaceRestrictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GcpNamespaceRestriction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GcpNamespaceRestriction{}, &GcpNamespaceRestrictionList{})
}
