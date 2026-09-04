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
DOCKER_REGISTRY = 074597099015.dkr.ecr.eu-central-1.amazonaws.com
BGDI_DOCKER_REGISTRY = 974517877189.dkr.ecr.eu-central-1.amazonaws.com
DOCKER_IMG_LOCAL_TAG := local-$(USER)-$(GIT_HASH_SHORT)
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
help: ## Display this help
# automatically generate the help page based on the documentation after each make target
# from https://gist.github.com/prwhite/8168133
# dockerbuild-%/dockerpush-% are pattern rules so they can't carry a ##
# comment awk can match against; their help lines are generated here instead,
# one per entry in DOCKERISED_BINS, and merged into the same sorted listing.
	@{ \
	awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "%-22s %s\n", $$1, $$2 }' $(MAKEFILE_LIST); \
	for b in $(DOCKERISED_BINS); do \
		printf "%-22s %s\n" "dockerbuild-$$b" "Build the Docker image for $$b"; \
		printf "%-22s %s\n" "dockerpush-$$b" "Push the Docker image for $$b"; \
		printf "%-22s %s\n" "dockerbuild-bgdi-$$b" "Build the Docker image for $$b using the BGDI Docker registry"; \
		printf "%-22s %s\n" "dockerpush-bgdi-$$b" "Push the Docker image for $$b on the BGDI registry"; \
	done; \
	} \
	| awk 'BEGIN { printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n" } \
	{ printf "  \033[36m%-22s\033[0m %s\n", $$1, substr($$0, index($$0,$$2)) }'


.PHONY: setup
setup: ## Install dependencies
	go mod tidy
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest


# linting target, calls upon yapf to make sure your code is easier to read and respects some conventions.

.PHONY: format
format: ## Format the go source code
	goimports -w .

.PHONY: lint
lint: ## Lint the go source code
	golangci-lint run

.PHONY: format-lint
format-lint: format lint ## Format and lint the go source code

.PHONY: govulncheck
govulncheck: govulncheck -show verbose ./... ## Golang vulnerability check

.PHONY: test
test: ## Runs tests
	go test ./... -v -count=1 # count=1 disables test caching

.PHONY: all
all: $(BINS) $(DOCKERFILES) ## Build all the Dockerfiles and Go binaries: $(DOCKERFILES) $(BINS)

# Docker related functions.

%/Dockerfile: Dockerfile.in
	echo "# THIS FILE IS AUTO-GENERATED, DO NOT EDIT OR COMMIT." > $@
	sed "s/SWISSGEO_GO_PACKAGE/$(subst /Dockerfile,,$@)/g" $< >> $@

.PHONY: dockerlogin
dockerlogin: ## Log in to the Docker registry for SWISSGEO services
	aws --profile swisstopo-swissgeo-builder ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(DOCKER_REGISTRY)

.PHONY: dockerlogin-bgdi
dockerlogin-bgdi: ## Log in to the Docker registry for BGDI services
	aws --profile swisstopo-bgdi-builder ecr get-login-password --region $(AWS_DEFAULT_REGION) | docker login --username AWS --password-stdin $(BGDI_DOCKER_REGISTRY)

# dockerbuild-<pkg>/dockerpush-<pkg> are generated as real explicit rules
# (one pair per DOCKERISED_BINS entry) rather than a "%" pattern rule.
# GNU Make only skips implicit-rule search - and so needs the FORCE trick -
# for targets that have NO explicit rule of their own; a target generated
# here via $(eval) has one, so .PHONY works on it directly.
define DOCKER_TARGETS
.PHONY: dockerbuild-$(1)
dockerbuild-$(1): $(1)/Dockerfile
	docker build \
		--build-arg BASE_IMAGE_VERSION="$$(GOLANG_VERSION)" \
		--build-arg GIT_HASH="$$(GIT_HASH)" \
		--build-arg GIT_BRANCH="$$(GIT_BRANCH)" \
		--build-arg GIT_DIRTY="$$(GIT_DIRTY)" \
		--build-arg VERSION="$$(GIT_TAG)" \
		--build-arg AUTHOR="$$(AUTHOR)" \
		-t $$(DOCKER_REGISTRY)/swissgeo/tool-golang-public/$(1):$$(DOCKER_IMG_LOCAL_TAG) \
		-f $(1)/Dockerfile .

.PHONY: dockerpush-$(1)
dockerpush-$(1): dockerbuild-$(1)
	docker push $$(DOCKER_REGISTRY)/swissgeo/tool-golang-public/$(1):$$(DOCKER_IMG_LOCAL_TAG)

## BGDI targets
.PHONY: dockerbuild-bgdi-$(1)
dockerbuild-bgdi-$(1): $(1)/Dockerfile
	docker build \
		--build-arg BASE_IMAGE_VERSION="$$(GOLANG_VERSION)" \
		--build-arg GIT_HASH="$$(GIT_HASH)" \
		--build-arg GIT_BRANCH="$$(GIT_BRANCH)" \
		--build-arg GIT_DIRTY="$$(GIT_DIRTY)" \
		--build-arg VERSION="$$(GIT_TAG)" \
		--build-arg AUTHOR="$$(AUTHOR)" \
		-t $$(BGDI_DOCKER_REGISTRY)/tool-golang-bgdi/$(1):$$(DOCKER_IMG_LOCAL_TAG) \
		-f $(1)/Dockerfile .

.PHONY: dockerpush-bgdi-$(1)
dockerpush-bgdi-$(1): dockerbuild-bgdi-$(1)
	docker push $$(BGDI_DOCKER_REGISTRY)/tool-golang-bgdi/$(1):$$(DOCKER_IMG_LOCAL_TAG)

endef


$(foreach bin,$(DOCKERISED_BINS),$(eval $(call DOCKER_TARGETS,$(bin))))
