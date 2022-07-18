package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatewayHostRouteSpec struct {
	// Fqdn of the GatewayHost
	Fqdn string `json:"fqdn"`
	// Route is the route for service
	Route Route `json:"route"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayHostRoute can be used to provide Route Specification for a service
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Namespaced,path=gatewayhostroutes,shortName=gwhostroute;gwhostroutes,singular=gatewayhostroute
type GatewayHostRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec GatewayHostRouteSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayHostRouteList is a list of GatewayHostRoutes
type GatewayHostRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GatewayHostRoute `json:"items"`
}
