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

// FakeHttpFilters implements HttpFilterInterface
type FakeHttpFilters struct {
	Fake *FakeEnrouteV1
	ns   string
}

var httpfiltersResource = schema.GroupVersionResource{Group: "enroute.saaras.io", Version: "v1", Resource: "httpfilters"}

var httpfiltersKind = schema.GroupVersionKind{Group: "enroute.saaras.io", Version: "v1", Kind: "HttpFilter"}

// Get takes name of the httpFilter, and returns the corresponding httpFilter object, and an error if there is any.
func (c *FakeHttpFilters) Get(ctx context.Context, name string, options v1.GetOptions) (result *enroutev1.HttpFilter, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(httpfiltersResource, c.ns, name), &enroutev1.HttpFilter{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.HttpFilter), err
}

// List takes label and field selectors, and returns the list of HttpFilters that match those selectors.
func (c *FakeHttpFilters) List(ctx context.Context, opts v1.ListOptions) (result *enroutev1.HttpFilterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(httpfiltersResource, httpfiltersKind, c.ns, opts), &enroutev1.HttpFilterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &enroutev1.HttpFilterList{ListMeta: obj.(*enroutev1.HttpFilterList).ListMeta}
	for _, item := range obj.(*enroutev1.HttpFilterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested httpFilters.
func (c *FakeHttpFilters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(httpfiltersResource, c.ns, opts))

}

// Create takes the representation of a httpFilter and creates it.  Returns the server's representation of the httpFilter, and an error, if there is any.
func (c *FakeHttpFilters) Create(ctx context.Context, httpFilter *enroutev1.HttpFilter, opts v1.CreateOptions) (result *enroutev1.HttpFilter, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(httpfiltersResource, c.ns, httpFilter), &enroutev1.HttpFilter{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.HttpFilter), err
}

// Update takes the representation of a httpFilter and updates it. Returns the server's representation of the httpFilter, and an error, if there is any.
func (c *FakeHttpFilters) Update(ctx context.Context, httpFilter *enroutev1.HttpFilter, opts v1.UpdateOptions) (result *enroutev1.HttpFilter, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(httpfiltersResource, c.ns, httpFilter), &enroutev1.HttpFilter{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.HttpFilter), err
}

// Delete takes name of the httpFilter and deletes it. Returns an error if one occurs.
func (c *FakeHttpFilters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(httpfiltersResource, c.ns, name, opts), &enroutev1.HttpFilter{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHttpFilters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(httpfiltersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &enroutev1.HttpFilterList{})
	return err
}

// Patch applies the patch and returns the patched httpFilter.
func (c *FakeHttpFilters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *enroutev1.HttpFilter, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(httpfiltersResource, c.ns, name, pt, data, subresources...), &enroutev1.HttpFilter{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.HttpFilter), err
}
