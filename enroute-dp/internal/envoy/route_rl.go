package envoy

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v31 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

func rateLimitActionSpecifierHeaderValueMatch() *envoy_config_route_v3.RateLimit_Action_HeaderValueMatch_ {
	return &envoy_config_route_v3.RateLimit_Action_HeaderValueMatch_{
		HeaderValueMatch: &envoy_config_route_v3.RateLimit_Action_HeaderValueMatch{
			DescriptorValue: "x",
			Headers:         nil,
		},
	}
}

func rateLimitActionSpecifierRequestHeaders(header_name, descriptor_key string) *envoy_config_route_v3.RateLimit_Action_RequestHeaders_ {
	return &envoy_config_route_v3.RateLimit_Action_RequestHeaders_{
		RequestHeaders: &envoy_config_route_v3.RateLimit_Action_RequestHeaders{
			HeaderName:    header_name,
			DescriptorKey: descriptor_key,
		},
	}
}

func rateLimitActionRequestHeaders(header_name, descriptor_key string) *envoy_config_route_v3.RateLimit_Action {
	return &envoy_config_route_v3.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierRequestHeaders(header_name, descriptor_key),
	}
}

func rateLimitActionSpecifierRemoteAddress() *envoy_config_route_v3.RateLimit_Action_RemoteAddress_ {
	return &envoy_config_route_v3.RateLimit_Action_RemoteAddress_{
		RemoteAddress: &envoy_config_route_v3.RateLimit_Action_RemoteAddress{},
	}
}

func rateLimitActionRemoteAddress() *envoy_config_route_v3.RateLimit_Action {
	return &envoy_config_route_v3.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierRemoteAddress(),
	}
}

func rateLimitActionSpecifierSourceCluster() *envoy_config_route_v3.RateLimit_Action_SourceCluster_ {
	return &envoy_config_route_v3.RateLimit_Action_SourceCluster_{
		SourceCluster: &envoy_config_route_v3.RateLimit_Action_SourceCluster{},
	}
}

func rateLimitActionSourceCluster() *envoy_config_route_v3.RateLimit_Action {
	return &envoy_config_route_v3.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierSourceCluster(),
	}
}

func rateLimitActionSpecifierDestinationCluster() *envoy_config_route_v3.RateLimit_Action_DestinationCluster_ {
	return &envoy_config_route_v3.RateLimit_Action_DestinationCluster_{
		DestinationCluster: &envoy_config_route_v3.RateLimit_Action_DestinationCluster{},
	}
}

func rateLimitActionDestinationCluster() *envoy_config_route_v3.RateLimit_Action {
	return &envoy_config_route_v3.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierDestinationCluster(),
	}
}

func rateLimitActionSpecifierGenericKey(descriptor_value string) *envoy_config_route_v3.RateLimit_Action_GenericKey_ {
	return &envoy_config_route_v3.RateLimit_Action_GenericKey_{
		GenericKey: &envoy_config_route_v3.RateLimit_Action_GenericKey{
			DescriptorValue: descriptor_value,
		},
	}
}

func rateLimitActionGenericKey(descriptor_value string) *envoy_config_route_v3.RateLimit_Action {
	return &envoy_config_route_v3.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierGenericKey(descriptor_value),
	}
}

func rateLimits(rl_filters []*dag.RouteFilter) []*envoy_config_route_v3.RateLimit {

	var rad saarasconfig.RouteActionDescriptors
	var err error
	var rla []*envoy_config_route_v3.RateLimit_Action
	var rrl_slice []*envoy_config_route_v3.RateLimit

	rrl_slice = make([]*envoy_config_route_v3.RateLimit, 0)

	for _, f := range rl_filters {
		if f == nil {
			continue
		}
		// f is of type cfg.SaarasRouteFilter
		// TODO: Fix the switch on filter type, we only parse rate-limit config if filter type is rate-limit
		switch f.Filter.Filter_type {
		case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
			rad, err = saarasconfig.UnmarshalRateLimitRouteFilterConfig(f.Filter.Filter_config)
			if err != nil {
				return nil
			}

			for _, oneRouteActionDescriptor := range rad.Descriptors {
				if oneRouteActionDescriptor.GenericKey != nil &&
					len(oneRouteActionDescriptor.GenericKey.DescriptorValue) > 0 {

					rla = append(rla,
						rateLimitActionGenericKey(oneRouteActionDescriptor.GenericKey.DescriptorValue))

				} else if oneRouteActionDescriptor.RequestHeaders != nil &&
					len(oneRouteActionDescriptor.RequestHeaders.HeaderName) > 0 {

					rla = append(rla,
						rateLimitActionRequestHeaders(
							oneRouteActionDescriptor.RequestHeaders.HeaderName,
							oneRouteActionDescriptor.RequestHeaders.DescriptorKey))

				} else if len(oneRouteActionDescriptor.SourceCluster) > 0 {
					rla = append(rla, rateLimitActionSourceCluster())
				} else if len(oneRouteActionDescriptor.DestinationCluster) > 0 {
					rla = append(rla, rateLimitActionDestinationCluster())
				} else if len(oneRouteActionDescriptor.RemoteAddress) > 0 {
					rla = append(rla, rateLimitActionRemoteAddress())
				}
			}

			rrl := envoy_config_route_v3.RateLimit{
				Stage:   u32nil(0),
				Actions: rla,
			}

			rrl_slice = append(rrl_slice, &rrl)
		default:
		}
	}

	return rrl_slice
}

func dagToEnvoyRateLimit(f *dag.RouteFilter, ra *envoy_config_route_v3.RouteAction) {
	var rad saarasconfig.RouteActionDescriptors
	var err error
	var rla []*envoy_config_route_v3.RateLimit_Action
	var rrl_slice []*envoy_config_route_v3.RateLimit

	rrl_slice = make([]*envoy_config_route_v3.RateLimit, 0)

	rad, err = saarasconfig.UnmarshalRateLimitRouteFilterConfig(f.Filter.Filter_config)
	if err != nil {
		return
	}

	for _, oneRouteActionDescriptor := range rad.Descriptors {
		if oneRouteActionDescriptor.GenericKey != nil &&
			len(oneRouteActionDescriptor.GenericKey.DescriptorValue) > 0 {

			rla = append(rla,
				rateLimitActionGenericKey(oneRouteActionDescriptor.GenericKey.DescriptorValue))

		} else if oneRouteActionDescriptor.RequestHeaders != nil &&
			len(oneRouteActionDescriptor.RequestHeaders.HeaderName) > 0 {

			rla = append(rla,
				rateLimitActionRequestHeaders(
					oneRouteActionDescriptor.RequestHeaders.HeaderName,
					oneRouteActionDescriptor.RequestHeaders.DescriptorKey))

		} else if len(oneRouteActionDescriptor.SourceCluster) > 0 {
			rla = append(rla, rateLimitActionSourceCluster())
		} else if len(oneRouteActionDescriptor.DestinationCluster) > 0 {
			rla = append(rla, rateLimitActionDestinationCluster())
		} else if len(oneRouteActionDescriptor.RemoteAddress) > 0 {
			rla = append(rla, rateLimitActionRemoteAddress())
		}
	}

	rrl := envoy_config_route_v3.RateLimit{
		Stage:   u32nil(0),
		Actions: rla,
	}

	rrl_slice = append(rrl_slice, &rrl)
	ra.RateLimits = rrl_slice
}

func dagToEnvoyHostRewrite(f *dag.RouteFilter, ra *envoy_config_route_v3.RouteAction) {
	// Host Rewrite takes several forms -
	//
	//    *RouteAction_HostRewriteLiteral
	//    *RouteAction_AutoHostRewrite
	//    *RouteAction_HostRewriteHeader
	//    *RouteAction_HostRewritePathRegex

	// The current code handles RouteAction_HostRewriteLiteral, RouteAction_HostRewritePathRegex
	hrc, err := saarasconfig.UnmarshalHostRewriteConfig(f.Filter.Filter_config)

	if err != nil {
		return
	}

	if len(hrc.Pattern_regex) > 0 {
		// RouteAction_HostRewritePathRegex
		ra.HostRewriteSpecifier = &envoy_config_route_v3.RouteAction_HostRewritePathRegex{
			HostRewritePathRegex: &v31.RegexMatchAndSubstitute{
				Pattern: &v31.RegexMatcher{
					EngineType: &v31.RegexMatcher_GoogleRe2{},
					Regex:      hrc.Pattern_regex,
				},
				Substitution: hrc.Substitution,
			},
		}
	} else {
		// RouteAction_HostRewriteLiteral
		ra.HostRewriteSpecifier = &envoy_config_route_v3.RouteAction_HostRewriteLiteral{
			HostRewriteLiteral: hrc.Substitution,
		}
	}
}

func SetupRouteRateLimits(r *dag.Route, ra *envoy_config_route_v3.RouteAction) {
	if r.RouteFilters != nil {
		ra.RateLimits = rateLimits(r.RouteFilters)
	}
}

func ProcessRouteFilters(r *dag.Route, ra *envoy_config_route_v3.RouteAction) {
	for _, f := range r.RouteFilters {

		switch f.Filter.Filter_type {
		case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
			dagToEnvoyRateLimit(f, ra)
		case saarasconfig.FILTER_TYPE_RT_HOST_REWRITE:
			dagToEnvoyHostRewrite(f, ra)
		default:
		}
	}
}
