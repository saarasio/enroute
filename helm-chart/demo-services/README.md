# demo-services

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

Chart for Workloads used in EnRoute demo - httpbin, echo, grpcbin

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| service.echo | object | `{"enable":true,"namespace":"echo","port":9001}` | echo service (websocket): https://github.com/jmalloc/echo-server https://hub.docker.com/r/jmalloc/echo-server |
| service.echo.enable | bool | `true` | enable/disable service installation |
| service.echo.namespace | string | `"echo"` | namespace to install service in |
| service.echo.port | int | `9001` | port on which the (kubernetes clusterip) service is accessible |
| service.grpc | object | `{"enable":true,"namespace":"grpc","port":9002}` | grpc service: https://github.com/moul/grpcbin |
| service.grpc.enable | bool | `true` | enable/disable service installation |
| service.grpc.namespace | string | `"grpc"` | namespace to install service in |
| service.grpc.port | int | `9002` | port on which the (kubernetes clusterip) service is accessible |
| service.httpbin | object | `{"enable":true,"namespace":"httpbin","port":9000}` | httpbin service: httpbin.org |
| service.httpbin.enable | bool | `true` | enable/disable service installation |
| service.httpbin.namespace | string | `"httpbin"` | namespace to install service in |
| service.httpbin.port | int | `9000` | port on which the (kubernetes clusterip) service is accessible |

