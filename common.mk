# =============================================================================
# common.mk — shared build and deploy targets
# =============================================================================
# Include this file from a project Makefile after setting:
#
#   DOCKER_IMAGE_NAME  (required)
#   BASE_IMAGE_NAME    (optional)
#   IMAGE_TAG          (default: latest)
#   BUILD_CONTEXT      (default: project dir via $(CURDIR) at include time)
#   TERRAFORM_DIR      (default: $(BUILD_CONTEXT)/terraform)
#   BUILD_TOOLS_DIR    (default: directory of this file)
# =============================================================================

# Resolve BUILD_TOOLS_DIR to the directory containing this file when not set.
BUILD_TOOLS_DIR ?= $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

BUILD_CONTEXT ?= $(CURDIR)
IMAGE_TAG     ?= latest
TERRAFORM_DIR ?= $(BUILD_CONTEXT)/terraform

.PHONY: build deploy help

help:
	@echo "Targets:"
	@echo "  build   Build and push Docker images (base + main)"
	@echo "  deploy  Deploy stack to Portainer via Terraform"

build:
	DOCKER_IMAGE_NAME=$(DOCKER_IMAGE_NAME) \
	BASE_IMAGE_NAME=$(BASE_IMAGE_NAME) \
	IMAGE_TAG=$(IMAGE_TAG) \
	BUILD_CONTEXT=$(BUILD_CONTEXT) \
	$(BUILD_TOOLS_DIR)/build-and-push.sh

# NOTE: deploy-portainer.sh resolves TERRAFORM_DIR relative to its own SCRIPT_DIR.
# We temporarily symlink the project terraform dir there for the duration of the call.
deploy:
	@bash -ec ' \
		if [ ! -d "$(BUILD_TOOLS_DIR)/terraform" ]; then \
			ln -s $(TERRAFORM_DIR) $(BUILD_TOOLS_DIR)/terraform; \
			trap "rm -f $(BUILD_TOOLS_DIR)/terraform" EXIT; \
		fi; \
		$(BUILD_TOOLS_DIR)/deploy-portainer.sh \
	'