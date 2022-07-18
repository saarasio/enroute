package saarasfilters

import (
	"github.com/saarasio/enroute/enroute-dp/saarasconfig"
	"github.com/sirupsen/logrus"
)

func SetConfigJson(log *logrus.Entry, filter_type string, filter_config string, args *map[string]interface{}, parse_config_success *bool) {
	*parse_config_success = false
	switch filter_type {
	case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
		cfg, err := saarasconfig.UnmarshalRateLimitRouteFilterConfig(filter_config)
		if err == nil {
			*parse_config_success = true
			(*args)["config_json"] = cfg
		} else {
			(*args)["config_json"] = struct {
				Descriptors [1]string `json:"descriptors"`
			}{
				Descriptors: [...]string{"{}"},
			}
			log.Errorf("Failed to decode RT Config [%+v] \n", filter_config)
		}
	case saarasconfig.FILTER_TYPE_HTTP_LUA, saarasconfig.FILTER_TYPE_VH_LUA:
		// Lua filter config contains lua script, which cannot be converted to json
		// convert it to { "config" : "..." } type
		type luaFilterConfig struct {
			Config string `json:"config"`
		}

		var lfc luaFilterConfig
		lfc.Config = filter_config
		log.Errorf("Setting config_json to [%+v] \n", lfc)
		*parse_config_success = true
		(*args)["config_json"] = lfc
	case saarasconfig.FILTER_TYPE_VH_CORS:
		cfg, err := saarasconfig.UnmarshalCorsFilterConfig(filter_config)
		if err == nil {
			*parse_config_success = true
			(*args)["config_json"] = cfg
		} else {
			var cors_config_filter saarasconfig.CorsFilterConfig
			(*args)["config_json"] = cors_config_filter
			log.Errorf("Failed to decode CORS Config [%+v] \n", filter_config)
		}

	default:
		// Unsupported filter
		log.Errorf("Unsupported filter type [%s]\n", filter_type)
	}
}

func IsFilterTypeValid(filter_type string) bool {
	switch filter_type {
	case saarasconfig.FILTER_TYPE_HTTP_LUA:
		return true
	case saarasconfig.FILTER_TYPE_VH_LUA:
		return true
	case saarasconfig.FILTER_TYPE_VH_CORS:
		return true
	case saarasconfig.FILTER_TYPE_RT_RATELIMIT:
		return true
	default:
		return false
	}
	return false
}
