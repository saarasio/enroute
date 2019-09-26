+++
date = "2017-04-02T22:01:15+01:00"
title = "Getting started with Enroute"
tags = ["markdown","prototype", "hugo"]
categories = ["design"]
description = "API basics"
draft = false
weight = 30
+++

<img data-src="https://cldup.com/3tov0aCFh8.png" class="lazyload">

## Why Enroute?

Enroute provides for a way to program multiple Envoy proxies. It defines well known abstractions like proxy, service, route, upstream and secret. It provides an API to work with these abstractions.

The configuration once defined, can be used to setup an application and associate it with one or more proxies.

<!-- <a href=""><img alt="Enroute" src="/img/EnrouteGettingStartedAPI.png"></a> -->
<a href=""><img alt="Enroute" src="/img/EnrouteGettingStartedAPI2.png"></a>

This approach allows Enroute to integrate with cloud infrastructure, discovery services and secret stores. 

Enroute lets a user track configuration changes over a period of time. Its config backup and restore let developer operations keep track of configuration changes and treat infrastructure as code.

## Understanding Enroute through a simple example

Next few steps show different aspects of enroute by programming the controller using APIs and running a proxy to consume that config.

#### Create objects on controller using API

For this example, a controller is already running on ```https://ingresspipe.io:8443``` We use the Enroute API to perform the following tasks -

###### Create **Proxy**

```
$ curl -s -X POST "https://ingresspipe.io:8443/proxy" -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Name":"proxy-p"}' | python -m json.tool
{
    "name": "proxy-p"
}
```

###### Create **Service**

```
$ curl -s -X POST "https://ingresspipe.io:8443/service" -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Service_Name":"l", "fqdn":"127.0.0.1"}' | python -m json.tool
{
    "data": {
        "insert_saaras_db_service": {
            "affected_rows": 1
        }
    }
}
```

###### Create **Route**

```
$ curl -s -X POST "https://ingresspipe.io:8443/service/l/route" -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Route_Name":"r", "Route_prefix":"/"}' | python -m json.tool
{
    "data": {
        "insert_saaras_db_route": {
            "affected_rows": 2
        }
    }
}
```

###### Create **Upstream**

```
$ curl -s -X POST "https://ingresspipe.io:8443/upstream" -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Upstream_name":"u", "Upstream_ip":"127.0.0.1", "Upstream_port": "8888", "Upstream_hc_path":"/", "Upstream_weight": "100"}' | python -m json.tool
{
    "data": {
        "insert_saaras_db_upstream": {
            "affected_rows": 1
        }
    }
}
```

#### Wire up objects

###### Create Route -> Upstream association

```
$ curl -s -X POST "https://ingresspipe.io:8443/service/l/route/r/upstream/u" -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "insert_saaras_db_route_upstream": {
            "affected_rows": 4
        }
    }
}
```

###### Create Proxy -> Service association

```
$ curl -s -X POST "https://ingresspipe.io:8443/proxy/proxy-p/service/l" -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "insert_saaras_db_proxy_service": {
            "affected_rows": 3
        }
    }
}
```

#### Run docker container with enroute image and point it to the controller

```
sudo docker run -e ENROUTE_NAME=proxy-p -e ENROUTE_CP_IP=ingresspipe.io -e ENROUTE_CP_PORT=8443 -e ENROUTE_CP_PROTO=HTTPS --net=host saarasio/enroute:latest
```

The above command starts envoy, connects to the controller and consumes configuration setup for **proxy-p**. It'll create a listener ```l```, route ```r```, upstream ```u``` on envoy proxy.
