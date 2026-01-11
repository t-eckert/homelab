#!/bin/bash
# Load spark CLI environment variables from Kubernetes secret
# Usage: source cluster/spark/load-env.sh

if ! kubectl get secret spark-cli-config -n spark &>/dev/null; then
    echo "Error: spark-cli-config secret not found in spark namespace"
    echo "Please create it first: kubectl apply -f cluster/spark/secret.yaml"
    return 1 2>/dev/null || exit 1
fi

export ANTHROPIC_API_KEY=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.ANTHROPIC_API_KEY}' | base64 -d)
export POSTGRES_PASSWORD=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)
export GITHUB_TOKEN=$(kubectl get secret spark-cli-config -n spark -o jsonpath='{.data.GITHUB_TOKEN}' | base64 -d)

echo "✓ Environment variables loaded from Kubernetes secret"
echo "  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY:0:10}..."
echo "  POSTGRES_PASSWORD: ***"
echo "  GITHUB_TOKEN: ${GITHUB_TOKEN:0:10}..."
