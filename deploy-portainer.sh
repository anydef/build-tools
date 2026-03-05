#!/bin/bash

set -e

# =============================================================================
# Portainer Stack Deployment via Terraform
# =============================================================================
# This script wraps Terraform deployment, loading secrets from .env and
# passing them as Terraform variables.
#
# The legacy API-based deployment script is available as:
# deploy-portainer-legacy.sh
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${TERRAFORM_DIR:-${SCRIPT_DIR}/terraform}"

# Load environment variables from .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Resolve secrets from .env.tpl via op inject
if [ -z "${_OP_LOADED:-}" ] && [ -f "$SCRIPT_DIR/.env.tpl" ]; then
    if ! command -v op &> /dev/null; then
        echo "Error: 'op' (1Password CLI) is not available to resolve .env.tpl"
        exit 1
    fi
    eval "$(op inject -i "$SCRIPT_DIR/.env.tpl")"
fi

# Also load a project-level .env.tpl from the working directory if it exists and
# differs from the build-tools dir. All resolved vars are auto-exported so that
# any TF_VAR_* entries are picked up by Terraform without further wiring.
if [ -z "${_OP_LOADED:-}" ] && [ "$PWD" != "$SCRIPT_DIR" ] && [ -f "$PWD/.env.tpl" ]; then
    set -a
    eval "$(op inject -i "$PWD/.env.tpl")"
    set +a
fi

# Validate required variables
if [ -z "$PORTAINER_URL" ] || [ -z "$PORTAINER_ACCESS_TOKEN" ] || [ -z "$DOCKER_REGISTRY" ]; then
    echo "Error: Missing required environment variables"
    echo "Required: PORTAINER_URL, PORTAINER_ACCESS_TOKEN, DOCKER_REGISTRY"
    exit 1
fi

echo "========================================"
echo "Portainer Stack Deployment (Terraform)"
echo "========================================"

# Export Terraform variables (TF_VAR_* are automatically picked up by Terraform)
export TF_VAR_portainer_url="${PORTAINER_URL%/}"  # Remove trailing slash
export TF_VAR_portainer_api_key="${PORTAINER_ACCESS_TOKEN}"
export TF_VAR_docker_registry="${DOCKER_REGISTRY}"
export TF_VAR_force_update="$(date +%s)"  # Timestamp to force stack update

echo "Configuration:"
echo "  Portainer URL: ${TF_VAR_portainer_url}"
echo "  Docker Registry: ${TF_VAR_docker_registry}"
echo "  Terraform Directory: ${TERRAFORM_DIR}"
echo "========================================"

# Check if Terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "Error: terraform is not installed or not in PATH"
    echo "Install from: https://www.terraform.io/downloads"
    exit 1
fi

# Clear cached providers/modules to ensure a clean init every deployment
echo "Clearing local Terraform cache..."
rm -rf "${TERRAFORM_DIR}/.terraform"
echo ""

# Initialize Terraform
echo "Initializing Terraform..."
terraform -chdir="${TERRAFORM_DIR}" init
echo ""

# Plan the deployment
echo "Planning deployment..."
terraform -chdir="${TERRAFORM_DIR}" plan
echo ""

# Ask for confirmation before applying
read -p "Apply this configuration? (yes/no): " confirm
if [ "$confirm" != "yes" ] && [ "$confirm" != "y" ]; then
    echo "Deployment cancelled."
    exit 0
fi

# Apply the configuration
echo ""
echo "Applying Terraform configuration..."
terraform -chdir="${TERRAFORM_DIR}" apply -auto-approve

# Display outputs
echo ""
echo "========================================"
echo "✓ Deployment completed successfully!"
echo "========================================"
echo ""

# Get outputs from Terraform
terraform -chdir="${TERRAFORM_DIR}" output

echo ""
echo "Next steps:"
echo "  - Access the application at the URL above"
echo "  - View the stack in Portainer UI"
echo "  - Check logs: docker logs helloworld-python"
echo ""
