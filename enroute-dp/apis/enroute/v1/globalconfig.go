package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GlobalConfigSpec defines the spec of the CRD
type GlobalConfigSpec struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Config string `json:"config,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalConfig is an Ingress CRD specificiation
type GlobalConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   GlobalConfigSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalConfigList is a list of GlobalConfig
type GlobalConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GlobalConfig `json:"items"`
}
