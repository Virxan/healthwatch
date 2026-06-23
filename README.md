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

## Setup on Windows (WSL)

This assumes you already have a WSL terminal open (Ubuntu or similar) but
nothing else installed yet. Six steps, all run **inside WSL** unless a
step explicitly says PowerShell.

### 0. Check you're on WSL2 with a recent build

From **PowerShell**:

```powershell
wsl --version
```

If it's missing or clearly old, update with `wsl --update` (PowerShell,
may need a re-open of the terminal afterwards). Recent WSL2 is required for
the next step (systemd support) and for Docker to work well.

### 1. Enable systemd

Nix's multi-user mode and Docker's service management both want a real init
system. WSL doesn't enable this by default. From inside WSL:

```sh
sudo tee /etc/wsl.conf > /dev/null << 'EOF'
[boot]
systemd=true
EOF
```

Then, from **PowerShell**:

```powershell
wsl --shutdown
```

Reopen your WSL terminal. Verify it took effect:

```sh
cat /proc/1/comm   # should print "systemd", not "init"
```

### 2. Base packages

```sh
sudo apt update
sudo apt install -y curl git unzip ca-certificates
```

### 3. Install Nix

The [Determinate Nix Installer](https://github.com/DeterminateSystems/nix-installer)
is the one to use here - it's explicitly tested on WSL2, sets up the
multi-user daemon via the systemd you just enabled, and turns flakes on by
default (the official installer makes you do that by hand).

```sh
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install
```

Close and reopen the terminal, then check:

```sh
nix --version
nix flake metadata github:NixOS/nixpkgs/nixos-unstable   # should resolve, confirms flakes work
```

### 4. Install Docker

k3d needs a container runtime to create cluster nodes. Two options - pick
one:

**Option A - native Docker Engine inside WSL (no Windows app, recommended
if you want everything in one Linux environment):**

```sh
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"
sudo systemctl enable --now docker
```

Close and reopen the terminal so the new group membership applies, then
check with `docker run hello-world`.

**Option B - Docker Desktop for Windows:** install it on the Windows side,
then in Docker Desktop's Settings -> Resources -> WSL Integration, enable
integration for your distro. No commands needed inside WSL; `docker` will
just be on your PATH there once that's toggled on.

### 5. Get the project onto the Linux filesystem

Important: don't unzip into `/mnt/c/...`. Files accessed through `/mnt/c`
cross the Windows/Linux filesystem boundary on every read, which is
noticeably slower and occasionally flaky for Nix and git. Keep it on the
WSL side:

```sh
mkdir -p ~/projects && cd ~/projects
unzip /mnt/c/Users/<you>/Downloads/healthwatch.zip   # adjust to wherever you saved the zip
cd healthwatch
```

From here on, follow [Quick start](#quick-start) below like on any other
Linux machine - `nix develop`, `just`, etc. all work the same way.

One WSL-specific thing to know in advance: when you get to step 4
(deploying into k3d) and run `just dashboard` or open the Argo CD UI,
`http://localhost:<port>` from your Windows browser generally just works -
recent WSL2 forwards localhost automatically, including ports published by
containers. If it doesn't for some reason, get the WSL VM's address with
`ip addr show eth0 | grep inet` and use that IP instead of `localhost` from
Windows.

## Setup on macOS / Linux

Just two things, no WSL detour needed:

- [Nix](https://nixos.org/download) with flakes enabled (the
  [Determinate installer](https://github.com/DeterminateSystems/nix-installer)
  works here too and saves you enabling flakes by hand)
- Docker, Podman, or Docker Desktop

Then jump to [Quick start](#quick-start).

---

Everything else (Go, k3d, kubectl, helm, argocd CLI, just, lefthook,
golangci-lint, gitleaks, syft, grype, dive, k6, yamllint, markdownlint) is
provided by the dev shell on every platform - that's the point of it.

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

(On WSL: both of these are plain `localhost` ports from a `kubectl`
process running inside WSL, so the automatic forwarding mentioned above
applies here too.)

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
