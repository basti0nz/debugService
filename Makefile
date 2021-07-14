EXE  := igpu-debug
PKG  := github.com/basti0nz/debugService
VER := 0.1
PATH := build:$(PATH)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

ifneq (,$(wildcard ./version))
    include version
    export
endif


$(EXE): go.mod *.go lib/*.go
	go build -v -ldflags "-X main.version=$(VER)" -o ./dist/$@ 


.PHONY: darwin linux
darwin linux:
	GOOS=$@ go build  -o ./dist/$(EXE)-$(VER)-$@-$(GOARCH)

.PHONY: clean
clean:
	rm -rf ./dist/



