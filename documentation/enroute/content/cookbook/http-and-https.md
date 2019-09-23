+++
date = "2017-04-02T22:01:15+01:00"
title = "One App - Two proxies"
tags = ["markdown","prototype", "hugo"]
categories = ["design"]
description = "Allow secure (HTTPS) and unsecure (HTTP) access to an application"
draft = false
weight = 30
+++

<img data-src="https://cldup.com/3tov0aCFh8.png" class="lazyload">

## Application
###### Application access requirement
This specifc use-case revolves around providing secure and un-secure access to an application. Internal consumption of application API needs bypassing SSL. External consumption of the application API happens over SSL

We demonstrate the use-case using the petstore example.

###### Application architecture
The customer use-case needs configuration of two proxies to achieve this. One runs the application without the certificate and another one runs it with the certificate.

<a href=""><img alt="Brand" src="/img/ApplicationArch.png"></a>

## Control Plane
###### Control Plane for proxy configuration
We spin up a control plane to create proxies and applications. Each of the data plane components connect to this control plane to read their config.

###### Control Plane API
The control plane exposes a RESTful API to administer proxies. The API provides complete control over the proxies. 

The admin API accepts content type application/json. 

###### Control Plane Abstractions
 - **Proxy** :  Proxy defines configuration associated with one proxy.
 - **Service** : Service is configuration associated with a listener. We associate service with a proxy to program it on a proxy.
 - **Route** : Route defines L7 routing rules.
 - **Upstream** : Upstream represents an endpoint to send traffic to


## Creating Proxies

**Proxy** is our high level abstraction that holds configuration related to a proxy. We initially create proxies on the control plane.


##### Create proxy to serve application over http

Proxy can be created using the following commands -

```
$ curl -s -X POST https://ingresspipe.io:8443/proxy -d '{"Name":"petstore"}' -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "name": "petstore"
}
```

To get details about the proxy -

```
$ curl -s https://ingresspipe.io:8443/proxy/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_proxy": [
            {
                "create_ts": "2019-09-17T22:03:23.307982+00:00",
                "proxy_id": 18,
                "proxy_name": "petstore",
                "update_ts": "2019-09-17T22:05:12.004973+00:00"
            }
        ]
    }
}
```

##### Create another proxy to serve traffic over https, check list of proxies

We repeat the steps above with "petstore-https" to create another proxy. Once created, check the list of proxies created -

```
$ curl -s https://ingresspipe.io:8443/proxy -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_proxy": [
            {
                "create_ts": "2019-09-17T22:03:23.307982+00:00",
                "proxy_id": 18,
                "proxy_name": "petstore",
                "update_ts": "2019-09-17T22:05:12.004973+00:00"
            },
            {
                "create_ts": "2019-09-17T22:27:35.555514+00:00",
                "proxy_id": 20,
                "proxy_name": "petstore-https",
                "update_ts": "2019-09-17T22:54:26.053712+00:00"
            }
        ]
    }
}
```

## Connecting data plane to proxies

###### Saaras data plane container

The Saaras data plane container comes pre-packaged with Envoy. It is accessible on docker at **saarasio/enroute**

###### Running data plane container

Running the data plane requires specifying the IP address and port of the control plane. The data plane container takes three arguments to determine details about the control plane.
 
 - **ENROUTE_NAME** - this is the name of the proxy created on the control plane
 - **ENROUTE_CP_IP** - this is the IP or the host to connect to the control plane
 - **ENROUTE_CP_PORT** - this is the port of the host to connect to the control plane
 - **ENROUTE_CP_PROTO** - the protocol to use to connect to the control plane

We create the data plane for petstore proxy using the following command -
```
docker run -e ENROUTE_NAME=petstore -e ENROUTE_CP_IP=ingresspipe.io -e ENROUTE_CP_PORT=8443 -e ENROUTE_CP_PROTO=HTTPS -p 80:8080 saarasio/enroute
```

By default listener inside the container is created on port 8080. This port is published to the host stack using the ``` -p 80:8080 ``` directive. We connect to the control plane over HTTPS.

We create the data plane for petstore-https proxy using the following command -

```
sudo docker run -e ENROUTE_NAME=petstore-https -e ENROUTE_CP_IP=ingresspipe.io -e ENROUTE_CP_PORT=8443 -e ENROUTE_CP_PROTO=HTTPS -p 443:8443 saarasio/enroute
```

## Programming the application on one proxy

An application is built using service -> route(s) -> upstream. The controller has APIs to work with these objects. Here we demonstrate how to work with each of these abstractions.

Under the hood, this configuration is streamed to the corresponding Envoy proxy over xDS APIs.

###### Create service

We start off by creating a service. A service can be created using the following API -

```
curl -s -X POST https://ingresspipe.io:8443/service -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Service_Name":"petstore", "Fqdn":"petstore.ingresspipe.io"}' | python -m json.tool
```

Here is how you view all the services -

```
curl http://ingresspipe.io:8443/service | python -m json.tool
```

To view just one service, qualify the above command with service name -

```
curl http://ingresspipe.io:8443/service/petstore | python -m json.tool
```

###### Create route(s) for service

Routes are rules to provide L7 routing information to Envoy. Routes do not exist indpendently and are associated with service. To create a route for the petstore, use the following command.

```
curl -s -X POST https://ingresspipe.io:8443/service/petstore/route -H "Content-Type: application/json" -d '{"Route_name":"default", "Route_prefix":"/"}' | python -m json.tool
```

You can also see a service, its routes and associated upstream using the **dump** api for a service. We can look at a service and its associated route using the following command -

```
curl -s https://ingresspipe.io:8443/service/dump/petstore -H "Authorization: Bearer treehugger" | python -m json.tool
{
    "data": {
        "saaras_db_service": [
            {
                "create_ts": "2019-09-12T00:14:21.68843+00:00",
                "fqdn": "petstore.ingresspipe.io",
                "routes": [
                    {
                        "route_id": 19,
                        "route_name": "default",
                        "route_prefix": "/",
                    },
                ],
                "service_id": 30,
                "service_name": "petstore",
                "service_secrets": []
            }
        ]
    }
}
```

###### Create upstream

Next we create an upstream that will serve this traffic -

```
```

###### Associate upstream with routes

###### Associate the service with petstore proxy

## Create complete application replica using *deepcopy*

## Secret

###### Create a secret
###### Attach the secret to a service

##  Test traffic
