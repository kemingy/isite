TARGET := isite
OUTPUT_DIR := ./bin
CMD_DIR := ./cmd
# Golang dir
GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

.DEFAULT_GOAL:=build

build:
	go build -trimpath -o $(OUTPUT_DIR)/$(TARGET) $(CMD_DIR)/$(TARGET)

format:
	go fmt ./...

lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR)

clean:
	@-rm -rf $(OUTPUT_DIR)
	@-rm -rf output
