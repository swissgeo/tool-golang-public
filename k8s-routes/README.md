# k8s-routes

This command lists the routes of the Traefik IngressRoutes, grouped by host and sorted by priority (highest first) within each host, to find out which route handles a request and to detect route conflicts.
The routes can either be rendered from the local kustomize overlays of a repository like swissgeo/infra-kubernetes, or be read from a cluster.

## Installation

```bash
go install github.com/swissgeo/tool-golang-public/k8s-routes@latest
```

Install tab completion

```bash
# Bash
k8s-routes completion bash > /usr/share/bash-completion/completions/k8s-routes

# ZSH
k8s-routes completion zsh > ~/.zsh/completion/_k8s-routes
```

Requires `kustomize` to list the local routes and `kubectl` with access to the cluster to list the live routes.

## Usage

### Output

```text
MATCH                                           PRIORITY  SERVICES          MIDDLEWARES         NAME
PathPrefix(`/status`)                           (21)      status                                ns/status

Host(`example.net`) && PathPrefix(`/api`)       1000      api               rate-limit,headers  ns/api (*)
Host(`www.example.net`) || Host(`example.net`)  (46)      shop,shop-canary                      ns/shop

Host(`www.example.net`) || Host(`example.net`)  (46)      shop,shop-canary                      ns/shop
Host(`www.example.net`) && Path(`/beta`)        5         beta                                  ns/beta (+)
```

- The hosts are separated by an empty line, only routes of the same host can conflict.
- A route matching several hosts is listed under each of them.
- The routes without `Host` or `HostRegexp` matcher are listed first, as they compete with the routes of all hosts.
- When no explicit priority is set, Traefik uses the length of the match rule as priority. Such computed priorities are shown in parentheses.

### Local routes

To list the routes rendered from the kustomize overlays of an env, from the root of the repository

```bash
k8s-routes local dev
```

By default the overlays are `services/*/overlays/{env}`, use `--glob` for another layout and `--root` to run it from another directory.
IngressRoutes without namespace get the path segment containing the first wildcard of the glob as namespace (e.g. the service name), like the ArgoCD application would do.

```bash
k8s-routes local int --root ~/infra-kubernetes --glob 'apps/*/{env}' --enable-helm
```

### Live routes

To list the routes deployed in a cluster (all namespaces), for the current kubectl context or a given one

```bash
k8s-routes live
k8s-routes live --context swissgeo/int-blue
```

To preview the effect of local changes not yet applied, use `--local`. The IngressRoutes rendered from the local overlays replace the live ones with the same namespace/name.
The env (dev|int|prod) is taken from the kube context name. Such routes are marked:

- `(*)` the IngressRoute differs from the live one
- `(+)` the IngressRoute does not exist live yet

IngressRoutes removed or renamed locally are still shown with their live version.

```bash
k8s-routes live --context swissgeo/int-blue --local
```

### Filter by host

To only show the routes of the hosts containing a string (case-insensitive), plus the routes without host as they compete with all hosts

```bash
k8s-routes live --host api.example
```

Colors are disabled when the output is not a terminal, or with `--no-color`.
