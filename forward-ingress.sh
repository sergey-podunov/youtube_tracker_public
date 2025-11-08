#!/usr/bin/env bash

NAMESPACE="${1:-default}"
LOCAL_PORT="${2:-8080}"

echo "Port forwarding youtube-tracker in namespace: $NAMESPACE"
echo "Access at: http://localhost:$LOCAL_PORT"
echo ""
echo "Press Ctrl+C to stop"

kubectl port-forward svc/ingress-nginx-controller $LOCAL_PORT:80 -n ingress-nginx
