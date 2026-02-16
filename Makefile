.PHONY: test test-cover clean build-plugin

BINARY := statusline
PLUGIN_DIR := plugins/statusline/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build-plugin:
	mkdir -p $(PLUGIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(PLUGIN_DIR)/$(BINARY)-darwin-amd64 ./cmd/statusline
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(PLUGIN_DIR)/$(BINARY)-darwin-arm64 ./cmd/statusline
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(PLUGIN_DIR)/$(BINARY)-linux-amd64 ./cmd/statusline
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(PLUGIN_DIR)/$(BINARY)-linux-arm64 ./cmd/statusline

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(PLUGIN_DIR) coverage.out coverage.html
