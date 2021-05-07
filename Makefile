
include .bingo/Variables.mk

# fallback just incase gitsemver doesn't exist
# note that `?=` means it will not run these commands
# if the value is already set
.GIT_VERSION ?= $(shell bash scripts/git-semver.sh)
.GIT_COMMIT ?= $(shell git rev-parse HEAD)

.PHONY: version
version:
	@echo $(.GIT_VERSION)

# Set an output prefix, which is the local directory if not specified
.PREFIX?=$(shell pwd)

# Setup name variables for the package/tool
.NAME := go-base

# Set the default go compiler
GO := go

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

env: ## Print debug information about your local environment
	@printf '%-15s:	 %s\n' 'git' "$(shell git version)" \
	&& printf '%-15s:	 %s\n' 'docker' "$(shell docker --version)" \
	&& printf '%-15s:	 %s\n' 'go' "$(shell go version)" \
	&& printf '%-15s:	 %s\n' 'gofmt' "$(shell which gofmt)" \
	&& printf '%-15s:	 %s\n' 'golangci-lint'	"$(GOLANGCI_LINT)" \
	&& printf '%-15s:	 %s\n' 'git-semver' "$(GIT_SEMVER)" \
	&& printf '%-15s:	 %s\n' 'app version' "$(.GIT_VERSION)"

.PHONY: setup-env
setup-env: $(GIT_SEMVER) $(GOLANGCI_LINT) ## Setup dev environment
	$(shell go mod download)

.PHONY: .run-test-db
.run-test-db: ## start the test db
	@echo "+ setup test db"
	@docker run --rm -d --name go-base-postgres \
	-p 0.0.0.0:5432:5432 \
	-e POSTGRES_PASSWORD=$$(cat pkg/db/test/password) \
	-e POSTGRES_USER=contiamo_test \
	-e POSTGRES_DB=postgres \
	postgres:alpine -c fsync=off -c full_page_writes=off -c synchronous_commit=off &>/dev/null
	@bash -c "export PGPASSWORD=$$(cat pkg/db/test/password); until psql -q -Ucontiamo_test -l -h localhost &>/dev/null; do echo -n .; sleep 1; done"
	@echo ""
	@echo "+ test db started"

.PHONY: .stop-test-db
.stop-test-db:  ## teardown the test db
	@echo "+ teardown test db"
	@docker rm -v -f go-base-postgres &>/dev/null

.PHONY: .test-ci
.test-ci:
	go test -v -cover ./...

.PHONY: changelog
changelog: ## Print git hitstory based changelog
	@./scripts/changelog.sh
	@echo ""

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed
	@echo "+ $@"
	@gofmt -s -l . | tee /dev/stderr

.PHONY: staticcheck
staticcheck: ## Verifies `staticcheck` passes
	@echo "+ $@"
	@staticcheck $(shell $(GO) list ./... | grep -v vendor) | grep -v '.pb.go:' | tee /dev/stderr


.PHONY: test
test: lint ## Runs the go tests
	@$(MAKE) .run-test-db
	@echo "+ $@"
	-@$(MAKE) .test-ci
	@$(MAKE) .stop-test-db

.PHONY: lint
lint: setup-env $(GOLANGCI_LINT) ## Verifies `golangci-lint` passes
	@echo "+ $@"
	@$(GOLANGCI_LINT) run ./...

.PHONY: lint-fix
lint-fix: setup-env $(GOLANGCI_LINT) ## Verifies `golangci-lint` passes
	@echo "+ $@"
	@$(GOLANGCI_LINT) run --fix ./...
