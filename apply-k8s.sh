#!/bin/bash

# Parse environment argument (default: default)
ENV="${1:-default}"

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

# Set the directory containing the Kubernetes manifests
K8S_DIR="./k8s"

# Check if the directory exists
if [ ! -d "$K8S_DIR" ]; then
  echo "Directory $K8S_DIR does not exist. Please create the directory or specify the correct path."
  exit 1
fi

# Apply base k8s resources (postgres, etc.)
echo "Applying base resources..."
for file in $(ls "$K8S_DIR" | grep -E '\.ya?ml$' | sort); do
  # Skip base and overlays directories
  if [[ -f "$K8S_DIR/$file" ]]; then
    echo "Applying $file..."
    kubectl apply -f "$K8S_DIR/$file"

    # Check if the apply command was successful
    if [ $? -ne 0 ]; then
      echo "Failed to apply $file. Exiting."
      exit 1
    fi
  fi
done

# Apply environment-specific app resources using Kustomize
echo "Applying $ENV environment app resources..."
kubectl apply -k "$K8S_DIR/overlays/$ENV"
if [ $? -ne 0 ]; then
  echo "Failed to apply $ENV overlay. Exiting."
  exit 1
fi

kubectl apply -f database/atlas-schema.yaml

echo "All files applied successfully."