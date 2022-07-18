// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	enroutev1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	versioned "github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned"
	internalinterfaces "github.com/saarasio/enroute/enroute-dp/apis/generated/informers/externalversions/internalinterfaces"
	v1 "github.com/saarasio/enroute/enroute-dp/apis/generated/listers/enroute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// GatewayHostRouteInformer provides access to a shared informer and lister for
// GatewayHostRoutes.
type GatewayHostRouteInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.GatewayHostRouteLister
}

type gatewayHostRouteInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewGatewayHostRouteInformer constructs a new informer for GatewayHostRoute type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewGatewayHostRouteInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredGatewayHostRouteInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredGatewayHostRouteInformer constructs a new informer for GatewayHostRoute type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredGatewayHostRouteInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EnrouteV1().GatewayHostRoutes(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EnrouteV1().GatewayHostRoutes(namespace).Watch(context.TODO(), options)
			},
		},
		&enroutev1.GatewayHostRoute{},
		resyncPeriod,
		indexers,
	)
}

func (f *gatewayHostRouteInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredGatewayHostRouteInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *gatewayHostRouteInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&enroutev1.GatewayHostRoute{}, f.defaultInformer)
}

func (f *gatewayHostRouteInformer) Lister() v1.GatewayHostRouteLister {
	return v1.NewGatewayHostRouteLister(f.Informer().GetIndexer())
}
