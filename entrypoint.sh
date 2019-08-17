/enroute/enroute serve --xds-port=8000 --xds-address=127.0.0.1 &
/usr/local/bin/envoy -c /enroute/config.json --service-node "service-node" --service-cluster "service-cluster"
