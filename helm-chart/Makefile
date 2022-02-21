options:
	@echo "Usage:"
	@echo "make [helm-package-enroute]        - build helm package for enroute"
	@echo "make [helm-package-service-policy] - build helm package for service-policy"
	@echo "make [helm-package-index]          - build helm index for packages"
	@echo "make [all]                         - build helm package fror enroute, enroute-crds and build index"

helm-package-enroute:
	helm package ./enroute

helm-package-service-policy:
	helm package ./service-policy

helm-package-index:
	helm repo index .

helm-lint:
	cd enroute && helm lint . && cd .. && cd service-policy && helm lint . && cd ..

helm-template:
	cd enroute && helm template . > ../enroute-out.yaml && cd ../service-policy && helm template . > ../service-policy-out.yaml && cd ..

all: helm-package-enroute helm-package-index helm-package-service-policy helm-lint
