[![Build Status](https://dev.azure.com/saaras-io/Enroute/_apis/build/status/saarasio.enroute?branchName=master)](https://dev.azure.com/saaras-io/Enroute/_build/latest?definitionId=6&branchName=master)[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


![](enroute.png)


# Universal Gateway with OpenAPI Spec and Advanced Rate-limiting built on Envoy Proxy

Run Any[**where**] - Any[**Platform | Cloud | Service**]

Universal Control Plane that drives Envoy (or a Fleet of Envoy's).
Runs as
[Kubernetes Ingress](https://getenroute.io/docs/getting-started-enroute-ingress-controller/) Gateway | [Standalone Gateway](https://getenroute.io/docs/getting-started-enroute-standalone-gateway/) | Stateless SideCar Gateways

<div class="row">
  <div class="column"><img src="https://getenroute.io/img/topology-saaras-k8s-ingress.png" alt="Kubernetes Ingress" width="400"/></div>
  <div class="column"><img src="https://getenroute.io/img/topology-saaras-standalone-gw.png" alt="Standalone" width="400"/></div>
</div>


### ```enroutectl```: Program OpenAPI Spec on Enroute Universal API Gateway in a minute

```enroutectl openapi --openapi-spec petstore.json --to-standalone-url http://localhost:1323/```

### Extend using Global HTTP Filters and Route Filters

You can associate additional plugin/filter functionality at global level or route level.

Filters/Plugins are supported for both Kubernetes Ingress Gateway and Standalone Gateway.

<img src="https://getenroute.io/img/EnrouteConfigModel3.png" alt="Config Model" width="400"/>

### Drive Enroute using REST or GraphQL APIs, Kubernetes CRDs or ```enroutectl``` cli

Enroute provides several options to populate the xDS cache for the underlying Envoy. 

##### It has simple REST APIs and a GraphQL interface.


```shell
    curl -X POST "http://localhost:1323/service"         \
      -d 'Service_Name=openapi-enroute'                  \
      -d 'Fqdn=saaras.io'
    
    curl -X POST "http://localhost:1323/service/openapi-enroute/route"  \
      -d 'Route_Name=root-slash'                                        \
      -d 'Route_prefix=/'
                             
    curl -X POST "http://localhost:1323/upstream"     \
      -d 'Upstream_name=openapi-upstream'             \
      -d 'Upstream_ip=openapi.example.com'            \
      -d 'Upstream_port=9001'                         \
      -d 'Upstream_hc_path=/'                         \
      -d 'Upstream_weight=100'
                            
```

Rate Limit ```Filter``` 

```shell
curl -s localhost:1323/filter/route_rl_1 | jq
{
  "data": {
    "saaras_db_filter": [
      {
        "filter_id": 139,
        "filter_name": "route_rl_1",
        "filter_type": "route_filter_ratelimit",
        "filter_config": {
          "descriptors": [
            {
              "generic_key": {
                "descriptor_value": "default"
              }
            }
          ]
        }
      }
    ]
  }
}
```

```GlobalConfig``` for Standalone Gateway

```shell
curl -s localhost:1323/globalconfig/t1 | jq
{
  "data": {
    "saaras_db_globalconfig": [
      {
        "globalconfig_id": 237,
        "globalconfig_name": "gc1",
        "globalconfig_type": "globalconfig_ratelimit",
        "config_json": {
          "domain": "enroute",
          "descriptors": [
            {
              "key": "generic_key",
              "value": "default",
              "rate_limit": {
                "unit": "second",
                "requests_per_unit": 10
              }
            }
          ]
        }
      }
    ]
  }
}
```

##### It can be programmed using the ```enroutectl``` CLI

Program an OpenAPI Spec on Enroute Standalone Gateway or Kubernetes Ingress Gateway

```shell
enroutectl openapi --openapi-spec petstore.json --to-standalone-url http://localhost:1323/
```

##### When running at Kubernetes Ingress, CRDs can be used to program it

Creating a ```GatewayHost``` that maps to a VirtualHost

```yaml
---
apiVersion: enroute.saaras.io/v1beta1

kind: GatewayHost
metadata:
  labels:
    app: httpbin
  name: httpbin
  namespace: enroute-gw-k8s
spec:
  virtualhost:
    fqdn: demo.saaras.io
    filters:
      - name: luatestfilter
        type: http_filter_lua
  routes:
    - match: /
      services:
        - name: httpbin
          port: 80
      filters:
        - name: rl2
          type: route_filter_ratelimit
---
```

Creating a ```RouteFilter``` that attaches to a ```GatewayHost``` Route

```yaml
apiVersion: enroute.saaras.io/v1beta1
kind: RouteFilter
metadata:
  labels:
    app: httpbin
  name: rl2
  namespace: enroute-gw-k8s
spec:
  name: rl2
  type: route_filter_ratelimit
  routeFilterConfig:
    config: |
        {
            "descriptors": [
              {
                "request_headers": {
                  "header_name": "x-app-key",
                  "descriptor_key": "x-app-key"
                }
              },
              {
                "remote_address": "{}"
              }
            ]
        }
---
```

Creating a ```HttpFilter``` that can be attached to a ```GatewayHost```

```yaml
apiVersion: enroute.saaras.io/v1beta1
kind: HttpFilter
metadata:
  labels:
    app: httpbin
  name: luatestfilter
  namespace: enroute-gw-k8s
spec:
  name: luatestfilter
  type: http_filter_lua
  httpFilterConfig:
    config: |
        function get_api_key(path, q_param_name)
            -- path = "/?api-key=valid-key"
            s, e = string.find(path, "?")
            if s ~= nil then
              for pre, q_params in string.gmatch(path, "(%S+)?(%S+)") do
                -- print(pre, q_params, path, s, e)
                for k, v in string.gmatch(q_params, "(%S+)=(%S+)") do
                  print(k, v)
                  if k == q_param_name then
                    return v
                  end
                end
              end
            end

            return nil
        end

        function envoy_on_request(request_handle)
           request_handle:logInfo("Begin: envoy_on_request()");

           hdr_x_app_key = "x-app-key"
           hdr_x_app_not_found = "x-app-notfound"
           q_param_name = "api-key"

           -- extract API key from header "x-app-key"
           headers = request_handle:headers()
           header_value = headers:get(hdr_x_app_key)

           if header_value ~= nil then
             request_handle:logInfo("envoy_on_request() API Key from header "..header_value);
           else
             request_handle:logInfo("envoy_on_request() API Key in header is nil");
           end

           -- extract API key from query param "api-key"
           path_in = headers:get(":path")
           api_key = get_api_key(path_in, q_param_name)

           if api_key ~= nil then
             request_handle:logInfo("envoy_on_request() API Key from query param "..api_key);
           else
             request_handle:logInfo("envoy_on_request() API Key from query param is nil");
           end

           -- If API key found, do nothing
           -- else set header x-app-key:x-app-notfound
           if header_value == nil then
               if api_key == nil then
                 headers:add(hdr_x_app_key, hdr_x_app_not_found)
               else
                 headers:add(hdr_x_app_key, api_key)
               end
           end

           request_handle:logInfo("End: envoy_on_request()");

        end

        function envoy_on_response(response_handle)
           response_handle:logInfo("Begin: envoy_on_response()");
           response_handle:logInfo("End: envoy_on_response()");
        end
---
```

Creating ```GlobalConfig``` to program advanced rate-limit engine config

```yaml

---
apiVersion: enroute.saaras.io/v1beta1

kind: GlobalConfig
metadata:
  labels:
    app: httpbin
  name: rl-global-config
  namespace: enroute-gw-k8s
spec:
  name: rl-global-config
  type: globalconfig_ratelimit
  config: |
        {

          "domain": "enroute",
          "descriptors" :
          [
            {
              "key" : "generic_key",
              "value" : "default",
              "rate_limit" :
              {
                "unit" : "second",
                "requests_per_unit" : 10
              }
            }
          ]
        }
---
```

### Grafana Telemetry for Standalone Gateway

![Grafana Telemetry on Standalone Gateway](https://getenroute.io/img/grafana-swagger.png)

### Why Enroute?

Digital transformation is a key initiative in organizations to [meet business requirements](https://getenroute.io/blog/devops-secops-k8s-cloud-adoption-micro-services/) . This need is driving cloud adoption with a more self-serve DevOps driven approach. Application and micro-services are run in Kubernetes and in public/private cloud with an automated continous delivery pipeline.

As applications undergo this change, traditional API gateways are retrofitted to meet the changing requirements. This has resulted in [multiple API gateways and different solutions](https://getenroute.io/blog/gateway-mesh/) that work only in a subset of use-cases. An API gateway that works for traditional as well as new cloud-native use-cases is critical as an application undergoes this transition.

Enroute is built from ground-up to support both traditional and cloud native use-cases. Enroute can be deployed in [multiple topolgies](https://getenroute.io/blog/enroute-topologies/) to meet the demands of todays application, regardless of where an application is in the cloud journey. The same gateway can be deployed outside kubernetes in [Standalone](https://getenroute.io/docs/getting-started-enroute-standalone-gateway/) mode for traditional use cases, a [kubernetes ingress controller](https://getenroute.io/docs/getting-started-enroute-ingress-controller/) and also inside kubernetes.

Enroute's powerful API brings automation and enables developer operations to treat infrastructure as code. Enroute natively supports [advanced rate-limiting and lua scripting](https://getenroute.io/cookbook/getting-started-advanced-rate-limiting/) as extensible filters or plugins that can be attached either for per-route traffic or all traffic. Enroute API Gateway control plane drives one or many data planes built using [Envoy Proxy](https://envoyproxy.io)

### Getting Started

Blogs, Cookbooks, getting started, examples and additional documentation can be found at

- [getenroute.io](https://getenroute.io)
- [FAQs](https://getenroute.io/faq/)
- [Introduction](https://getenroute.io/docs/enroute-universal-api-gateway/)
- [Getting Started](https://getenroute.io/docs/)
- [Cookbook](https://getenroute.io/cookbook/)
- [Blog](https://getenroute.io/blog/)

