SHELL = /bin/bash

.DEFAULT_GOAL := help

CURRENT_DIR := $(shell pwd)

BINS = $(shell find * -name main.go -printf 'bin/%h\n')

# Docker metadata
GOLANG_VERSION ?= `GOENV_GOMOD_VERSION_ENABLE=1 goenv local`
GIT_HASH = `git rev-parse HEAD`
GIT_HASH_SHORT = `git rev-parse --short HEAD`
GIT_BRANCH = `git symbolic-ref HEAD --short 2>/dev/null`
GIT_DIRTY = `git status --porcelain`
GIT_TAG = `git describe --tags || echo "no version info"`
AUTHOR = $(USER)

# Docker variables
DOCKER_REGISTRY = 974517877189.dkr.ecr.eu-central-1.amazonaws.com
DOCKER_IMG_LOCAL_TAG := local-$(USER)-$(GIT_HASH_SHORT)
# TODO(PB-2197): add codebuild once the corresponding ECR is ready
DOCKERISED_BINS = codebuild e2e-tests
DOCKERFILES := $(patsubst %,%/Dockerfile,$(DOCKERISED_BINS))

# AWS variables
AWS_DEFAULT_REGION = eu-central-1

bin:
	mkdir -p $@

bin/%: % bin FORCE
	go build -o $@ ./$<

.PHONY:
clean:
	$(RM) -r bin/ */Dockerfile

# Allows to .PHONY'se implicit rules without enumerating them all.
# https://www.gnu.org/software/make/manual/html_node/Force-Targets.html
FORCE:

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo
	@echo "Some possible targets:"
	@echo "- setup                  Install dependencies"
	@echo "- all                    Build all the Dockerfiles and Go binaries: $(DOCKERFILES) $(BINS)"
	@echo -e " \033[1mFORMATING, LINTING AND TESTING TOOLS TARGETS\033[0m "
	@echo "- format                 Format the go source code"
	@echo "- lint                   Lint the go source code"
	@echo "- format-lint            Format and lint the go source code"
	@echo "- govulncheck            Golang vulnerability check"
	@echo "- test                   Runs tests"
	@echo "- clean                  Clean the build directory."
	@echo -e " \033[1mDocker TARGETS\033[0m "
	@echo "- dockerlogin            Login to the AWS ECR registry for pulling/pushing docker images"
	@echo "- dockerbuild-e2e-tests  Build the tool e2e-tests locally (with tag := $(DOCKER_IMG_LOCAL_TAG))"
	@echo "- dockerbuild-codebuild  Build the tool codebuild locally (with tag := $(DOCKER_IMG_LOCAL_TAG))"
	@echo "- dockerpush-e2e-tests   Build and push the tool e2e-tests (with tag := $(DOCKER_IMG_LOCAL_TAG))"


.PHONY: setup
setup:
	go mod tidy
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest


# linting target, calls upon yapf to make sure your code is easier to read and respects some conventions.

.PHONY: format
format:
	goimports -w .

.PHONY: lint
lint:
	golangci-lint run

.PHONY: format-lint
format-lint: format lint

.PHONY: govulncheck
govulncheck: govulncheck -show verbose ./...

.PHONY: test
test:
	go test ./... -v -count=1 # count=1 disables test caching

.PHONY: all
all: $(BINS) $(DOCKERFILES)

# Docker related functions.

%/Dockerfile: Dockerfile.in
	echo "# THIS FILE IS AUTO-GENERATED, DO NOT EDIT OR COMMIT." > $@
	sed "s/BGDI_GO_PACKAGE/$(subst /Dockerfile,,$@)/g" $< >> $@

.PHONY: dockerlogin
dockerlogin:
	aws --profile swisstopo-bgdi-builder ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(DOCKER_REGISTRY)


dockerbuild-%: %/Dockerfile FORCE
	$(eval PACKAGE=$(subst dockerbuild-,,$@))
	docker build \
		--build-arg BASE_IMAGE_VERSION="$(GOLANG_VERSION)" \
		--build-arg GIT_HASH="$(GIT_HASH)" \
		--build-arg GIT_BRANCH="$(GIT_BRANCH)" \
		--build-arg GIT_DIRTY="$(GIT_DIRTY)" \
		--build-arg VERSION="$(GIT_TAG)" \
		--build-arg AUTHOR="$(AUTHOR)" \
		-t $(DOCKER_REGISTRY)/tool-golang-bgdi/$(PACKAGE):$(DOCKER_IMG_LOCAL_TAG) \
		-f $(PACKAGE)/Dockerfile .

dockerpush-%: dockerbuild-% FORCE
	$(eval PACKAGE=$(subst dockerpush-,,$@))
	docker push $(DOCKER_REGISTRY)/tool-golang-bgdi/$(PACKAGE):$(DOCKER_IMG_LOCAL_TAG)
