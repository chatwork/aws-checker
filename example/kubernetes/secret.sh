#!/usr/bin/env bash

set -e

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
env_file="${dir}/.env.secret"

kubectl create secret generic aws-checker --dry-run=client --from-env-file="${env_file}" --output=yaml

echo "---"

kubectl create secret generic datadog-secret --from-literal api-key="${DD_API_KEY}" --dry-run=client --output=yaml
