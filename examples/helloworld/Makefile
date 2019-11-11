export GO111MODULE=on

.PHONY: build

build:
	go install -v -tags "oidc gcp" ./...

build-pb:
	protoc -I pb pb/echo.proto --go_out=plugins=grpc:pb
