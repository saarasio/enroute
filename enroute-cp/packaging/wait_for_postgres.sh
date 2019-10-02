#!/bin/sh

set -e

log() {
    TIMESTAMP=$(date -u "+%Y-%m-%dT%H:%M:%S.000+0000")
    MESSAGE=$1
    echo "{\"timestamp\":\"$TIMESTAMP\",\"level\":\"info\",\"type\":\"startup\",\"detail\":{\"kind\":\"migration-apply\",\"info\":\"$MESSAGE\"}}"
}


SERVER_TIMEOUT=30
POSTGRES_SERVER_PORT=5432

# wait for a port to be ready
wait_for_port() {
    local PORT=$1
    log "waiting $SERVER_TIMEOUT for $PORT to be ready"
    for i in `seq 1 $SERVER_TIMEOUT`;
    do
        nc -z localhost $PORT > /dev/null 2>&1 && log "port $PORT is ready" && return
        sleep 1
    done
    log "failed waiting for $PORT" && exit 1
}

wait_for_port $POSTGRES_SERVER_PORT
