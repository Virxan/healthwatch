# Healthwatch task runner. Run `just` with no arguments to list everything.

cluster := "healthwatch"
namespace := "healthwatch"
image := "result-container"

default:
    @just --list

# --- build & run ------------------------------------------------------

# Build the healthwatch binary into ./bin
build:
    mkdir -p bin
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/healthwatch ./cmd/healthwatch

# Run healthwatch locally against config/targets.yaml on :8080
run:
    go run ./cmd/healthwatch -config config/targets.yaml -addr :8080

# --- formatting & linting ----------------------------------------------

# Format all Go files in place
fmt:
    gofumpt -l -w .

# Fail if any file is not formatted (used in git hooks and CI)
fmt-check:
    @test -z "$(gofumpt -l .)" || (echo "not gofumpt-formatted:"; gofumpt -l .; exit 1)

# Run golangci-lint
lint:
    golangci-lint run ./...

# Lint YAML files (k8s manifests, Argo CD, CI, hooks config)
lint-yaml:
    yamllint -c .yamllint.yaml deploy/ argocd/ .github/workflows/ lefthook.yml .golangci.yml .yamllint.yaml

# Lint Markdown files
lint-md:
    markdownlint -c .markdownlint.json '**/*.md' --ignore node_modules

# Everything fmt-check + lint + secrets, the same checks CI enforces
lint-all: fmt-check lint lint-yaml lint-md secrets

# --- tests --------------------------------------------------------------

# Unit tests + Cucumber/godog checker specs (no network, no live server)
test:
    go test ./... -race

# Just the Cucumber/godog specs under features/
test-bdd:
    go test ./features/... -v

# End-to-end: build, run a real instance, point the API contract specs
# at it, then tear it down. Pass a URL to target a k3d deployment
# instead, e.g.: just test-e2e http://localhost:8080
test-e2e BASE_URL="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -n "{{BASE_URL}}" ]; then
        HEALTHWATCH_E2E=1 HEALTHWATCH_BASE_URL="{{BASE_URL}}" go test ./features/api/... -v
        exit 0
    fi
    just build
    ./bin/healthwatch -config config/targets.yaml -addr :8099 &
    pid=$!
    trap 'kill $pid 2>/dev/null || true' EXIT
    sleep 1
    HEALTHWATCH_E2E=1 HEALTHWATCH_BASE_URL=http://localhost:8099 go test ./features/api/... -v

# Load/benchmark test against a running instance (requires k6)
bench BASE_URL="http://localhost:8080":
    BASE_URL={{BASE_URL}} k6 run test/k6/load.js

# --- security & supply chain --------------------------------------------

# Scan the working tree for committed secrets
secrets:
    gitleaks detect --source . --no-banner --redact

# Generate an SBOM for the container image (requires `just container` first)
sbom:
    syft "docker-archive:{{image}}" -o spdx-json=sbom.spdx.json
    syft "docker-archive:{{image}}" -o table

# Scan the container image for known CVEs (requires `just container` first)
cve:
    grype "docker-archive:{{image}}" --fail-on medium

# Inspect container layers for minimality
dive:
    dive "docker-archive:{{image}}"

# --- container -----------------------------------------------------------

# Build the distroless, non-root container image with Nix
container:
    nix build .#container -o {{image}}
    @echo "image written to ./{{image}} (docker-archive format)"

# --- local k3d cluster + Argo CD -----------------------------------------

# Create a local k3d cluster and install Argo CD into it
k3d-up:
    k3d cluster create {{cluster}} --port "8080:80@loadbalancer" --wait
    kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
    kubectl -n argocd wait --for=condition=available --timeout=180s deployment/argocd-server
    @echo "Argo CD is up. Run 'just argocd-password' for the initial admin password."

# Delete the local k3d cluster
k3d-down:
    k3d cluster delete {{cluster}}

# Print the Argo CD initial admin password
argocd-password:
    kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
    @echo ""

# Load a locally-built image straight into the k3d nodes (no registry needed)
import-image:
    k3d image import {{image}} -c {{cluster}}

# Apply the raw k8s manifests directly (fast inner loop, bypasses Argo CD/git)
deploy:
    kubectl apply -k deploy/

# Apply the Argo CD Application so it starts tracking the git repo
# (update argocd/application.yaml's repoURL to your fork first)
argocd-app:
    kubectl apply -f argocd/application.yaml

# Port-forward the dashboard/API to http://localhost:8080
dashboard:
    kubectl -n {{namespace}} port-forward svc/healthwatch 8080:8080

# --- housekeeping ---------------------------------------------------------

clean:
    rm -rf bin result result-container sbom.spdx.json
