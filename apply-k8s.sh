#!/bin/bash

# Parse environment argument (default: default)
ENV="${1:-default}"

CURRENT_CTX=$(kubectl config current-context)
if [[ "$CURRENT_CTX" == *"gke"* ]]; then
  echo "⚠️ WARNING: You are connected to PROD (GKE). Continue? (y/n)"
  read -r confirm
  if [[ $confirm != "y" ]]; then exit 1; fi
fi

# Validate environment
if [[ "$ENV" != "default" && "$ENV" != "prod" ]]; then
  echo "Error: Invalid environment '$ENV'. Use 'default' or 'prod'."
  echo "Usage: $0 [default|prod]"
  exit 1
fi

echo "Deploying to environment: $ENV"

# Check if nginx Ingress Controller is already installed
if ! kubectl get namespace ingress-nginx > /dev/null 2>&1; then
  echo "nginx Ingress Controller not found, installing..."
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/baremetal/deploy.yaml
  if [ $? -ne 0 ]; then
    echo "Failed to install nginx Ingress Controller. Exiting."
    exit 1
  fi
  echo "Waiting for Ingress Controller to be ready..."
  kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=120s
else
  echo "nginx Ingress Controller is already installed."
fi

# Check if Atlas Operator is already installed via Helm
if ! helm status atlas-operator > /dev/null 2>&1; then
  echo "Atlas Operator not found, installing..."
  helm install atlas-operator oci://ghcr.io/ariga/charts/atlas-operator
  if [ $? -ne 0 ]; then
    echo "Failed to install Atlas Operator. Exiting."
    exit 1
  fi
else
  echo "Atlas Operator is already installed."
fi

# Check if Sealed Secrets controller is already installed via Helm
if ! helm status sealed-secrets-controller -n kube-system > /dev/null 2>&1; then
  echo "Sealed Secrets controller not found, installing..."
  helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
  helm install sealed-secrets-controller sealed-secrets/sealed-secrets \
    --namespace kube-system \
    --set-string fullnameOverride=sealed-secrets-controller
  if [ $? -ne 0 ]; then
    echo "Failed to install Sealed Secrets controller. Exiting."
    exit 1
  fi
  echo "Waiting for Sealed Secrets controller to be ready..."
  kubectl wait --namespace kube-system \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/name=sealed-secrets \
    --timeout=120s
else
  echo "Sealed Secrets controller is already installed."
fi

# Set the directory containing the Kubernetes manifests
K8S_DIR="./k8s"

# Check if the directory exists
if [ ! -d "$K8S_DIR" ]; then
  echo "Directory $K8S_DIR does not exist. Please create the directory or specify the correct path."
  exit 1
fi

# Apply environment-specific sealed secrets (before base resources so secrets exist for deployments)
SEALED_DIR="$K8S_DIR/sealed-secrets/$ENV"
if [ -d "$SEALED_DIR" ]; then
  echo "Applying sealed secrets for $ENV..."
  for file in $(ls "$SEALED_DIR" | grep -E '\.ya?ml$' | sort); do
    echo "Applying sealed secret $file..."
    kubectl apply -f "$SEALED_DIR/$file"
    if [ $? -ne 0 ]; then
      echo "Failed to apply sealed secret $file. Exiting."
      exit 1
    fi
  done
else
  echo "Warning: No sealed secrets directory found at $SEALED_DIR"
fi

# Apply base infrastructure resources (namespaces, postgres, etc.)
echo "Applying infrastructure resources..."
kubectl apply -k "$K8S_DIR/infrastructure"
if [ $? -ne 0 ]; then
  echo "Failed to apply infrastructure resources. Exiting."
  exit 1
fi

# Apply Atlas schema
echo "Applying Atlas schema..."
kubectl apply -k "$K8S_DIR/atlas-schema"
if [ $? -ne 0 ]; then
  echo "Failed to apply Atlas schema. Exiting."
  exit 1
fi

# Apply environment-specific app resources using Kustomize
echo "Applying $ENV environment app resources..."
kubectl apply -k "$K8S_DIR/overlays/$ENV"
if [ $? -ne 0 ]; then
  echo "Failed to apply $ENV overlay. Exiting."
  exit 1
fi

echo "All files applied successfully."