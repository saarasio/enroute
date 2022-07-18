package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenericHttpFilterConfig struct {
	Config string `json:"config,omitempty"`
}

// HttpFilterSpec defines the spec of the CRD
type HttpFilterSpec struct {
	Name             string                  `json:"name,omitempty"`
	Type             string                  `json:"type,omitempty"`
	HttpFilterConfig GenericHttpFilterConfig `json:"httpFilterConfig,omitempty"`
	// Service that the filter communicates with to provide the filter functionality
	// Eg: jwt server that hosts external JWKS
	Service Service `json:"services,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HttpFilter is an Ingress CRD specificiation
type HttpFilter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec HttpFilterSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HttpFilterList is a list of HttpFilter
type HttpFilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []HttpFilter `json:"items"`
}
