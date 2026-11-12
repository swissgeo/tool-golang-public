# aws-ssm

This command can be used to search, retrieve, update and create AWS SSM Parameters across different aws accounts.

## Installation

```bash
go install github.com/swissgeo/tool-golang-public/aws-ssm@latest
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

### Update or create a parameter

> [!WARNING]
> Avoid using `--create` to create new parameters, create them with Terraform instead.
> Parameters created with this command are not managed as code: they are not reviewed,
> not documented in the infrastructure repository and Terraform is not aware of them.
> Use this command mainly to update the value of existing parameters, e.g. to set a secret
> that should not be stored in the Terraform code.

```bash
# Update an existing parameter, fails if the parameter does not exist
aws-ssm set --profile PROFILE --name NAME

# Create a new parameter, fails if the parameter already exists
aws-ssm set --profile PROFILE --name NAME --create --type SecureString
```

- The value is asked interactively twice with hidden input. It is never passed on the command
  line, so it does not appear in the shell history or in `ps`. The command is meant for human
  users only and requires a terminal.
- The value must be a single line. Multi-line values (e.g. PEM certificates or private keys)
  are not supported: when pasted, the first line is taken as the value and the second as the
  confirmation. Values longer than about 4 KB are silently truncated by the terminal line
  buffer, so they are not supported either (only relevant for Advanced parameters, which allow
  up to 8 KB).
- `--type` (`String`, `StringList` or `SecureString`) is mandatory with `--create` and not
  allowed without it. The type of an existing parameter cannot be changed, on update the
  existing type, data type and KMS key are kept.
- On creation the tags `CreatedBy` (current user) and `Created` (local time in ISO 8601 format)
  are added. Tags are not changed on update.
- A newly created parameter only shows up in the name completion once the completion cache has
  expired (see above).
