#!/bin/bash

set -e

log() {
    TIMESTAMP=$(date -u "+%Y-%m-%dT%H:%M:%S.000+0000")
    MESSAGE=$1
    echo "{\"timestamp\":\"$TIMESTAMP\",\"level\":\"info\",\"type\":\"startup\",\"detail\":{\"kind\":\"migration-apply\",\"info\":\"$MESSAGE\"}}"
}

if [ -z ${HASURA_GRAPHQL_MIGRATIONS_SERVER_TIMEOUT+x} ]; then
    log "server timeout is not set defaulting to 60 seconds"
    HASURA_GRAPHQL_MIGRATIONS_SERVER_TIMEOUT=60
fi


# wait for a port to be ready
wait_for_port() {
    local PORT=$1
    log "waiting $HASURA_GRAPHQL_MIGRATIONS_SERVER_TIMEOUT for $PORT to be ready"
    for i in `seq 1 $HASURA_GRAPHQL_MIGRATIONS_SERVER_TIMEOUT`;
    do
        nc -z localhost $PORT > /dev/null 2>&1 && log "port $PORT is ready" && return
        sleep 1
    done
    log "failed waiting for $PORT" && exit 1
}

wait_for_port 1323
wait_for_port 8888
wait_for_port 6379

/bin/enroute bootstrap --xds-address 127.0.0.1 --xds-port 8001 /supervisord/config.json

/bin/enroute serve --xds-port=8001 --xds-address=127.0.0.1 --enroute-cp-ip localhost --enroute-cp-port 8888 --enroute-cp-proto http --enroute-name gw --enable-ratelimit &

/bin/envoy -c /supervisord/config.json --service-node "service-node-enroute-gw" --service-cluster "gw" --log-level trace
