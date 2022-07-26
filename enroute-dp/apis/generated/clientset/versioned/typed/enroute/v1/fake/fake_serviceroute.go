// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	enroutev1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServiceRoutes implements ServiceRouteInterface
type FakeServiceRoutes struct {
	Fake *FakeEnrouteV1
	ns   string
}

var serviceroutesResource = schema.GroupVersionResource{Group: "enroute.saaras.io", Version: "v1", Resource: "serviceroutes"}

var serviceroutesKind = schema.GroupVersionKind{Group: "enroute.saaras.io", Version: "v1", Kind: "ServiceRoute"}

// Get takes name of the serviceRoute, and returns the corresponding serviceRoute object, and an error if there is any.
func (c *FakeServiceRoutes) Get(ctx context.Context, name string, options v1.GetOptions) (result *enroutev1.ServiceRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(serviceroutesResource, c.ns, name), &enroutev1.ServiceRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.ServiceRoute), err
}

// List takes label and field selectors, and returns the list of ServiceRoutes that match those selectors.
func (c *FakeServiceRoutes) List(ctx context.Context, opts v1.ListOptions) (result *enroutev1.ServiceRouteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(serviceroutesResource, serviceroutesKind, c.ns, opts), &enroutev1.ServiceRouteList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &enroutev1.ServiceRouteList{ListMeta: obj.(*enroutev1.ServiceRouteList).ListMeta}
	for _, item := range obj.(*enroutev1.ServiceRouteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested serviceRoutes.
func (c *FakeServiceRoutes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(serviceroutesResource, c.ns, opts))

}

// Create takes the representation of a serviceRoute and creates it.  Returns the server's representation of the serviceRoute, and an error, if there is any.
func (c *FakeServiceRoutes) Create(ctx context.Context, serviceRoute *enroutev1.ServiceRoute, opts v1.CreateOptions) (result *enroutev1.ServiceRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(serviceroutesResource, c.ns, serviceRoute), &enroutev1.ServiceRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.ServiceRoute), err
}

// Update takes the representation of a serviceRoute and updates it. Returns the server's representation of the serviceRoute, and an error, if there is any.
func (c *FakeServiceRoutes) Update(ctx context.Context, serviceRoute *enroutev1.ServiceRoute, opts v1.UpdateOptions) (result *enroutev1.ServiceRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(serviceroutesResource, c.ns, serviceRoute), &enroutev1.ServiceRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.ServiceRoute), err
}

// Delete takes name of the serviceRoute and deletes it. Returns an error if one occurs.
func (c *FakeServiceRoutes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(serviceroutesResource, c.ns, name, opts), &enroutev1.ServiceRoute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeServiceRoutes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(serviceroutesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &enroutev1.ServiceRouteList{})
	return err
}

// Patch applies the patch and returns the patched serviceRoute.
func (c *FakeServiceRoutes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *enroutev1.ServiceRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(serviceroutesResource, c.ns, name, pt, data, subresources...), &enroutev1.ServiceRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.ServiceRoute), err
}
