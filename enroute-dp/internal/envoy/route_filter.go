package envoy

import (
	//"strings"
	//"encoding/json"
	//"github.com/pkg/errors"
	"github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/saarasio/enroute/enroute-dp/internal/dag"
	//"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	"github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	//"github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	cfg "github.com/saarasio/enroute/enroute-dp/saarasconfig"
)

var test_cors_config = `
{
		"match_condition" : {
			"regex" : "*"
		},
		"access_control_allow_methods" : "GET",
		"access_control_allow_headers" : "x-forwarded-for",
		"access_control_expose_headers" : "*",
		"access_control_max_age" : "60s"
}
`

func SaarasConfigToEnvoyConfig(cfg *cfg.CorsFilterConfig) *envoy_config_route_v3.CorsPolicy {

	if cfg == nil {
		return nil
	}

	smlist := make([]*envoy_type_matcher_v3.StringMatcher, 0)

	if len(cfg.MatchCondition.Regex) > 0 {
		sm := &envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_SafeRegex{
				SafeRegex: &envoy_type_matcher_v3.RegexMatcher{
					EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{
						GoogleRe2: &envoy_type_matcher_v3.RegexMatcher_GoogleRE2{},
					},
					Regex: cfg.MatchCondition.Regex,
				},
			},
		}

		smlist = append(smlist, sm)
	}

	if len(cfg.MatchCondition.Prefix) > 0 {
		sm := &envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Prefix{
				Prefix: cfg.MatchCondition.Prefix,
			},
		}
		smlist = append(smlist, sm)
	}

	if len(cfg.MatchCondition.Exact) > 0 {
		sm := &envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{
				Exact: cfg.MatchCondition.Exact,
			},
		}
		smlist = append(smlist, sm)
	}

	if len(cfg.MatchCondition.Suffix) > 0 {
		sm := &envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Suffix{
				Suffix: cfg.MatchCondition.Suffix,
			},
		}
		smlist = append(smlist, sm)
	}

	if len(cfg.MatchCondition.Contains) > 0 {
		sm := &envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Contains{
				Contains: cfg.MatchCondition.Contains,
			},
		}
		smlist = append(smlist, sm)
	}

	c_cfg := envoy_config_route_v3.CorsPolicy{
		AllowOriginStringMatch: smlist,
		AllowMethods:           cfg.AccessControlAllowMethods,
		AllowHeaders:           cfg.AccessControlAllowHeaders,
		ExposeHeaders:          cfg.AccessControlExposeHeaders,
		MaxAge:                 cfg.AccessControlMaxAge,
	}

	return &c_cfg
}

func corsConfig(vh *dag.VirtualHost) *envoy_config_route_v3.CorsPolicy {

	if vh != nil {
		// VH Cors
		cors_http_filter := dag.GetVHHttpFilterConfigIfPresent(cfg.FILTER_TYPE_HTTP_CORS, vh)

		// No CORS filter
		if cors_http_filter == nil {
			return nil
		}

		filter_config := cors_http_filter.Filter.Filter_config

		// Unmarshal config
		unmsh_cfg, e := cfg.UnmarshalCorsFilterConfig(filter_config)

		if e != nil {
			return nil
		}

		c_cfg := SaarasConfigToEnvoyConfig(unmsh_cfg)

		return c_cfg
	}

	return nil
}

func SetupEnvoyFilters(vh *dag.VirtualHost, vhost *envoy_config_route_v3.VirtualHost, isVh bool, r *dag.Route) {
	c := corsConfig(vh)
	vhost.Cors = c
}
