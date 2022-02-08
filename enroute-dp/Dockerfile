FROM 			envoyproxy/envoy:v1.21.0
WORKDIR 		/enroute
COPY 			enroute /enroute
COPY 			redis-server /bin
COPY 			entrypoint.sh /enroute

# write bootstrap config
RUN 			/enroute/enroute bootstrap --xds-address 127.0.0.1 --xds-port 8001 config.json
ENTRYPOINT 	/enroute/entrypoint.sh
