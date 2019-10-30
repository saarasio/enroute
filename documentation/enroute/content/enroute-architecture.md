+++
date = "2017-04-02T22:01:15+01:00"
title = "Enroute Architecture"
tags = ["markdown","prototype", "hugo"]
categories = ["design"]
description = "Enroute Internals"
draft = false
weight = 30
+++


<h3> How does Enroute simplify config? </h3>
<p> 
Enroute provides for a way to program multiple Envoy proxies. It defines well known abstractions like proxy, service, route, upstream and secret. It provides an API to work with these abstractions.  
</p>

<p>
The configuration once defined, can be associated with one or more proxies.
</p>

<!-- <a href=""><img alt="Enroute" src="/img/EnrouteGettingStartedAPI.png"></a> -->
<a href=""><img alt="Enroute" src="/img/EnrouteGettingStartedAPI2.png"></a>

<p>
This approach allows Enroute to integrate with cloud infrastructure, discovery services and secret stores. 

</p>

<p>
Enroute lets a user track configuration changes over a period of time. Its config backup and restore let developer operations keep track of configuration changes and treat infrastructure as code.
</p>

<h3> Enroute cloud integration hooks </h3>

<p>
Enroute is built to integrate with external cloud service discovery. Additionally it provides configuration capability to import secrets from a secret store into Envoy.
</p>

<p>
Application logic can then be inserted with artifacts and services discovered from the environment in which Enroute is running
</p>

<a href=""><img alt="Enroute" src="/img/CloudIntegration.png"></a>

