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

if [ -z "$ENROUTE_CP_PROTO" ]
then
	echo "\$ENROUTE_CP_PROTO is not set. Please set before running"
	exit	1
else
	echo "\$ENROUTE_CP_PROTO set to $ENROUTE_CP_PROTO"
fi

#log() {
#    TIMESTAMP=$(date -u "+%Y-%m-%dT%H:%M:%S.000+0000")
#    MESSAGE=$1
#    echo "{\"timestamp\":\"$TIMESTAMP\",\"level\":\"info\",\"type\":\"startup\",\"detail\":{\"kind\":\"enroute-dp\",\"info\":\"$MESSAGE\"}}"
#}
#
#
## wait for a port to be ready
#wait_for_port() {
#    local PORT=$1
#    local TIMEOUT=15
#    log "waiting for $PORT to be ready"
#    for i in `seq 1 TIMEOUT`;
#    do
#        nc -z localhost $PORT > /dev/null 2>&1 && log "port $PORT is ready" && return
#        sleep 1
#    done
#    log "failed waiting for $PORT" && exit 1
#}
#
# Start redis server and make it available for envoy and enroute
/bin/redis-server --port 6379 --loglevel verbose &
# TODO: Test this
# wait_for_port 6379
sleep 5
/enroute/enroute serve --xds-port=8001 --xds-address=127.0.0.1 --enroute-cp-ip $ENROUTE_CP_IP --enroute-cp-port $ENROUTE_CP_PORT --enroute-cp-proto $ENROUTE_CP_PROTO --enroute-name $ENROUTE_NAME --enable-ratelimit &
sleep 5
/usr/local/bin/envoy -c /enroute/config.json --service-node "service-node" --service-cluster "$ENROUTE_NAME" --log-level trace 
