# service-host-route

![Version: 0.2.0](https://img.shields.io/badge/Version-0.2.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.11.0](https://img.shields.io/badge/AppVersion-v0.11.0-informational?style=flat-square)

Host (GatewayHost), Route (ServiceRoute) config

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| filters.route.circuitbreakers | object | `{"enable":false,"max_connections":100,"max_pending_requests":101,"max_requests":102,"max_retries":103}` | enable/configure circuit breakers for this route |
| filters.route.directresponse | object | `{"enable":false}` | used to send a direct response |
| filters.route.hostrewrite.enable | bool | `false` |  |
| filters.route.hostrewrite.pattern_regex | string | `nil` |  |
| filters.route.hostrewrite.substitution | string | `"newhost.com"` | if `pattern_regex` is empty, simply replace host with the value specified in `substitution` if `pattern_regex` is not empty, match groups in pattern can be used to rewrite this host https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto#envoy-v3-api-field-config-route-v3-routeaction-host-rewrite-path-regex |
| filters.route.outlierdetection | object | `{"consecutive_5xx":5,"consecutive_gateway_failure":5,"enable":false,"enforcing_consecutive_5xx":5,"enforcing_consecutive_gateway_failure":5}` | enable/configure outliner detection settings for this route |
| filters.route.ratelimit.enable | bool | `false` | enable configuration to send rate-limit descriptors for this route to global rate-limit engine example descriptors from template file are installed Note: these may have to be fine-tuned for the use-case |
| filters.route.redirect | object | `{"enable":false,"host_redirect":"enroutedemo.com","path_redirect":"/get","port_redirect":8081,"prefix_rewrite":"/get_rewrite","regex_redirect":"redirect","response_code":302,"scheme_redirect":"http","strip_query":false}` | redirect a request using these settings TODO |
| filters.virtualhost.cors.access_control_allow_headers | string | `"Content-Type"` |  |
| filters.virtualhost.cors.access_control_allow_methods | string | `"GET, OPTIONS"` |  |
| filters.virtualhost.cors.access_control_expose_headers | string | `"*"` |  |
| filters.virtualhost.cors.access_control_max_age | int | `120` |  |
| filters.virtualhost.cors.enable | bool | `false` | when enabled, cors filter is associated with this virtualhost |
| filters.virtualhost.cors.match_condition_regex | string | `"\\\\*"` |  |
| filters.virtualhost.lua.enable | bool | `false` | when enabled, lua filter is associated with this virtualhost |
| filters.virtualhost.lua.scriptfile | string | `"files/script.lua"` |  |
| filters.virtualhost.rbac | object | `{"enable":false}` | when enabled, cors filter is associated with this virtualhost |
| routeonly | bool | `false` | when set to true, create `ServiceRoute` when set to false, create `GatewayHost` A `GatewayHost` creates a Host with Fqdn and a Route  eg: GatewayHost(fqdn='foo.com', route='/bar') creates Host(fqdn='foo.com'), Route('/bar) A `ServiceRoute` creates a Route and associates it with an existing Host  eg: ServiceRoute (fqdn='foo.com', route='/baz') creates Route('/baz) and associates it with Host('/foo') A `ServiceRoute` is used when a Host is already created using `ServiceRoute` |
| service.fqdn | string | `""` | fqdn for the service being configured When `ServiceRoute` is created, a Host with this Fqdn is created When `ServiceRoute` is created, a route is associated with a Host with this Fqdn |
| service.httphealthcheck | object | `{"enable":false,"healthy_threshold_count":3,"host":"hc","interval_seconds":5,"path":"/","timeout_seconds":3,"unhealthy_threshold_count":3}` | Define healthcheck for this service |
| service.httphealthcheck.enable | bool | `false` | When enabled, health checks are installed |
| service.name | string | `"httpbin"` | Name of the service |
| service.port | int | `9000` | Port on which the clusterIP service is accessible |
| service.prefix | string | `"/"` | L7 prefix on which to make the service routable |
| service.protocol | string | `nil` | Set protocol to "h2c" for a grpc service, else leave empty |

