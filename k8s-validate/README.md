# k8s-validate

This command can be use to validate all kubernetes kustomization folders down the current working directory.
It is mainly used by the CI of kubernetes manifest repositories like geoadmin/infra-kubernetes and geoadmin/infra-kubernetes-internal.

Currently it only supports kustomize but could be extended in future to also supports helm if needed.

## Installation

```bash
go install github.com/geoadmin/tool-golang-bgdi/k8s-validate@latest
```

Install tab completion

```bash
# Bash
k8s-validate completion bash > /usr/share/bash-completion/completions/k8s-validate

# ZSH
k8s-validate completion zsh > ~/.zsh/completion/_k8s-validate
```

## Usage

```bash
k8s-validate
```

To fail at the first failure

```bash
k8s-validate --fail-fast
```
