PROJECT = enroute-cp
REGISTRY ?= saarasio
IMAGE := $(REGISTRY)/$(PROJECT)
SRCDIRS := ./webapp
PKGS := $(shell GO111MODULE=on go list -mod=readonly ./docs/...)

TAG_LATEST ?= false
GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)
VERSION ?= $(GIT_REF)

export GO111MODULE=on

test:
	go test -mod=readonly ./...

test-race: | test
	go test -race -mod=readonly ./...

install:
	go install -v -tags "oidc gcp" ./...

download:
	go mod download

apidocs:
	go install github.com/swaggo/swag/cmd/swag
	cd ../enroute-cp && swag init -g webhttp.go && cd ..

container:
	docker build . -t $(IMAGE):$(VERSION)
	docker build . -t $(IMAGE):latest

push: container
	docker push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
endif

gofmt:
	@echo Checking code is gofmted
	@test -z "$(shell gofmt -s -l -d -e $(SRCDIRS) | tee /dev/stderr)"
