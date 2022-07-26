package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the group name for the Contour API
	GroupName = "enroute.saaras.io"
)

var (
	// SchemeBuilder collects the scheme builder functions for the Enroute API
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme applies the SchemeBuilder functions to a specified scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// SchemeGroupVersion is the GroupVersion for the Contour API
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

// Resource gets an Contour GroupResource for a specified resource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&GatewayHost{},
		&GatewayHostList{},
		&ServiceRoute{},
		&ServiceRouteList{},
		&GlobalConfig{},
		&GlobalConfigList{},
		&HttpFilter{},
		&HttpFilterList{},
		&RouteFilter{},
		&RouteFilterList{},
		&TLSCertificateDelegation{},
		&TLSCertificateDelegationList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
