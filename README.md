# Healthwatch

A small website health-check aggregator: a Go + PostgreSQL backend that
watches a list of URLs (the "items") for reachability, latency and TLS
certificate expiry, fronted by a Vue 3 + Tailwind dashboard. Built as an
SDLC tooling exercise - reproducible dev shell (Nix), a Taskfile that
exposes every action, git hooks (lefthook + gitleaks), a distroless/
non-root container built with Nix, supply-chain checks (Syft + Grype),
tests at four levels (Go unit, Testcontainers integration, k6, Hurl), a
CI pipeline that enforces all of it, and a bonus GitOps deployment to a
local Kubernetes cluster via Argo CD.

The `items` table and the `GET/POST/DELETE /items` routes are
deliberately literal matches for the brief's CRUD requirement - what
makes this Healthwatch rather than a generic todo list is that creating
an item (`name` + `url`) immediately checks the URL over HTTP(S), and a
background scheduler re-checks every item every 30s, persisting status,
latency and TLS expiry back onto the same row. The dashboard adds live
search, status filters, summary stats, per-row and bulk delete, a dark
theme, and a `task seed` that loads ~125 demo sites with a realistic
up/down mix.

## Architecture

```text
                 ┌──────────────┐        ┌──────────────────────────┐
  dev:   Vite ───┤ proxy /api/* ├───────▶│                          │
  prod:  (nothing, Go serves    │        │   backend (Go + pgx)     │
         the Vue build itself)  │        │   GET  /health           │
                 └──────────────┘        │   GET  /items            │
                                          │   POST /items            │
                                          └────────────┬─────────────┘
                                                        │ pgx/v5
                                                        ▼
                                          ┌──────────────────────────┐
                                          │   PostgreSQL (items)     │
                                          └──────────────────────────┘
```

Locally: `docker-compose` runs Postgres, the Go binary runs directly on
the host, Vite serves the frontend with hot reload and proxies `/api/*`
to the backend.

In Kubernetes (`task e2e` or the Argo CD path): both Postgres and the
backend run as pods in the `healthwatch` namespace (`k8s/deployment.yaml`);
the backend image is a Nix-built, distroless, non-root container with the
Vue build embedded inside it via `go:embed` - no Vite server in
production at all.

## Setup on Windows (WSL)

Six steps, all run **inside WSL** unless noted otherwise. Already did
this for a previous version of this project? You can skip straight to
[Quick start](#quick-start) - nothing here has changed.

1. **Check WSL2 is current** - from PowerShell: `wsl --version` (update
   with `wsl --update` if it looks old).
2. **Enable systemd** (needed for Nix's multi-user daemon and for
   Docker's service):

   ```sh
   sudo tee /etc/wsl.conf > /dev/null << 'EOF'
   [boot]
   systemd=true
   EOF
   ```

   Then from PowerShell: `wsl --shutdown`, then reopen your WSL terminal
   and confirm with `cat /proc/1/comm` (should print `systemd`).
3. **Base packages**: `sudo apt update && sudo apt install -y curl git unzip ca-certificates`
4. **Install Nix** (the [Determinate installer](https://github.com/DeterminateSystems/nix-installer)
   is tested on WSL2 and turns flakes on by default):

   ```sh
   curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install
   ```

5. **Install Docker** (native engine, using the systemd from step 2):

   ```sh
   curl -fsSL https://get.docker.com | sudo sh
   sudo usermod -aG docker "$USER"
   sudo systemctl enable --now docker
   ```

   Close/reopen the terminal, then check with `docker run hello-world`.
6. **Clone the repo onto the Linux filesystem** - not `/mnt/c/...`. Files
   accessed through `/mnt/c` cross the Windows/Linux boundary on every
   read, which is measurably slower and has caused real, reproducible
   failures for this exact project (`git apply` mangling file
   permissions, `k3d image import` failing outright on a multi-MB file
   mid-transfer). Use the WSL side:

   ```sh
   mkdir -p ~/projects && cd ~/projects
   git clone <your-fork-url> healthwatch
   cd healthwatch
   ```

## Setup on macOS / Linux

Just Nix (flakes) and Docker - no WSL detour needed. Then jump to
[Quick start](#quick-start).

## Quick start

```sh
nix develop      # one command, full toolchain (go, node, k3d, task, hurl, k6...)
task             # list every available task
```

`backend/go.sum` is committed, so `nix develop` and every build step
work straight away. You only need to regenerate it if you change Go
dependencies:

```sh
cd backend && go mod tidy && cd ..
```

### 1. Run it locally

```sh
task db          # starts Postgres via docker-compose
task seed        # optional: ~125 demo sites with a realistic up/down mix
task build-frontend
task run         # http://localhost:8080
```

`task seed` is idempotent (it clears the table first), so you can re-run
it any time to get back to a clean, populated dashboard. It picks URLs
that settle fast - real, reliable sites resolve "up", `*.invalid` hosts
fail DNS instantly and settle "down" - so the 30s scheduler sweep never
saturates a small VM with slow timeouts.

The Go binary embeds the built frontend at compile time (`go:embed`), so
after `task build-frontend` you must restart `task run` for the new
build to be served. For iterating on the UI, skip that loop and run
Vite's dev server instead - it hot-reloads and proxies `/api/*` to the
backend on `:8080`:

```sh
cd frontend && npm install && npm run dev   # http://localhost:5173
```

Reaching the dashboard from another machine (e.g. a VM accessed from the
host) may need the port opened in the firewall, e.g. on RHEL-family
distros: `sudo firewall-cmd --add-port=8080/tcp --permanent && sudo firewall-cmd --reload`.

### 2. Test

```sh
task test               # Go unit tests, no Docker needed
task test-integration   # Testcontainers: spins up a real, throwaway Postgres
task test-api           # Hurl, against the instance from `task run`
task bench               # k6, 10 VUs / 15s, p95 < 200ms
task test-all             # all of the above
```

### 3. Lint and build the container

```sh
task lint
task container
```

`nix/package.nix` already carries the real `npmDepsHash` and
`vendorHash`, so `task container` builds out of the box. If you ever
change dependencies (`frontend/package-lock.json` or `backend/go.sum`),
the build fails with a message like `error: hash mismatch ... got:
sha256-XXXX...` - that's Nix telling you the new hash. Paste the two
`got:` values back into `nix/package.nix`'s `npmDepsHash` and
`vendorHash`, then run it again.

```sh
task sbom
task cve
```

### 4. Full e2e in k3d

```sh
task e2e
```

Creates a throwaway k3d cluster, builds and loads the image, deploys
Postgres + the backend (`k8s/deployment.yaml`), runs the Hurl and k6
suites against the real cluster, then tears everything down -
whether the tests passed or not.

### 5. GitOps with Argo CD (bonus)

This is the persistent version of the e2e cluster, kept around and
synced from your git remote instead of torn down after one test run.

```sh
k3d cluster create healthwatch --wait
task argocd-setup                      # installs Argo CD, prints the admin password
```

Push this repo to your own fork, update `k8s/argocd-app.yaml`'s
`spec.source.repoURL` to point at it, then:

```sh
git push
task container
docker load < result-container
k3d image import healthwatch-backend:latest -c healthwatch
kubectl apply -f k8s/argocd-app.yaml
```

Argo CD now tracks `k8s/` and re-syncs on every push
(`syncPolicy.automated` in `k8s/argocd-app.yaml`).

```sh
kubectl -n healthwatch port-forward svc/backend 8080:8080
kubectl -n argocd port-forward svc/argocd-server 8443:443   # Argo CD UI
```

## Configuration

| Variable       | Required | Description                                          |
| -------------- | -------- | ----------------------------------------------------- |
| `DATABASE_URL` | yes      | Postgres connection string, e.g. `postgres://user:pass@host:5432/db?sslmode=disable` |

The backend creates the `items` table itself on startup if it doesn't
exist (one idempotent `CREATE TABLE IF NOT EXISTS` - no migration
framework needed at this size).

## API

| Method | Path                  | Also at  | Description                          |
| ------ | --------------------- | -------- | ------------------------------------- |
| GET    | `/health`             | `/api/health` | `{"status":"ok"}` or 503 `{"status":"down","error":"..."}` - pings the database, not the watched sites |
| GET    | `/items`              | `/api/items`   | All watched sites, with their latest check result, JSON array |
| POST   | `/items`              | `/api/items`   | `{"name":"...","url":"https://..."}` → 201 with the created item (already checked once), or 400 if name/url is missing or url isn't a valid http(s) URL |
| DELETE | `/items`              | `/api/items`   | Remove every watched site at once (the dashboard's "vider la base") → 200 `{"deleted":N}` |
| DELETE | `/items/{id}`         | `/api/items/{id}` | Remove a single watched site (per-row trash button) → 204, or 404 if no item has that id, or 400 if the id isn't an integer |
| GET    | `/*`                  | -        | The built Vue frontend                |

Routes are registered at both the bare path and under `/api/` so the
exact same frontend code (`fetch('/api/items')`) works unmodified
whether Vite is proxying in dev or Go is serving everything in prod -
see `backend/handlers.go`.

Every item also carries `last_status`, `last_http_status`,
`last_latency_ms`, `last_checked_at`, `tls_days_remaining` and
`last_error` once it's been checked at least once (all `null`/omitted
before that - which only lasts a moment, since creating an item checks
it immediately).

## Project layout

```text
backend/
  main.go, main_test.go      entrypoint + handler unit tests (MemoryStore)
  handlers.go, web.go        HTTP routes, embedded frontend
  checker.go, checker_test.go    one HTTP+TLS check
  scheduler.go, scheduler_test.go    re-checks every item every 30s
  db/                        Store interface, PGStore (pgx), MemoryStore
  db/seed.sql                ~125 demo sites, loaded by `task seed`
  tests/integration/         Testcontainers, real Postgres (-tags integration)
  tests/k6/, tests/hurl/     load test, API contract test
  web/dist/                  Vite build output lands here (go:embed target)
frontend/                    Vue 3 + Vite + Tailwind
nix/                         dev-shell, package (Go+npm build), container, treefmt
k8s/                         Deployment/Service/Secret for app+Postgres, Argo CD Application
tests/e2e/                   e2e.sh (k3d cycle), setup-argocd.sh
docker-compose.yml           local Postgres for `task run`
Taskfile.yaml                every task
```

## Design notes / deviations from the brief

A few small, deliberate departures from the file list as written, each
forced by a real constraint rather than taste:

- **`backend/db/` package**: `tests/integration/db_test.go` lives in a
  different directory than `backend/`, so it's necessarily a different
  Go package - it can't import `package main`. The Store/PGStore/
  MemoryStore types live in `backend/db` for exactly this reason; `main`
  imports them like any other dependency.
- **`frontend/eslint.config.js`**: not in the original file list, but
  `lint-frontend` needs something to actually run - ESLint 9's flat
  config is the standard choice for a new Vue project.
- **Tailwind v3, not v4**: v4 replaced `tailwind.config.js` +
  `postcss.config.js` with a CSS-first setup, which would make two
  explicitly required files redundant. Pinned to v3 specifically so
  those two files keep doing real work.
- **`go 1.25` in `go.mod`, `go` (unpinned) in `nix/dev-shell.nix`**:
  pinning the dev shell to an exact `go_1_25` package is what breaks the
  moment that version goes end-of-life and gets removed from nixpkgs (it
  already has, once, during this project's lifetime) - the unpinned
  `go` attribute always tracks whatever's current, while `go.mod`'s `go
  1.25` directive still declares the minimum language version like the
  brief asks. A newer toolchain happily builds an older `go` directive.
- **No `.yamllint.yaml`**: dropped - there's no `lint-yaml` task in the
  required list, so a config for a linter nothing calls would just be
  clutter.
- **`backend/go.sum` is committed**: it was generated in an environment
  with full internet access, because `testcontainers-go`'s dependency
  tree is deep enough that producing one inside a restricted sandbox
  wasn't reliable. Regenerate it with `go mod tidy` only when you change
  dependencies; otherwise it behaves like any other Go project's go.sum.
