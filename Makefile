BINARY_NAME := csl-bench
BUILD_DIR   := bin
GOFLAGS     := -trimpath
LDFLAGS     := -s -w

.PHONY: all build test lint clean deps bootstrap

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

bootstrap:
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
