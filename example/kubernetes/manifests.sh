#!/usr/bin/env bash

set -e

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

"$dir"/configmap.sh
echo "---"
"$dir"/secret.sh
echo "---"
cat "$dir"/aws-checker.yaml
