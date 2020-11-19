.GIT_COMMIT=$(shell git rev-parse HEAD)
.GIT_VERSION=$(shell git describe --tags 2>/dev/null || echo "$(.GIT_COMMIT)")
.GIT_UNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(.GIT_UNTRACKEDCHANGES),)
	GITCOMMIT := $(GITCOMMIT)-dirty
endif

# Set an output prefix, which is the local directory if not specified
.PREFIX?=$(shell pwd)

# Setup name variables for the package/tool
.NAME := go-base

# Set the default go compiler
GO := go

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: env
env: ## Print debug information about your local environment
	@echo git: $(shell git version)
	@echo go: $(shell go version)
	@echo golangci-lint: $(shell which golangci-lint)
	@echo gofmt: $(shell which gofmt)
	@echo staticcheck: $(shell which staticcheck)

.PHONY: setup-env
setup-env:
	$(shell go mod download)
	$(shell curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s v1.20.0)

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
	@git --no-pager log --no-merges --pretty=format:"%h : %s (by %an)" $(shell git describe --tags --abbrev=0)...HEAD
	@echo ""

# .PHONY: clean
# clean: ## Cleanup any build binaries or packages
# 	rm -rf bin


.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed
	@echo "+ $@"
	@gofmt -s -l . | tee /dev/stderr

.PHONY: staticcheck
staticcheck: ## Verifies `staticcheck` passes
	@echo "+ $@"
	@staticcheck $(shell $(GO) list ./... | grep -v vendor) | grep -v '.pb.go:' | tee /dev/stderr


.PHONY: test
test: ## Runs the go tests
	@$(MAKE) .run-test-db
	@echo "+ $@"
	-@$(MAKE) .test-ci
	@$(MAKE) .stop-test-db

.PHONY: lint
lint: setup-env ## Verifies `golangci-lint` passes
	@echo "+ $@"
	@./bin/golangci-lint run  ./...




