TARGET := isite
OUTPUT_DIR := ./bin
CMD_DIR := ./cmd

# Golang dir
ROOT := github.com/kemingy/isite
GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

# Version
GIT_TAG ?= $(shell git describe --tags --dirty=.dirty --always)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)

BUILD_FLAGS ?= -s -w \
  -X $(ROOT)/pkg/version.gitTag=$(GIT_TAG) \
  -X $(ROOT)/pkg/version.buildDate=$(BUILD_DATE) \
  -X $(ROOT)/pkg/version.gitCommit=$(GIT_COMMIT) \

.DEFAULT_GOAL:=build

build:
	@go build -trimpath -o $(OUTPUT_DIR)/$(TARGET) -ldflags "$(BUILD_FLAGS)" $(CMD_DIR)/$(TARGET)

format:
	@go fmt ./...

lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR)

clean:
	@-rm -rf $(OUTPUT_DIR)
	@-rm -rf output

update:
	@go get -u ./...
	@go mod tidy
