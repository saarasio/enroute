package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayHostSpec defines the spec of the CRD
type GatewayHostSpec struct {
	// Virtualhost appears at most once. If it is present, the object is considered
	// to be a "root".
	VirtualHost *VirtualHost `json:"virtualhost,omitempty"`
	// Routes are the ingress routes. If TCPProxy is present, Routes is ignored.
	Routes []Route `json:"routes"`
	// TCPProxy holds TCP proxy information.
	TCPProxy *TCPProxy `json:"tcpproxy,omitempty"`
}

type RouteAttachedFilter struct {
	// Name of the filter attached to this route
	Name string `json:"name,omitempty"`
	// Type of the filter attached to this route
	Type string `json:"type,omitempty"`
}

type HostAttachedFilter struct {
	// Name of the filter attached to this VirtualHost
	Name string `json:"name,omitempty"`
	// Type of the filter attached to this VirtualHost
	Type string `json:"type,omitempty"`
}

// VirtualHost appears at most once. If it is present, the object is considered
// to be a "root".
type VirtualHost struct {
	// The fully qualified domain name of the root of the ingress tree
	// all leaves of the DAG rooted at this object relate to the fqdn
	Fqdn string `json:"fqdn"`
	// If present describes tls properties. The CNI names that will be matched on
	// are described in fqdn, the tls.secretName secret must contain a
	// matching certificate
	TLS *TLS `json:"tls,omitempty"`

	// Filters attached to this VirtualHost
	Filters []HostAttachedFilter `json:"filters,omitempty"`
}

// TLS describes tls properties. The CNI names that will be matched on
// are described in fqdn, the tls.secretName secret must contain a
// matching certificate unless tls.passthrough is set to true.
type TLS struct {
	// required, the name of a secret in the current namespace
	SecretName string `json:"secretName,omitempty"`
	// Minimum TLS version this vhost should negotiate
	MinimumProtocolVersion string `json:"minimumProtocolVersion,omitempty"`
	// If Passthrough is set to true, the SecretName will be ignored
	// and the encrypted handshake will be passed through to the
	// backing cluster.
	Passthrough bool `json:"passthrough,omitempty"`
}

// HeaderCondition specifies the header condition to match.
// Name is required. Only one of Present or Contains must
// be provided.
type HeaderCondition struct {

	// Name is the name of the header to match on. Name is required.
	// Header names are case insensitive.
	Name string `json:"name"`

	// Present is true if the Header is present in the request.
	// +optional
	Present bool `json:"present,omitempty"`

	// Contains is true if the Header containing this string is present
	// in the request.
	// +optional
	Contains string `json:"contains,omitempty"`

	// NotContains is true if the Header containing this string is not present
	// in the request.
	// +optional
	NotContains string `json:"notcontains,omitempty"`

	// Exact is true if the Header containing this string matches exactly
	// in the request.
	// +optional
	Exact string `json:"exact,omitempty"`

	// NotExact is true if the Header containing this string doesn't match exactly
	// in the request.
	// +optional
	NotExact string `json:"notexact,omitempty"`
}

// Condition are policies that are applied on top of GatewayHost.
// One of Prefix or Header must be provided.
type Condition struct {
	// Prefix defines a prefix match for a request.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Header specifies the header condition to match.
	// +optional
	Header *HeaderCondition `json:"header,omitempty"`
}

// Route contains the set of routes for a virtual host
type Route struct {
	// Conditions are a set of routing properties that is applied to an GatewayHost in a namespace.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
	// Services are the services to proxy traffic
	Services []Service `json:"services,omitempty"`
	// Delegate specifies that this route should be delegated to another GatewayHost
	Delegate *Delegate `json:"delegate,omitempty"`
	// Enables websocket support for the route
	EnableWebsockets bool `json:"enableWebsockets,omitempty"`
	// Allow this path to respond to insecure requests over HTTP which are normally
	// not permitted when a `virtualhost.tls` block is present.
	PermitInsecure bool `json:"permitInsecure,omitempty"`
	// Indicates that during forwarding, the matched prefix (or path) should be swapped with this value
	PrefixRewrite string `json:"prefixRewrite,omitempty"`
	// The timeout policy for this route
	TimeoutPolicy *TimeoutPolicy `json:"timeoutPolicy,omitempty"`
	// The retry policy for this route
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	// Filters attached to this route
	Filters []RouteAttachedFilter `json:"filters,omitempty"`

	// Disable external authorization for this route
	DisableExtAuthz bool `json:"disableExtauth,omitempty"`
}

// TCPProxy contains the set of services to proxy TCP connections.
type TCPProxy struct {
	// Services are the services to proxy traffic
	Services []Service `json:"services,omitempty"`
	// Delegate specifies that this tcpproxy should be delegated to another GatewayHost
	Delegate *Delegate `json:"delegate,omitempty"`
}

// Service defines an upstream to proxy traffic to
type Service struct {
	// Name is the name of Kubernetes service to proxy traffic.
	// Names defined here will be used to look up corresponding endpoints which contain the ips to route.
	Name string `json:"name"`
	// Port (defined as Integer) to proxy traffic to since a service can have multiple defined
	//
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65536
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +kubebuilder:validation:ExclusiveMaximum=true
	Port int `json:"port"`
	// Protocol may be used to specify (or override) the protocol used to reach this Service.
	// Values may be tls, h2, h2c. If omitted, protocol-selection falls back on Service annotations.
	// +kubebuilder:validation:Enum=h2;h2c;tls
	// +optional
	Protocol string `json:"protocol,omitempty"`
	// Weight defines percentage of traffic to balance traffic
	// +optional
	Weight uint32 `json:"weight,omitempty"`
	// HealthCheck defines optional healthchecks on the upstream service
	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
	// LB Algorithm to apply (see https://github.com/saarasio/enroute/enroute-dp/blob/master/design/gatewayhost-design.md#load-balancing)
	// +optional
	Strategy string `json:"strategy,omitempty"`
	// UpstreamValidation defines how to verify the backend service's certificate
	// +optional
	UpstreamValidation *UpstreamValidation `json:"validation,omitempty"`
	// ClientValidation defines a way to provide client's identity encoded in SAN in a certificate.
	// The certificate to send to backend service that it'll verify
	// +optional
	ClientValidation *UpstreamValidation `json:"clientvalidation,omitempty"`
}

// Delegate allows for delegating VHosts to other GatewayHosts
type Delegate struct {
	// Name of the GatewayHost
	Name string `json:"name"`
	// Namespace of the GatewayHost
	Namespace string `json:"namespace,omitempty"`
}

// HealthCheck defines optional healthchecks on the upstream service
type HealthCheck struct {
	// HTTP endpoint used to perform health checks on upstream service
	Path string `json:"path"`
	// The value of the host header in the HTTP health check request.
	// If left empty (default value), the name "contour-envoy-healthcheck"
	// will be used.
	Host string `json:"host,omitempty"`
	// The interval (seconds) between health checks
	IntervalSeconds int64 `json:"intervalSeconds"`
	// The time to wait (seconds) for a health check response
	TimeoutSeconds int64 `json:"timeoutSeconds"`
	// The number of unhealthy health checks required before a host is marked unhealthy
	UnhealthyThresholdCount uint32 `json:"unhealthyThresholdCount"`
	// The number of healthy health checks required before a host is marked healthy
	HealthyThresholdCount uint32 `json:"healthyThresholdCount"`
}

// TimeoutPolicy define the attributes associated with timeout
type TimeoutPolicy struct {
	// Timeout for receiving a response from the server after processing a request from client.
	// If not supplied the timeout duration is undefined.
	Request string `json:"request"`

	// +optional
	Idle string `json:"idle,omitempty"`

	// +optional
	// Idle Timeout for upstream connections, duration for which there are no active connections
	// default is 1h, to disable, set it to 0
	ClusterIdle string `json:"cluster_idle,omitempty"`

	// +optional
	// Timeout for new network connections, if not set default value is 5s
	ClusterConnect string `json:"cluster_connect,omitempty"`

	// +optional
	// Maximum duration for connection
	ClusterMaxConnectionDuration string `json:"cluster_max_duration,omitempty"`
}

// RetryPolicy define the attributes associated with retrying policy
type RetryPolicy struct {
	// NumRetries is maximum allowed number of retries.
	// If not supplied, the number of retries is zero.
	NumRetries uint32 `json:"count"`
	// PerTryTimeout specifies the timeout per retry attempt.
	// Ignored if NumRetries is not supplied.
	PerTryTimeout string `json:"perTryTimeout,omitempty"`
	// RetryOn specifies the conditions on which to retry a request.
	// +optional
	RetryOn string `json:"retryOn,omitempty"`

	// HTTP status codes that should trigger a retry in addition to those specified by RetryOn.
	// +optional
	RetriableStatusCodes []uint32 `json:"retriableStatusCodes,omitempty"`
}

// UpstreamValidation defines how to verify the backend service's certificate
type UpstreamValidation struct {
	// Name of the Kubernetes secret be used to validate the certificate presented by the backend
	CACertificate string `json:"caSecret"`
	// Key which is expected to be present in the 'subjectAltName' of the presented certificate
	SubjectName string `json:"subjectName"`
}

// Status reports the current state of the GatewayHost
type Status struct {
	CurrentStatus string `json:"currentStatus"`
	Description   string `json:"description"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayHost is an Ingress CRD specificiation
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="FQDN",type="string",JSONPath=".spec.virtualhost.fqdn",description="Fully qualified domain name"
// +kubebuilder:printcolumn:name="TLS Secret",type="string",JSONPath=".spec.virtualhost.tls.secretName",description="Secret with TLS credentials"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.currentStatus",description="The current status of the HTTPProxy"
// +kubebuilder:printcolumn:name="Status Description",type="string",JSONPath=".status.description",description="Description of the current status"
// +kubebuilder:resource:scope=Namespaced,path=gatewayhosts,shortName=gwhost;gwhosts,singular=gatewayhost
// +kubebuilder:subresource:status
type GatewayHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec GatewayHostSpec `json:"spec"`
	// +optional
	Status `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GatewayHostList is a list of GatewayHosts
type GatewayHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GatewayHost `json:"items"`
}
