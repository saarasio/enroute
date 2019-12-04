![](enroute.png)

# Enroute
Enroute is lightweight control plane to stream xDS (LDS, RDS, CDS, EDS, SDS) config to Envoy proxies.

Enroute is built as a light weight controller that can be deployed anywhere - on premise, on the cloud, inside kubernetes, outside kubernetes, greenfield and brownfield deployments. Enroute is minimal functionality to run in all these environments.

The architecture doesn't force a role on proxy (as a side-car, an intermediate or an edge proxy). Enroute was built to provide generic APIs to realize the target architecture.

This flexibility allows Enroute to assume the role of a generic control plane or an API gateway built.

Blogs, Cookbooks, getting started, examples and additional documentation can be found at - [getenroute.io](https://getenroute.io)

