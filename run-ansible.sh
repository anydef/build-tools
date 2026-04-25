#!/bin/bash

set -e

# =============================================================================
# run-ansible.sh — run an Ansible playbook with 1Password secrets injection
# =============================================================================
# Usage:
#   run-ansible.sh <playbook-path> [extra ansible-playbook args]
#
# Environment variables:
#   ANSIBLE_INVENTORY   Path to the inventory file (required)
#   ENV_FILE            Path to .env.tpl to load (defaults to $PWD/.env.tpl)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PLAYBOOK="${1:?Usage: run-ansible.sh <playbook-path> [extra args]}"
shift

# Load secrets from .env.tpl via load-env-tpl.sh.
# OP Connect credentials are only fetched when there is actually a .env.tpl to load.
_ENV_FILE="${ENV_FILE:-$PWD/.env.tpl}"
if [ -z "${_OP_LOADED:-}" ] && [ -f "${_ENV_FILE}" ]; then
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
    source "${SCRIPT_DIR}/load-env-tpl.sh" "${_ENV_FILE}"
fi

ansible-playbook \
    -i "${ANSIBLE_INVENTORY:?ANSIBLE_INVENTORY is required}" \
    "${PLAYBOOK}" \
    "$@"
