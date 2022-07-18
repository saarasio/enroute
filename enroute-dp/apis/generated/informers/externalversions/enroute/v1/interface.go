// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/saarasio/enroute/enroute-dp/apis/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// GatewayHosts returns a GatewayHostInformer.
	GatewayHosts() GatewayHostInformer
	// GatewayHostRoutes returns a GatewayHostRouteInformer.
	GatewayHostRoutes() GatewayHostRouteInformer
	// GlobalConfigs returns a GlobalConfigInformer.
	GlobalConfigs() GlobalConfigInformer
	// HttpFilters returns a HttpFilterInformer.
	HttpFilters() HttpFilterInformer
	// PolicyOverlays returns a PolicyOverlayInformer.
	PolicyOverlays() PolicyOverlayInformer
	// RouteFilters returns a RouteFilterInformer.
	RouteFilters() RouteFilterInformer
	// TLSCertificateDelegations returns a TLSCertificateDelegationInformer.
	TLSCertificateDelegations() TLSCertificateDelegationInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// GatewayHosts returns a GatewayHostInformer.
func (v *version) GatewayHosts() GatewayHostInformer {
	return &gatewayHostInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// GatewayHostRoutes returns a GatewayHostRouteInformer.
func (v *version) GatewayHostRoutes() GatewayHostRouteInformer {
	return &gatewayHostRouteInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// GlobalConfigs returns a GlobalConfigInformer.
func (v *version) GlobalConfigs() GlobalConfigInformer {
	return &globalConfigInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// HttpFilters returns a HttpFilterInformer.
func (v *version) HttpFilters() HttpFilterInformer {
	return &httpFilterInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PolicyOverlays returns a PolicyOverlayInformer.
func (v *version) PolicyOverlays() PolicyOverlayInformer {
	return &policyOverlayInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// RouteFilters returns a RouteFilterInformer.
func (v *version) RouteFilters() RouteFilterInformer {
	return &routeFilterInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// TLSCertificateDelegations returns a TLSCertificateDelegationInformer.
func (v *version) TLSCertificateDelegations() TLSCertificateDelegationInformer {
	return &tLSCertificateDelegationInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
