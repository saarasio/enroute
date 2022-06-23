package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenericPolicyOverlayConfig struct {
	Config string `json:"config,omitempty"`
}

// PolicyOverlaySpec defines the spec of the CRD
type PolicyOverlaySpec struct {
	Name                string                     `json:"name,omitempty"`
	Type                string                     `json:"type,omitempty"`
	PolicyOverlayConfig GenericPolicyOverlayConfig `json:"policyOverlayConfig,omitempty"`
	// Service that the filter communicates with to provide the filter functionality
	// Eg: jwt server that hosts external JWKS
	Service Service `json:"services,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyOverlay is an Ingress CRD specificiation
type PolicyOverlay struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PolicyOverlaySpec `json:"spec"`
	Status `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyOverlayList is a list of PolicyOverlay
type PolicyOverlayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []PolicyOverlay `json:"items"`
}
