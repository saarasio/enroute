#FROM golang:1.12.7 AS build
#FROM envoyproxy/envoy:latest
FROM envoyproxy/envoy:v1.11.1
RUN apt-get update
#CMD /usr/local/bin/envoy -c /etc/envoy.yaml

ARG ENROUTE_PATH

# Install golang
RUN apt-get -y install wget
RUN apt-get -y install vim
RUN apt-get -y install curl
RUN wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.12.9.linux-amd64.tar.gz
ENV PATH "$PATH:/usr/local/go/bin"

WORKDIR /enroute

ENV GOPROXY=https://gocenter.io
COPY go.mod ./
RUN go mod download

#TODO: Remove hardcoded copy. This copy assumes that the built binary is in local directory
COPY enroute /enroute

RUN /enroute/enroute bootstrap --xds-address 127.0.0.1 --xds-port 8001 config.json
#CMD /enroute/enroute serve && /usr/local/bin/envoy -c /enroute/config.json --service-node "service-node" --service-cluster "service-cluster"
COPY entrypoint.sh /enroute
ENTRYPOINT /enroute/entrypoint.sh

#COPY cmd cmd
#COPY internal internal
#COPY apis apis
#RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-ldflags=-w go build -o /go/bin/enroute -ldflags=-s -v github.com/saarasio/enroute/cmd/enroute
#
#FROM scratch AS final
#COPY --from=build /go/bin/contour /bin/contour
