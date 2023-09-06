TARGET := isite
OUTPUT_DIR := ./bin
CMD_DIR := ./cmd

build:
	go build -trimpath -o $(OUTPUT_DIR)/$(TARGET) $(CMD_DIR)/$(TARGET)

format:
	go fmt ./...
