# DeepTalk 服务端构建脚本
# 默认 CGO_ENABLED=0，使用纯 Go 的 SQLite 驱动，可直接交叉编译。
APP        ?= deeptalk
BIN_DIR    ?= bin
UI_DIR     ?= ../ui
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v1.0.0)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

CGO ?= 0
GOFLAGS := -trimpath
LDFLAGS := -s -w \
  -X main.version=$(VERSION) \
  -X main.commit=$(COMMIT) \
  -X main.buildTime=$(BUILD_TIME)

# 依赖走中国镜像；缓存放在仓库内，避免 CI 与沙箱环境写入失败
GOPROXY     ?= https://goproxy.cn,direct
GOSUMDB     ?= sum.golang.google.cn
GOCACHE     ?= $(CURDIR)/../.gocache
GOMODCACHE  ?= $(CURDIR)/../.gomodcache
export GOPROXY GOSUMDB GOCACHE GOMODCACHE

.PHONY: all build build-linux build-linux-arm64 build-windows build-darwin build-all run dev test fmt vet tidy clean ui ui-build embed

all: build

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP) .

build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP)-linux-amd64 .

build-linux-arm64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP)-linux-arm64 .

build-windows:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP)-windows-amd64.exe .

build-darwin:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP)-darwin-arm64 .

build-all: build-linux build-linux-arm64 build-windows build-darwin
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(APP)-darwin-amd64 .

run:
	go run .

dev:
	go run .

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

# 构建两个前端并拷贝到内嵌目录；后端负责输出 index.html
ui-build:
	cd $(UI_DIR) && pnpm install && pnpm build

embed: ui-build
	rm -rf web/dist/c web/dist/b
	mkdir -p web/dist/c web/dist/b
	cp -R $(UI_DIR)/c/dist/. web/dist/c/
	cp -R $(UI_DIR)/b/dist/. web/dist/b/

# 一次性完成前端构建、静态资源内嵌与后端编译
dist: embed build

clean:
	rm -rf $(BIN_DIR)
