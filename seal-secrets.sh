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
#   ./seal-secrets.sh                              # Seal all secrets for default environment
#   ./seal-secrets.sh prod                         # Seal all secrets for prod environment
#   ./seal-secrets.sh default/postgres-secrets.yaml # Seal a single file (env/filename)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SINGLE_FILE=""

# Check if argument contains a path separator (env/file)
if [[ "${1:-}" == */* ]]; then
  ENV="${1%%/*}"
  SINGLE_FILE="$SCRIPT_DIR/k8s/secrets-plain/$1"
else
  ENV="${1:-default}"
fi

SECRETS_DIR="$SCRIPT_DIR/k8s/secrets-plain/$ENV"
SEALED_DIR="$SCRIPT_DIR/k8s/sealed-secrets/$ENV"

echo "Sealing secrets for environment: $ENV"

# Verify kubeseal is available
if ! command -v kubeseal &> /dev/null; then
  echo "Error: kubeseal CLI not found. Install it first:"
  echo "  brew install kubeseal   # macOS"
  echo "  # or download from https://github.com/bitnami-labs/sealed-secrets/releases"
  exit 1
fi

# Verify controller is reachable
if ! kubeseal --fetch-cert --controller-name=sealed-secrets-controller --controller-namespace=sealed-secrets > /dev/null 2>&1; then
  echo "Error: Cannot reach SealedSecrets controller. Make sure:"
  echo "  1. The controller is installed in the cluster"
  echo "  2. kubectl is configured to reach the cluster"
  exit 1
fi

if [ ! -d "$SECRETS_DIR" ]; then
  echo "Error: Secrets directory not found at $SECRETS_DIR"
  echo "Create it and add plain secret YAML files:"
  echo "  mkdir -p $SECRETS_DIR"
  echo "  cp k8s/secrets-plain/*.example $SECRETS_DIR/"
  echo "  # Rename .example to .yaml and fill in real values"
  exit 1
fi

mkdir -p "$SEALED_DIR"

# Build file list: single file or all files in the directory
if [ -n "$SINGLE_FILE" ]; then
  if [ ! -f "$SINGLE_FILE" ]; then
    echo "Error: File not found: $SINGLE_FILE"
    exit 1
  fi
  FILE_LIST=("$SINGLE_FILE")
else
  FILE_LIST=()
  for f in "$SECRETS_DIR"/*.yaml "$SECRETS_DIR"/*.yml; do
    [ -f "$f" ] || continue
    [[ "$f" == *.example ]] && continue
    FILE_LIST+=("$f")
  done
fi

if [ ${#FILE_LIST[@]} -eq 0 ]; then
  echo "No plain secret files found in $SECRETS_DIR/"
  echo "Copy a .example template and fill in real values:"
  echo "  cp k8s/secrets-plain/postgres-secrets.yaml.example $SECRETS_DIR/postgres-secrets.yaml"
  exit 1
fi

for secret_file in "${FILE_LIST[@]}"; do
  filename=$(basename "$secret_file")
  # Derive sealed filename: e.g., postgres-secrets.yaml -> 000-postgres-sealed-secret.yaml
  base="${filename%.*}"
  sealed_filename="000-${base%-secret*}-sealed-secret.yaml"

  echo "Sealing $filename -> $sealed_filename (environment: $ENV)"

  # Create a temp file with the namespace set
  tmp_file=$(mktemp)
  trap "rm -f $tmp_file" EXIT

  # Set the namespace in the secret before sealing
  if command -v yq &> /dev/null; then
    yq ".metadata.namespace = \"default\"" "$secret_file" > "$tmp_file"
  else
    # Fallback: use sed to add/replace namespace
    sed "s|^\(  name:.*\)|\1\n  namespace: default|" "$secret_file" | \
      sed '/^  namespace:/{ n; /^  namespace:/d; }' > "$tmp_file"
  fi

  kubeseal --format yaml \
    --controller-name=sealed-secrets-controller \
    --controller-namespace=sealed-secrets \
    < "$tmp_file" \
    > "$SEALED_DIR/$sealed_filename"

  rm -f "$tmp_file"

  echo "  Created $SEALED_DIR/$sealed_filename"
done

echo ""
echo "Done. Sealed secrets are in $SEALED_DIR/"
echo "Remember: DO NOT commit the plain secret files in $SECRETS_DIR/"
