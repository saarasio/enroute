/enroute/enroute serve --xds-port=8001 --xds-address=127.0.0.1 --enroute-cp-ip localhost --enroute-cp-port 8081 &
sleep 5
/usr/local/bin/envoy -c /enroute/config.json --service-node "service-node" --service-cluster "service-cluster" --log-level trace 
