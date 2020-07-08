package envoy

import (
	// "fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	//ratelimithttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rate_limit/v2"
)

func rateLimitActionSpecifierHeaderValueMatch() *route.RateLimit_Action_HeaderValueMatch_ {
	return &route.RateLimit_Action_HeaderValueMatch_{
		HeaderValueMatch: &route.RateLimit_Action_HeaderValueMatch{
			DescriptorValue: "x",
			Headers:         nil,
		},
	}
}

//func rateLimitActionSpecifierRequestHeaders(header_name, descriptor_key string) *route.RateLimit_Action_RequestHeaders_ {
//	return &route.RateLimit_Action_RequestHeaders_{
//		RequestHeaders: &route.RateLimit_Action_RequestHeaders{
//			HeaderName:    "user-agent",
//			DescriptorKey: "useragent",
//		},
//	}
//}
//func rateLimitActionSpecifierGenericKey(generic_key string) *route.RateLimit_Action_GenericKey_ {
//	return &route.RateLimit_Action_GenericKey_{
//		GenericKey: &route.RateLimit_Action_GenericKey{
//			DescriptorValue: "default",
//		},
//	}
//}

func rateLimitActionSpecifierRequestHeaders(header_name, descriptor_key string) *route.RateLimit_Action_RequestHeaders_ {
	return &route.RateLimit_Action_RequestHeaders_{
		RequestHeaders: &route.RateLimit_Action_RequestHeaders{
			HeaderName:    header_name,
			DescriptorKey: descriptor_key,
		},
	}
}

func rateLimitActionRequestHeaders(header_name, descriptor_key string) *route.RateLimit_Action {
	return &route.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierRequestHeaders(header_name, descriptor_key),
	}
}

func rateLimitActionSpecifierRemoteAddress() *route.RateLimit_Action_RemoteAddress_ {
	return &route.RateLimit_Action_RemoteAddress_{
		RemoteAddress: &route.RateLimit_Action_RemoteAddress{},
	}
}

func rateLimitActionRemoteAddress() *route.RateLimit_Action {
	return &route.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierRemoteAddress(),
	}
}

func rateLimitActionSpecifierSourceCluster() *route.RateLimit_Action_SourceCluster_ {
	return &route.RateLimit_Action_SourceCluster_{
		SourceCluster: &route.RateLimit_Action_SourceCluster{},
	}
}

func rateLimitActionSourceCluster() *route.RateLimit_Action {
	return &route.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierSourceCluster(),
	}
}

func rateLimitActionSpecifierDestinationCluster() *route.RateLimit_Action_DestinationCluster_ {
	return &route.RateLimit_Action_DestinationCluster_{
		DestinationCluster: &route.RateLimit_Action_DestinationCluster{},
	}
}

func rateLimitActionDestinationCluster() *route.RateLimit_Action {
	return &route.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierDestinationCluster(),
	}
}

func rateLimitActionSpecifierGenericKey(descriptor_value string) *route.RateLimit_Action_GenericKey_ {
	return &route.RateLimit_Action_GenericKey_{
		GenericKey: &route.RateLimit_Action_GenericKey{
			DescriptorValue: descriptor_value,
		},
	}
}

func rateLimitActionGenericKey(descriptor_value string) *route.RateLimit_Action {
	return &route.RateLimit_Action{
		ActionSpecifier: rateLimitActionSpecifierGenericKey(descriptor_value),
	}
}

func rateLimits(rl *dag.RouteFilter) []*route.RateLimit {

	var rad saarasconfig.RouteActionDescriptors
	var err error
	var rla []*route.RateLimit_Action
	var rrl_slice []*route.RateLimit

	rrl_slice = make([]*route.RateLimit, 0)
	// fmt.Printf("rateLimits(): RouteFilter [%+v]\n", rl)

	for _, f := range rl.Filters {
		// fmt.Printf("rateLimits(): processing filter [%+v]\n", f)
		if f == nil {
			continue
		}
		// f is of type cfg.SaarasRouteFilter
		// TODO: Fix the switch on filter type, we only parse rate-limit config if filter type is rate-limit
		switch f.Filter_type {
		case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
			// fmt.Printf("rateLimits(): Unmarshalling [%+s]\n", f.Filter_config)
			rad, err = saarasconfig.UnmarshalRateLimitRouteFilterConfig(f.Filter_config)
			// fmt.Printf("rateLimits(): Route action descriptor [%+v]\n", rad)
			if err != nil {
				//TODO
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

			rrl := route.RateLimit{
				Stage:   u32(0),
				Actions: rla,
			}

			rrl_slice = append(rrl_slice, &rrl)
		default:
		}
	}

	return rrl_slice

	//	return []*route.RateLimit{
	//		{
	//			Stage: u32(0),
	//			Actions: []*route.RateLimit_Action{
	//				rateLimitAction3(),
	//				rateLimitAction(),
	//			},
	//		},
	//		{
	//			Stage: u32(0),
	//			Actions: []*route.RateLimit_Action{
	//				rateLimitAction(),
	//			},
	//		},
	//	}

	// return []*route.RateLimit{}
}

func SetupRouteRateLimits(r *dag.Route, ra *route.RouteAction) {
	// fmt.Printf("SetupRouteRateLimits() DAG Route [%+v]\n", r)
	// fmt.Printf("SetupRouteRateLimits() DAG Route filters [%+v]\n", r.RouteFilters)
	if r.RouteFilters != nil {
		// fmt.Printf("SetupRouteRateLimits() DAG Route filter list [%+v]\n", r.RouteFilters.Filters)
		ra.RateLimits = rateLimits(r.RouteFilters)
	}
	// fmt.Printf("SetupRouteRateLimits() Done Route filters [%+v]\n", r)
}
