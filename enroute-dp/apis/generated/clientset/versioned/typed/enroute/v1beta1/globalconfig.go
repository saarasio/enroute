// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2021 Saaras Inc.

/*
Copyright 2019  Heptio

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1beta1

import (
	"context"
	"time"

	v1beta1 "github.com/saarasio/enroute/enroute-dp/apis/enroute/v1beta1"
	scheme "github.com/saarasio/enroute/enroute-dp/apis/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GlobalConfigsGetter has a method to return a GlobalConfigInterface.
// A group's client should implement this interface.
type GlobalConfigsGetter interface {
	GlobalConfigs(namespace string) GlobalConfigInterface
}

// GlobalConfigInterface has methods to work with GlobalConfig resources.
type GlobalConfigInterface interface {
	Create(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.CreateOptions) (*v1beta1.GlobalConfig, error)
	Update(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.UpdateOptions) (*v1beta1.GlobalConfig, error)
	UpdateStatus(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.UpdateOptions) (*v1beta1.GlobalConfig, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.GlobalConfig, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta1.GlobalConfigList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.GlobalConfig, err error)
	GlobalConfigExpansion
}

// globalConfigs implements GlobalConfigInterface
type globalConfigs struct {
	client rest.Interface
	ns     string
}

// newGlobalConfigs returns a GlobalConfigs
func newGlobalConfigs(c *EnrouteV1beta1Client, namespace string) *globalConfigs {
	return &globalConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the globalConfig, and returns the corresponding globalConfig object, and an error if there is any.
func (c *globalConfigs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.GlobalConfig, err error) {
	result = &v1beta1.GlobalConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("globalconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalConfigs that match those selectors.
func (c *globalConfigs) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.GlobalConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta1.GlobalConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("globalconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalConfigs.
func (c *globalConfigs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("globalconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a globalConfig and creates it.  Returns the server's representation of the globalConfig, and an error, if there is any.
func (c *globalConfigs) Create(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.CreateOptions) (result *v1beta1.GlobalConfig, err error) {
	result = &v1beta1.GlobalConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("globalconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalConfig).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a globalConfig and updates it. Returns the server's representation of the globalConfig, and an error, if there is any.
func (c *globalConfigs) Update(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.UpdateOptions) (result *v1beta1.GlobalConfig, err error) {
	result = &v1beta1.GlobalConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("globalconfigs").
		Name(globalConfig.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalConfig).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *globalConfigs) UpdateStatus(ctx context.Context, globalConfig *v1beta1.GlobalConfig, opts v1.UpdateOptions) (result *v1beta1.GlobalConfig, err error) {
	result = &v1beta1.GlobalConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("globalconfigs").
		Name(globalConfig.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalConfig).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the globalConfig and deletes it. Returns an error if one occurs.
func (c *globalConfigs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("globalconfigs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalConfigs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("globalconfigs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched globalConfig.
func (c *globalConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.GlobalConfig, err error) {
	result = &v1beta1.GlobalConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("globalconfigs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
