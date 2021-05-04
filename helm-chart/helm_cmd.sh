helm install --set ociSettings.enable=true --set filters.jwt.enable=true --set filters.cors.enable=true --set envoySettings.logLevel=trace enroute-demo ./enroute
