SHELL = /bin/bash

.DEFAULT_GOAL := help

CURRENT_DIR := $(shell pwd)

# Docker metadata
GIT_HASH = `git rev-parse HEAD`
GIT_HASH_SHORT = `git rev-parse --short HEAD`
GIT_BRANCH = `git symbolic-ref HEAD --short 2>/dev/null`
GIT_DIRTY = `git status --porcelain`
GIT_TAG = `git describe --tags || echo "no version info"`
AUTHOR = $(USER)

# Docker variables
DOCKER_REGISTRY = 974517877189.dkr.ecr.eu-central-1.amazonaws.com
E2E_TESTS_DOCKER_IMG_LOCAL_TAG := $(DOCKER_REGISTRY)/tool-golang-bgdi/e2e-tests:local-$(USER)-$(GIT_HASH_SHORT)

# AWS variables
AWS_DEFAULT_REGION = eu-central-1


all: help


.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo
	@echo "Possible targets:"
	@echo "- setup                  Install dependencies"
	@echo -e " \033[1mFORMATING, LINTING AND TESTING TOOLS TARGETS\033[0m "
	@echo "- format             	Format the go source code"
	@echo "- lint               	Lint the go source code"
	@echo -e " \033[1mDocker TARGETS\033[0m "
	@echo "- dockerlogin        	Login to the AWS ECR registry for pulling/pushing docker images"
	@echo "- dockerbuild-e2e-tests  Build the tool e2e-tests locally (with tag := $(E2E_TESTS_DOCKER_IMG_LOCAL_TAG))"
	@echo "- dockerpush-e2e-tests   Build and push the tool e2e-tests (with tag := $(E2E_TESTS_DOCKER_IMG_LOCAL_TAG))"


.PHONY: setup
setup:
	go mod tidy
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest


# linting target, calls upon yapf to make sure your code is easier to read and respects some conventions.

.PHONY: format
format:
	goimports -w .


.PHONY: lint
lint:
	golangci-lint run
	govulncheck -show verbose ./...


# Docker related functions.

.PHONY: dockerlogin
dockerlogin:
	aws --profile swisstopo-bgdi-builder ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(DOCKER_REGISTRY)


.PHONY: dockerbuild-e2e-tests
dockerbuild-e2e-tests:
	docker build \
		--build-arg GIT_HASH="$(GIT_HASH)" \
		--build-arg GIT_BRANCH="$(GIT_BRANCH)" \
		--build-arg GIT_DIRTY="$(GIT_DIRTY)" \
		--build-arg VERSION="$(GIT_TAG)" \
		--build-arg AUTHOR="$(AUTHOR)" -t $(E2E_TESTS_DOCKER_IMG_LOCAL_TAG) -f e2e-tests/Dockerfile .


.PHONY: dockerpush-e2e-tests
dockerpush-e2e-tests: dockerbuild-e2e-tests
	docker push $(E2E_TESTS_DOCKER_IMG_LOCAL_TAG)
