# Admin API port (1323 -> 1323) - Run config against this port
# Enroute control plane port (8888 -> 8080) - Provide this port when bootstrapping data plane
docker run -v db_data:/var/lib/postgresql/11/main -p 8888:8888 -p 1323:1323 -e WEBAPP_SECRET="treehugger" gcr.io/enroute-10102020/enroute-cp
