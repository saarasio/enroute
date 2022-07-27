// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServiceRouteLister helps list ServiceRoutes.
// All objects returned here must be treated as read-only.
type ServiceRouteLister interface {
	// List lists all ServiceRoutes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ServiceRoute, err error)
	// ServiceRoutes returns an object that can list and get ServiceRoutes.
	ServiceRoutes(namespace string) ServiceRouteNamespaceLister
	ServiceRouteListerExpansion
}

// serviceRouteLister implements the ServiceRouteLister interface.
type serviceRouteLister struct {
	indexer cache.Indexer
}

// NewServiceRouteLister returns a new ServiceRouteLister.
func NewServiceRouteLister(indexer cache.Indexer) ServiceRouteLister {
	return &serviceRouteLister{indexer: indexer}
}

// List lists all ServiceRoutes in the indexer.
func (s *serviceRouteLister) List(selector labels.Selector) (ret []*v1.ServiceRoute, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ServiceRoute))
	})
	return ret, err
}

// ServiceRoutes returns an object that can list and get ServiceRoutes.
func (s *serviceRouteLister) ServiceRoutes(namespace string) ServiceRouteNamespaceLister {
	return serviceRouteNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServiceRouteNamespaceLister helps list and get ServiceRoutes.
// All objects returned here must be treated as read-only.
type ServiceRouteNamespaceLister interface {
	// List lists all ServiceRoutes in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ServiceRoute, err error)
	// Get retrieves the ServiceRoute from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.ServiceRoute, error)
	ServiceRouteNamespaceListerExpansion
}

// serviceRouteNamespaceLister implements the ServiceRouteNamespaceLister
// interface.
type serviceRouteNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServiceRoutes in the indexer for a given namespace.
func (s serviceRouteNamespaceLister) List(selector labels.Selector) (ret []*v1.ServiceRoute, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ServiceRoute))
	})
	return ret, err
}

// Get retrieves the ServiceRoute from the indexer for a given namespace and name.
func (s serviceRouteNamespaceLister) Get(name string) (*v1.ServiceRoute, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("serviceroute"), name)
	}
	return obj.(*v1.ServiceRoute), nil
}