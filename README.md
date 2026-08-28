# Golang CLI Tools for SWISSGEO

| Branch | Status |
|--------|-----------|
| main | ![Build Status](https://codebuild.eu-central-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoidEVDZVVLWjRsT2xIWHFWYUs0Y1Eyc0xyNlhMT0VaWlNnSzFRS0pxUVVaQmgweW5vUXYxUEtXS2tLUTdNU3Fxc29QZFcwcFVRSHhMWGljdUl3N21xUURrPSIsIml2UGFyYW1ldGVyU3BlYyI6InN5NFptUWl2RkdKSHFITmYiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=main) |

This repository contains CLI tools written in golang to manage SWISSGEO services and infrastructure. It is also used to manage the legacy BGDI services and infrastructure and
has been ported from https://github.com/geoadmin/tool-golang-bgdi.

- [Repository structure](#repository-structure)
- [Naming convention](#naming-convention)
- [Formatting and linting](#formatting-and-linting)
- [Create new application skeleton](#create-new-application-skeleton)
- [CLI commands](#cli-commands)

## Repository structure

```text
tool-golang-public
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

See individual directories for the documentation of the respective command.
