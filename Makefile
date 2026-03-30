APP := multi-ping
DIST_DIR := dist
CMD_DIR := ./cmd/$(APP)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -extldflags=-static
TAR ?= tar
SHA256 ?= sha256sum

.PHONY: build build-linux-amd64 build-linux-arm64 package checksums clean

build: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-amd64 $(CMD_DIR)

build-linux-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-arm64 $(CMD_DIR)

package: build
	$(TAR) -C $(DIST_DIR) -czf $(DIST_DIR)/$(APP)-linux-amd64.tar.gz $(APP)-linux-amd64
	$(TAR) -C $(DIST_DIR) -czf $(DIST_DIR)/$(APP)-linux-arm64.tar.gz $(APP)-linux-arm64

checksums: package
	cd $(DIST_DIR) && $(SHA256) $(APP)-linux-amd64.tar.gz $(APP)-linux-arm64.tar.gz > checksums.txt

clean:
	rm -rf $(DIST_DIR)
