#!/bin/bash
set -euo pipefail

ENV=${1:-default}

if [[ "$ENV" != "default" && "$ENV" != "prod" ]]; then
  echo "Usage: $0 [default|prod]"
  exit 1
fi

echo "Environment: $ENV"

CURRENT_CTX=$(kubectl config current-context)
if [[ "$CURRENT_CTX" == *"gke"* ]]; then
  echo "WARNING: You are connected to PROD (GKE). Continue? (y/n)"
  read -r confirm
  if [[ $confirm != "y" ]]; then exit 1; fi
fi

echo "Installing ArgoCD..."
kubectl apply -k k8s/argocd/install/

echo "Waiting for ArgoCD server to be ready..."
kubectl wait --namespace argocd \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/name=argocd-server \
  --timeout=300s

echo "Applying ArgoCD ConfigMap customizations..."
kubectl apply -f k8s/argocd/argocd-cm-patch.yaml

echo "Deploying root application (App of Apps) for '$ENV' environment..."
kubectl apply -k k8s/argocd/apps/overlays/${ENV}

echo ""
echo "ArgoCD installed successfully."
echo ""
echo "Access the UI:"
echo "  kubectl port-forward svc/argocd-server -n argocd 8443:443"
echo ""
echo "Initial admin password:"
echo "  kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d && echo"