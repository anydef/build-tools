#!/bin/bash

set -e

# =============================================================================
# Terraform-only Deployment
# =============================================================================
# Lightweight deploy script that runs terraform init/plan/apply without
# requiring Portainer or Docker registry variables.
# Suitable for infrastructure-only projects (DNS, HAProxy, etc.).
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${TERRAFORM_DIR:-${SCRIPT_DIR}/terraform}"

# Load environment variables from .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Load 1Password Connect credentials when not running in CI and not already set.
if [ -z "${GITHUB_ACTIONS:-}" ] && [ -z "${GITEA_ACTIONS:-}" ] && [ -z "${OP_CONNECT_HOST:-}" ]; then
    if ! command -v op &> /dev/null; then
        echo "Error: 'op' (1Password CLI) is not available to load Connect credentials"
        exit 1
    fi
    export OP_CONNECT_HOST="$(op read 'op://HomeLab/1password-connect/hostname')"
    export OP_CONNECT_TOKEN="$(op read 'op://HomeLab/1password-connect/token')"
fi

# Ensure OP_CONNECT_HOST has a URL scheme (op read returns bare host:port).
if [ -n "${OP_CONNECT_HOST:-}" ] && [[ "${OP_CONNECT_HOST}" != http://* ]] && [[ "${OP_CONNECT_HOST}" != https://* ]]; then
    export OP_CONNECT_HOST="http://${OP_CONNECT_HOST}"
fi

# Resolve secrets from .env.tpl via op inject.
_inject_env_tpl() {
    local file="$1"
    grep -v '^[[:space:]]*#' "$file" | op inject
}

if [ -z "${_OP_LOADED:-}" ] && [ -f "$SCRIPT_DIR/.env.tpl" ]; then
    if ! command -v op &> /dev/null; then
        echo "Error: 'op' (1Password CLI) is not available to resolve .env.tpl"
        exit 1
    fi
    eval "$(_inject_env_tpl "$SCRIPT_DIR/.env.tpl")"
fi

if [ -z "${_OP_LOADED:-}" ] && [ "$PWD" != "$SCRIPT_DIR" ] && [ -f "$PWD/.env.tpl" ]; then
    set -a
    eval "$(_inject_env_tpl "$PWD/.env.tpl")"
    set +a
fi

echo "========================================"
echo "Terraform Deployment"
echo "========================================"
echo "  Terraform Directory: ${TERRAFORM_DIR}"
echo "========================================"

if ! command -v terraform &> /dev/null; then
    echo "Error: terraform is not installed or not in PATH"
    exit 1
fi

# Clear cached providers/modules to ensure a clean init every deployment
echo "Clearing local Terraform cache..."
rm -rf "${TERRAFORM_DIR}/.terraform"
echo ""

echo "Initializing Terraform..."
terraform -chdir="${TERRAFORM_DIR}" init
echo ""

echo "Planning deployment..."
terraform -chdir="${TERRAFORM_DIR}" plan
echo ""

read -p "Apply this configuration? (yes/no): " confirm
if [ "$confirm" != "yes" ] && [ "$confirm" != "y" ]; then
    echo "Deployment cancelled."
    exit 0
fi

echo ""
echo "Applying Terraform configuration..."
terraform -chdir="${TERRAFORM_DIR}" apply -auto-approve

echo ""
echo "========================================"
echo "✓ Deployment completed successfully!"
echo "========================================"