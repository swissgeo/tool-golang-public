# aws-ssm

This command can be use to search and retrieve AWS SSM Parameters across different aws accounts.

## Installation

```bash
export GOPRIVATE=github.com/geoadmin
go install github.com/geoadmin/tool-golang-bgdi/aws-ssm@latest
```

Install tab completion

```bash
# Bash
aws-ssm completion bash > /usr/share/bash-completion/completions/aws-ssm

# ZSH
aws-ssm completion zsh > ~/.zsh/completion/aws-ssm
```

To improve completion performance, the parameter names are cached in `$HOME/.cache/aws-ssm/completions` for 24 hours. Remove this file to reset the cache.

## Usage

```bash
aws-ssm --help
```
