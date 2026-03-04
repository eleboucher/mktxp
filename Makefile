BINARY_NAME=mktxp
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/eleboucher/mktxp/internal/version.Version=$(VERSION) \
				 -X github.com/eleboucher/mktxp/internal/version.GitCommit=$(GIT_COMMIT) \
				 -X github.com/eleboucher/mktxp/internal/version.BuildDate=$(BUILD_DATE)"
BUILD_DIR=./build

.PHONY: build test lint clean docker

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

docker:
	docker build -t mktxp-go:$(VERSION) .
