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

package contour

import (
	"sort"
	"strings"
	"sync"

	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/proto"
	"github.com/saarasio/enroute/enroute-dp/internal/envoy"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scache "k8s.io/client-go/tools/cache"
)

// A EndpointsTranslator translates Kubernetes Endpoints objects into Envoy
// ClusterLoadAssignment objects.
type EndpointsTranslator struct {
	logrus.FieldLogger
	clusterLoadAssignmentCache
	Cond
}

func (e *EndpointsTranslator) OnAdd(obj interface{}, isInInitialList bool) {
	switch obj := obj.(type) {
	case *v1.Endpoints:
		e.addEndpoints(obj)
	default:
		e.Errorf("OnAdd unexpected type %T: %#v", obj, obj)
	}
}

func (e *EndpointsTranslator) OnUpdate(oldObj, newObj interface{}) {
	switch newObj := newObj.(type) {
	case *v1.Endpoints:
		oldObj, ok := oldObj.(*v1.Endpoints)
		if !ok {
			e.Errorf("OnUpdate endpoints %#v received invalid oldObj %T; %#v", newObj, oldObj, oldObj)
			return
		}
		e.updateEndpoints(oldObj, newObj)
	default:
		e.Errorf("OnUpdate unexpected type %T: %#v", newObj, newObj)
	}
}

func (e *EndpointsTranslator) OnDelete(obj interface{}) {
	switch obj := obj.(type) {
	case *v1.Endpoints:
		e.removeEndpoints(obj)
	case k8scache.DeletedFinalStateUnknown:
		e.OnDelete(obj.Obj) // recurse into ourselves with the tombstoned value
	default:
		e.Errorf("OnDelete unexpected type %T: %#v", obj, obj)
	}
}

func (e *EndpointsTranslator) Contents() []proto.Message {
	values := e.clusterLoadAssignmentCache.Contents()
	sort.Stable(clusterLoadAssignmentsByName(values))
	return values
}

func (e *EndpointsTranslator) Query(names []string) []proto.Message {
	e.clusterLoadAssignmentCache.mu.Lock()
	defer e.clusterLoadAssignmentCache.mu.Unlock()
	var values []proto.Message
	for _, n := range names {
		v, ok := e.entries[n]
		if !ok {
			v = &envoy_config_endpoint_v3.ClusterLoadAssignment{
				ClusterName: n,
			}
		}
		values = append(values, v)
	}
	sort.Stable(clusterLoadAssignmentsByName(values))
	return values
}

type clusterLoadAssignmentsByName []proto.Message

func (c clusterLoadAssignmentsByName) Len() int      { return len(c) }
func (c clusterLoadAssignmentsByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c clusterLoadAssignmentsByName) Less(i, j int) bool {
	return c[i].(*envoy_config_endpoint_v3.ClusterLoadAssignment).ClusterName < c[j].(*envoy_config_endpoint_v3.ClusterLoadAssignment).ClusterName
}

func (*EndpointsTranslator) TypeURL() string { return resource.EndpointType }

func (e *EndpointsTranslator) addEndpoints(ep *v1.Endpoints) {
	e.recomputeClusterLoadAssignment(nil, ep)
}

func (e *EndpointsTranslator) updateEndpoints(oldep, newep *v1.Endpoints) {
	if len(newep.Subsets) == 0 && len(oldep.Subsets) == 0 {
		// if there are no endpoints in this object, and the old
		// object also had zero endpoints, ignore this update
		// to avoid sending a noop notification to watchers.
		return
	}
	e.recomputeClusterLoadAssignment(oldep, newep)
}

func (e *EndpointsTranslator) removeEndpoints(ep *v1.Endpoints) {
	e.recomputeClusterLoadAssignment(ep, nil)
}

// recomputeClusterLoadAssignment recomputes the EDS cache taking into account old and new endpoints.
func (e *EndpointsTranslator) recomputeClusterLoadAssignment(oldep, newep *v1.Endpoints) {
	// skip computation if either old and new services or endpoints are equal (thus also handling nil)
	if oldep == newep {
		return
	}

	defer e.Notify()

	if oldep == nil {
		oldep = &v1.Endpoints{
			ObjectMeta: newep.ObjectMeta,
		}
	}

	if newep == nil {
		newep = &v1.Endpoints{
			ObjectMeta: oldep.ObjectMeta,
		}
	}

	clas := make(map[string]*envoy_config_endpoint_v3.ClusterLoadAssignment)
	// add or update endpoints
	for _, s := range newep.Subsets {
		// skip any subsets that don't have ready addresses
		if len(s.Addresses) == 0 {
			continue
		}

		for _, p := range s.Ports {
			// TODO(dfc) check protocol, don't add UDP enties by mistake

			// if this endpoint's service's port has a name, then the endpoint
			// controller will apply the name here. The name may appear once per subset.
			portname := p.Name
			cla, ok := clas[portname]
			if !ok {
				cla = &envoy_config_endpoint_v3.ClusterLoadAssignment{
					ClusterName: servicename(newep.ObjectMeta, portname),
					Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{{
						LbEndpoints: make([]*envoy_config_endpoint_v3.LbEndpoint, 0, 1),
					}},
				}
				clas[portname] = cla
			}
			for _, a := range s.Addresses {
				addr := envoy.SocketAddress(a.IP, int(p.Port))
				cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, envoy.LBEndpoint(addr))
			}
		}
	}

	// iterate all the defined clusters and add or update them.
	for _, a := range clas {
		e.Add(a)
	}

	// iterate over the ports in the old spec, remove any that are not
	// mentioned in clas
	for _, s := range oldep.Subsets {
		if len(s.Addresses) == 0 {
			continue
		}
		for _, p := range s.Ports {
			// if this endpoint's service's port has a name, then the endpoint
			// controller will apply the name here. The name may appear once per subset.
			portname := p.Name
			if _, ok := clas[portname]; !ok {
				// port is not present in the list added / updated, so remove it
				e.Remove(servicename(oldep.ObjectMeta, portname))
			}
		}
	}
}

type clusterLoadAssignmentCache struct {
	mu      sync.Mutex
	entries map[string]*envoy_config_endpoint_v3.ClusterLoadAssignment
}

// Add adds an entry to the cache. If a ClusterLoadAssignment with the same
// name exists, it is replaced.
func (c *clusterLoadAssignmentCache) Add(a *envoy_config_endpoint_v3.ClusterLoadAssignment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]*envoy_config_endpoint_v3.ClusterLoadAssignment)
	}
	c.entries[a.ClusterName] = a
}

// Remove removes the named entry from the cache. If the entry
// is not present in the cache, the operation is a no-op.
func (c *clusterLoadAssignmentCache) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, name)
}

// Contents returns a copy of the contents of the cache.
func (c *clusterLoadAssignmentCache) Contents() []proto.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	var values []proto.Message
	for _, v := range c.entries {
		values = append(values, v)
	}
	return values
}

// servicename returns the name of the cluster this meta and port
// refers to. The CDS name of the cluster may include additional suffixes
// but these are not known to EDS.
func servicename(meta metav1.ObjectMeta, portname string) string {
	name := []string{
		meta.Namespace,
		meta.Name,
		portname,
	}
	if portname == "" {
		name = name[:2]
	}
	return strings.Join(name, "/")
}
