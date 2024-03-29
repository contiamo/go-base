# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.5.2. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for git-semver variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(GIT_SEMVER)
#	@echo "Running git-semver"
#	@$(GIT_SEMVER) <flags/args..>
#
GIT_SEMVER := $(GOBIN)/git-semver-v6.0.1
$(GIT_SEMVER): $(BINGO_DIR)/git-semver.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/git-semver-v6.0.1"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=git-semver.mod -o=$(GOBIN)/git-semver-v6.0.1 "github.com/mdomke/git-semver/v6"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.50.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.50.1"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.50.1 "github.com/golangci/golangci-lint/cmd/golangci-lint"

