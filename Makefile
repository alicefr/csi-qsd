BIN_DIR=bin
BIN_CSI=csi-qsd
BIN_QSD_CLI=qsd-client
DIR :=  $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
TEST_DIR=$(DIR)test-qmp

.PHONY: build
build: clean-$(BIN_DIR)/$(BIN_CSI)
	go build -o $(BIN_DIR)/qsd-server ./cmd/qsd
	go build -o $(BIN_DIR)/driver ./cmd/driver

.PHONY: build-$(BIN_QSD_CLI)
build-client: 
	go build -o $(BIN_DIR)/$(BIN_QMP_CLI) ./qsd-client

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

clean-$(BIN_DIR)/$(BIN_CSI):
	rm -f $(BIN_DIR)/$(BIN_CSI)
