package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenericRouteFilterConfig struct {
	Config string `json:"config,omitempty"`
}

// RouteFilterSpec defines the spec of the CRD
type RouteFilterSpec struct {
	Name              string                   `json:"name,omitempty"`
	Type              string                   `json:"type,omitempty"`
	RouteFilterConfig GenericRouteFilterConfig `json:"routeFilterConfig,omitempty"`
	// Service that the filter may need to communicate with to provide the filter functionality
	// Eg: jwt server that hosts external JWKS
	Service Service `json:"services,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RouteFilter is an Ingress CRD specificiation
type RouteFilter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   RouteFilterSpec `json:"spec"`
	Status `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RouteFilterList is a list of RouteFilter
type RouteFilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []RouteFilter `json:"items"`
}
