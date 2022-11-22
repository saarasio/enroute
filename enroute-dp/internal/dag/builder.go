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

// Package dag provides a data model, in the form of a directed acyclic graph,
// of the relationship between Kubernetes Ingress, Service, and Secret objects.
package dag

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	net_v1 "k8s.io/api/networking/v1"

	_ "github.com/davecgh/go-spew/spew"
	"github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	gatewayhostv1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	StatusValid    = "valid"
	StatusInvalid  = "invalid"
	StatusOrphaned = "orphaned"
)

// BuildDAG returns a new DAG from the supplied KubernetesCache.
func BuildDAG(kc *KubernetesCache) *DAG {
	log := logrus.StandardLogger()
	log.SetLevel(logrus.DebugLevel)
	builder := &builder{source: kc, log: log}
	builder.reset()
	log.Debugf("dag:builder:BuildDAG(): Recompute DAG, Reprogram Envoy")
	return builder.compute()
}

// reset (re)inialises the internal state of the builder.
func (b *builder) reset() {
	b.services = make(map[servicemeta]Service, len(b.services))
	b.secrets = make(map[Meta]*Secret, len(b.secrets))
	b.listeners = make(map[int]*Listener, len(b.listeners))

	b.routefilters = make(map[RouteFilterMeta]*RouteFilter, len(b.routefilters))
	b.httpfilters = make(map[HttpFilterMeta]*HttpFilter, len(b.httpfilters))

	b.statuses = make(map[Meta]Status, len(b.statuses))
}

// A builder holds the state of one invocation of Builder.Build.
// Once used, the builder should be discarded.
type builder struct {
	source *KubernetesCache

	services  map[servicemeta]Service
	secrets   map[Meta]*Secret
	listeners map[int]*Listener

	routefilters map[RouteFilterMeta]*RouteFilter
	httpfilters  map[HttpFilterMeta]*HttpFilter

	orphaned map[Meta]bool

	statuses map[Meta]Status
	log      logrus.FieldLogger
}

func (b *builder) debugPrintServices(m Meta, port net_v1.ServiceBackendPort) {
	if logger.EL.ELogger != nil && logger.EL.ELogger.GetLevel() >= logrus.DebugLevel {
		for key, svc := range b.source.services {
			logger.EL.ELogger.Debugf(
				"dag:builder:debugPrintServices(): Have service meta-key [%+v] svc [%+v]\n", key, svc)
		}
	}
}

// lookupHTTPService returns a HTTPService that matches the Meta and port supplied.
func (b *builder) lookupHTTPService(m Meta, port net_v1.ServiceBackendPort) *HTTPService {
	s := b.lookupService(m, port)
	switch s := s.(type) {
	case *HTTPService:
		return s
	case nil:
		svc, ok := b.source.services[m]
		if !ok {
			b.debugPrintServices(m, port)
			return nil
		}
		for i := range svc.Spec.Ports {
			p := &svc.Spec.Ports[i]
			if p.Port == port.Number {
				return b.addHTTPService(svc, p)
			}
			// When comparing names, one of them should be non-empty for comparison to be valid
			if port.Name == p.Name && ((port.Name != "") || (p.Name != "")) {
				return b.addHTTPService(svc, p)
			}
		}
		b.debugPrintServices(m, port)
		return nil
	default:
		// some other type
		b.debugPrintServices(m, port)
		return nil
	}
}

// lookupTCPService returns a TCPService that matches the Meta and port supplied.
func (b *builder) lookupTCPService(m Meta, port net_v1.ServiceBackendPort) *TCPService {
	s := b.lookupService(m, port)
	switch s := s.(type) {
	case *TCPService:
		return s
	case nil:
		svc, ok := b.source.services[m]
		if !ok {
			return nil
		}
		for i := range svc.Spec.Ports {
			p := &svc.Spec.Ports[i]
			if p.Port == port.Number {
				return b.addTCPService(svc, p)
			}
			if port.Name == p.Name {
				return b.addTCPService(svc, p)
			}
		}
		return nil
	default:
		// some other type
		return nil
	}
}
func (b *builder) lookupService(m Meta, port net_v1.ServiceBackendPort) Service {
	if port.Number < 0 || port.Number > 65535 {
		// can't handle, give up
		return nil
	}
	sm := servicemeta{
		name:      m.name,
		namespace: m.namespace,
		port:      port.Number,
	}
	s, ok := b.services[sm]
	if !ok {
		return nil // avoid typed nil
	}
	return s
}

func (b *builder) addHTTPService(svc *v1.Service, port *v1.ServicePort) *HTTPService {
	if b.services == nil {
		b.services = make(map[servicemeta]Service)
	}
	up := parseUpstreamProtocols(svc.Annotations, annotationUpstreamProtocol, "h2", "h2c", "tls")
	protocol := up[port.Name]
	if protocol == "" {
		protocol = up[strconv.Itoa(int(port.Port))]
	}

	s := &HTTPService{
		TCPService: TCPService{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			ServicePort: port,

			MaxConnections:     maxConnections(svc),
			MaxPendingRequests: maxPendingRequests(svc),
			MaxRequests:        maxRequests(svc),
			MaxRetries:         maxRetries(svc),
			ExternalName:       externalName(svc),
		},
		Protocol: protocol,
	}
	b.services[s.toMeta()] = s
	return s
}

func externalName(svc *v1.Service) string {
	if svc.Spec.Type != v1.ServiceTypeExternalName {
		return ""
	}
	return svc.Spec.ExternalName
}

func (b *builder) addTCPService(svc *v1.Service, port *v1.ServicePort) *TCPService {
	if b.services == nil {
		b.services = make(map[servicemeta]Service)
	}
	s := &TCPService{
		Name:        svc.Name,
		Namespace:   svc.Namespace,
		ServicePort: port,

		MaxConnections:     maxConnections(svc),
		MaxPendingRequests: maxPendingRequests(svc),
		MaxRequests:        maxRequests(svc),
		MaxRetries:         maxRetries(svc),
	}
	b.services[s.toMeta()] = s
	return s
}

// lookupSecret returns a Secret if present or nil if the underlying kubernetes
// secret fails validation or is missing.
func (b *builder) lookupSecret(m Meta, validate func(*v1.Secret) bool) *Secret {

	if s, ok := b.secrets[m]; ok {
		return s
	}
	sec, ok := b.source.secrets[m]
	if !ok {
		if logger.EL.ELogger != nil {
			logger.EL.ELogger.Infof("dag:builder:lookupSecret(): secret [%+v] not found in k8s cache\n", m)
		}
		return nil
	}
	if !validate(sec) {
		if logger.EL.ELogger != nil {
			logger.EL.ELogger.Infof("dag:builder:lookupSecret(): secret [%+v] in k8s cache but invalid\n", m)
		}
		return nil
	}
	s := &Secret{
		Object: sec,
	}
	if b.secrets == nil {
		b.secrets = make(map[Meta]*Secret)
	}
	b.secrets[s.toMeta()] = s
	return s
}

func (b *builder) lookupVirtualHost(name string) *VirtualHost {
	l := b.listener(80)
	vh, ok := l.VirtualHosts[name]
	if !ok {
		vh := &VirtualHost{
			Name: name,
		}
		l.VirtualHosts[vh.Name] = vh
		return vh
	}
	return vh.(*VirtualHost)
}

func (b *builder) lookupSecureVirtualHost(name string) *SecureVirtualHost {
	l := b.listener(443)
	svh, ok := l.VirtualHosts[name]
	if !ok {
		svh := &SecureVirtualHost{
			VirtualHost: VirtualHost{
				Name: name,
			},
		}
		l.VirtualHosts[svh.VirtualHost.Name] = svh
		return svh
	}
	return svh.(*SecureVirtualHost)
}

// listener returns a listener for the supplied port.
// TODO: the port value is not actually used as a port
// anywhere. It's only used to choose between
// 80 (for insecure) or 443 (for secure). This should be
// fixed, see https://github.com/saarasio/enroute/enroute-dp/issues/1135
func (b *builder) listener(port int) *Listener {
	l, ok := b.listeners[port]
	if !ok {
		l = &Listener{
			Port:         port,
			VirtualHosts: make(map[string]Vertex),
		}
		if b.listeners == nil {
			b.listeners = make(map[int]*Listener)
		}
		b.listeners[l.Port] = l
	}
	return l
}

func (b *builder) compute() *DAG {
	b.source.mu.RLock() // blocks mutation of the underlying cache until compute is done.
	defer b.source.mu.RUnlock()

	if logger.EL.ELogger != nil {
		logger.EL.ELogger.Debugf("dag:builder:compute:DAG compute()\n")
	}

	// setup secure vhosts if there is a matching secret
	// we do this first so that the set of active secure vhosts is stable
	// during computeIngresses.
	b.computeSecureVirtualhosts()

	b.computeIngresses()

	b.computeGatewayHosts()

	return b.DAG()
}

// route builds a dag.Route for the supplied Ingress.
func route(ingress *net_v1.Ingress, path string, service *HTTPService) *Route {
	wr := websocketRoutes(ingress)
	r := &Route{
		HTTPSUpgrade:  tlsRequired(ingress),
		Websocket:     wr[path],
		TimeoutPolicy: ingressTimeoutPolicy(ingress),
		RetryPolicy:   ingressRetryPolicy(ingress),
		Clusters: []*Cluster{{
			Upstream: service,
		}},
	}

	if strings.ContainsAny(path, "^+*[]%") {
		// path smells like a regex
		r.PathCondition = &RegexCondition{Regex: path}
		return r
	}

	r.PathCondition = &PrefixCondition{Prefix: path}
	return r
}

// isBlank indicates if a string contains nothing but blank characters.
func isBlank(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// minProtoVersion returns the TLS protocol version specified by an ingress annotation
// or default if non present.
func minProtoVersion(version string) tlsv3.TlsParameters_TlsProtocol {
	switch version {
	case "1.3":
		return tlsv3.TlsParameters_TLSv1_3
	case "1.2":
		return tlsv3.TlsParameters_TLSv1_2
	default:
		// any other value is interpreted as TLS/1.1
		return tlsv3.TlsParameters_TLSv1_1
	}
}

// validGatewayHosts returns a slice of *gatewayhostv1.GatewayHost objects.
// invalid GatewayHost objects are excluded from the slice and a corresponding entry
// added via setStatus.
func (b *builder) validGatewayHosts() []*gatewayhostv1.GatewayHost {
	// ensure that a given fqdn is only referenced in a single gatewayhost resource
	var valid []*gatewayhostv1.GatewayHost
	fqdnIngressroutes := make(map[string][]*gatewayhostv1.GatewayHost)
	for _, ir := range b.source.gatewayhosts {
		if ir.Spec.VirtualHost == nil {
			valid = append(valid, ir)
			continue
		}
		fqdnIngressroutes[ir.Spec.VirtualHost.Fqdn] = append(fqdnIngressroutes[ir.Spec.VirtualHost.Fqdn], ir)
	}

	for fqdn, irs := range fqdnIngressroutes {
		switch len(irs) {
		case 1:
			valid = append(valid, irs[0])
		default:
			// multiple irs use the same fqdn. mark them as invalid.
			var conflicting []string
			for _, ir := range irs {
				conflicting = append(conflicting, fmt.Sprintf("%s/%s", ir.Namespace, ir.Name))
			}
			sort.Strings(conflicting) // sort for test stability
			msg := fmt.Sprintf(
				"fqdn %q is used in multiple GatewayHosts: %s",
				fqdn, strings.Join(conflicting, ", "))
			for _, ir := range irs {
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: msg, Vhost: fqdn})
			}
		}
	}
	return valid
}

// computeSecureVirtualhosts populates tls parameters of
// secure virtual hosts.
func (b *builder) computeSecureVirtualhosts() {
	for _, ing := range b.source.ingresses {
		for _, tls := range ing.Spec.TLS {
			m := splitSecret(tls.SecretName, ing.Namespace)
			if sec := b.lookupSecret(m, validSecret); sec != nil && b.delegationPermitted(m, ing.Namespace) {
				for _, host := range tls.Hosts {
					svhost := b.lookupSecureVirtualHost(host)
					svhost.Secret = sec
					version := compatAnnotation(ing, "tls-minimum-protocol-version")
					svhost.MinProtoVersion = minProtoVersion(version)
				}
			}
		}
	}
}

// splitSecret splits a secretName into its namespace and name components.
// If there is no namespace prefix, the default namespace is returned.
func splitSecret(secret, defns string) Meta {
	v := strings.SplitN(secret, "/", 2)
	switch len(v) {
	case 1:
		// no prefix
		return Meta{
			name:      v[0],
			namespace: defns,
		}
	default:
		return Meta{
			name:      v[1],
			namespace: stringOrDefault(v[0], defns),
		}
	}
}

func (b *builder) delegationPermitted(secret Meta, to string) bool {
	contains := func(haystack []string, needle string) bool {
		if len(haystack) == 1 && haystack[0] == "*" {
			return true
		}
		for _, h := range haystack {
			if h == needle {
				return true
			}
		}
		return false
	}

	if secret.namespace == to {
		// secret is in the same namespace as target
		return true
	}
	for _, d := range b.source.delegations {
		if d.Namespace != secret.namespace {
			continue
		}
		for _, d := range d.Spec.Delegations {
			if contains(d.TargetNamespaces, to) {
				if secret.name == d.SecretName {
					return true
				}
			}
		}
	}
	return false
}

func (b *builder) computeIngresses() {
	// deconstruct each ingress into routes and virtualhost entries
	for _, ing := range b.source.ingresses {

		// rewrite the default ingress to a stock ingress rule.
		rules := rulesFromSpec(ing.Spec)

		for _, rule := range rules {
			host := rule.Host
			if strings.Contains(host, "*") {
				// reject hosts with wildcard characters.
				continue
			}
			if host == "" {
				// if host name is blank, rewrite to Envoy's * default host.
				host = "*"
			}

			for _, httppath := range httppaths(rule) {
				path := stringOrDefault(httppath.Path, "/")
				be := httppath.Backend
				m := Meta{name: be.Service.Name, namespace: ing.Namespace}
				// XXX: We now read the protocol directly from gatewayhostv1.Service
				// s.Protocol = be.Protocol
				s := b.lookupHTTPService(m, be.Service.Port)
				if s == nil {
					continue
				}

				r := route(ing, path, s)

				// should we create port 80 routes for this ingress
				if tlsRequired(ing) || httpAllowed(ing) {
					b.lookupVirtualHost(host).addRoute(r)
				}

				if b.secureVirtualhostExists(host) && host != "*" {
					b.lookupSecureVirtualHost(host).addRoute(r)
				}
			}
		}
	}
}

func stringOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (b *builder) computeGatewayHosts() {
	for _, ir := range b.validGatewayHosts() {
		if ir.Spec.VirtualHost == nil {
			// mark delegate gatewayhost orphaned.
			b.setOrphaned(ir)
			continue
		}

		// ensure root gatewayhost lives in allowed namespace
		if !b.rootAllowed(ir) {
			b.setStatus(Status{Object: ir, Status: StatusInvalid,
				Description: "root GatewayHost cannot be defined in this namespace"})
			continue
		}

		host := ir.Spec.VirtualHost.Fqdn
		if isBlank(host) {
			b.setStatus(Status{Object: ir, Status: StatusInvalid,
				Description: "Spec.VirtualHost.Fqdn must be specified",
				Vhost:       host})
			continue
		}

		// Allow wildcard host if non-TLS
		if ir.Spec.VirtualHost.TLS != nil && strings.Contains(host, "*") {
			b.setStatus(Status{Object: ir, Status: StatusInvalid,
				Description: fmt.Sprintf("Spec.VirtualHost.Fqdn %q cannot use wildcards", host),
				Vhost:       host})
			continue
		}

		var enforceTLS, passthrough bool
		if tls := ir.Spec.VirtualHost.TLS; tls != nil {
			// attach secrets to TLS enabled vhosts
			m := splitSecret(tls.SecretName, ir.Namespace)
			sec := b.lookupSecret(m, validSecret)
			secretInvalidOrNotFound := sec == nil
			if sec != nil && b.delegationPermitted(m, ir.Namespace) {
				svhost := b.lookupSecureVirtualHost(host)
				svhost.Secret = sec
				svhost.MinProtoVersion = minProtoVersion(ir.Spec.VirtualHost.TLS.MinimumProtocolVersion)
				enforceTLS = true
				b.SetupHttpFilters(&svhost.VirtualHost, ir)
			}
			// passthrough is true if tls.secretName is not present, and
			// tls.passthrough is set to true.
			passthrough = isBlank(tls.SecretName) && tls.Passthrough

			// If not passthrough and secret is invalid, then set status
			if secretInvalidOrNotFound && !passthrough {
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: fmt.Sprintf("TLS Secret [%s] not found or is malformed", tls.SecretName),
					Vhost:       host})
			}
		}

		switch {
		case ir.Spec.TCPProxy != nil && (passthrough || enforceTLS):
			b.processTCPProxy(ir, nil, host)
		case ir.Spec.Routes != nil:
			vh := b.lookupVirtualHost(host)
			b.SetupHttpFilters(vh, ir)
			b.processRoutes(ir, nil, host, enforceTLS)
		}
	}
}

func (b *builder) secureVirtualhostExists(host string) bool {
	_, ok := b.listener(443).VirtualHosts[host]
	return ok
}

// rulesFromSpec merges the IngressSpec's Rules with a synthetic
// rule representing the default backend.
func rulesFromSpec(spec net_v1.IngressSpec) []net_v1.IngressRule {
	rules := spec.Rules
	if backend := spec.DefaultBackend; backend != nil {
		rule := defaultBackendRule(backend)
		rules = append(rules, rule)
	}
	return rules
}

// defaultBackendRule returns an IngressRule that represents the IngressBackend.
func defaultBackendRule(be *net_v1.IngressBackend) net_v1.IngressRule {
	return net_v1.IngressRule{
		IngressRuleValue: net_v1.IngressRuleValue{
			HTTP: &net_v1.HTTPIngressRuleValue{
				Paths: []net_v1.HTTPIngressPath{{
					Backend: net_v1.IngressBackend{
						Service: &net_v1.IngressServiceBackend{
							Name: be.Service.Name,
							Port: be.Service.Port,
						},
					},
				}},
			},
		},
	}
}

// DAG returns a *DAG representing the current state of this builder.
func (b *builder) DAG() *DAG {
	var dag DAG
	for _, l := range b.listeners {
		for k, vh := range l.VirtualHosts {
			switch vh := vh.(type) {
			case *VirtualHost:
				// suppress virtual hosts without routes.
				if len(vh.Routes) < 1 {
					delete(l.VirtualHosts, k)
				}
			case *SecureVirtualHost:
				// suppress secure virtual hosts without secrets or tcpproxy.
				if vh.Secret == nil && vh.TCPProxy == nil {
					delete(l.VirtualHosts, k)
				}
			}
		}
		// suppress empty listeners
		if len(l.VirtualHosts) > 0 {
			dag.roots = append(dag.roots, l)
		}
	}
	for meta := range b.orphaned {
		ir, ok := b.source.gatewayhosts[meta]
		if ok {
			b.setStatus(Status{
				Object: ir, Status: StatusOrphaned,
				Description: "this GatewayHost is not part of a delegation chain from a root GatewayHost"})
		}
	}
	dag.statuses = b.statuses

	return &dag
}

// setStatus assigns a status to an object.
func (b *builder) setStatus(st Status) {
	m := Meta{name: st.Object.Name, namespace: st.Object.Namespace}
	if b.statuses == nil {
		b.statuses = make(map[Meta]Status)
	}
	if _, ok := b.statuses[m]; !ok {
		b.statuses[m] = st
	}
}

// setOrphaned records an gatewayhost as orphaned.
func (b *builder) setOrphaned(ir *gatewayhostv1.GatewayHost) {
	if b.orphaned == nil {
		b.orphaned = make(map[Meta]bool)
	}
	m := Meta{name: ir.Name, namespace: ir.Namespace}
	b.orphaned[m] = true
}

// rootAllowed returns true if the gatewayhost lives in a permitted root namespace.
func (b *builder) rootAllowed(ir *gatewayhostv1.GatewayHost) bool {
	if len(b.source.GatewayHostRootNamespaces) == 0 {
		return true
	}
	for _, ns := range b.source.GatewayHostRootNamespaces {
		if ns == ir.Namespace {
			return true
		}
	}
	return false
}

// validSecret returns true if the Secret contains certificate and private key material.
func validSecret(s *v1.Secret) bool {
	return s.Type == v1.SecretTypeTLS && len(s.Data[v1.TLSCertKey]) > 0 && len(s.Data[v1.TLSPrivateKeyKey]) > 0
}

// validCert returns true if the Secret contains certificate and private key material.
func validCert(s *v1.Secret) bool {
	return s.Type == v1.SecretTypeTLS && len(s.Data[v1.TLSCertKey]) > 0
}

func validCA(s *v1.Secret) bool {
	return len(s.Data["ca.crt"]) > 0
}

func getProtocol(service gatewayhostv1.Service, s *HTTPService) (string, error) {
	// Determine the protocol to use to speak to this Cluster.
	var protocol string
	if service.Protocol != "" {
		protocol = service.Protocol
		switch protocol {
		case "h2c", "h2", "tls":
			// If we did not set the protocol on the DAG service, copy it from GatewayHost Service to DAG Service
			s.Protocol = protocol
		default:
			return "", fmt.Errorf("unsupported protocol: %v", protocol)
		}
	} else {
		protocol = s.Protocol
	}

	return protocol, nil
}

func (b *builder) processOneRoute(ir *gatewayhostv1.GatewayHost, route gatewayhostv1.Route, host string, enforceTLS bool, ns string) error {
	pathConditionValid, errMesg := pathConditionsValid(route.Conditions, "route")

	if !pathConditionValid {
		if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
			logger.EL.ELogger.Debugf(
				"dag:builder:processServiceRoute() Path Condition Invalid: Host [%s] Err [%s] Conditions [%+v]",
				host, errMesg, route.Conditions)
		}
		return fmt.Errorf("%s", errMesg)
	}

	// Look for duplicate exact match headers on this route
	if !headerConditionsAreValid(route.Conditions) {
		if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
			logger.EL.ELogger.Debugf(
				"dag:builder:processRoutes() cannot specify duplicate header 'exact match' conditions in the same route - GatewayHost [%s] ", ir.Spec.VirtualHost.Fqdn)
		}
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: "cannot specify duplicate header 'exact match' conditions in the same route", Vhost: host})
	}

	// Route has a delegate, check doesn't have a service
	if len(route.Services) > 0 && route.Delegate != nil {
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: "cannot specify services and delegate in the same route", Vhost: host})
		return fmt.Errorf("cannot specify services and delegate in the same route")
	}

	if len(route.Services) > 0 {
		r := &Route{
			PathCondition:    mergePathConditions(route.Conditions),
			HeaderConditions: MergeHeaderConditions(route.Conditions),
			Websocket:        route.EnableWebsockets,
			HTTPSUpgrade:     routeEnforceTLS(enforceTLS, route.PermitInsecure),
			PrefixRewrite:    route.PrefixRewrite,
			TimeoutPolicy:    timeoutPolicy(route.TimeoutPolicy),
			RetryPolicy:      retryPolicy(route.RetryPolicy),
			DisableExtAuthz:  route.DisableExtAuthz,
		}

		b.SetupRouteFilters(r, &route, ns)
		routeServiceFilters := b.RouteServiceFilters(r, &route, ns)

		for _, service := range route.Services {
			if service.Port < 1 || service.Port > 65535 {
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: fmt.Sprintf("service %q: port must be in the range 1-65535",
						service.Name), Vhost: host})
				return fmt.Errorf("service %q: port must be in the range 1-65535", service.Name)
			}
			if service.Weight < 0 && logger.EL.ELogger != nil {
				logger.EL.ELogger.Infof("bad service weight [%s] [%d]\n",
					service.Name, service.Weight)
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: fmt.Sprintf("service %q: weight must be greater than or equal to zero",
						service.Name), Vhost: host})
				return fmt.Errorf("service %q: weight must be greater than or equal to zero", service.Name)
			}
			m := Meta{name: service.Name, namespace: ns}
			s := b.lookupHTTPService(m, net_v1.ServiceBackendPort{Number: int32(service.Port)})

			if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil && logger.EL.ELogger.GetLevel() >= logrus.DebugLevel {
				logger.EL.ELogger.Debugf("dag:builder:processRoutes() Valid: GatewayHost [%s] Looked up Service [%s:%d]\n",
					ir.Spec.VirtualHost.Fqdn, service.Name, service.Port)
			}

			if s == nil {
				if logger.EL.ELogger != nil {
					logger.EL.ELogger.Debugf("dag:builder:processRoutes() bad service [%s] - invalid or missing\n", service.Name)
				}
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: fmt.Sprintf("Service [%s:%d] is invalid or missing", service.Name, service.Port), Vhost: host})
				return fmt.Errorf("service [%s:%d]: is invalid or missing", service.Name, service.Port)
			}

			var uv, cv *UpstreamValidation
			var err error

			if logger.EL.ELogger != nil {
				logger.EL.ELogger.Debugf("dag:builder:processRoutes() service [%s] - protocol [%s]\n", service.Name, s.Protocol)
			}

			protocol, err := getProtocol(service, s)

			// XXX: Ensure that our HTTPService on DAG has the right protocol set
			s.Protocol = protocol

			if err != nil {
				if logger.EL.ELogger != nil {
					logger.EL.ELogger.Debugf(
						"bad service [%s] - error in getting protocol\n", service.Name)
				}
			}

			if protocol == "tls" {
				// we can only validate TLS connections to services that talk TLS
				uv, err = b.lookupUpstreamValidation(ir, host, &service, ns)

				if err != nil {
					// Do not add route/upstream if we cannot validate upstream validation context
					if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
						logger.EL.ELogger.Infof(
							"dag:builder:processRoutes() Valid: GatewayHost [%s] TLS Service [%s:%d] Upstream Validation err [%s]\n",
							ir.Spec.VirtualHost.Fqdn, service.Name, service.Port, err)
					}

					return fmt.Errorf("service [%s:%d]: Error in upstream validation", service.Name, service.Port)
				}

				// client validation
				cv, err = b.lookupClientValidation(ir, host, &service, ns)

				if err != nil {
					// Do not add route/upstream if we cannot validate upstream validation context
					if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
						logger.EL.ELogger.Infof(
							"dag:builder:processRoutes() Valid: GatewayHost [%s] TLS Service [%s:%d] Client Validation err [%s]\n",
							ir.Spec.VirtualHost.Fqdn, service.Name, service.Port, err)
					}

					return fmt.Errorf("service [%s:%d]: Error in client validation", service.Name, service.Port)
				}
			}
			// When talking to an ExternalName (DNS) service, explicitly set SNI to that name
			r.Clusters = append(r.Clusters, &Cluster{
				Upstream:             s,
				LoadBalancerStrategy: service.Strategy,
				Weight:               service.Weight,
				HealthCheck:          service.HealthCheck,
				UpstreamValidation:   uv,
				ClientValidation:     cv,
				ClusterFilters:       routeServiceFilters,
				SNI:                  s.ExternalName,
			})
			if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
				logger.EL.ELogger.Infof(
					"dag:builder:processRoutes() Valid: GatewayHost [%s] Setup Cluster \n", ir.Spec.VirtualHost.Fqdn)
			}
		}

		b.lookupVirtualHost(host).addRoute(r)
		b.lookupSecureVirtualHost(host).addRoute(r)
	}

	return nil
}

func (b *builder) processServiceRoute(ir *gatewayhostv1.GatewayHost, ghr *gatewayhostv1.ServiceRoute, host string, enforceTLS bool) {

	if ghr == nil {
		return
	}

	route := ghr.Spec.Route
	ns := ghr.Namespace

	b.processOneRoute(ir, route, host, enforceTLS, ns)
}

// Process routes for one GatewayHost
func (b *builder) processRoutes(ir *gatewayhostv1.GatewayHost, visited []*gatewayhostv1.GatewayHost, host string, enforceTLS bool) {
	visited = append(visited, ir)

	if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
		logger.EL.ELogger.Debugf("dag:builder:processRoutes() GatewayHost [%s]", ir.Spec.VirtualHost.Fqdn)
	}

	ns := ir.Namespace

	for _, route := range ir.Spec.Routes {

		err := b.processOneRoute(ir, route, host, enforceTLS, ns)

		if err != nil {
			return
		}

		// Process ServiceRoute for this GatewayHost
		for _, ghroute := range b.source.serviceroutes {
			if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
				logger.EL.ELogger.Infof(
					"dag:builder:processRoutes() Processing ServiceRoute [%v] \n", ghroute)
			}

			if host == ghroute.Spec.Fqdn {
				if ir != nil && ir.Spec.VirtualHost != nil && logger.EL.ELogger != nil {
					logger.EL.ELogger.Infof(
						"dag:builder:processRoutes() ServiceRoute fqdn Match [%s] \n", host)
				}

				b.processServiceRoute(ir, ghroute, host, enforceTLS)
			}
		}

		if route.Delegate == nil {
			// not a delegate route
			continue
		}

		namespace := route.Delegate.Namespace
		if namespace == "" {
			// we are delegating to another GatewayHost in the same namespace
			namespace = ir.Namespace
		}

		if dest, ok := b.source.gatewayhosts[Meta{name: route.Delegate.Name, namespace: namespace}]; ok {
			// dest is not an orphaned ingress route, as there is an GatewayHost that points to it
			delete(b.orphaned, Meta{name: dest.Name, namespace: dest.Namespace})

			// ensure we are not following an edge that produces a cycle
			var path []string
			for _, vir := range visited {
				path = append(path, fmt.Sprintf("%s/%s", vir.Namespace, vir.Name))
			}
			for _, vir := range visited {
				if dest.Name == vir.Name && dest.Namespace == vir.Namespace {
					path = append(path, fmt.Sprintf("%s/%s", dest.Namespace, dest.Name))
					description := fmt.Sprintf("route creates a delegation cycle: %s", strings.Join(path, " -> "))
					b.setStatus(Status{Object: ir, Status: StatusInvalid,
						Description: description, Vhost: host})
					return
				}
			}

			// follow the link and process the target ingress route
			b.processRoutes(dest, visited, host, enforceTLS)
		}
	}

	b.setStatus(Status{Object: ir, Status: StatusValid, Description: "valid GatewayHost", Vhost: host})
}

// TODO(dfc) needs unit tests; we should pass in some kind of context object that encasulates all the properties we need for reporting
// status here, the ir, the host, the route, etc. I'm thinking something like logrus' WithField.

func (b *builder) lookupUpstreamValidation(ir *gatewayhostv1.GatewayHost, host string, service *gatewayhostv1.Service, namespace string) (*UpstreamValidation, error) {
	uv := service.UpstreamValidation
	if uv == nil {
		// no upstream validation requested, nothing to do
		return nil, nil
	}

	cacert := b.lookupSecret(Meta{name: uv.CACertificate, namespace: namespace}, validCA)
	if cacert == nil {
		// UpstreamValidation is requested, but cert is missing or not configured
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: fmt.Sprintf("service %q: upstreamValidation requested but secret not found or misconfigured", service.Name), Vhost: host})
		return nil, fmt.Errorf("service %q: upstreamValidation requested but secret not found or misconfigured", service.Name)
	}

	if uv.SubjectName == "" {
		// UpstreamValidation is requested, but SAN is not provided
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: fmt.Sprintf("service %q: upstreamValidation requested but subject alt name not found or misconfigured", service.Name), Vhost: host})
		return nil, fmt.Errorf("service %q: upstreamValidation requested but subject alt name not found or misconfigured", service.Name)
	}

	return &UpstreamValidation{
		CACertificate: cacert,
		SubjectName:   uv.SubjectName,
	}, nil
}

func (b *builder) lookupClientValidation(ir *gatewayhostv1.GatewayHost, host string, service *gatewayhostv1.Service, namespace string) (*UpstreamValidation, error) {
	cv := service.ClientValidation
	if cv == nil {
		// no client validation requested, nothing to do
		return nil, nil
	}

	cacert := b.lookupSecret(Meta{name: cv.CACertificate, namespace: namespace}, validSecret)
	if cacert == nil {
		// UpstreamValidation is requested, but cert is missing or not configured
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: fmt.Sprintf("service %q: clientValidation requested but secret not found or misconfigured", service.Name), Vhost: host})
		return nil, fmt.Errorf("service %q: clientValidation requested but secret not found or misconfigured", service.Name)
	}

	// TODO: We probably don't need to specify the SAN since the cert has the SAN that the backend service will verify
	//if cv.SubjectName == "" {
	//	// UpstreamValidation is requested, but SAN is not provided
	//	b.setStatus(Status{Object: ir, Status: StatusInvalid, Description: fmt.Sprintf("service %q: upstreamValidation requested but subject alt name not found or misconfigured", service.Name), Vhost: host})
	//	return nil, fmt.Errorf("service %q: upstreamValidation requested but subject alt name not found or misconfigured", service.Name)
	//}

	return &UpstreamValidation{
		CACertificate: cacert,
	}, nil
}

func (b *builder) processTCPProxy(ir *gatewayhostv1.GatewayHost, visited []*gatewayhostv1.GatewayHost, host string) {
	visited = append(visited, ir)

	// tcpproxy cannot both delegate and point to services
	tcpproxy := ir.Spec.TCPProxy
	if len(tcpproxy.Services) > 0 && tcpproxy.Delegate != nil {
		b.setStatus(Status{Object: ir, Status: StatusInvalid,
			Description: "tcpproxy: cannot specify services and delegate in the same tcpproxy", Vhost: host})
		return
	}

	if len(tcpproxy.Services) > 0 {
		var proxy TCPProxy
		for _, service := range tcpproxy.Services {
			m := Meta{name: service.Name, namespace: ir.Namespace}
			s := b.lookupTCPService(m, net_v1.ServiceBackendPort{Number: int32(service.Port)})
			if s == nil {
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: fmt.Sprintf("tcpproxy: service %s/%s/%d: not found",
						ir.Namespace, service.Name, service.Port), Vhost: host})
				return
			}
			proxy.Clusters = append(proxy.Clusters, &Cluster{
				Upstream:             s,
				LoadBalancerStrategy: service.Strategy,
			})
		}
		b.lookupSecureVirtualHost(host).VirtualHost.TCPProxy = &proxy
		b.setStatus(Status{Object: ir, Status: StatusValid, Description: "valid GatewayHost", Vhost: host})
		return
	}

	if tcpproxy.Delegate == nil {
		// not a delegate tcpproxy
		return
	}

	namespace := tcpproxy.Delegate.Namespace
	if namespace == "" {
		// we are delegating to another GatewayHost in the same namespace
		namespace = ir.Namespace
	}

	if dest, ok := b.source.gatewayhosts[Meta{name: tcpproxy.Delegate.Name, namespace: namespace}]; ok {
		// dest is not an orphaned ingress route, as there is an GatewayHost that points to it
		delete(b.orphaned, Meta{name: dest.Name, namespace: dest.Namespace})

		// ensure we are not following an edge that produces a cycle
		var path []string
		for _, vir := range visited {
			path = append(path, fmt.Sprintf("%s/%s", vir.Namespace, vir.Name))
		}
		for _, vir := range visited {
			if dest.Name == vir.Name && dest.Namespace == vir.Namespace {
				path = append(path, fmt.Sprintf("%s/%s", dest.Namespace, dest.Name))
				description := fmt.Sprintf("tcpproxy creates a delegation cycle: %s", strings.Join(path, " -> "))
				b.setStatus(Status{Object: ir, Status: StatusInvalid,
					Description: description, Vhost: host})
				return
			}
		}

		// follow the link and process the target ingress route
		b.processTCPProxy(dest, visited, host)
	}

	b.setStatus(Status{Object: ir, Status: StatusValid,
		Description: "valid GatewayHost", Vhost: host})
}

// routeEnforceTLS determines if the route should redirect the user to a secure TLS listener
func routeEnforceTLS(enforceTLS, permitInsecure bool) bool {
	return enforceTLS && !permitInsecure
}

// httppaths returns a slice of HTTPIngressPath values for a given IngressRule.
// In the case that the IngressRule contains no valid HTTPIngressPaths, a
// nil slice is returned.
func httppaths(rule net_v1.IngressRule) []net_v1.HTTPIngressPath {
	if rule.IngressRuleValue.HTTP == nil {
		// rule.IngressRuleValue.HTTP value is optional.
		return nil
	}
	return rule.IngressRuleValue.HTTP.Paths
}

// matchesPathPrefix checks whether the given path matches the given prefix
func matchesPathPrefix(path, prefix string) bool {
	if len(prefix) == 0 {
		return true
	}
	// an empty string cannot have a prefix
	if len(path) == 0 {
		return false
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	if path[len(path)-1] != '/' {
		path += "/"
	}
	return strings.HasPrefix(path, prefix)
}

// Status contains the status for an GatewayHost (valid / invalid / orphan, etc)
type Status struct {
	Object      *gatewayhostv1.GatewayHost
	Status      string
	Description string
	Vhost       string
}
