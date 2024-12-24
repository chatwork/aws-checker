#!/usr/bin/env bash

set -e

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
env_file="${dir}/.env.config"

kubectl create configmap aws-checker --dry-run=client --from-env-file="${env_file}" --output=yaml
