BIN_DIR=bin
BIN_CSI=csi-qsd
BIN_QSD_CLI=qsd-client
DIR :=  $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
TEST_DIR=$(DIR)test-qmp

IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
GEN_FILE_PATH=github.com/alicefr/csi-qsd/api/v1

.PHONY: build
build:
	go build -o $(BIN_DIR)/qsd-server ./cmd/qsd
	go build -o $(BIN_DIR)/driver ./cmd/driver
	go build -o $(BIN_DIR)/$(BIN_QMP_CLI) ./cmd/qsd-client
	go build -o $(BIN_DIR)/metadata ./cmd/metadata

.PHONY: test
test:
	@GO111MODULE=on go test -mod=vendor -v ./...

.PHONY: images
images: image-qsd image-driver image-metadata

.PHONY: image-qsd
image-qsd: build
	docker build -t qsd/qsd -f dockerfiles/qsd/Dockerfile .

.PHONY: image-driver
image-driver: build
	docker build -t qsd/driver -f	dockerfiles/driver/Dockerfile .

.PHONY: image-metadata
image-metadata: build
	docker build -t qsd/metadata -f dockerfiles/metadata/Dockerfile .

.PHONY: generate
generate: controller-gen 
	protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative  --experimental_allow_proto3_optional \
	pkg/qsd/qsd.proto \
	pkg/metadata/metadata.proto
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="$(GEN_FILE_PATH)"

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="$(GEN_FILE_PATH)" output:crd:artifacts:config=config/crd/bases



.PHONY: cluster-up
cluster-up:	
	mkdir -p /tmp/kind-cluster-test
	hack/cluster-create.sh

.PHONY: cluster-deploy
cluster-deploy:	images
	hack/cluster-deploy.sh

.PHONY: cluster-down
cluster-down:
	hack/cluster-delete.sh

.PHONY: up
up:
	rm -rf test-qsd
	mkdir -p test-qsd
	docker run --rm -ti \
		--name qsd \
		--security-opt label=disable  \
		-p 4444:4444 -v $(PWD)/test-qsd:/var/run \
		-u $(shell id -u ${USER}):$(shell id -g ${USER}) \
		qsd/qsd

.PHONY: vendor
vendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: clean-test-dir
clean-test-dir:
	@rm -rf $(TEST_DIR)

.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)
	

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

##@ Build
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

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

