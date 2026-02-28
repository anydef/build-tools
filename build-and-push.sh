#!/bin/bash

set -xe

# =============================================================================
# Build and Push Docker Images
# =============================================================================
# Intended for use as an include/target in Makefiles.
#
# Required variables (set externally or via .env):
#   DOCKER_IMAGE_NAME   - main image name (without registry prefix)
#   DOCKER_REGISTRY     - Docker registry URL
#
# Optional variables:
#   BASE_IMAGE_NAME     - base image name; building is skipped if
#                         Dockerfile.<BASE_IMAGE_NAME> does not exist
#   IMAGE_TAG           - image tag (default: latest)
#   BUILD_CONTEXT       - Docker build context dir (default: SCRIPT_DIR)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_CONTEXT="${BUILD_CONTEXT:-$SCRIPT_DIR}"

# Load environment variables from .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Resolve secrets from .env.tpl via op run (re-exec with all secrets injected)
# _OP_LOADED guards against infinite re-execution after op run re-invokes this script.
if [ -f "$SCRIPT_DIR/.env.tpl" ] && [ -z "$_OP_LOADED" ]; then
    if ! command -v op &> /dev/null; then
        echo "Error: 'op' (1Password CLI) is not available to resolve .env.tpl"
        exit 1
    fi
    export _OP_LOADED=1
    exec op run --env-file "$SCRIPT_DIR/.env.tpl" -- "$0" "$@"
fi

# Validate required variables
if [ -z "$DOCKER_IMAGE_NAME" ]; then
    echo "Error: DOCKER_IMAGE_NAME is not set"
    exit 1
fi
if [ -z "$DOCKER_REGISTRY" ]; then
    echo "Error: DOCKER_REGISTRY is not set (set it directly or provide .env / .env.tpl)"
    exit 1
fi

IMAGE_TAG="${IMAGE_TAG:-latest}"
FULL_IMAGE_NAME="${DOCKER_REGISTRY}/${DOCKER_IMAGE_NAME}:${IMAGE_TAG}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Determine whether a base image build is needed
BASE_DOCKERFILE=""
if [ -n "$BASE_IMAGE_NAME" ]; then
    CANDIDATE="${BUILD_CONTEXT}/Dockerfile.${BASE_IMAGE_NAME}"
    if [ -f "$CANDIDATE" ]; then
        BASE_DOCKERFILE="$CANDIDATE"
    fi
fi

FULL_BASE_IMAGE=""
if [ -n "$BASE_DOCKERFILE" ]; then
    FULL_BASE_IMAGE="${DOCKER_REGISTRY}/${BASE_IMAGE_NAME}:${IMAGE_TAG}"
fi

# ---- Header ----------------------------------------------------------------
echo -e "${BLUE}========================================"
echo "Building and Pushing Docker Images"
echo "========================================"
echo -e "Registry:   ${YELLOW}${DOCKER_REGISTRY}${NC}"
if [ -n "$FULL_BASE_IMAGE" ]; then
    echo -e "Base image: ${YELLOW}${FULL_BASE_IMAGE}${NC}"
else
    echo -e "Base image: ${YELLOW}(none)${NC}"
fi
echo -e "Main image: ${YELLOW}${FULL_IMAGE_NAME}${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# ---- Base image (optional) -------------------------------------------------
STEP=1
TOTAL_STEPS=2
if [ -n "$BASE_DOCKERFILE" ]; then
    TOTAL_STEPS=4

    echo -e "${BLUE}[${STEP}/${TOTAL_STEPS}] Building base image...${NC}"
    docker build -f "$BASE_DOCKERFILE" -t "${FULL_BASE_IMAGE}" "$BUILD_CONTEXT"
    echo -e "${GREEN}✓ Base image built successfully${NC}"
    echo ""
    STEP=$((STEP + 1))

    echo -e "${BLUE}[${STEP}/${TOTAL_STEPS}] Pushing base image to registry...${NC}"
    docker push "${FULL_BASE_IMAGE}"
    echo -e "${GREEN}✓ Base image pushed successfully${NC}"
    echo ""
    STEP=$((STEP + 1))
fi

# ---- Main image ------------------------------------------------------------
echo -e "${BLUE}[${STEP}/${TOTAL_STEPS}] Building main image...${NC}"
docker build --build-arg DOCKER_REGISTRY="${DOCKER_REGISTRY}" -t "${FULL_IMAGE_NAME}" "$BUILD_CONTEXT"
echo -e "${GREEN}✓ Main image built successfully${NC}"
echo ""
STEP=$((STEP + 1))

echo -e "${BLUE}[${STEP}/${TOTAL_STEPS}] Pushing main image to registry...${NC}"
docker push "${FULL_IMAGE_NAME}"
echo -e "${GREEN}✓ Main image pushed successfully${NC}"
echo ""

# ---- Summary ---------------------------------------------------------------
echo -e "${GREEN}========================================"
echo "✓ Build and Push Complete"
echo "========================================"
echo -e "Registry images:${NC}"
if [ -n "$FULL_BASE_IMAGE" ]; then
    echo "  - ${FULL_BASE_IMAGE}"
fi
echo "  - ${FULL_IMAGE_NAME}"
echo -e "${GREEN}========================================${NC}"
