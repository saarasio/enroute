package ratelim

type RateLimitGlobalConfig struct {
    Domain string
}

func UnmarshalRateLimitGlobalConfig(config_string string) (RateLimitGlobalConfig, error) {
    var cfg RateLimitGlobalConfig
    return cfg, nil
}
