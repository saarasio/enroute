sudo docker run -e ENROUTE_NAME=adminproxy -e ENROUTE_CP_IP=127.0.0.1 -e ENROUTE_CP_PORT=8081 -e ENROUTE_CP_PROTO=HTTP --net=host saarasio/enroute:latest
