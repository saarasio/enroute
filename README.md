[![Build Status](https://dev.azure.com/saaras-io/Enroute/_apis/build/status/saarasio.enroute?branchName=master)](https://dev.azure.com/saaras-io/Enroute/_build/latest?definitionId=6&branchName=master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Twitter](https://img.shields.io/twitter/follow/SaarasInc?label=Follow%20EnRoute&style=social)

![](https://getenroute.io/images/enroute-logo.svg)

EnRoute makes it easy to run Envoy as an API Gateway. You can use it for microservices running inside Kubernetes or any service running standalone when there is no Kubernetes.

![](https://getenroute.io/img/APIGatewayIngressWithFiltersWithHelm.jpeg)

What makes it easy is simple REST APIs to configure the Standalone gateway or CRDs to configure the Kubernetes Ingress Gateway. Plugins provide the ability to add fine-grained route-level or global policies and traffic control.

As Envoy is being widely accepted as a next-gen proxy, EnRoute is deployed in production use at companies. EnRoute is an actively maintained project and community edition supports Advanced Rate Limiting.

#### How is EnRoute different?

EnRoute is an API gateway with batteries included. EnRoute is oriented towards DevOps and integration with CI/CD pipelines. It is completely automatable and there is an API for everything. 

EnRoute state management is flexible. For Kubernetes Ingress API Gateway, the state is completely managed inside Kubernetes. For Kubernetes, the state is stored in CRDs and state management is completely Kubernetes-native without any external databases. EnRoute supports GitOps even when running as a stateless docker container. 

EnRoute is the only gateway on Envoy proxy that works for both Kubernetes Ingress and Standalone use-cases. Typically solutions either target one or the other. A majority of users have a mix of workloads, and this capability comes in handy, especially with the same consistent policy model across all deployments. And running Envoy makes it a super performant solution.

![](https://getenroute.io/img/APIGatewayStandaloneAndIngressWithFilters.jpeg)

## Features

EnRoute is built on high performance feature rich Envoy and provides the following features.

* **Run Anywhere** - Any Platform, Any Cloud - EnRoute can integrate with any cloud for any service or can protect services running inside Kubernetes
* **Native Kubernetes** - Use CRDs to configure EnRoute Ingress API Gateway without any external store.
* **Canary Release** - EnRoute OSS supports canary releases
* **Advanced Rate Limiting** - EnRoute community edition supports advanced per-user, different rate limits for authenticated/unauthenticated user, IP based rate-limiting and several advanced configurations.
* **Multiple Load Balancing Algorithms** - EnRoute can be effectively programmed to use different load balancing mechanisms like Round Robin, Least Request, Random, Ring Hash
* **Circuit Breakers** - EnRoute can program underlying Envoy circuit Breakers
* **Health Checks** - Health checking including custom health check for upstream services
* **Service Discovery** - Discover external services in cloud or service mesh like consul to populate Standalone or Kubernetes Ingress Gateway
* **Tracing** - Zipkin, Jaeger support
* **gRPC** - Native support for gRPC
* **Websockets** - Support for Websocket services
* **SSL** - Terminate SSL connections either at Kubernetes Ingress or using a Docker gateway
* **Cipher Selection** - Select ciphers used to terminate SSL connections
* **JWT Validation** - Validate incoming JWT tokens
* **OIDC** - Open ID Connect support

[Complete list of features](https://getenroute.io/features)


## Getting Started

* Use helm to get started with [Kubernetes Ingress API Gateway](https://getenroute.io/docs/ingress-filter-legos-secure-microservices-apis-using-helm-envoy/)
* Use docker to get started with [Standalone Gateway](https://getenroute.io/docs/getting-started-enroute-standalone-gateway/)
* Use ```enroutectl``` to [program OpenAPI spec on gateway](https://getenroute.io/cookbook/openapi-swagger-spec-autoprogram-api-gateway-30-seconds-no-code/)

Blogs, Cookbooks, getting started, examples and additional documentation can be found at

- [getenroute.io](https://getenroute.io)
- [FAQs](https://getenroute.io/faq/)
- [Introduction](https://getenroute.io/docs/enroute-universal-api-gateway/)
- [Cookbook](https://getenroute.io/cookbook/)
- [Blog](https://getenroute.io/blog/)

### Extend using Global HTTP Filters and Route Filters

You can associate additional plugin/filter functionality at global level or route level.

Filters/Plugins are supported for both Kubernetes Ingress Gateway and Standalone Gateway.

<img src="https://getenroute.io/img/EnrouteConfigModel3.png" alt="Config Model" width="400"/>

### Community

[Periodic Office Hours](https://www.meetup.com/enroute-universal-api-gateway-periodic-office-hours/events/rtqbdsycccbsb/)

[Community Slack](https://join.slack.com/t/saaras-io/shared_invite/zt-o5nzx78x-bjm7XEyRFRFkMSZzBX12mA)
[Community Discord](https://discord.gg/p9Nu9Uk)

### Enterprise Support and Demo
EnRoute has an [enterprise version](https://getenroute.io/features) that provides additional support and features 
