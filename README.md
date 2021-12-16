[![Build Status](https://dev.azure.com/saaras-io/Enroute/_apis/build/status/saarasio.enroute?branchName=master)](https://dev.azure.com/saaras-io/Enroute/_build/latest?definitionId=6&branchName=master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Twitter](https://img.shields.io/twitter/follow/SaarasInc?label=Follow%20EnRoute&style=social)](https://twitter.com/intent/follow?screen_name=SaarasInc)

![](https://getenroute.io/images/enroute-logo.svg)

EnRoute makes is simple to expose services running in Kubernetes using one helm command. L7 policy can be specified using simple switches to enable/disable functionality

![](https://getenroute.io/img/APIGatewayIngressWithFiltersWithHelm.jpeg)

EnRoute can also use ACME protocol to get, verify and install a Let's Encrypt Certificate in just one step

![](https://getenroute.io/img/onestep-fast-20.gif)

## Getting Started Kubernetes Ingress Gateway

Getting started is extremely simple and can be achieved in less than a minute. Once the helm repositories are setup, all it takes a simple command to set it up.

[Getting Started Kubernetes Ingress Guide](https://getenroute.io/docs/ingress-filter-legos-secure-microservices-apis-using-helm-envoy/)

## Getting Started Docker Gateway

EnRoute is the only gateway on Envoy proxy that works for both Kubernetes Ingress and Standalone use-cases. Typically solutions either target one or the other. A majority of users have a mix of workloads, and this capability comes in handy, especially with the same consistent policy model across all deployments. And running Envoy makes it a super performant solution.

![](https://getenroute.io/img/APIGatewayStandaloneAndIngressWithFilters.jpeg)

[Getting Started EnRoute Standalone Guide](https://getenroute.io/reference/getting-started/getting-started-enroute-standalone-gateway/)

* EnRoute Standalone can also be setup using the [getting started bash script](https://github.com/saarasio/gettingstarted)

* EnRoute APIs can be invoked from any language. [An example in golang can be found here](https://github.com/saarasio/api-ratelimit)

* Use ```enroutectl``` to [program OpenAPI spec on gateway](https://getenroute.io/cookbook/openapi-swagger-spec-autoprogram-api-gateway-30-seconds-no-code/)

## Features

[Complete list of features](https://getenroute.io/features)

## Community

- [Periodic Office Hours](https://www.meetup.com/enroute-universal-api-gateway-periodic-office-hours/events/rtqbdsycccbsb/)

- [Community Slack](https://join.slack.com/t/saaras-io/shared_invite/zt-pz1qay34-9UNGwJWTOMG5jolGrbWH~g)

## Enterprise Support and Demo

EnRoute has an [enterprise version](https://getenroute.io/features) that provides additional support and features 
