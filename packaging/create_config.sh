#!/bin/bash

set -e

# Variables to access webapp
AUTHORIZATION="treeseverywhere10"
ADMIN_HOST_URL="http://localhost:1323"

# Variables to configure proxy
PROXY_NAME="adminproxy"

# Variables to configure service
SERVICE_NAME="adminservice"
SERVICE_FQDN="enroute-controller.local"

# Variables to configure route
DOCS_ROUTE_NAME="route_docs"
DOCS_ROUTE_PREFIX="/"

# Variables to configure upstream
UPSTREAM_NAME="docs"
UPSTREAM_IP="127.0.0.1"
UPSTREAM_PORT="1313"
UPSTREAM_WEIGHT="100"
UPSTREAM_HC_PATH="/"

# Variables for secret
SECRET_NAME="wildcard-ingresspipe-io"
SECRET_KEY_FILE="/home/ubuntu/enroute/documentation/certs/wildcard.ingresspipe.io.privkey.pem"
SECRET_CERT_FILE="/home/ubuntu/enroute/documentation/certs/wildcard.ingresspipe.io.fullchain.pem"

log() {
    TIMESTAMP=$(date -u "+%Y-%m-%dT%H:%M:%S.000+0000")
    MESSAGE=$1
    echo "{\"timestamp\":\"$TIMESTAMP\",\"level\":\"info\",\"type\":\"startup\",\"detail\":{\"kind\":\"bootstrap-admin\",\"info\":\"$MESSAGE\"}}"
}

create_service_route_upstream() {
	echo "Begin - create_service_route_upstream()"

	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_PROXY_NAME=$3
	local LOCAL_SERVICE_NAME=$4
	local LOCAL_SERVICE_FQDN=$5
	local LOCAL_DOCS_ROUTE_NAME=$6
	local LOCAL_DOCS_ROUTE_PREFIX=$7
	local LOCAL_UPSTREAM_NAME=$8
	local LOCAL_UPSTREAM_IP=$9
	local LOCAL_UPSTREAM_PORT=${10}
	local LOCAL_UPSTREAM_WEIGHT=${11}
	local LOCAL_UPSTREAM_HC_PATH=${12}

	log "CREATE PROXY ${LOCAL_PROXY_NAME}"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/proxy \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" \
		-d '{"Name":"'${LOCAL_PROXY_NAME}'"}' | jq
	
	log "CREATE SERVICE ${LOCAL_SERVICE_NAME}"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/service \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" \
		-d '{"Service_Name":"'${LOCAL_SERVICE_NAME}'", "fqdn":"'${LOCAL_SERVICE_FQDN}'"}' | jq
	
	log "CREATE ROUTE TO SERVE DOCS"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" \
		-d '{"Route_Name":"'${LOCAL_DOCS_ROUTE_NAME}'", "Route_prefix":"'${LOCAL_DOCS_ROUTE_PREFIX}'"}' | jq
	
	log "CREATE UPSTREAM TO SERVE DOCS - IP:${LOCAL_UPSTREAM_IP} PORT:${LOCAL_UPSTREAM_PORT}"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/upstream \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" \
	        -d '{"Upstream_name":"'${LOCAL_UPSTREAM_NAME}'", "Upstream_ip":"'${LOCAL_UPSTREAM_IP}'", "upstream_port":"'${LOCAL_UPSTREAM_PORT}'", "Upstream_hc_path":"'${UPSTREAM_HC_PATH}'", "Upstream_weight":"'${LOCAL_UPSTREAM_WEIGHT}'"}' | jq
	
	log "ATTACH upstream to route"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route/${LOCAL_DOCS_ROUTE_NAME}/upstream/${LOCAL_UPSTREAM_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq
	
	log "Dump service ${LOCAL_SERVICE_NAME}"
	
	curl -s ${LOCAL_ADMIN_HOST_URL}/service/dump/${LOCAL_SERVICE_NAME} -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	echo "End - create_service_route_upstream()"
}

associate_proxy_service() {
	echo "Begin - associate_proxy_service()"

	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_PROXY_NAME=$3
	local LOCAL_SERVICE_NAME=$4

	log "Associate SERVICE ${LOCAL_SERVICE_NAME} <--> PROXY ${LOCAL_PROXY_NAME}"
	
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/proxy/${LOCAL_PROXY_NAME}/service/${LOCAL_SERVICE_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" | jq

	echo "End - associate_proxy_service()"
}


create_secret() {
	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_SECRET_NAME=$3
	local LOCAL_SECRET_KEY_FILE=$4
	local LOCAL_SECRET_CERT_FILE=$5

	log "CREATE SECRET ${LOCAL_SECRET_NAME}"

	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/secret \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" \
		-H "Content-Type: application/json" \
		-d '{"Secret_Name":"'${LOCAL_SECRET_NAME}'"}' | jq

	log "curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/secret/${LOCAL_SECRET_NAME}/key -F 'Secret_key=@'${LOCAL_SECRET_KEY_FILE}"
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/secret/${LOCAL_SECRET_NAME}/key -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" -F 'Secret_key=@'${LOCAL_SECRET_KEY_FILE} | jq

	log "curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/secret/${LOCAL_SECRET_NAME}/cert -F 'Secret_cert=@'${LOCAL_SECRET_CERT_FILE}"
	curl -s -X POST ${LOCAL_ADMIN_HOST_URL}/secret/${LOCAL_SECRET_NAME}/cert -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" -F 'Secret_cert=@'${LOCAL_SECRET_CERT_FILE} | jq
}

list_secret() {
	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	curl -s ${LOCAL_ADMIN_HOST_URL}/secret -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq
}

delete_service_route_upstream() {
	echo "Begin - delete_service_route_upstream()"

	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_PROXY_NAME=$3
	local LOCAL_SERVICE_NAME=$4
	local LOCAL_DOCS_ROUTE_NAME=$5
	local LOCAL_UPSTREAM_NAME=$6

	log "curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route/${LOCAL_DOCS_ROUTE_NAME}/upstream/${LOCAL_UPSTREAM_NAME}"

	curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route/${LOCAL_DOCS_ROUTE_NAME}/upstream/${LOCAL_UPSTREAM_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	log "curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/upstream/${LOCAL_UPSTREAM_NAME}"

	curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/upstream/${LOCAL_UPSTREAM_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	log "curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route/${LOCAL_DOCS_ROUTE_NAME}"

	curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}/route/${LOCAL_DOCS_ROUTE_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	log "curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME}"

	curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/service/${LOCAL_SERVICE_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	log "curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/proxy/${LOCAL_PROXY_NAME}"

	curl -s -X DELETE ${LOCAL_ADMIN_HOST_URL}/proxy/${LOCAL_PROXY_NAME} \
		-H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq

	echo "End - delete_service_route_upstream()"
}

dump_proxy() {
	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2

	log "Dump proxies"
	curl -s ${LOCAL_ADMIN_HOST_URL}/proxy/dump -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq
}

dump_service() {
	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_SERVICE_NAME=$3

	log "Dump service"
	curl -s ${LOCAL_ADMIN_HOST_URL}/service/dump/${LOCAL_SERVICE_NAME} -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq
}

dump_upstream() {
	local LOCAL_AUTHORIZATION=$1
	local LOCAL_ADMIN_HOST_URL=$2
	local LOCAL_UPSTREAM_NAME=$3

	log "Dump upstream"
	curl -s ${LOCAL_ADMIN_HOST_URL}/upstream/${LOCAL_UPSTREAM_NAME} -H "Authorization: Bearer ${LOCAL_AUTHORIZATION}" | jq
}

#create_service_route_upstream ${AUTHORIZATION} ${ADMIN_HOST_URL} ${PROXY_NAME} ${SERVICE_NAME} ${SERVICE_FQDN} ${DOCS_ROUTE_NAME} ${DOCS_ROUTE_PREFIX} ${UPSTREAM_NAME} ${UPSTREAM_IP} ${UPSTREAM_PORT} ${UPSTREAM_WEIGHT} ${UPSTREAM_HC_PATH}
#dump_proxy ${AUTHORIZATION} ${ADMIN_HOST_URL}
#dump_service ${AUTHORIZATION} ${ADMIN_HOST_URL} ${SERVICE_NAME}
#dump_upstream ${AUTHORIZATION} ${ADMIN_HOST_URL} ${UPSTREAM_NAME}
#
#delete_service_route_upstream ${AUTHORIZATION} ${ADMIN_HOST_URL} ${PROXY_NAME} ${SERVICE_NAME} ${DOCS_ROUTE_NAME} ${UPSTREAM_NAME}
#dump_proxy ${AUTHORIZATION} ${ADMIN_HOST_URL}
#dump_service ${AUTHORIZATION} ${ADMIN_HOST_URL} ${SERVICE_NAME}

#create_secret ${AUTHORIZATION} ${ADMIN_HOST_URL} ${SECRET_NAME} ${SECRET_KEY_FILE} ${SECRET_CERT_FILE}
#list_secret ${AUTHORIZATION} ${ADMIN_HOST_URL}

case "$1" in
    create-service-route-upstream)
		create_service_route_upstream ${AUTHORIZATION} ${ADMIN_HOST_URL} ${PROXY_NAME} ${SERVICE_NAME} ${SERVICE_FQDN} ${DOCS_ROUTE_NAME} ${DOCS_ROUTE_PREFIX} ${UPSTREAM_NAME} ${UPSTREAM_IP} ${UPSTREAM_PORT} ${UPSTREAM_WEIGHT} ${UPSTREAM_HC_PATH}
            ;;
    delete-service-route-upstream)
		delete_service_route_upstream ${AUTHORIZATION} ${ADMIN_HOST_URL} ${PROXY_NAME} ${SERVICE_NAME} ${DOCS_ROUTE_NAME} ${UPSTREAM_NAME}
            ;;
    show-service-route-upstream)
		dump_proxy ${AUTHORIZATION} ${ADMIN_HOST_URL}
            ;;
    associate-service-to-proxy)
        associate_proxy_service ${AUTHORIZATION} ${ADMIN_HOST_URL} ${PROXY_NAME} ${SERVICE_NAME}
            ;;
	create-secret)
		create_secret ${AUTHORIZATION} ${ADMIN_HOST_URL} ${SECRET_NAME} ${SECRET_KEY_FILE} ${SECRET_CERT_FILE}
            ;;
	show-secret)
		list_secret ${AUTHORIZATION} ${ADMIN_HOST_URL}
	    ;;
    show-service)
		dump_service ${AUTHORIZATION} ${ADMIN_HOST_URL} ${SERVICE_NAME}
            ;;
               *)
            echo $"Usage: $0 {create-service-route-upstream|delete-service-route-upstream|associate-service-to-proxy|show-service-route-upstream|show-service|show-secret}"
            exit 1
 
esac
