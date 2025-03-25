SHELL = /bin/bash

.DEFAULT_GOAL := help

CURRENT_DIR := $(shell pwd)


all: help


.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo
	@echo "Possible targets:"
	@echo -e " \033[1mFORMATING, LINTING AND TESTING TOOLS TARGETS\033[0m "
	@echo "- format             Format the go source code"
	@echo "- lint               Lint the go source code"


# linting target, calls upon yapf to make sure your code is easier to read and respects some conventions.

.PHONY: format
format:
	goimports -w .


.PHONY: lint
lint:
	golangci-lint run
