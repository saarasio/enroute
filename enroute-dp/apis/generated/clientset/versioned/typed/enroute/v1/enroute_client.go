// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"net/http"

	v1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type EnrouteV1Interface interface {
	RESTClient() rest.Interface
	GatewayHostsGetter
	GatewayHostRoutesGetter
	GlobalConfigsGetter
	HttpFiltersGetter
	PolicyOverlaysGetter
	RouteFiltersGetter
	TLSCertificateDelegationsGetter
}

// EnrouteV1Client is used to interact with features provided by the enroute.saaras.io group.
type EnrouteV1Client struct {
	restClient rest.Interface
}

func (c *EnrouteV1Client) GatewayHosts(namespace string) GatewayHostInterface {
	return newGatewayHosts(c, namespace)
}

func (c *EnrouteV1Client) GatewayHostRoutes(namespace string) GatewayHostRouteInterface {
	return newGatewayHostRoutes(c, namespace)
}

func (c *EnrouteV1Client) GlobalConfigs(namespace string) GlobalConfigInterface {
	return newGlobalConfigs(c, namespace)
}

func (c *EnrouteV1Client) HttpFilters(namespace string) HttpFilterInterface {
	return newHttpFilters(c, namespace)
}

func (c *EnrouteV1Client) PolicyOverlays(namespace string) PolicyOverlayInterface {
	return newPolicyOverlays(c, namespace)
}

func (c *EnrouteV1Client) RouteFilters(namespace string) RouteFilterInterface {
	return newRouteFilters(c, namespace)
}

func (c *EnrouteV1Client) TLSCertificateDelegations(namespace string) TLSCertificateDelegationInterface {
	return newTLSCertificateDelegations(c, namespace)
}

// NewForConfig creates a new EnrouteV1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*EnrouteV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new EnrouteV1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*EnrouteV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &EnrouteV1Client{client}, nil
}

// NewForConfigOrDie creates a new EnrouteV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *EnrouteV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new EnrouteV1Client for the given RESTClient.
func New(c rest.Interface) *EnrouteV1Client {
	return &EnrouteV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *EnrouteV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
