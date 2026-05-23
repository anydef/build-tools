# =============================================================================
# common.just — shared recipes for tower services (mirror of common.mk)
# =============================================================================
# Import from a service justfile after setting:
#
#   docker_image_name  (required for `build`)
#   base_image_name    (optional; selects Dockerfile.<base_image_name>)
#   image_tag          (default: latest)
#   build_context      (default: justfile_directory())
#   terraform_dir      (default: build_context/terraform)
#   ansible_inventory  (default: build_context/hosts.yaml)
#   ansible_playbook_dir (default: build_context/ansible)
#   build_tools_dir    (REQUIRED — caller must provide; typically from bootstrap.just)
#
# Variables defined in the importing justfile take precedence (shallower wins).
# =============================================================================

build_context         := justfile_directory()
image_tag             := env_var_or_default("IMAGE_TAG", "latest")
terraform_dir         := build_context / "terraform"
ansible_inventory     := build_context / "hosts.yaml"
ansible_playbook_dir  := build_context / "ansible"

docker_image_name := env_var_or_default("DOCKER_IMAGE_NAME", "")
base_image_name   := env_var_or_default("BASE_IMAGE_NAME", "")

# Build and push the project's Docker image(s) via build-and-push.sh.
build:
    DOCKER_IMAGE_NAME={{docker_image_name}} \
    BASE_IMAGE_NAME={{base_image_name}} \
    IMAGE_TAG={{image_tag}} \
    BUILD_CONTEXT={{build_context}} \
    {{build_tools_dir}}/build-and-push.sh

# Deploy a Portainer stack via Terraform.
deploy:
    TERRAFORM_DIR={{terraform_dir}} {{build_tools_dir}}/deploy-portainer.sh

# Deploy infrastructure (DNS, HAProxy, etc.) via Terraform only — no Portainer.
deploy-tf:
    TERRAFORM_DIR={{terraform_dir}} {{build_tools_dir}}/deploy-terraform.sh

# Run an Ansible playbook from ansible_playbook_dir.
# Usage: just run-ansible <playbook-filename> [extra ansible-playbook args]
# Example: just run-ansible foo.yaml --tags argocd
run-ansible playbook *args:
    ANSIBLE_INVENTORY={{ansible_inventory}} \
    ENV_FILE={{build_context}}/.env.tpl \
    {{build_tools_dir}}/run-ansible.sh {{ansible_playbook_dir}}/{{playbook}} {{args}}
