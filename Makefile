
# Copyright Red Hat
SHELL := /bin/bash

export PROJECT_DIR            = $(shell 'pwd')
export PROJECT_NAME			  = $(shell basename ${PROJECT_DIR})

# Version to apply to generated artifacts (for bundling/publishing). # This value is set by
# GitHub workflows on push to main and tagging and is not expected to be bumped here.
export VERSION ?= 0.0.1

# Image URL to use all building/pushing image targets
IMAGE_TAG_BASE ?= quay.io/identitatem/$(PROJECT_NAME)
export IMG ?= ${IMAGE_TAG_BASE}:${VERSION}

GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
IMG_COVERAGE ?= ${PROJECT_NAME}-coverage:${GIT_BRANCH}
IMG_E2E_TEST ?= ${PROJECT_NAME}-e2e-test:${GIT_BRANCH}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1"

# Bundle Prereqs
BUNDLE_IMG ?= ${IMAGE_TAG_BASE}-bundle:${VERSION}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# enable Go modules
export GO111MODULE=on

# Catalog Deploy Namespace
CATALOG_DEPLOY_NAMESPACE ?= idp-mgmt-config

# Global things
OS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(shell uname -m | sed 's/x86_64/amd64/g')


# Credentials for Bundle Push
DOCKER_USER ?=
DOCKER_PASS ?=

# For cypress E2E tests
BROWSER ?= chrome

#### UTILITIES #####



# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef


.PHONY: controller-gen
## Find or download controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	@( \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	)
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif


.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.8.7 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

CURL := $(shell which curl 2> /dev/null)
YQ_VERSION ?= v4.5.1
YQ_URL ?= https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(OS)_$(ARCH)
YQ ?= ${PWD}/yq
.PHONY: yq/install
## Install yq to ${YQ} (defaults to current directory)
yq/install: %install:
	@[ -x $(YQ) ] || ( \
		echo "Installing YQ $(YQ_VERSION) ($(YQ_PLATFORM)_$(YQ_ARCH)) from $(YQ_URL)" && \
		curl '-#' -fL -o $(YQ) $(YQ_URL) && \
		chmod +x $(YQ) \
		)
	$(YQ) --version


OPERATOR_SDK ?= ${PWD}/operator-sdk
.PHONY: operatorsdk
## Install operator-sdk to ${OPERATOR_SDK} (defaults to the current directory)
operatorsdk:
	@curl '-#' -fL -o ${OPERATOR_SDK} https://github.com/operator-framework/operator-sdk/releases/download/v1.16.0/operator-sdk_${OS}_${ARCH} && \
		chmod +x ${OPERATOR_SDK}



.PHONY: kubebuilder-tools
## Find or download kubebuilder
kubebuilder-tools:
ifeq (, $(shell which kubebuilder))
	@( \
		set -ex ;\
		KUBEBUILDER_TMP_DIR=$$(mktemp -d) ;\
		cd $$KUBEBUILDER_TMP_DIR ;\
		curl -L -o $$KUBEBUILDER_TMP_DIR/kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/3.1.0/$$(go env GOOS)/$$(go env GOARCH) ;\
		chmod +x $$KUBEBUILDER_TMP_DIR/kubebuilder && mv $$KUBEBUILDER_TMP_DIR/kubebuilder /usr/local/bin/ ;\
	)
endif


OPM = ./bin/opm
.PHONY: opm
## Download opm locally if necessary.
opm:
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.19.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif


# See https://book.kubebuilder.io/reference/envtest.html.
#    kubebuilder 2.3.x contained kubebuilder and etc in a tgz
#    kubebuilder 3.x only had the kubebuilder, not etcd, so we had to download a different way
# After running this make target, you will need to either:
# - export KUBEBUILDER_ASSETS=$HOME/kubebuilder/bin
# OR
# - sudo mv $HOME/kubebuilder /usr/local
#
# This will allow you to run `make test`
.PHONY: envtest-tools
## Install envtest tools to allow you to run `make test`
envtest-tools:
ifeq (, $(shell which etcd))
		@{ \
			set -ex ;\
			ENVTEST_TMP_DIR=$$(mktemp -d) ;\
			cd $$ENVTEST_TMP_DIR ;\
			K8S_VERSION=1.19.2 ;\
			curl -sSLo envtest-bins.tar.gz https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$$K8S_VERSION-$$(go env GOOS)-$$(go env GOARCH).tar.gz ;\
			tar xf envtest-bins.tar.gz ;\
			mv $$ENVTEST_TMP_DIR/kubebuilder $$HOME ;\
			rm -rf $$ENVTEST_TMP_DIR ;\
		}
endif



#### BUNDLING AND PUBLISHING ####



# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=quay.io/identitatem/idp-mgmt-config-bundle:0.1.1,quay.io/identitatem/idp-mgmt-config-bundle:0.0.2).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=quay.io/identitatem/idp-mgmt-config-catalog:0.1.1).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set PREV_BUNDLE_INDEX_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin PREV_BUNDLE_INDEX_IMG), undefined)
FROM_INDEX_OPT := --from-index $(PREV_BUNDLE_INDEX_IMG)
endif

.PHONY: publish
## Build and push the operator, bundle, and catalog
publish: docker-login docker-build docker-push patch-bundle-version bundle-build bundle-push catalog-build catalog-push
	if [[ "${PUSH_LATEST}" = true ]]; then \
		echo "Tagging operator image as latest and pushing"; \
		$(MAKE) docker-push-latest; \
	fi;

.PHONY: docker-login
## Log in to the docker registry for ${BUNDLE_IMG}
docker-login:
	@docker login ${BUNDLE_IMG} -u ${DOCKER_USER} -p ${DOCKER_PASS}


.PHONY: bundle
## Generate bundle manifests and metadata, patch the webhook deployment name, then validate generated files [NOTE: validate bundle is skipped for now].
bundle: manifests kustomize yq/install operatorsdk
	echo IMG=${IMG}
	${OPERATOR_SDK} generate kustomize manifests --interactive=false -q
	$(eval REPLACES := $(shell echo ${PREV_BUNDLE_INDEX_IMG} | cut -d : -f 2))
	echo ${REPLACES}
	if [[ -n "${REPLACES}" ]]; then \
	echo idp-mgmt-operator.${REPLACES}; \
	  sed -i.bak "s/PREV_CATALOG_VERSION/idp-mgmt-operator.${REPLACES}/g" config/manifests/bases/idp-mgmt-operator.clusterserviceversion.yaml; \
	else \
	  sed -i.bak "s/PREV_CATALOG_VERSION//g" config/manifests/bases/idp-mgmt-operator.clusterserviceversion.yaml; \
	fi;
	cd config/installer && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | ${OPERATOR_SDK} generate bundle -q --overwrite --version $(VERSION)
	mv config/manifests/bases/idp-mgmt-operator.clusterserviceversion.yaml.bak config/manifests/bases/idp-mgmt-operator.clusterserviceversion.yaml


## Patch up the bundle manifest wtih correct image, version and replaces
.PHONY: patch-bundle-version
patch-bundle-version: yq/install
	${YQ} eval -i '.spec.install.spec.deployments[0].spec.template.spec.containers[0].image = env(IMG)' bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml
	${YQ} eval -i '.spec.version = env(VERSION)' bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml
	BUNDLE_NAME=$(shell echo "idp-mgmt-operator.v${VERSION}") ${YQ} eval -i '.metadata.name = env(BUNDLE_NAME)' bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml
	if [[ -z "${PREV_BUNDLE_INDEX_IMG}" ]]; then \
		sed -i.bak '/replaces: idp-mgmt-operator\.v/d' bundle/manifests/idp-mgmt-operator.clusterserviceversion.yaml; \
	fi;

.PHONY: bundle-build
## Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .


.PHONY: bundle-push
## Push the bundle image.
bundle-push: docker-login
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)


# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
## Build a catalog image.
catalog-build: opm
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)


.PHONY: catalog-push
## Push a catalog image.
catalog-push: docker-login
	$(MAKE) docker-push IMG=$(CATALOG_IMG)


.PHONY: deploy-catalog
## Deploy the catalogsource to a cluster
deploy-catalog:
	@cat bundle-deploy/catalogsource.yaml \
		| sed -e "s;__CATALOG_DEPLOY_NAMESPACE__;${CATALOG_DEPLOY_NAMESPACE};g" -e "s;__CATALOG_IMG__;${CATALOG_IMG};g" > .tmp_catalog.yaml; \
		kubectl apply -f .tmp_catalog.yaml; \
		rm -f .tmp_catalog.yaml



#### BUILD, TEST, AND DEPLOY ####



all: manager

check: check-copyright

check-copyright:
	@build/check-copyright.sh

test: fmt vet manifests
	@go test ./... -coverprofile cover.out -coverpkg ./... &&\
	COVERAGE=`go tool cover -func="cover.out" | grep "total:" | awk '{ print $$3 }' | sed 's/[][()><%]/ /g'` &&\
	echo "-------------------------------------------------------------------------" &&\
	echo "TOTAL COVERAGE IS $$COVERAGE%" &&\
	echo "-------------------------------------------------------------------------" &&\
	go tool cover -html "cover.out" -o ${PROJECT_DIR}/cover.html

# Build manager binary
manager: fmt vet
	go build -o bin/idp-mgmt main.go


# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet manifests
	go run ./main.go


run-coverage: fmt vet manifests
	go test -covermode=atomic -coverpkg=github.com/identitatem/${PROJECT_NAME}/controllers/... -tags testrunmain -run "^TestRunMain$$" -coverprofile=cover.out .

# Install CRDs into a cluster
install: manifests
	$(KUSTOMIZE) build config/crd | kubectl apply -f -


# Uninstall CRDs from a cluster
uninstall: manifests
	$(KUSTOMIZE) build config/crd | kubectl delete -f -


# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: kustomize manifests
	cp config/installer/kustomization.yaml config/installer/kustomization.yaml.tmp
	cd config/installer && kustomize edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	mv config/installer/kustomization.yaml.tmp config/installer/kustomization.yaml


# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-coverage: kustomize manifests
	cp config/installer-coverage/kustomization.yaml config/installer-coverage/kustomization.yaml.tmp
	cd config/installer-coverage && $(KUSTOMIZE) edit set image controller=${IMG_COVERAGE}
	$(KUSTOMIZE) build config/default-coverage | kubectl apply -f -
	mv config/installer-coverage/kustomization.yaml.tmp config/installer-coverage/kustomization.yaml


undeploy:
	kubectl delete --wait=true -k config/default


undeploy-coverage:
	kubectl delete --wait=true -k config/default-coverage


# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen yq/install
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." && \
	${YQ} e '.metadata.name = "idp-mgmt-operator-manager-role"' config/rbac/role.yaml > deploy/idp-mgmt-operator/clusterrole.yaml && \
	${YQ} e '.metadata.name = "leader-election-operator-role" | .metadata.namespace = "{{ .Namespace }}"' config/rbac/leader_election_role.yaml > deploy/idp-mgmt-operator/leader_election_role.yaml

# Run go fmt against code
fmt:
	go fmt ./...


# Run go vet against code
vet:
	go vet ./...


# Build the docker image
docker-build: test
	docker build . -t ${IMG}


# Build the docker image
docker-build-coverage: docker-build
	docker build . \
	--build-arg DOCKER_BASE_IMAGE=${IMG} \
	-f Dockerfile-coverage \
	-t ${IMG_COVERAGE}

# Push the docker image
docker-push:
	docker push ${IMG}

# Tag the IMG as latest and docker push
docker-push-latest:
	docker tag ${IMG} ${IMAGE_TAG_BASE}:latest
	$(MAKE) docker-push IMG=${IMAGE_TAG_BASE}:latest

functional-test-crds:
	@for FILE in "test/config/crd/external"; do kubectl apply -f $$FILE;done


functional-test-full: docker-build-coverage
	@build/run-functional-tests.sh $(IMG_COVERAGE)


functional-test-full-clean:
	@build/run-functional-tests-clean.sh


functional-test:
	@echo running functional tests
	ginkgo -tags functional -v --slowSpecThreshold=30 test/functional -- -v=5

.PHONY: build-e2e-test-image
build-e2e-test-image:
	@echo "Building $(IMAGE_E2E_TEST)"
	docker build . \
	-f Dockerfile.cypress \
	-t ${IMG_E2E_TEST}

#.PHONY: run-e2e-test-image
#run-e2e-test-image:
#	docker run \
#	-e BROWSER=$(BROWSER) \
	-v $(shell pwd)/results/:/results/ \
	${IMG_E2E_TEST}

e2e-ginkgo-test:
	@echo running e2e ginkgo tests
	ginkgo -tags e2e -v test/e2e -- -v=5
	