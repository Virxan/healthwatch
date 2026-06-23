#!/usr/bin/env bash
# Installs Argo CD into an existing k3d cluster. Idempotent - safe to
# re-run.
set -euo pipefail

NAMESPACE="argocd"

echo "==> Installing Argo CD into namespace '${NAMESPACE}'"
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# --server-side: the official install manifest's CRDs (notably
# applicationsets.argoproj.io) exceed the 256KB annotation size limit
# that client-side `kubectl apply` hits when storing the previous config
# as an annotation for 3-way diffing.
kubectl apply --server-side -n "${NAMESPACE}" \
  -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

echo "==> Waiting for argocd-server to become available"
kubectl -n "${NAMESPACE}" wait --for=condition=available --timeout=180s deployment/argocd-server

echo "==> Argo CD is up. Initial admin password:"
kubectl -n "${NAMESPACE}" get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
echo
