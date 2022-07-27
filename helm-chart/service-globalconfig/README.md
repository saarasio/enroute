# service-globalconfig

![Version: 0.2.0](https://img.shields.io/badge/Version-0.2.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.11.0](https://img.shields.io/badge/AppVersion-v0.11.0-informational?style=flat-square)

Global Config and Global Filters for EnRoute

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| filters.cors | object | `{"enable":false}` | cors filter configuration |
| filters.cors.enable | bool | `false` | when enabled, global cors filter config is created |
| filters.extauthz | object | `{"allowed_authorization_headers":["\"ext-authz-example-header\"","\"x-auth-accountId\"","\"x-auth-userId\"","\"x-auth-userId\""],"allowed_request_headers":["\"x-stamp\"","\"requested-status\"","\"x_forwarded_for\"","\"requested-cookie\""],"auth_service":"ext-authz","auth_service_port":8080,"auth_service_proto":"http","body_allow_partial":true,"body_max_bytes":409,"enable":false,"failure_mode_allow":true,"pack_raw_bytes":false,"path_prefix":null,"status_on_error":403,"timeout":10,"url":"https://ext-authz-ns.ext-auth:8443"}` | ext_authz filter configuration https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter#config-http-filters-ext-authz |
| filters.extauthz.allowed_authorization_headers | list | `["\"ext-authz-example-header\"","\"x-auth-accountId\"","\"x-auth-userId\"","\"x-auth-userId\""]` | list of response headers from auth service that are forwarded to upstream https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ext_authz/v3/ext_authz.proto.html#envoy-v3-api-field-extensions-filters-http-ext-authz-v3-authorizationresponse-allowed-upstream-headers |
| filters.extauthz.allowed_request_headers | list | `["\"x-stamp\"","\"requested-status\"","\"x_forwarded_for\"","\"requested-cookie\""]` | a list of allowed request headers may be supplied https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ext_authz/v3/ext_authz.proto.html#envoy-v3-api-field-extensions-filters-http-ext-authz-v3-authorizationrequest-allowed-headers |
| filters.extauthz.auth_service | string | `"ext-authz"` | unused, use url field |
| filters.extauthz.auth_service_port | int | `8080` | unused, use url field |
| filters.extauthz.auth_service_proto | string | `"http"` | valid values are (http or grpc), used to communicate with external auth service |
| filters.extauthz.body_allow_partial | bool | `true` | invoke auth filter when maximum bytes are reached |
| filters.extauthz.body_max_bytes | int | `409` | defines the maximum bytes that will be buffered for this filter, else returns 413 |
| filters.extauthz.enable | bool | `false` | when enabled, global ext_authz filter config is installed |
| filters.extauthz.failure_mode_allow | bool | `true` | when set, requests are allowed to upstream even when there is a failure to communicate with external auth service |
| filters.extauthz.path_prefix | string | `nil` | prepend path value when sending requests to external authorization service |
| filters.extauthz.status_on_error | int | `403` | http status to return when network error in reaching external auth service |
| filters.extauthz.url | string | `"https://ext-authz-ns.ext-auth:8443"` | URL to reach external authz service Uses the form <scheme>://<namespace>.<service-name>:<service-port> scheme can be grpc or https, if no port is specified, port 443 is used for https |
| filters.healthcheck | object | `{"enable":false,"path":"/healthz"}` | HealthCheck filter configuration |
| filters.healthcheck.path | string | `"/healthz"` | Path on which healthchecks can be performed |
| filters.jwt | object | `{"audience":"api-identifier","enable":false,"issuer":{"create":false,"external_name":"saaras.auth0.com","service_name":"jwt-issuer-auth0","service_port":443,"service_protocol":"tls"},"issuer_url":"https://saaras.auth0.com/","jwks_uri":"https://saaras.auth0.com/.well-known/jwks.json","jwt_forward_header_name":"x-jwt-token","jwt_service_name":"jwt-issuer-auth0","jwt_service_port":443,"name":"auth0"}` | jwt filter configuration https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/jwt_authn_filter#config-http-filters-jwt-authn |
| filters.jwt.audience | string | `"api-identifier"` | audience allowed to access |
| filters.jwt.enable | bool | `false` | when enabled, global jwt filter config is installed |
| filters.jwt.issuer | object | `{"create":false,"external_name":"saaras.auth0.com","service_name":"jwt-issuer-auth0","service_port":443,"service_protocol":"tls"}` | Issuer service created to access JWT provider These settings are used to create an ExternalName service |
| filters.jwt.issuer.create | bool | `false` | Create ExternalName issuer service |
| filters.jwt.issuer.external_name | string | `"saaras.auth0.com"` | DNS of external service |
| filters.jwt.issuer.service_name | string | `"jwt-issuer-auth0"` | name of ExternalName issuer service |
| filters.jwt.issuer.service_port | int | `443` | port for ExternalName issuer service |
| filters.jwt.issuer.service_protocol | string | `"tls"` | protocol used to communicate with external service |
| filters.jwt.issuer_url | string | `"https://saaras.auth0.com/"` | JWT issuer URL, the principal that issued the JWT, usually a URL or an email address |
| filters.jwt.jwks_uri | string | `"https://saaras.auth0.com/.well-known/jwks.json"` | JWKS provider URI to reach provider |
| filters.jwt.jwt_forward_header_name | string | `"x-jwt-token"` | Header name in which the JWT token is forwarded to upstream |
| filters.jwt.jwt_service_name | string | `"jwt-issuer-auth0"` | Service name used to access the JWKS provider |
| filters.jwt.jwt_service_port | int | `443` | Port used to access the JWKS provider |
| filters.jwt.name | string | `"auth0"` | Name of JWKS provider |
| filters.lua | object | `{"enable":false,"scriptfile":"files/script.lua"}` | lua filter configuration |
| filters.lua.enable | bool | `false` | when enabled, a lua filter is installed with basic script |
| filters.lua.scriptfile | string | `"files/script.lua"` | not used |
| filters.opa | object | `{"enable":false}` | OPA filter configuration |
| filters.ratelimit | object | `{"enable":true}` | Rate Limit engine config |
| filters.ratelimit.enable | bool | `true` | when enabled, Rate Limit engine global config is created |
| filters.wasm | object | `{"enable":false,"image_url":"oci://saarasio/vvx-json"}` | wasm filter configuration |
| filters.wasm.image_url | string | `"oci://saarasio/vvx-json"` | url to remote oci image that has a wasm plugin packaged in it |
| mesh.linkerD | bool | `false` |  |

