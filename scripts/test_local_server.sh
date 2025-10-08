#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/test_local_server.sh [--host URL] [--token TOKEN]

Runs a full local test against a running CoderForge API server:
 - Builds the dev provider binary
 - Uses .terraformrc dev override for local provider
 - For each example (function, container_registry):
   init -> validate -> plan -> apply -> output -> destroy

Options:
  --host   Local API base URL (default: http://127.0.0.1:8080)
  --token  API token to use (default: dev-token)

Environment overrides:
  CODERFORGE_API_URL, CODERFORGE_HOST, CODERFORGE_CLOUD_TOKEN, CODERFORGE_TOKEN
EOF
}

ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${ROOT_DIR}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
  ROOT_DIR="${SCRIPT_DIR%/scripts}"
fi

HOST_URL="http://127.0.0.1:8080"
TOKEN="dev-token"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      HOST_URL="$2"; shift 2 ;;
    --token)
      TOKEN="$2"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      echo "Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "go is required on PATH" >&2
  exit 2
fi

if ! command -v terraform >/dev/null 2>&1; then
  echo "terraform is required on PATH" >&2
  exit 2
fi

export TF_CLI_CONFIG_FILE="${ROOT_DIR}/.terraformrc"
export CODERFORGE_API_URL="${HOST_URL}"
export CODERFORGE_CLOUD_TOKEN="${TOKEN}"

echo "Building dev provider ..."
(
  cd "${ROOT_DIR}"
  go build -o registry.terraform.io/coderforge/coderforge
)

run_example() {
  local example_dir="$1"
  echo "\n=== Testing example: ${example_dir} ==="
  (
    cd "${ROOT_DIR}/${example_dir}"
    terraform init -input=false -upgrade
    terraform validate
    terraform plan -out=tfplan
    terraform apply -auto-approve tfplan
    terraform output -json || true
    terraform destroy -auto-approve || {
      echo "WARN: destroy failed; attempting re-run once" >&2
      terraform destroy -auto-approve || true
    }
  )
}

run_example examples/resources/function
run_example examples/resources/container_registry

echo "\nAll local tests completed successfully."

