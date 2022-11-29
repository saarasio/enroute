PROJECT = enroute-dp
REGISTRY ?= saarasio
IMAGE := $(REGISTRY)/$(PROJECT)
SRCDIRS := ./cmd ./internal ./apis
PKGS := $(shell GO111MODULE=on go list -mod=readonly ./cmd/... ./internal/...)

TAG_LATEST ?= false
GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)
VERSION ?= $(GIT_REF)



export GO111MODULE=on

BVAL = $(value BTYPE)
TAGS = "oidc gcp $(BVAL)"
LDFLAGS = "-X main.GitCommit=$(GIT_COMMIT) -X main.MakeTime=$(MAKE_TIME)"


test:
	go test -mod=readonly ./...

test-race: | test
	go test -race -mod=readonly ./...

install:
	$(eval GIT_COMMIT=$(shell git log -1 --pretty=format:"%H"))
	echo GIT_COMMIT '$(GIT_COMMIT)'

	$(eval MAKE_TIME=$(shell date +"+%FT%T%z"))
	echo MAKE_TIME '$(MAKE_TIME)'

	go install -v -tags $(TAGS) -ldflags $(LDFLAGS) ./...

download:
	go mod download

apidocs:
	go install github.com/swaggo/swag/cmd/swag
	cd ../enroute-cp && swag init -g webhttp.go && cd ..

container:
	docker build . -t $(IMAGE):$(VERSION)
	docker build . -t $(IMAGE):latest

updategenerated: ## Update generated CRD code
	@echo Updating generated CRD code...
	@(bash hack/update-generated-crd-code.sh)

push: container
	docker push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
endif

yamllint:
	docker run --rm -ti -v $(CURDIR):/workdir giantswarm/yamllint examples/ site/examples/ 

gofmt:
	@echo Checking code is gofmted
	@test -z "$(shell gofmt -s -l -d -e $(SRCDIRS) | tee /dev/stderr)"

rmcrds:
	kubectl get customresourcedefinitions.apiextensions.k8s.io | grep saaras | cut -d ' ' -f 1 | xargs kubectl delete customresourcedefinitions.apiextensions.k8s.io
