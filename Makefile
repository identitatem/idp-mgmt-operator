
# Copyright Contributors to the Open Cluster Management project

export PROJECT_DIR            = $(shell 'pwd')
export PROJECT_NAME			  = $(shell basename ${PROJECT_DIR})

# Image URL to use all building/pushing image targets
IMG ?= ${PROJECT_NAME}:latest
IMG_COVERAGE ?= ${PROJECT_NAME}-coverage:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# enable Go modules
export GO111MODULE=on

all: manager

check: check-copyright

check-copyright:
	@build/check-copyright.sh

test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

run-coverage: generate fmt vet manifests
	go test -covermode=atomic -coverpkg=github.com/identitatem/${PROJECT_NAME}/controllers/... -tags testrunmain -run "^TestRunMain$$" -coverprofile=cover.out . 

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-coverage: manifests
	cd config/manager && kustomize edit set image controller=${IMG_COVERAGE}
	kustomize build config/default | kubectl apply -f -

undeploy:
	kubectl delete --wait=true -k config/default

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

generate-clients: generate
	./hack/update-codegen.sh

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Build the docker image
docker-build-coverage: test docker-build
	docker build . \
	--build-arg DOCKER_BASE_IMAGE=${IMG} \
	-f Dockerfile-coverage \
	-t ${IMG_COVERAGE}


# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

functional-test-full: docker-build-coverage
	@build/run-functional-tests.sh $(IMG_COVERAGE)

functional-test-full-clean: 
	@build/run-functional-tests-clean.sh

functional-test:
	@echo running functional tests
	ginkgo -tags functional -v --slowSpecThreshold=30 test/functional -- -v=5
