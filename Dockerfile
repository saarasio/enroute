#FROM golang:1.12.7 AS build
FROM envoyproxy/envoy:latest
RUN apt-get update
#CMD /usr/local/bin/envoy -c /etc/envoy.yaml

# Install golang
RUN apt-get -y install wget
RUN wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.12.9.linux-amd64.tar.gz
ENV PATH "$PATH:/usr/local/go/bin"

WORKDIR /contour

ENV GOPROXY=https://gocenter.io
COPY go.mod ./
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY apis apis
#RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-ldflags=-w go build -o /go/bin/enroute -ldflags=-s -v github.com/saarasio/enroute/cmd/enroute
#
#FROM scratch AS final
#COPY --from=build /go/bin/contour /bin/contour
