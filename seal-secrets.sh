#!/bin/bash
set -euo pipefail

# seal-secrets.sh - Converts plain Secret YAMLs into SealedSecret YAMLs
#
# Prerequisites:
#   - kubeseal CLI installed (brew install kubeseal)
#   - kubectl configured with access to the target cluster
#   - SealedSecrets controller running in the cluster
#
# Usage:
#   ./seal-secrets.sh              # Seal for default namespace
#   ./seal-secrets.sh prod         # Seal for prod namespace

NAMESPACE="${1:-default}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="$SCRIPT_DIR/k8s"
SECRETS_DIR="$SCRIPT_DIR/k8s/secrets-plain"

echo "Sealing secrets for namespace: $NAMESPACE"

# Verify kubeseal is available
if ! command -v kubeseal &> /dev/null; then
  echo "Error: kubeseal CLI not found. Install it first:"
  echo "  brew install kubeseal   # macOS"
  echo "  # or download from https://github.com/bitnami-labs/sealed-secrets/releases"
  exit 1
fi

# Verify controller is reachable
if ! kubeseal --fetch-cert > /dev/null 2>&1; then
  echo "Error: Cannot reach SealedSecrets controller. Make sure:"
  echo "  1. The controller is installed in the cluster"
  echo "  2. kubectl is configured to reach the cluster"
  exit 1
fi

FOUND=0

for secret_file in "$SECRETS_DIR"/*.yaml "$SECRETS_DIR"/*.yml; do
  [ -f "$secret_file" ] || continue

  # Skip .example files
  if [[ "$secret_file" == *.example ]]; then
    continue
  fi

  FOUND=1
  filename=$(basename "$secret_file")
  # Derive sealed filename: e.g., postgres-secrets.yaml -> 000-postgres-sealed-secret.yaml
  base="${filename%.*}"
  sealed_filename="000-${base%-secret*}-sealed-secret.yaml"

  echo "Sealing $filename -> $sealed_filename (namespace: $NAMESPACE)"

  # Create a temp file with the namespace set
  tmp_file=$(mktemp)
  trap "rm -f $tmp_file" EXIT

  # Set the namespace in the secret before sealing
  if command -v yq &> /dev/null; then
    yq ".metadata.namespace = \"$NAMESPACE\"" "$secret_file" > "$tmp_file"
  else
    # Fallback: use sed to add/replace namespace
    sed "s|^\(  name:.*\)|\1\n  namespace: $NAMESPACE|" "$secret_file" | \
      sed '/^  namespace:/{ n; /^  namespace:/d; }' > "$tmp_file"
  fi

  kubeseal --format yaml \
    --controller-name=sealed-secrets-controller \
    --controller-namespace=kube-system \
    < "$tmp_file" \
    > "$K8S_DIR/$sealed_filename"

  rm -f "$tmp_file"

  echo "  Created $K8S_DIR/$sealed_filename"
done

if [ "$FOUND" -eq 0 ]; then
  echo "No plain secret files found in $SECRETS_DIR/"
  echo "Copy a .example file to .yaml and fill in real values first:"
  echo "  cp $SECRETS_DIR/postgres-secrets.yaml.example $SECRETS_DIR/postgres-secrets.yaml"
  exit 1
fi

echo ""
echo "Done. Sealed secrets are in $K8S_DIR/"
echo "Remember: DO NOT commit the plain secret files in $SECRETS_DIR/"
