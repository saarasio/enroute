# Admin API port (80 -> 1323) - Run config against this port
# Enroute control plane port (8080 -> 8080) - Provide this port when bootstrapping data plane
sudo docker run gcr.io/enroute-10102020/enroute-cp -p 80:1323 -p 8080:8080
