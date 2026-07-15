PROJECTNAME=$(shell basename "$(PWD)")
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT_ID?=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-X main.Version=$(VERSION) -X main.CommitID=$(COMMIT_ID) -X main.BuildTime=$(BUILD_TIME)
SOL_DIR=./solidity

CENT_EMITTER_ADDR?=0x1
CENT_CHAIN_ID?=0x1
CENT_TO?=0x1234567890
CENT_TOKEN_ID?=0x5
CENT_METADATA?=0x0

.PHONY: help build install
all: help

help: Makefile
	@echo
	@echo "Choose a make command to run in "$(PROJECTNAME)":"
	@echo
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
	@echo

get:
	@echo "  >  \033[32mDownloading & Installing all the modules...\033[0m "
	go mod tidy && go mod download

build:
	@echo "  >  \033[32mBuilding compass...\033[0m "
	cd cmd/compass && go build -ldflags "$(LDFLAGS)" -o ../../build/compass

dev:
	@echo "  >  \033[32mBuilding compass-dev...\033[0m "
	cd cmd/compass && env GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o ../../build/compass-dev

install:
	@echo "  >  \033[32mInstalling compass...\033[0m "
	cd cmd/compass && go install -ldflags "$(LDFLAGS)"
