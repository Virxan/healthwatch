# Healthwatch

A small, dependency-light website health-check aggregator. It periodically
checks a list of HTTP(S) targets for reachability, latency and TLS
certificate expiry, and serves the results over a JSON API and a minimal
dashboard.

Built as the SDLC tooling exercise: reproducible dev shell (Nix), a single
task runner (Just), git hooks (lefthook), a distroless/non-root container
built with Nix, supply-chain checks (gitleaks, Syft, Grype), tests at two
levels (Go unit tests + Cucumber/godog specs), a CI pipeline that enforces
all of it, and GitOps deployment to a local Kubernetes cluster via Argo CD.

## Architecture

```text
Nix dev shell -> Git repo -+-> CI (lint, test, build, scan) -> GHCR
                            +-> Argo CD --(GitOps sync)--> k3d cluster
                                                              |
                                                   Healthwatch pod
                                            scheduler -> checker -> store -> API
                                                              |
                                                    target websites (HTTP+TLS)
```

The binary has no scheduler dependency beyond itself: each target gets its
own goroutine ticking at its configured interval (`internal/scheduler`), the
check itself measures latency and TLS expiry (`internal/checker`), results
land in an in-memory store (`internal/store`), and `internal/api` serves them.

## Prerequisites

- [Nix](https://nixos.org/download) with flakes enabled
- Docker or Podman (k3d needs a container runtime to create cluster nodes)
- A GitHub account, if you want Argo CD to actually sync (see below)

Everything else (Go, k3d, kubectl, helm, argocd CLI, just, lefthook,
golangci-lint, gitleaks, syft, grype, dive, k6) is provided by the dev shell.

## Quick start

```sh
nix develop          # one command, full toolchain
just                 # list every available task
```

### 1. Run it locally, no Kubernetes at all

```sh
just run             # serves on http://localhost:8080
```

Open `http://localhost:8080` for the dashboard, or `curl localhost:8080/api/v1/checks`.

### 2. Run the test suite

```sh
just test            # unit tests + Cucumber/godog checker specs (no network)
just test-bdd        # just the Cucumber specs, verbose Gherkin output
just test-e2e        # builds, runs a real instance, runs the API contract specs against it
just lint            # golangci-lint
just secrets         # gitleaks
```

### 3. Build and inspect the container

```sh
just container       # nix build .#container -> ./result-container (docker-archive)
just sbom             # SBOM via Syft (table + sbom.spdx.json)
just cve              # CVE scan via Grype, fails the task above medium severity
just dive             # inspect layers for minimality
```

The first `just container` will fail with a message like:

```text
error: hash mismatch ... got: sha256-XXXXXXX...
```

That's expected: `vendorHash` in `flake.nix` is a placeholder. Copy the
`got:` hash into `flake.nix`'s `vendorHash` field and re-run - this is the
normal Nix workflow whenever `go.sum` changes.

### 4. Deploy to a local Kubernetes cluster (k3d) via Argo CD

This is the full GitOps loop. Argo CD needs a git remote it can reach, so
push this repo to your own GitHub (or GitLab, etc.) first:

```sh
git remote add origin https://github.com/<you>/healthwatch.git
git push -u origin main
```

Then update `argocd/application.yaml`'s `spec.source.repoURL` to point at
that remote, and:

```sh
just k3d-up               # creates the k3d cluster + installs Argo CD
just argocd-password      # initial admin password (user: admin)
just container            # build the image with Nix
just import-image         # load it straight into the k3d nodes, no registry needed
just argocd-app            # tell Argo CD to start tracking deploy/
```

Argo CD will reconcile `deploy/` against the cluster automatically from then
on (`syncPolicy.automated` in `argocd/application.yaml`) - every push to
`main` rolls out on its own.

Access things:

```sh
just dashboard                                 # port-forward :8080 -> the app
kubectl -n argocd port-forward svc/argocd-server 8443:443   # Argo CD UI on https://localhost:8443
```

For a quick inner loop without touching git/Argo CD at all, `just deploy`
applies `deploy/` directly with `kubectl apply -k`.

### 5. Load test

```sh
just bench                                      # against localhost:8080
just bench http://localhost:8080                 # same, explicit
```

## Configuration

Targets live in `config/targets.yaml` (local run) or the `healthwatch-targets`
ConfigMap in `deploy/configmap.yaml` (cluster run):

```yaml
targets:
  - name: example
    url: https://example.com
    interval_seconds: 30   # default: 30
    timeout_seconds: 5     # default: 5
```

## API

| Method | Path                    | Description                                  |
| ------ | ------------------------ | --------------------------------------------- |
| GET    | `/healthz`               | Liveness/readiness for the process itself     |
| GET    | `/api/v1/checks`         | Latest result for every target (JSON array)   |
| GET    | `/api/v1/checks/{name}`  | Latest result for one target                  |
| GET    | `/`                      | HTML dashboard                                 |

## Project layout

```text
cmd/healthwatch/        entrypoint
internal/config/        targets.yaml loading + validation
internal/checker/       one HTTP+TLS check
internal/scheduler/     per-target ticking loop
internal/store/         in-memory result store
internal/api/           JSON API + HTML dashboard
features/checker/       Cucumber specs for the checker (no network)
features/api/           Cucumber API-contract specs (opt-in, hits a live instance)
test/k6/                load test script
deploy/                 k8s manifests Argo CD tracks
argocd/                 the Argo CD Application resource
flake.nix               dev shell + container build
justfile                task runner
lefthook.yml            git hooks
.github/workflows/      CI
```

## Design notes / trade-offs

- **No Argo Workflows**: scheduling lives in the binary itself
  (goroutines + tickers), not in a CronWorkflow. Simpler to reason about
  and debug, at the cost of the scheduler not being independently
  observable/restartable from the cluster's GitOps tooling.
- **No SQL database**: the store is in-memory only, by design - it avoids
  a cgo driver, keeps the binary fully static, and keeps the SBOM/CVE
  surface minimal. Restarting the pod resets history; if you need
  persistence across restarts, that's the natural next extension point
  (`store.Store` is an interface for exactly this reason).
- **`go-yaml` and `godog` instead of `gopkg.in/yaml.v3` directly**: both
  resolve straight from `github.com`, with no dependency on the
  `gopkg.in` redirect service being reachable - one fewer external
  service in the dependency chain. `go.mod`'s `replace` directives pin
  the two `gopkg.in`-imported transitive dependencies godog pulls in to
  their GitHub mirrors for the same reason.
