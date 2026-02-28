# build-tools

Shared build and deployment tooling for Docker + Portainer projects.

Provides:
- **`common.mk`** — reusable Makefile targets for building/pushing Docker images and deploying to Portainer
- **`build-and-push.sh`** — Docker build and push script with optional base-image support
- **`deploy-portainer.sh`** — Terraform-based Portainer stack deployment
- **`terraform/portainer-stack`** — reusable Terraform module that creates a Portainer stack

---

## Prerequisites

| Tool | Purpose |
|---|---|
| `docker` | Build and push images |
| `terraform` | Deploy Portainer stacks |
| `op` (1Password CLI) | Inject secrets from `.env.tpl` |

---

## Secret management

Copy `.env.tpl` into your project and fill in the 1Password references:

```bash
PORTAINER_URL="op://HomeLab/Portainer Tower/website"
PORTAINER_ACCESS_TOKEN="op://HomeLab/Portainer Tower/access_token"
DOCKER_REGISTRY="op://HomeLab/DockerRegistry/hostname"
```

The scripts automatically run themselves via `op run --env-file .env.tpl` to inject secrets at runtime. For local development without 1Password, create a plain `.env` file with the same keys set to real values — this takes precedence.

---

## Makefile integration

### 1. Define your variables, then include `common.mk`

The recommended pattern is to auto-clone build-tools on first use:

```makefile
# =============================================================================
# my-service
# =============================================================================

DOCKER_IMAGE_NAME := my-service
BASE_IMAGE_NAME   := my-service-base   # optional — omit if no base image
IMAGE_TAG         := latest
BUILD_CONTEXT     := $(CURDIR)
TERRAFORM_DIR     := $(CURDIR)/terraform

BUILD_TOOLS_DIR := .build/build-tools

-include $(BUILD_TOOLS_DIR)/common.mk
$(BUILD_TOOLS_DIR)/common.mk:
	git clone --depth=1 https://github.com/anydef/build-tools $(BUILD_TOOLS_DIR)
```

The `-include` (note the leading `-`) silently skips the missing file on the first run, which triggers the pattern rule to clone the repo. On every subsequent run the file exists and is included normally.

Add `.build/` to your `.gitignore`:

```
/.build/
```

### 2. Available variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DOCKER_IMAGE_NAME` | yes | — | Name of the main Docker image |
| `BASE_IMAGE_NAME` | no | — | Name of the base image (triggers `Dockerfile.base` build) |
| `IMAGE_TAG` | no | `latest` | Tag applied to all built images |
| `BUILD_CONTEXT` | no | `$(CURDIR)` | Docker build context directory |
| `TERRAFORM_DIR` | no | `$(BUILD_CONTEXT)/terraform` | Path to your project's terraform directory |
| `BUILD_TOOLS_DIR` | no | auto-detected | Path to this cloned repo |

### 3. Available targets

```
make help    # List available targets
make build   # Build and push Docker image(s)
make deploy  # Deploy stack to Portainer via Terraform
```

### 4. Base image (optional)

If `BASE_IMAGE_NAME` is set, the build script looks for a `Dockerfile.base` in the build context and builds it first, pushing it as `$DOCKER_REGISTRY/$BASE_IMAGE_NAME:$IMAGE_TAG`. Your main `Dockerfile` can then reference it:

```dockerfile
ARG DOCKER_REGISTRY
FROM ${DOCKER_REGISTRY}/my-service-base:latest AS builder
```

This caches heavy dependencies (e.g. compiling Rust crates) separately from application code changes.

---

## Terraform module

The `terraform/portainer-stack` module creates a Portainer stack from a Docker Compose file.

### Import via local path (after auto-clone)

```hcl
module "portainer_stack" {
  source = "../.build/build-tools/terraform/portainer-stack"

  stack_name         = var.stack_name
  endpoint_id        = var.endpoint_id
  stack_file_content = file("${path.module}/../docker-compose.yml")
  docker_registry    = var.docker_registry
  force_update       = var.force_update
}
```

### Import directly from GitHub

```hcl
module "portainer_stack" {
  source = "github.com/anydef/build-tools//terraform/portainer-stack?ref=main"

  stack_name         = var.stack_name
  endpoint_id        = var.endpoint_id
  stack_file_content = file("${path.module}/../docker-compose.yml")
  docker_registry    = var.docker_registry
  force_update       = var.force_update
}
```

### Module inputs

| Variable | Required | Type | Description |
|---|---|---|---|
| `stack_name` | yes | string | Name of the Portainer stack |
| `endpoint_id` | yes | number | Portainer endpoint ID |
| `stack_file_content` | yes | string | Docker Compose file content (use `file()`) |
| `docker_registry` | yes | string | Docker registry address |
| `force_update` | no | string | Set to a new value (e.g. timestamp) to force stack recreation |

### Module outputs

| Output | Description |
|---|---|
| `stack_id` | ID of the deployed Portainer stack |
| `stack_name` | Name of the deployed stack |

### Minimal project terraform layout

```
my-service/
└── terraform/
    ├── provider.tf
    ├── variables.tf
    ├── main.tf
    └── output.tf
```

**provider.tf**
```hcl
terraform {
  required_version = ">= 1.0"
  required_providers {
    portainer = {
      source  = "portainer/portainer"
      version = "~> 1.0"
    }
  }
}

provider "portainer" {
  endpoint = var.portainer_url
  api_key  = var.portainer_api_key
}
```

**variables.tf**
```hcl
variable "portainer_url"     { type = string }
variable "portainer_api_key" { type = string; sensitive = true }
variable "docker_registry"   { type = string }
variable "stack_name"        { type = string; default = "my-service" }
variable "endpoint_id"       { type = number; default = 1 }
variable "force_update"      { type = string; default = "" }
```

**main.tf**
```hcl
module "portainer_stack" {
  source = "../.build/build-tools/terraform/portainer-stack"

  stack_name         = var.stack_name
  endpoint_id        = var.endpoint_id
  stack_file_content = file("${path.module}/../docker-compose.yml")
  docker_registry    = var.docker_registry
  force_update       = var.force_update
}
```

**output.tf**
```hcl
output "stack_id"   { value = module.portainer_stack.stack_id }
output "stack_name" { value = module.portainer_stack.stack_name }
```

---

## End-to-end workflow

```bash
# 1. First run — auto-clones build-tools, then builds and pushes images
make build

# 2. Deploy (or re-deploy) the stack to Portainer
make deploy
```

`make deploy` will:
1. Initialize Terraform if needed
2. Show a plan and prompt for confirmation
3. Apply with `-auto-approve` after confirmation
4. Print Terraform outputs

---

## Reference project

[resawod-scheduler](https://github.com/anydef/resawod-scheduler) is a full working example of a project using this repo.