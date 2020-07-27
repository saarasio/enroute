[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


![](enroute.png)

# Universal Gateway with OpenAPI Spec and Advanced Rate-limiting built on Envoy Proxy

Run Any[**where**] - Any[**Platform | Cloud | Service**]

One Control Plane to run Envoy Proxy as
[Kubernetes Ingress](https://getenroute.io/docs/getting-started-enroute-ingress-controller/) Gateway | [Standalone Gateway](https://getenroute.io/docs/getting-started-enroute-standalone-gateway/) | Fleet of Envoy proxies as Stateless SideCar Gateway

<div class="row">
  <div class="column"><img src="https://getenroute.io/img/topology-saaras-k8s-ingress.png" alt="Kubernetes Ingress" width="300"/></div>
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

