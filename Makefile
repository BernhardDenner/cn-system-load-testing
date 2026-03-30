BINARY_NAME := csl-bench
BUILD_DIR   := bin
VERSION     := 0.3.0
IMAGE_NAME  := bernhard97/csl-bench
IMAGE_TAG   := $(VERSION)
GOFLAGS     := -trimpath
LDFLAGS     := -s -w -X main.version=$(VERSION)

.PHONY: all build test lint clean deps bootstrap image

all: build

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/csl-bench

test:
	$(shell go env GOPATH)/bin/ginkgo -r --randomize-all --race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

deps:
	go mod download

image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE_NAME):$(IMAGE_TAG) .

push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

bootstrap:
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
