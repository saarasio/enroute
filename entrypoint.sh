if [ -z "$ENROUTE_NAME" ]
then
	echo "\$ENROUTE_NAME is not set. Please set before running"
	exit	1
else
	echo "\$ENROUTE_NAME set to $ENROUTE_NAME"
fi

if [ -z "$ENROUTE_CP_IP" ]
then
	echo "\$ENROUTE_CP_IP is not set. Please set before running"
	exit	1
else
	echo "\$ENROUTE_CP_IP set to $ENROUTE_CP_IP"
fi

if [ -z "$ENROUTE_CP_PORT" ]
then
	echo "\$ENROUTE_CP_PORT is not set. Please set before running"
	exit	1
else
	echo "\$ENROUTE_CP_PORT set to $ENROUTE_CP_PORT"
fi

/enroute/enroute serve --xds-port=8001 --xds-address=127.0.0.1 --enroute-cp-ip $ENROUTE_CP_IP --enroute-cp-port $ENROUTE_CP_PORT --enroute-cp-proto $ENROUTE_CP_PROTO --enroute-name $ENROUTE_NAME &
sleep 5
/usr/local/bin/envoy -c /enroute/config.json --service-node "service-node" --service-cluster "$ENROUTE_NAME" --log-level trace 
