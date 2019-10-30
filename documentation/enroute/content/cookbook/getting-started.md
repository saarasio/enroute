+++
date = "2017-04-02T22:01:15+01:00"
title = "Getting started with Enroute"
tags = ["markdown","prototype", "hugo"]
categories = ["design"]
description = "API Basics"
draft = false
weight = 20
+++

## What is Enroute?

Enroute is a way to configure and manage multiple Envoy proxies. To achieve this, Enroute is broken up in two sofware components - 

* enroute-cp: Enroute Control Plane 

* enroute-dp: Enroute Data Plane 

Enroute-dp runs along with an Envoy proxy while providing configuration updates to it. It communicates with Enroute-cp to read the config.

Enroute-cp provides a REST API to end-user to configure the data plane Envoys.

## Understanding Enroute through a simple example

In the next few steps, we show how easy it is to configure Envoy. At a high level, we'll perform the following steps -

* Run the Enroute control plane (enroute-cp)
* Create config on enroute-cp: Create Proxy ```proxy-p```, Service ```s```, Route ```r```, Upstream ```u``` 
* Wire up the objects: ``` proxy-p -> s -> r -> u ```
* Run the Enroute data plane (enroute-dp) with name ```proxy-p```. Enroute-dp also runs an Envoy instance along with it.
* The above steps will configure the Envoy proxy with configuration provided on the Enroute control plane

#### Start the enroute control plane - ```enroute-cp```
The controller is packaged as a docker image that can be run using the following command -

```
sudo docker run 					\
    -p 8887:1323 					\
    -p 8888:8888 					\
    -e WEBAPP_SECRET="treeseverywhere" 			\
    -v db_data:/var/lib/postgresql/11/main 		\
    gcr.io/enroute-10102020/enroute-cp:latest
```
There are two port mappings in the above command -
```
    (host:cont)
        -----------
    (8887:1323) Forwards the admin API port on host
    (8888:8888) Forwards port for data-plane to connect to control-plane
```

#### Create objects on controller using API

We associate the IP address of the host on which the control-plane is running to ***enroute-controller.local***. An entry for ***enroute-controller.local*** can either be made in the DNS or the IP address of the controller can be used instead. 

We use the Enroute API to perform the following tasks -

###### Create **Proxy**

```
$ curl -s -X POST "http://enroute-controller.local:8887/proxy"         \
    -H "Authorization: Bearer treeseverywhere"             	       \
    -d 'Name=proxy-p' | jq
{
    "name": "proxy-p"
}
```

###### Create **Service**

```
$ curl -s -X POST "http://enroute-controller.local:8887/service"    	\
    -H "Authorization: Bearer treeseverywhere"             		\
    -d 'Service_Name=l'                         			\
    -d 'fqdn=127.0.0.1' | jq
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
$ curl -s -X POST "http://enroute-controller.local:8887/service/l/route"	\
    -H "Authorization: Bearer treeseverywhere"             			\
    -d 'Route_Name=r'                         					\
    -d 'Route_prefix=/' | jq
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
$ curl -s -X POST "http://enroute-controller.local:8887/upstream"     	\
    -H "Authorization: Bearer treeseverywhere"             		\
    -d 'Upstream_name=u'                         			\
    -d 'Upstream_ip=127.0.0.1'                     			\
    -d 'Upstream_port=9001'                     			\
    -d 'Upstream_hc_path=/'                     			\
    -d 'Upstream_weight=100' | jq
{
    "data": {
        "insert_saaras_db_upstream": {
            "affected_rows": 1
        }
    }
}
```

#### Wire up objects

###### Associate a Route to an upstream : Create Route -> Upstream association

```
$ curl -s -X POST "http://enroute-controller.local:8887/service/l/route/r/upstream/u" \
    -H "Authorization: Bearer treeseverywhere" | jq
{
    "data": {
        "insert_saaras_db_route_upstream": {
            "affected_rows": 4
        }
    }
}
```

###### Associate a Service to a Proxy : Create Proxy -> Service association

```
$ curl -s -X POST "http://enroute-controller.local:8887/proxy/proxy-p/service/l" \
    -H "Authorization: Bearer treeseverywhere" | jq
{
    "data": {
        "insert_saaras_db_proxy_service": {
            "affected_rows": 3
        }
    }
}
```

### Run the data plane ```enroute-dp```

```
sudo docker run                     			\
    -e ENROUTE_NAME=proxy-p             		\
    -e ENROUTE_CP_IP=enroute-controller.local     	\
    -e ENROUTE_CP_PORT=8888             		\
    -e ENROUTE_CP_PROTO=HTTP             		\
    -p 8081:8080                     			\
    -p 9001:9001                     			\
    gcr.io/enroute-10102020/enroute-dp
```

The above command starts envoy, connects to the controller on port ```8888``` and host ```enroute-controller.local```.

It consumes configuration setup for **proxy-p** - which creates a listener ```l```, route ```r```, upstream ```u``` on envoy proxy.

There are two ports forwarded from the ```enroute-dp``` container to the host -
```
 (host:cont)
 -----------
 (8081:8080) - Forwards the listener port to host. The listener l is created on port 8080 in the container
 (9001:9001) - Forwards the envoy admin port
```

#### Test the listener, check envoy stats
Send a request to the listener which gets routed to the envoy admin interface (note the ```Upstream.Upstream_port``` configured to ```9001```)

```
curl 127.0.0.1:8081/admin
```

The envoy admin interface can be also accessed using the following command -

```
curl -vvv 127.0.0.1:9001/admin
```

Note that the curl commands are run on the machine running enroute-dp. Also note that the fqdn is configured to ```127.0.0.1```
