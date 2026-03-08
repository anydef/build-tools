#!/bin/bash

set -e

# =============================================================================
# Load secrets from .env.tpl files via op inject
# =============================================================================
# Usage:
#   load-env-tpl.sh [file1.env.tpl [file2.env.tpl ...]]
#
# With no arguments, loads:
#   1. <SCRIPT_DIR>/.env.tpl  (build-tools base secrets)
#   2. $PWD/.env.tpl          (project-level secrets, if PWD != SCRIPT_DIR)
#
# Resolves op:// references via the op CLI, strips surrounding quotes from
# values, exports vars to the current shell, and — when $GITHUB_ENV is set —
# appends them in heredoc format for cross-step availability in CI.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v op &> /dev/null; then
    echo "Error: 'op' (1Password CLI) is not installed or not in PATH"
    exit 1
fi

# Build list of files to process
if [ "$#" -gt 0 ]; then
    FILES=("$@")
else
    FILES=()
    [ -f "$SCRIPT_DIR/.env.tpl" ] && FILES+=("$SCRIPT_DIR/.env.tpl")
    if [ "$PWD" != "$SCRIPT_DIR" ] && [ -f "$PWD/.env.tpl" ]; then
        FILES+=("$PWD/.env.tpl")
    fi
fi

if [ "${#FILES[@]}" -eq 0 ]; then
    echo "Warning: no .env.tpl files found to load"
    exit 0
fi

_load_file() {
    local file="$1"
    echo "Loading secrets from $file..."

    while IFS= read -r line || [ -n "$line" ]; do
        # Skip blank lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue

        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"

            # Strip surrounding double or single quotes
            if [[ "$value" =~ ^\"(.*)\"$ ]]; then
                value="${BASH_REMATCH[1]}"
            elif [[ "$value" =~ ^\'(.*)\'$ ]]; then
                value="${BASH_REMATCH[1]}"
            fi

            export "$key=$value"

            # Propagate to subsequent CI steps via GITHUB_ENV heredoc format
            if [ -n "${GITHUB_ENV:-}" ]; then
                printf '%s<<%s\n%s\n%s\n' "$key" "__EOF_${key}__" "$value" "__EOF_${key}__" >> "$GITHUB_ENV"
            fi
        fi
    # Strip comment lines before passing to op inject so that commented-out
    # op:// references are not resolved.
    done < <(grep -v '^[[:space:]]*#' "$file" | op inject)
}

for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "Error: file not found: $file"
        exit 1
    fi
    _load_file "$file"
done