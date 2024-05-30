ORG     ?= $(shell basename $(realpath ..))
PKGS    := $(shell go list ./...)

TAG  ?= $(shell git describe --tags --abbrev=0 HEAD)
LAST = $(shell git describe --tags --abbrev=0 HEAD^)
BODY = "`git log ${LAST}..HEAD --oneline --decorate` `printf '\n\#\#\# [Build Info](${BUILD_URL})'`"
DATE_FMT = +"%Y-%m-%dT%H:%M:%S%z"
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)")
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

# The ldflags for the Go build process to set the version related data
GO_BUILD_VERSION_LDFLAGS=\
  -X go.szostok.io/version.version=$(TAG) \
  -X go.szostok.io/version.buildDate=$(BUILD_DATE) \
  -X go.szostok.io/version.commit=$(shell git rev-parse --short HEAD) \
  -X go.szostok.io/version.commitDate=$(shell git log -1 --date=format:"%Y-%m-%dT%H:%M:%S%z" --format=%cd) \
  -X go.szostok.io/version.dirtyBuild=false

build:
	go build -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" ${TARGETS}
.PHONY: build

fmt:
	go fmt ${PKGS}
.PHONY: fmt

check:
	go vet ${PKGS}
.PHONY: check

static-linux:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=readonly go build -o "dist/hpkl_linux_amd64" -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" ${TARGETS}
.PHONY: static-linux

static-linux-amd64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=readonly go build -o "dist/hpkl_linux_amd64" -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" ${TARGETS}
.PHONY: static-linux-amd64

static-linux-arm64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOFLAGS=-mod=readonly go build -o "dist/hpkl_linux_arm64" -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" ${TARGETS}
.PHONY: static-linux-arm64

install:
	env CGO_ENABLED=0 go install -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" ${TARGETS}
.PHONY: install

clean:
	rm dist/hpkl_*
.PHONY: clean
