# Set shell to bash explicitly
SHELL := /bin/bash

LOCAL_BIN := $(CURDIR)/bin
RUSH_BIN := $(LOCAL_BIN)/rush
GOLANGCI_BIN := $(LOCAL_BIN)/golangci-lint
GOLANGCI_TAG := 1.63.4
BUILD_ENVPARMS := CGO_ENABLED=0
BUILD_GOOS := $(shell go env GOHOSTOS)
BUILD_GOARCH := $(shell go env GOHOSTARCH)
UNAME := $(shell uname)
PROJECT_ID := nvme-exporter
CMD_LIST := $(shell ls ./cmd | sed -e 's/^/.\/cmd\//')
BUILD_SUFFIX_LINUX := -linux-amd64

install-lint:
	$(info Installing golangci-lint v$(GOLANGCI_TAG))
	@GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_TAG)

install-rush:
ifeq ($(wildcard $(RUSH_BIN)),)
	$(info Installing rush)
	@GOBIN=$(LOCAL_BIN) go install github.com/shenwei356/rush@v0.5.6
endif

.bin-deps: install-rush install-lint
bin-deps: .bin-deps

.lint:
	$(info Running lint against changed files...)
	@$(GOLANGCI_BIN) run \
		--config=.golangci.yaml \
		--sort-results \
		--max-issues-per-linter=1000 \
		--max-same-issues=1000 \
		./...

lint: .lint

.test:
	$(info Running tests...)
	@go test ./...

test: .test

.build:
	echo $(CMD_LIST) | $(RUSH_BIN) \
		--keep-order \
		--immediate-output \
		--jobs 4 \
		--record-delimiter=" " \
		--trim=b \
		'$(BUILD_ENVPARMS) GOOS=$(BUILD_GOOS) GOARCH=$(BUILD_GOARCH) go build -o="$(LOCAL_BIN)/{%}$(BUILD_SUFFIX)" {}'

build: .build

.build-linux: BUILD_GOOS=linux
.build-linux: BUILD_GOARCH=amd64
.build-linux: BUILD_SUFFIX=$(BUILD_SUFFIX_LINUX)
.build-linux: build

build-linux: .build-linux
	s3cmd -q -c ~/.aws/minio-island put bin/*$(BUILD_SUFFIX_LINUX) s3://binary

.PHONY: \
	install-lint install-rush \
	.bin-deps bin-deps \
	.lint lint \
	.test test \
	.build build \
	.build-linux build-linux
