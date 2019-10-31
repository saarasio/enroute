+++
date = "2019-10-30T22:01:15+01:00"
title = "Enroute Containers: Enroute-cp, Enroute-dp, Enroute-gw"
tags = ["markdown","prototype", "hugo"]
categories = ["design"]
description = "Enroute Packages"
draft = false
weight = 30
+++

<h3 class="section-head" id="h-livetabs"><a href="#h-livetabs">Enroute Packages</a></h3>
<p>Enroute architecture provides flexibility to deploy it in multiple ways. We describe here the containers that are available to deploy Enroute</p>
<br/>

<p>
Enroute control plane provides the API to configure one or more data planes. 
Enroute data plane connects to the control plane to read its config.
Enroute gateway has control plane and one instance of data plane all packaged as one container. 
</p>

<br/>
<div class="example">
  <nav data-component="tabs" data-live=".tab-live" id="livetabs"></nav>
  <div class="tab-live" data-title="Enroute-cp" id="tab-cp">
    <h5>Enroute-cp</h5>
    <p>Control plane</p>
  </div>
  <div class="tab-live" data-title="Enroute-dp" id="tab-dp">
    <h5>Enroute-dp</h5>
    <p>Data plane</p>
  </div>
  <div class="tab-live" data-title="Enroute-gw" id="tab-gw">
    <h5>Enroute-gw</h5>
    <p>Gateway</p>
  </div>




