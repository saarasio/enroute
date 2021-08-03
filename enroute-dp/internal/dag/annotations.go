// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.

// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dag

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Object interface {
	metav1.ObjectMetaAccessor
}

const (
	// set docs/annotations.md for details of how these annotations
	// are applied by Contour.

	annotationRequestTimeout     = "enroute.saaras.io/request-timeout"
	annotationResponseTimeout    = "enroute.saaras.io/response-timeout"
	annotationWebsocketRoutes    = "enroute.saaras.io/websocket-routes"
	annotationUpstreamProtocol   = "enroute.saaras.io/upstream-protocol"
	annotationMaxConnections     = "enroute.saaras.io/max-connections"
	annotationMaxPendingRequests = "enroute.saaras.io/max-pending-requests"
	annotationMaxRequests        = "enroute.saaras.io/max-requests"
	annotationMaxRetries         = "enroute.saaras.io/max-retries"
	annotationRetryOn            = "enroute.saaras.io/retry-on"
	annotationNumRetries         = "enroute.saaras.io/num-retries"
	annotationPerTryTimeout      = "enroute.saaras.io/per-try-timeout"
)

// '0' is returned if the annotation is absent or unparseable.
func maxConnections(o Object) uint32 {
	return parseUInt32(compatAnnotation(o, "max-connections"))
}

func maxPendingRequests(o Object) uint32 {
	return parseUInt32(compatAnnotation(o, "max-pending-requests"))
}

func maxRequests(o Object) uint32 {
	return parseUInt32(compatAnnotation(o, "max-requests"))
}

func maxRetries(o Object) uint32 {
	return parseUInt32(compatAnnotation(o, "max-retries"))
}

func compatAnnotation(o Object, key string) string {
	a := o.GetObjectMeta().GetAnnotations()

	return a["enroute.saaras.io/"+key]
}

// parseUInt32 parses the supplied string as if it were a uint32.
// If the value is not present, or malformed, or outside uint32's range, zero is returned.
func parseUInt32(s string) uint32 {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(v)
}

// parseAnnotation parses the annotation map for the supplied key.
// If the value is not present, or malformed, then zero is returned.
func parseAnnotation(annotations map[string]string, annotation string) uint32 {
	v, err := strconv.ParseUint(annotations[annotation], 10, 32)
	if err != nil {
		return 0
	}
	return uint32(v)
}

// parseUpstreamProtocols parses the annotations map for a enroute.saaras.io/upstream-protocol.{protocol}
// where 'protocol' identifies which protocol must be used in the upstream.
// If the value is not present, or malformed, then an empty map is returned.
func parseUpstreamProtocols(annotations map[string]string, annotation string, protocols ...string) map[string]string {
	up := make(map[string]string)
	for _, protocol := range protocols {
		ports := annotations[fmt.Sprintf("%s.%s", annotation, protocol)]
		for _, v := range strings.Split(ports, ",") {
			port := strings.TrimSpace(v)
			if port != "" {
				up[port] = protocol
			}
		}
	}
	return up
}

// httpAllowed returns true unless the kubernetes.io/ingress.allow-http annotation is
// present and set to false.
func httpAllowed(i *v1.Ingress) bool {
	return !(i.Annotations["kubernetes.io/ingress.allow-http"] == "false")
}

// tlsRequired returns true if the ingress.kubernetes.io/force-ssl-redirect annotation is
// present and set to true.
func tlsRequired(i *v1.Ingress) bool {
	return i.Annotations["ingress.kubernetes.io/force-ssl-redirect"] == "true"
}

func websocketRoutes(i *v1.Ingress) map[string]bool {
	routes := make(map[string]bool)
	for _, v := range strings.Split(i.Annotations[annotationWebsocketRoutes], ",") {
		route := strings.TrimSpace(v)
		if route != "" {
			routes[route] = true
		}
	}
	return routes
}

// MinProtoVersion returns the TLS protocol version specified by an ingress annotation
// or default if non present.
func MinProtoVersion(version string) envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol {
	switch version {
	case "1.3":
		return envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_3
	case "1.2":
		return envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_2
	default:
		// any other value is interpreted as TLS/1.1
		return envoy_extensions_transport_sockets_tls_v3.TlsParameters_TLSv1_1
	}
}

// perTryTimeout returns the duration envoy will wait per retry cycle.
func perTryTimeout(i *v1.Ingress) time.Duration {
	return parseTimeout(compatAnnotation(i, "per-try-timeout"))
}

// numRetries returns the number of retries specified by the "enroute.saaras.io/num-retries"
func numRetries(i *v1.Ingress) uint32 {
	return parseUInt32(compatAnnotation(i, "num-retries"))
}
