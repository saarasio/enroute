// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package sync

import (
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/saarasio/enroute/enroutectl/config"
	"sort"
	"strconv"
)

type EnrouteSync struct {
	EnrouteCtlUUID string
	EnDrv
}

// Add Auriga Service, Route and Upstream to Enroute
func (s *EnrouteSync) addServiceTree(svc config.ProxyServices) *EnStatus {

	enstatus := &EnStatus{}
	ps := PostServiceArg{
		Service_Name: svc.Service.ServiceName,
		Fqdn:         svc.Service.Fqdn,
	}

	s.EnrouteCreateService(&ps)

	for _, svc_rt := range svc.Service.Routes {
		pr := PostRouteArg{
			Route_Name:   svc_rt.RouteName,
			Route_Prefix: svc_rt.RoutePrefix,
			Route_Config: svc_rt.RouteConfig,
		}

		s.EnrouteCreateRoute(ps.Service_Name, &pr)

		for _, svc_rt_u := range svc_rt.RouteUpstreams {

			// Create upstream
			pu := PostUpstreamArg{
				Upstream_name:    svc_rt_u.Upstream.UpstreamName,
				Upstream_ip:      svc_rt_u.Upstream.UpstreamIP,
				Upstream_port:    strconv.Itoa(svc_rt_u.Upstream.UpstreamPort),
				Upstream_weight:  "100",
				Upstream_hc_path: svc_rt_u.Upstream.UpstreamHcPath,
			}

			enstatus = s.EnrouteAddUpstream(&pu)

			// Associate upstream
			s.EnrouteAssociateUpstreamToServiceRoute(svc.Service.ServiceName, svc_rt.RouteName, svc_rt_u.Upstream.UpstreamName)
		}
	}

	return enstatus
}

func (s *EnrouteSync) SyncAddMissingUpstream(au_svc *config.ProxyServices, au_svc_route, en_svc_route *config.Routes) *EnStatus {

	enstatus := &EnStatus{}
	for _, au_svc_rt_u := range au_svc_route.RouteUpstreams {
		found := false
		for _, en_svc_rt_u := range en_svc_route.RouteUpstreams {
			if au_svc_rt_u.Upstream.UpstreamName == en_svc_rt_u.Upstream.UpstreamName {
				found = true
			}
		}

		if !found {
			pu := PostUpstreamArg{
				Upstream_name:    au_svc_rt_u.Upstream.UpstreamName,
				Upstream_ip:      au_svc_rt_u.Upstream.UpstreamIP,
				Upstream_port:    strconv.Itoa(au_svc_rt_u.Upstream.UpstreamPort),
				Upstream_weight:  "100",
				Upstream_hc_path: au_svc_rt_u.Upstream.UpstreamHcPath,
			}

			// TODO: Collect status
			// This will just fail if it already exists
			// API call to create upstream is idempotent
			enstatus = s.EnrouteAddUpstream(&pu)
			s.EnrouteAssociateUpstreamToServiceRoute(au_svc.Service.ServiceName, au_svc_route.RouteName, au_svc_rt_u.Upstream.UpstreamName)
		}
	}

	return enstatus
}

func (s *EnrouteSync) SyncAddMissingRouteTree(from_svc, to_svc *config.ProxyServices) *EnStatus {
	enstatus := &EnStatus{}
	for _, from_svc_rt := range from_svc.Service.Routes {
		found := false
		for _, to_svc_rt := range to_svc.Service.Routes {
			if to_svc_rt.RouteName == from_svc_rt.RouteName {
				found = true
			}
		}

		// If route not found, add route and upstreams and associate
		if !found {
			rarg := PostRouteArg{
				Route_Name:   from_svc_rt.RouteName,
				Route_Prefix: from_svc_rt.RoutePrefix,
				Route_Config: from_svc_rt.RouteConfig,
			}
			s.EnrouteCreateRoute(to_svc.Service.ServiceName, &rarg)

			for _, from_svc_rt_u := range from_svc_rt.RouteUpstreams {
				pu := PostUpstreamArg{
					Upstream_name:    from_svc_rt_u.Upstream.UpstreamName,
					Upstream_ip:      from_svc_rt_u.Upstream.UpstreamIP,
					Upstream_port:    strconv.Itoa(from_svc_rt_u.Upstream.UpstreamPort),
					Upstream_weight:  "100",
					Upstream_hc_path: from_svc_rt_u.Upstream.UpstreamHcPath,
				}

				// TODO: Collect status
				enstatus = s.EnrouteAddUpstream(&pu)
				s.EnrouteAssociateUpstreamToServiceRoute(to_svc.Service.ServiceName, from_svc_rt.RouteName, from_svc_rt_u.Upstream.UpstreamName)
			}
		}
	}

	return enstatus
}

func (s *EnrouteSync) SyncProxyServicesAdd(services_to, services_from []config.ProxyServices) {
	for _, from_service := range services_from {
		found := false
		for _, to_svc := range services_to {
			if from_service.Service.ServiceName == to_svc.Service.ServiceName {
				found = true
			}
		}

		if !found {
			s.addServiceTree(from_service)
		}
	}
}

// TODO: Needs test
func routeMatchConditionsEqual(ra_mc, rb_mc saarasconfig.RouteMatchConditions) bool {
	if ra_mc.Prefix == rb_mc.Prefix {
		sort.Stable(saarasconfig.RouteMatchConditionsByHeaderNameVal(ra_mc.MatchConditions))
		sort.Stable(saarasconfig.RouteMatchConditionsByHeaderNameVal(rb_mc.MatchConditions))

		ra_cond := ra_mc.MatchConditions
		rb_cond := rb_mc.MatchConditions

		if len(ra_cond) == len(rb_cond) {
			for idx, c := range ra_cond {
				if c.Name != rb_cond[idx].Name {
					return false
				}
				if c.Exact != rb_cond[idx].Exact {
					return false
				}
			}
		}
	}
	return true
}

// TODO: Needs test
func routesEqual(ra, rb *config.Routes) bool {
	if ra.RouteName == rb.RouteName {

		if len(ra.RoutePrefix) > 0 && len(rb.RoutePrefix) == len(ra.RoutePrefix) && ra.RoutePrefix == rb.RoutePrefix {
			return true
		}

		ra_mc, err1 := saarasconfig.UnmarshalRouteMatchCondition(ra.RouteConfig)
		rb_mc, err2 := saarasconfig.UnmarshalRouteMatchCondition(ra.RouteConfig)

		if err1 != nil || err2 != nil {
			return false
		}

		return routeMatchConditionsEqual(ra_mc, rb_mc)
	}

	return false
}

func (s *EnrouteSync) SyncProxyAdd(cfg_to, cfg_from *config.EnrouteConfig) {

	var svc_to, svc_from []config.ProxyServices

	if cfg_from == nil {
		// Nothing to add
		return
	}

	if cfg_to != nil &&
		cfg_to.Data.SaarasDbProxy != nil &&
		len(cfg_to.Data.SaarasDbProxy) > 0 &&
		cfg_to.Data.SaarasDbProxy[0].ProxyServices != nil {
		svc_to = cfg_to.Data.SaarasDbProxy[0].ProxyServices
	} else {

		// Create Proxy, idempotent if already present
		p := PostProxyArg{
			Name: "gw",
		}
		s.EnrouteCreateProxy(&p)

		// Create an empty service list
		svc_to = make([]config.ProxyServices, 0)
	}

	if len(cfg_from.Data.SaarasDbProxy) > 0 {
		svc_from = cfg_from.Data.SaarasDbProxy[0].ProxyServices
	}

	sort.Stable(config.ServicesByName(svc_to))
	sort.Stable(config.ServicesByName(svc_from))

	s.SyncProxyServicesAdd(svc_to, svc_from)

	// for _, f_svc := range svc_from {
	//     for _, t_svc := range svc_to {
	//         if f_svc.Service.ServiceName == t_svc.Service.ServiceName {
	//             s.SyncAddMissingRouteTree(&t_svc, &f_svc)

	//             // Routes synced, check for upstreams
	//             if len(f_svc.Service.Routes) > 0 {
	//                 for _, t_svc_route := range t_svc.Service.Routes {
	//                     for _, f_svc_route := range f_svc.Service.Routes {

	//                         if routesEqual(&t_svc_route, &f_svc_route) {
	//                             // Matching routes, check for missing upstreams and add
	//                             s.SyncAddMissingUpstream(&f_svc, &f_svc_route, &t_svc_route)
	//                         }
	//                     }
	//                 }
	//             }

	//         }
	//     }
	// }
}

func (s *EnrouteSync) Sync(to, from *config.EnrouteConfig) {
	s.SyncProxyAdd(to, from)
	//s.SyncProxyDelete(to, from)
	//s.SyncProxyUpdate(to, from)
}
