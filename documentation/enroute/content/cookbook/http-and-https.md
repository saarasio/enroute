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

###### We'll walk through the following list of steps to achieve this objective
* Start control plane (For this example, we assume it is already running on https://ingresspipe.io:8443
* Create proxies, one for petstore, another for petstore-https
* Create service, one for petstore, another for petstore-https
* Associate route and upstream with petstore and petstore-https service
* Create secret and attach secret to petstore-https
* Associate psetstore service to petstore proxy
* Associate psetstore-https service to petstore-https proxy
* Send test traffic to petstore-http and petstore-https

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
{
    "data": {
        "saaras_db_service": [
            {
                "create_ts": "2019-09-12T00:14:21.68843+00:00",
                "fqdn": "petstore.ingresspipe.io",
                "service_id": 30,
                "service_name": "petstore",
                "update_ts": "2019-09-17T22:29:31.623226+00:00"
            }
        ]
    }
}
```

Here is how you view all the services -

```

$ curl -s https://ingresspipe.io:8443/service -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_service": [
            {
                "create_ts": "2019-09-12T00:14:21.68843+00:00",
                "fqdn": "petstore.ingresspipe.io",
                "service_id": 30,
                "service_name": "petstore",
                "update_ts": "2019-09-17T22:29:31.623226+00:00"
            },
            {
                "create_ts": "2019-09-17T22:53:27.129213+00:00",
                "fqdn": "petstore-https.ingresspipe.io",
                "service_id": 94,
                "service_name": "petstore-https",
                "update_ts": "2019-09-17T23:23:37.433466+00:00"
            }
        ]
    }
}

```

To view just one service, qualify the above command with service name -

```
$ curl -s https://ingresspipe.io:8443/service/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_service": [
            {
                "create_ts": "2019-09-12T00:14:21.68843+00:00",
                "fqdn": "petstore.ingresspipe.io",
                "service_id": 30,
                "service_name": "petstore",
                "update_ts": "2019-09-17T22:29:31.623226+00:00"
            }
        ]
    }
}

```

###### Create route(s) for service

Routes are rules to provide L7 routing information to Envoy. To create a route for the petstore, use the following command.

```
$ curl -s -X POST https://ingresspipe.io:8443/service/petstore/route -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Route_name":"default", "Route_prefix":"/"}' | python -m json.tool
{
    "data": {
        "insert_saaras_db_route": {
            "affected_rows": 2
        }
    }
}

```

You can also see a service, its routes and associated upstream using the **dump** api for a service. We can look at a service and its associated route using the following command -

```
curl -s https://ingresspipe.io:8443/service/dump/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
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
$ curl -s -X POST https://ingresspipe.io:8443/upstream -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" -d '{"Upstream_name":"default", "Upstream_ip":"172.31.18.10", "Upstream_port":"8088", "Upstream_weight":"100", "Upstream_hc_path": "/" }' | python -m json.tool
{
    "data": {
        "insert_saaras_db_upstream": {
            "affected_rows": 1
        }
    }
}
```

###### View the upstream

Use the following command to view the upstream -

```

$ curl -s  https://ingresspipe.io:8443/upstream/default -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_upstream": [
            {
                "create_ts": "2019-09-12T00:38:05.564381+00:00",
                "update_ts": "2019-09-17T22:53:27.139382+00:00",
                "upstream_hc_healthythresholdcount": 0,
                "upstream_hc_host": "",
                "upstream_hc_intervalseconds": 0,
                "upstream_hc_path": "/",
                "upstream_hc_timeoutseconds": 0,
                "upstream_hc_unhealthythresholdcount": 0,
                "upstream_id": 16,
                "upstream_ip": "172.31.18.10",
                "upstream_name": "default",
                "upstream_port": 8088,
                "upstream_strategy": "",
                "upstream_validation_cacertificate": "",
                "upstream_validation_subjectname": "",
                "upstream_weight": 100
            }
        ]
    }
}

```

###### Associate upstream with routes

We have created a service, a route for this service. Next we created an upstream. However this upstream is not associated with the route. Here we associate the upstream with the route -

```
$ curl -s -X POST https://ingresspipe.io:8443/service/petstore/route/default/upstream/default -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "insert_saaras_db_route_upstream": {
            "affected_rows": 4
        }
    }
}
```

###### Dump service to view (service -> route -> upstream)

```
$ curl -s https://ingresspipe.io:8443/service/dump/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
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
                        "route_upstreams": [
                            {
                                "upstream": {
                                    "upstream_id": 16,
                                    "upstream_ip": "172.31.18.10",
                                    "upstream_name": "default",
                                    "upstream_port": 8088
                                }
                            }
                        ]
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

###### Associate the service with petstore proxy

```
curl -s -X POST https://ingresspipe.io:8443/proxy/petstore/service/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

###### Dump petstore proxy config

```
$ curl -s https://ingresspipe.io:8443/proxy/dump/petstore -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_proxy": [
            {
                "create_ts": "2019-09-17T22:03:23.307982+00:00",
                "proxy_id": 18,
                "proxy_name": "petstore",
                "proxy_services": [
                    {
                        "service": {
                            "create_ts": "2019-09-12T00:14:21.68843+00:00",
                            "fqdn": "petstore.ingresspipe.io",
                            "routes": [
                                {
                                    "create_ts": "2019-09-12T00:19:06.462858+00:00",
                                    "route_id": 19,
                                    "route_name": "default",
                                    "route_prefix": "/",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T00:38:05.564381+00:00",
                                                "update_ts": "2019-09-17T22:53:27.139382+00:00",
                                                "upstream_hc_healthythresholdcount": 0,
                                                "upstream_hc_host": "",
                                                "upstream_hc_intervalseconds": 0,
                                                "upstream_hc_path": "/",
                                                "upstream_hc_timeoutseconds": 0,
                                                "upstream_hc_unhealthythresholdcount": 0,
                                                "upstream_id": 16,
                                                "upstream_ip": "172.31.18.10",
                                                "upstream_name": "default",
                                                "upstream_port": 8088,
                                                "upstream_strategy": "",
                                                "upstream_validation_cacertificate": "",
                                                "upstream_validation_subjectname": "",
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-23T22:22:09.688875+00:00"
                                },
                                {
                                    "create_ts": "2019-09-12T01:50:11.47308+00:00",
                                    "route_id": 22,
                                    "route_name": "proxy_admin",
                                    "route_prefix": "/admin",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-12T02:01:23.085169+00:00"
                                },
                                {
                                    "create_ts": "2019-09-12T02:06:49.882298+00:00",
                                    "route_id": 26,
                                    "route_name": "proxy_listeners",
                                    "route_prefix": "/listeners",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-12T02:07:32.451421+00:00"
                                },
                                {
                                    "create_ts": "2019-09-12T02:08:00.49249+00:00",
                                    "route_id": 28,
                                    "route_name": "proxy_clusters",
                                    "route_prefix": "/clusters",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-12T02:08:08.684786+00:00"
                                }
                            ],
                            "service_id": 30,
                            "service_name": "petstore",
                            "service_secrets": [],
                            "update_ts": "2019-09-23T22:22:09.688875+00:00"
                        }
                    }
                ],
                "update_ts": "2019-09-17T22:05:12.004973+00:00"
            }
        ]
    }
}
```

## Create service to serve application over HTTPS - use *deepcopy*

```
curl -s -X POST https://ingresspipe.io:8443/service/deepcopy/petstore/petstore-https -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

## Create and associate Secret to the newly created service

Next we show how create a secret and upload a key and a cert for that secret

###### Create a secret

1. **Create a secret**

```
curl -s -X POST https://ingresspipe.io:8443/secret -d '{"Secret_name":"petstore-https-secret"}' -H "Content-Type: application/json" -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

2. **Upload a key**

```
curl -vvv -X POST -F 'Secret_key=@privkey.pem' https://ingresspipe.io:8443/secret/petstore-https-secret/key -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

3. **Upload a cert**

```
curl -vvv -X POST -F 'Secret_cert=@fullchain.pem' https://ingresspipe.io:8443/secret/petstore-https-secret/cert -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

###### Attach the secret to a service

```
curl -s -X POST https://ingresspipe.io:8443/service/petstore-https/secret/petstore-https-secret -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

###### Associate the service with petstore-https proxy

```
curl -s -X POST https://ingresspipe.io:8443/proxy/petstore-https/service/petstore-https -H "Authorization: Bearer treeseverywhere" | python -m json.tool
```

## Dump petstore-https proxy config

```
$ curl -s https://ingresspipe.io:8443/proxy/dump/petstore-https -H "Authorization: Bearer treeseverywhere" | python -m json.tool
{
    "data": {
        "saaras_db_proxy": [
            {
                "create_ts": "2019-09-17T22:27:35.555514+00:00",
                "proxy_id": 20,
                "proxy_name": "petstore-https",
                "proxy_services": [
                    {
                        "service": {
                            "create_ts": "2019-09-17T22:53:27.129213+00:00",
                            "fqdn": "petstore-https.ingresspipe.io",
                            "routes": [
                                {
                                    "create_ts": "2019-09-17T22:53:27.133695+00:00",
                                    "route_id": 72,
                                    "route_name": "default",
                                    "route_prefix": "/",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T00:38:05.564381+00:00",
                                                "update_ts": "2019-09-17T22:53:27.139382+00:00",
                                                "upstream_hc_healthythresholdcount": 0,
                                                "upstream_hc_host": "",
                                                "upstream_hc_intervalseconds": 0,
                                                "upstream_hc_path": "/",
                                                "upstream_hc_timeoutseconds": 0,
                                                "upstream_hc_unhealthythresholdcount": 0,
                                                "upstream_id": 16,
                                                "upstream_ip": "172.31.18.10",
                                                "upstream_name": "default",
                                                "upstream_port": 8088,
                                                "upstream_strategy": "",
                                                "upstream_validation_cacertificate": "",
                                                "upstream_validation_subjectname": "",
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-17T22:53:27.139382+00:00"
                                },
                                {
                                    "create_ts": "2019-09-17T22:53:27.14846+00:00",
                                    "route_id": 74,
                                    "route_name": "proxy_admin",
                                    "route_prefix": "/admin",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-17T22:53:27.151782+00:00"
                                },
                                {
                                    "create_ts": "2019-09-17T22:53:27.162287+00:00",
                                    "route_id": 76,
                                    "route_name": "proxy_listeners",
                                    "route_prefix": "/listeners",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-17T22:53:27.165597+00:00"
                                },
                                {
                                    "create_ts": "2019-09-17T22:53:27.171819+00:00",
                                    "route_id": 78,
                                    "route_name": "proxy_clusters",
                                    "route_prefix": "/clusters",
                                    "route_upstreams": [
                                        {
                                            "upstream": {
                                                "create_ts": "2019-09-12T01:58:29.441053+00:00",
                                                "update_ts": "2019-09-17T22:53:27.174974+00:00",
                                                "upstream_hc_healthythresholdcount": null,
                                                "upstream_hc_host": null,
                                                "upstream_hc_intervalseconds": null,
                                                "upstream_hc_path": "/admin",
                                                "upstream_hc_timeoutseconds": null,
                                                "upstream_hc_unhealthythresholdcount": null,
                                                "upstream_id": 21,
                                                "upstream_ip": "127.0.0.1",
                                                "upstream_name": "proxy_admin",
                                                "upstream_port": 9001,
                                                "upstream_strategy": null,
                                                "upstream_validation_cacertificate": null,
                                                "upstream_validation_subjectname": null,
                                                "upstream_weight": 100
                                            }
                                        }
                                    ],
                                    "update_ts": "2019-09-17T22:53:27.174974+00:00"
                                }
                            ],
                            "service_id": 94,
                            "service_name": "petstore-https",
                            "service_secrets": [
                                {
                                    "secret": {
                                        "create_ts": "2019-09-17T23:10:48.299743+00:00",
                                        "secret_cert": "-----BEGIN CERTIFICATE-----
                                          MIIFdDCCBFygAwIBAgISA8jvDUl7GQ1VfzT3GBpMpMAdMA0GCSqGSIb3DQEBCwUA
                                          MEoxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MSMwIQYDVQQD
                                          ExpMZXQncyBFbmNyeXB0IEF1dGhvcml0eSBYMzAeFw0xOTA5MTcyMjA4MjBaFw0x
                                          OTEyMTYyMjA4MjBaMCgxJjAkBgNVBAMTHXBldHN0b3JlLWh0dHBzLmluZ3Jlc3Nw
                                          aXBlLmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvvsGNbgelrwQ
                                          r8x3znQg8GOwae69/FAeUcVwqz+OPieKmvq/Y3OMC5L2gPrEv/yynut3wyxhNRZI
                                          Fm8UV8s1rqiS1LiMumO3FED4etSM3uYRYo7vz6fVWnmdl5Lgw8BH07tS3cC/afI/
                                          ZW6sdnk09PYH/dfYIRf/oyvxXVW5y6gtcU7/ACKKpX5QrIP+0wPyNYftZSley0s7
                                          fV44kZ3LosXWIEILzxvhmNK4OH9yXdvQXL9d30NIydSG2tbwsTWRs6jBaqkMafAN
                                          eypegRwT/sAeOxRe1WSEb6QycyaHYnpF3c45LsSRMgeneq6FphOfwgWYEpzna2iU
                                          ElH06b2QWQIDAQABo4ICdDCCAnAwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG
                                          CCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBRD7RrZ
                                          Ic9CKq2l47JbFhdAxyV90jAfBgNVHSMEGDAWgBSoSmpjBH3duubRObemRWXv86js
                                          oTBvBggrBgEFBQcBAQRjMGEwLgYIKwYBBQUHMAGGImh0dHA6Ly9vY3NwLmludC14
                                          My5sZXRzZW5jcnlwdC5vcmcwLwYIKwYBBQUHMAKGI2h0dHA6Ly9jZXJ0LmludC14
                                          My5sZXRzZW5jcnlwdC5vcmcvMCgGA1UdEQQhMB+CHXBldHN0b3JlLWh0dHBzLmlu
                                          Z3Jlc3NwaXBlLmlvMEwGA1UdIARFMEMwCAYGZ4EMAQIBMDcGCysGAQQBgt8TAQEB
                                          MCgwJgYIKwYBBQUHAgEWGmh0dHA6Ly9jcHMubGV0c2VuY3J5cHQub3JnMIIBBgYK
                                          KwYBBAHWeQIEAgSB9wSB9ADyAHcAdH7agzGtMxCRIZzOJU9CcMK//V5CIAjGNzV5
                                          5hB7zFYAAAFtQXvg3AAABAMASDBGAiEAsLBIXPgQZL8K86PaKKqN2sDvsWgzw7mT
                                          QhJoI6wkY8UCIQDmp69yj3O2IH8TS89ueMaJbe9Ls+fddpIhqcbIEnfsQAB3ACk8
                                          UZZUyDlluqpQ/FgH1Ldvv1h6KXLcpMMM9OVFR/R4AAABbUF74PQAAAQDAEgwRgIh
                                          APkNCQIDsUM6VY/Q8xsaNjTnSNVa9WwrqUvfOObYgSvjAiEA8SNeKn1D7zyMeQuD
                                          3zU1akhs9UtTi6GNnBvh9tPdPAwwDQYJKoZIhvcNAQELBQADggEBAAaR8tbgr7Sk
                                          eF2nHPKvKL1GHVjhH/DRWWyUA1e+EGznXBFCeYJNnfsDtbnF9X6T7IkIqDkvVI5G
                                          QTi6SKNtj/IXOmgibtoCYKSrv7gbxA8t0+4C7H7PyS/+ApO4gHJqG81PdTtQSvIa
                                          G+vJ7EynKASXvj1jpWsRAbws63XVzXvZN4IZPVWWs32FxWTbiNfL8iylbRmoWpr5
                                          iTfvLa+Oh+nGqBtwGMlnTjjwoGIz7lshhGOdhr7tMkf5BS/zOOxRGWaE57eIvQi8
                                          9l3WIRzrrPRKRIqwB72ITg87hCy9DJSwYmtfwe1l4JtWWyrFfXVhhV5kcheSuf5a
                                          Yef9y9ufFZ4=
                                          -----END CERTIFICATE-----
                                          -----BEGIN CERTIFICATE-----
                                          MIIEkjCCA3qgAwIBAgIQCgFBQgAAAVOFc2oLheynCDANBgkqhkiG9w0BAQsFADA/
                                          MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
                                          DkRTVCBSb290IENBIFgzMB4XDTE2MDMxNzE2NDA0NloXDTIxMDMxNzE2NDA0Nlow
                                          SjELMAkGA1UEBhMCVVMxFjAUBgNVBAoTDUxldCdzIEVuY3J5cHQxIzAhBgNVBAMT
                                          GkxldCdzIEVuY3J5cHQgQXV0aG9yaXR5IFgzMIIBIjANBgkqhkiG9w0BAQEFAAOC
                                          AQ8AMIIBCgKCAQEAnNMM8FrlLke3cl03g7NoYzDq1zUmGSXhvb418XCSL7e4S0EF
                                          q6meNQhY7LEqxGiHC6PjdeTm86dicbp5gWAf15Gan/PQeGdxyGkOlZHP/uaZ6WA8
                                          SMx+yk13EiSdRxta67nsHjcAHJyse6cF6s5K671B5TaYucv9bTyWaN8jKkKQDIZ0
                                          Z8h/pZq4UmEUEz9l6YKHy9v6Dlb2honzhT+Xhq+w3Brvaw2VFn3EK6BlspkENnWA
                                          a6xK8xuQSXgvopZPKiAlKQTGdMDQMc2PMTiVFrqoM7hD8bEfwzB/onkxEz0tNvjj
                                          /PIzark5McWvxI0NHWQWM6r6hCm21AvA2H3DkwIDAQABo4IBfTCCAXkwEgYDVR0T
                                          AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwfwYIKwYBBQUHAQEEczBxMDIG
                                          CCsGAQUFBzABhiZodHRwOi8vaXNyZy50cnVzdGlkLm9jc3AuaWRlbnRydXN0LmNv
                                          bTA7BggrBgEFBQcwAoYvaHR0cDovL2FwcHMuaWRlbnRydXN0LmNvbS9yb290cy9k
                                          c3Ryb290Y2F4My5wN2MwHwYDVR0jBBgwFoAUxKexpHsscfrb4UuQdf/EFWCFiRAw
                                          VAYDVR0gBE0wSzAIBgZngQwBAgEwPwYLKwYBBAGC3xMBAQEwMDAuBggrBgEFBQcC
                                          ARYiaHR0cDovL2Nwcy5yb290LXgxLmxldHNlbmNyeXB0Lm9yZzA8BgNVHR8ENTAz
                                          MDGgL6AthitodHRwOi8vY3JsLmlkZW50cnVzdC5jb20vRFNUUk9PVENBWDNDUkwu
                                          Y3JsMB0GA1UdDgQWBBSoSmpjBH3duubRObemRWXv86jsoTANBgkqhkiG9w0BAQsF
                                          AAOCAQEA3TPXEfNjWDjdGBX7CVW+dla5cEilaUcne8IkCJLxWh9KEik3JHRRHGJo
                                          uM2VcGfl96S8TihRzZvoroed6ti6WqEBmtzw3Wodatg+VyOeph4EYpr/1wXKtx8/
                                          wApIvJSwtmVi4MFU5aMqrSDE6ea73Mj2tcMyo5jMd6jmeWUHK8so/joWUoHOUgwu
                                          X4Po1QYz+3dszkDqMp4fklxBwXRsW10KXzPMTZ+sOPAveyxindmjkW8lGy+QsRlG
                                          PfZ+G6Z6h7mjem0Y+iWlkYcV4PIWL1iwBi8saCbGS5jN2p8M+X+Q7UNKEkROb3N6
                                          KOqkqm57TH2H3eDJAkSnh6/DNFu0Qg==
                                          -----END CERTIFICATE-----
                                        ",
                                        "secret_id": 5,
                                        "secret_key": "-----BEGIN PRIVATE KEY-----
                                          MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC++wY1uB6WvBCv
                                          zHfOdCDwY7Bp7r38UB5RxXCrP44+J4qa+r9jc4wLkvaA+sS//LKe63fDLGE1FkgW
                                          bxRXyzWuqJLUuIy6Y7cUQPh61Ize5hFiju/Pp9VaeZ2XkuDDwEfTu1LdwL9p8j9l
                                          bqx2eTT09gf919ghF/+jK/FdVbnLqC1xTv8AIoqlflCsg/7TA/I1h+1lKV7LSzt9
                                          XjiRncuixdYgQgvPG+GY0rg4f3Jd29Bcv13fQ0jJ1Iba1vCxNZGzqMFqqQxp8A17
                                          Kl6BHBP+wB47FF7VZIRvpDJzJodiekXdzjkuxJEyB6d6roWmE5/CBZgSnOdraJQS
                                          UfTpvZBZAgMBAAECggEAI81vZpaztVJgVnSgaSXAHxCxO8qz9x8V8AJxkskBY4mK
                                          JG+pfX1l3a2ZZKieRdebrMs70mz5dDhPH1WHnMXNtIaJsDNAvph+898SNgSuvAKp
                                          c66UKnuuNZ3i+01fsZLUZE8Tw9qkh7oQRHWxAyzJzrpo2R+jtuCG3hIY14SApjr4
                                          EoBddLvaC6GXb2wfzeX2GGw4TBTTaxqeV9sbloq6WyekP3VSvt9Plkc/CoPN0Gab
                                          0bnENEDbX5c6zklItBpVgb+sS/D9au1F8SkyElYutSpiwt7hgSbd+QLSghhcpQL6
                                          U0o4wMc40UggxfdsHeVJuP6s7ygEYsLeDsg3r8IamQKBgQDdWpKa0vDRobky4b6j
                                          jHyxch7WiqrNJCzYXZzb5VHpzHVufs9fCEAjxoHTe4pSdVezgCj010FFVqRG+fCP
                                          eilS2VC0s/VPFl4VmR6cgbaINd+Y0XdyFbaD4Z8do7yg0cscizT+yss1qT4hQs3r
                                          J39uYMaeuAT8lpGHF3j7sczI5wKBgQDc320WbsLQPNQYunxcW8ljjXp+Z954URqb
                                          P5BIQqXLdgG83yIzAL3aIFpeGJ8akd4/cZQpT+oEcNeaEefxVaf8wdC8KrxhGtaW
                                          G6gXRXHGuXoMMzHmSecc79pAQaoFSqOXCxNUlRDeq5CY4tFZaivZvBX4Bhn9XCCr
                                          rf74TV10vwKBgQCW+rk2axyhD9r/TqS2bxN6AOnx0eFQTRVdevSLtC2b9749YLdX
                                          DYyaGkLhGcmuFqV8JLVK0yuM/NzOIJqpclyPSvTWXEy85ffEaY1MmNkErSJW3MDJ
                                          CvBToefi0pTNaGtOi9DY3T+f2VEsZKGJfIZZph6zkbatBpI6f5MgshSJDwKBgFHG
                                          RtUvXOFMJBqjsLdhJEa/csKqIivZm0gvWHPoeQnDPxF2a2sGs0O3Br4fz4g+yVIj
                                          8v74n2PVg31/c6heVju2ZlnEWMp67UfWJX24ME+rDAzIR4lDg1WrV9rCdPhQkhCy
                                          AQ4nwn8udfKkx22baXDLujaBy82J9m6ZlPTJb/hxAoGBAJCJgdL1qrqae6wuq9jW
                                          ua+yN+EAmdsGwKJ7Z47et/nAW5JQcDInmxeyOZcgnd/8NnqBCY2J+niRo+fJhNEk
                                          GS7XA/g4/qciPreAEqRZ1tROVqfHcgFkM3DCAIU5S4qDBAOFmEF/7o9UzN3DH/XP
                                          vRgHw3tvqapryvZcZwujhDvk
                                          -----END PRIVATE KEY-----
                                        ",
                                        "secret_name": "petstore-https-secret",
                                        "update_ts": "2019-09-17T23:23:37.433466+00:00"
                                    }
                                }
                            ],
                            "update_ts": "2019-09-17T23:23:37.433466+00:00"
                        }
                    }
                ],
                "update_ts": "2019-09-17T22:54:26.053712+00:00"
            }
        ]
    }
}
```

##  Test traffic

#### Test petstore app

```
$ curl -vvv petstore.ingresspipe.io
* Rebuilt URL to: petstore.ingresspipe.io/
*   Trying 13.52.165.162...
* TCP_NODELAY set
* Connected to petstore.ingresspipe.io (13.52.165.162) port 80 (#0)
> GET / HTTP/1.1
> Host: petstore.ingresspipe.io
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< last-modified: Mon, 13 Aug 2018 06:26:44 GMT
< accept-ranges: bytes
< content-type: text/html;charset=UTF-8
< content-language: en-US
< content-length: 1832
< date: Mon, 23 Sep 2019 23:02:18 GMT
< x-envoy-upstream-service-time: 2
< server: envoy
<
<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link href="https://fonts.googleapis.com/css?family=Open+Sans:400,700|Source+Code+Pro:300,600|Titillium+Web:400,600,700" rel="stylesheet">
    <link rel="stylesheet" type="text/css" href="./swagger-ui/swagger-ui.css" >
    <link rel="icon" type="image/png" href="./swagger-ui/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="./swagger-ui/favicon-16x16.png" sizes="16x16" />
    <style>
        html
        {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }

        *,
        *:before,
        *:after
        {
            box-sizing: inherit;
        }

        body
        {
            margin:0;
            background: #fafafa;
        }
    </style>
</head>

<body>
<div id="swagger-ui"></div>

<script src="./swagger-ui/swagger-ui-bundle.js"> </script>
<script src="./swagger-ui/swagger-ui-standalone-preset.js"> </script>
<script>
    window.onload = function() {

        // Build a system
        const ui = SwaggerUIBundle({
            url: "/openapi.json",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
            oauth2RedirectUrl: "/swagger-ui/oauth2-redirect.html"
        })

        ui.initOAuth({
            clientId: "sample-client-id",
            clientSecret: "secret",
            scopeSeparator: " "
        })

        window.ui = ui
    }
</script>
</body>
* Connection #0 to host petstore.ingresspipe.io left intact
</html>

```

#### Test petstore https app

```
$ curl -vvv https://petstore-https.ingresspipe.io
* Rebuilt URL to: https://petstore-https.ingresspipe.io/
*   Trying 13.52.165.162...
* TCP_NODELAY set
* Connected to petstore-https.ingresspipe.io (13.52.165.162) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* Cipher selection: ALL:!EXPORT:!EXPORT40:!EXPORT56:!aNULL:!LOW:!RC4:@STRENGTH
* successfully set certificate verify locations:
*   CAfile: /etc/ssl/cert.pem
  CApath: none
* TLSv1.2 (OUT), TLS handshake, Client hello (1):
* TLSv1.2 (IN), TLS handshake, Server hello (2):
* TLSv1.2 (IN), TLS handshake, Certificate (11):
* TLSv1.2 (IN), TLS handshake, Server key exchange (12):
* TLSv1.2 (IN), TLS handshake, Server finished (14):
* TLSv1.2 (OUT), TLS handshake, Client key exchange (16):
* TLSv1.2 (OUT), TLS change cipher, Client hello (1):
* TLSv1.2 (OUT), TLS handshake, Finished (20):
* TLSv1.2 (IN), TLS change cipher, Client hello (1):
* TLSv1.2 (IN), TLS handshake, Finished (20):
* SSL connection using TLSv1.2 / ECDHE-RSA-AES128-GCM-SHA256
* ALPN, server accepted to use h2
* Server certificate:
*  subject: CN=petstore-https.ingresspipe.io
*  start date: Sep 17 22:08:20 2019 GMT
*  expire date: Dec 16 22:08:20 2019 GMT
*  subjectAltName: host "petstore-https.ingresspipe.io" matched cert's "petstore-https.ingresspipe.io"
*  issuer: C=US; O=Let's Encrypt; CN=Let's Encrypt Authority X3
*  SSL certificate verify ok.
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Using Stream ID: 1 (easy handle 0x7fdf42009c00)
> GET / HTTP/2
> Host: petstore-https.ingresspipe.io
> User-Agent: curl/7.54.0
> Accept: */*
>
* Connection state changed (MAX_CONCURRENT_STREAMS updated)!
< HTTP/2 200
< last-modified: Mon, 13 Aug 2018 06:26:44 GMT
< accept-ranges: bytes
< content-type: text/html;charset=UTF-8
< content-language: en-US
< content-length: 1832
< date: Mon, 23 Sep 2019 23:02:31 GMT
< x-envoy-upstream-service-time: 2
< server: envoy
<
<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link href="https://fonts.googleapis.com/css?family=Open+Sans:400,700|Source+Code+Pro:300,600|Titillium+Web:400,600,700" rel="stylesheet">
    <link rel="stylesheet" type="text/css" href="./swagger-ui/swagger-ui.css" >
    <link rel="icon" type="image/png" href="./swagger-ui/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="./swagger-ui/favicon-16x16.png" sizes="16x16" />
    <style>
        html
        {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }

        *,
        *:before,
        *:after
        {
            box-sizing: inherit;
        }

        body
        {
            margin:0;
            background: #fafafa;
        }
    </style>
</head>

<body>
<div id="swagger-ui"></div>

<script src="./swagger-ui/swagger-ui-bundle.js"> </script>
<script src="./swagger-ui/swagger-ui-standalone-preset.js"> </script>
<script>
    window.onload = function() {

        // Build a system
        const ui = SwaggerUIBundle({
            url: "/openapi.json",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
            oauth2RedirectUrl: "/swagger-ui/oauth2-redirect.html"
        })

        ui.initOAuth({
            clientId: "sample-client-id",
            clientSecret: "secret",
            scopeSeparator: " "
        })

        window.ui = ui
    }
</script>
</body>
* Connection #0 to host petstore-https.ingresspipe.io left intact

```
