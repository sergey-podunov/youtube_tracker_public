#!/bin/bash

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

# Loop through files in alphabetical order and apply them one by one
for file in $(ls "$K8S_DIR" | sort); do
  # Ensure only .yaml or .yml files are processed
  if [[ $file == *.yaml || $file == *.yml ]]; then
    echo "Applying $file..."
    kubectl apply -f "$K8S_DIR/$file"

    # Check if the apply command was successful
    if [ $? -ne 0 ]; then
      echo "Failed to apply $file. Exiting."
      exit 1
    fi
  fi
done

kubectl apply -f database/atlas-schema.yaml

echo "All files applied successfully."