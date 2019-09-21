+++
draft= false
title = "FAQ"
description = "frequently asked questions"
+++

## What is Enroute ?

Enroute is a L7 routing controller for Envoy proxies. Complete configuration of multiple Envoy proxies can be managed using Enroute Controller.

## What's the dual plane architecture ?

Enroute is implemented as a control plane that is the central repository for all Envoy config. The data plane component implements the xDS server. The dual plane architecture is also resilient across failure of connectivity between control plane and data plane.

## Enroute vs Service Mesh

Enroute was built to configure Envoy used in multiple roles across an application delivery path. The architecture doesn't force a role on proxy (as a side-car or an edge proxy). Enroute was built to provide generic APIs to realize the target architecture.
