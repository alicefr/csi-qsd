BIN_DIR=bin
BIN_CSI=csi-qsd
BIN_QSD_CLI=qsd-client
DIR :=  $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
TEST_DIR=$(DIR)test-qmp

.PHONY: build
build:
	go build -o $(BIN_DIR)/qsd-server ./cmd/qsd
	go build -o $(BIN_DIR)/driver ./cmd/driver
	go build -o $(BIN_DIR)/$(BIN_QMP_CLI) ./cmd/qsd-client

.PHONY: test
test:
	@GO111MODULE=on go test -mod=vendor -v ./...

.PHONY: images
images: image-qsd  image-driver

.PHONY: image-qsd
image-qsd: build
	docker build -t qsd/qsd -f dockerfiles/qsd/Dockerfile .

.PHONY: image-driver
image-driver: build
	docker build -t qsd/driver -f	dockerfiles/driver/Dockerfile .

.PHONY: generate
generate:
	protoc --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative  --experimental_allow_proto3_optional \
	pkg/qsd/qsd.proto 

.PHONY: cluster-up
cluster-up:	
	hack/cluster-create.sh

.PHONY: cluster-deploy
cluster-deploy:	images
	hack/cluster-deploy.sh

.PHONY: cluster-down
cluster-down:
	hack/cluster-delete.sh

.PHONY: up
up:
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

