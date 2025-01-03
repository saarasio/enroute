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

// FakePolicyOverlays implements PolicyOverlayInterface
type FakePolicyOverlays struct {
	Fake *FakeEnrouteV1
	ns   string
}

var policyoverlaysResource = schema.GroupVersionResource{Group: "enroute.saaras.io", Version: "v1", Resource: "policyoverlays"}

var policyoverlaysKind = schema.GroupVersionKind{Group: "enroute.saaras.io", Version: "v1", Kind: "PolicyOverlay"}

// Get takes name of the policyOverlay, and returns the corresponding policyOverlay object, and an error if there is any.
func (c *FakePolicyOverlays) Get(ctx context.Context, name string, options v1.GetOptions) (result *enroutev1.PolicyOverlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(policyoverlaysResource, c.ns, name), &enroutev1.PolicyOverlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.PolicyOverlay), err
}

// List takes label and field selectors, and returns the list of PolicyOverlays that match those selectors.
func (c *FakePolicyOverlays) List(ctx context.Context, opts v1.ListOptions) (result *enroutev1.PolicyOverlayList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(policyoverlaysResource, policyoverlaysKind, c.ns, opts), &enroutev1.PolicyOverlayList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &enroutev1.PolicyOverlayList{ListMeta: obj.(*enroutev1.PolicyOverlayList).ListMeta}
	for _, item := range obj.(*enroutev1.PolicyOverlayList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested policyOverlays.
func (c *FakePolicyOverlays) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(policyoverlaysResource, c.ns, opts))

}

// Create takes the representation of a policyOverlay and creates it.  Returns the server's representation of the policyOverlay, and an error, if there is any.
func (c *FakePolicyOverlays) Create(ctx context.Context, policyOverlay *enroutev1.PolicyOverlay, opts v1.CreateOptions) (result *enroutev1.PolicyOverlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(policyoverlaysResource, c.ns, policyOverlay), &enroutev1.PolicyOverlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.PolicyOverlay), err
}

// Update takes the representation of a policyOverlay and updates it. Returns the server's representation of the policyOverlay, and an error, if there is any.
func (c *FakePolicyOverlays) Update(ctx context.Context, policyOverlay *enroutev1.PolicyOverlay, opts v1.UpdateOptions) (result *enroutev1.PolicyOverlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(policyoverlaysResource, c.ns, policyOverlay), &enroutev1.PolicyOverlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.PolicyOverlay), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePolicyOverlays) UpdateStatus(ctx context.Context, policyOverlay *enroutev1.PolicyOverlay, opts v1.UpdateOptions) (*enroutev1.PolicyOverlay, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(policyoverlaysResource, "status", c.ns, policyOverlay), &enroutev1.PolicyOverlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.PolicyOverlay), err
}

// Delete takes name of the policyOverlay and deletes it. Returns an error if one occurs.
func (c *FakePolicyOverlays) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(policyoverlaysResource, c.ns, name, opts), &enroutev1.PolicyOverlay{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePolicyOverlays) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(policyoverlaysResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &enroutev1.PolicyOverlayList{})
	return err
}

// Patch applies the patch and returns the patched policyOverlay.
func (c *FakePolicyOverlays) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *enroutev1.PolicyOverlay, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(policyoverlaysResource, c.ns, name, pt, data, subresources...), &enroutev1.PolicyOverlay{})

	if obj == nil {
		return nil, err
	}
	return obj.(*enroutev1.PolicyOverlay), err
}
