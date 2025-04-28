# Golang CLI Tools for BGDI

| Branch | Status |
|--------|-----------|
| master | ![Build Status](https://codebuild.eu-central-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoiSklmL2ZFYzE2QXRVZzloVFo4dFYrdHh3a2pzZXZOYnYxSXpVVzRRbUlzUDJ6OEpSMWREaHo5d01hYUFpdjR3V05ORkljcG96aUlJTG8wOWZoMituTzlNPSIsIml2UGFyYW1ldGVyU3BlYyI6InBTdWJDZjh1bXNaR1pZSGwiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=master) |

This repository contains CLI tools written in golang to manage BGDI services and infrastructure.

- [Repository structure](#repository-structure)
- [Naming convention](#naming-convention)
- [Formatting and linting](#formatting-and-linting)
- [Create new application skeleton](#create-new-application-skeleton)
- [CLI commands](#cli-commands)

## Repository structure

```text
tool-golang-bgdi
    |-- TOOL_NAME
    .      |-- main.go
           |-- README.md
           |-- cmd
                |-- root.go
```

## Naming convention

`TOOL_NAME` should be in kebab case

## Formatting and linting

Code should be formatted with `goimports` and linted with `golangci-lint`

```bash
goimports -w .
```

```bash
golangci-lint run
```

To simplify the linting and formatting a makefile is available

```bash
make format
make lint
```

## Create new application skeleton

```bash
go install github.com/spf13/cobra-cli@latest
cobra-cli init <app-name>
```

## CLI commands

- [k8s-validate](./k8s-validate/README.md)
- [e2e-tests](./e2e-tests/README.md)
- [cloudfront-logs](./cloudfront-logs/README.md)
