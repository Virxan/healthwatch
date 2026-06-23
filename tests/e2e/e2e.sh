#!/usr/bin/env bash
# Spins up a throwaway k3d cluster, deploys the backend + Postgres into
# it, runs the Hurl and k6 suites against it, then tears the cluster
# down - whether the tests passed or not.
set -euo pipefail

CLUSTER="healthwatch-e2e"
NAMESPACE="healthwatch"
PF_PID=""

cleanup() {
  if [ -n "${PF_PID}" ]; then
    kill "${PF_PID}" 2>/dev/null || true
  fi
  echo "==> Cleaning up cluster '${CLUSTER}'"
  k3d cluster delete "${CLUSTER}" >/dev/null 2>&1 || true
  rm -f result-container-e2e
}
trap cleanup EXIT

echo "==> Creating k3d cluster '${CLUSTER}'"
k3d cluster create "${CLUSTER}" --wait

echo "==> Building the backend image with Nix"
nix build .#container -o result-container-e2e

echo "==> Loading the image into Docker, then into the cluster"
docker load < result-container-e2e
k3d image import healthwatch-backend:latest -c "${CLUSTER}"

echo "==> Deploying Postgres + backend"
kubectl apply -f k8s/deployment.yaml

echo "==> Waiting for postgres and backend to be ready"
kubectl -n "${NAMESPACE}" wait --for=condition=available --timeout=120s deployment/postgres
kubectl -n "${NAMESPACE}" wait --for=condition=available --timeout=120s deployment/backend

echo "==> Port-forwarding backend to localhost:18080"
kubectl -n "${NAMESPACE}" port-forward svc/backend 18080:8080 &
PF_PID=$!
sleep 3

BASE_URL="http://localhost:18080"

echo "==> Running Hurl API tests against ${BASE_URL}"
hurl --variable base_url="${BASE_URL}" --test backend/tests/hurl/api.hurl

echo "==> Running k6 load test against ${BASE_URL}"
BASE_URL="${BASE_URL}" k6 run backend/tests/k6/bench.js

echo "==> e2e tests passed"
