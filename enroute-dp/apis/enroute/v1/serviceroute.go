package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceRouteSpec struct {
	// Fqdn of the GatewayHost
	Fqdn string `json:"fqdn"`
	// Route is the route for service
	Route Route `json:"route"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceRoute can be used to provide Route Specification for a service
// +k8s:openapi-gen=true
// +kubebuilder:resource:scope=Namespaced,path=serviceroutes,shortName=svcroute;svcroutes,singular=serviceroute
type ServiceRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ServiceRouteSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceRouteList is a list of ServiceRoutes
type ServiceRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ServiceRoute `json:"items"`
}
