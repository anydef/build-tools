resource "portainer_stack" "this" {
  name               = var.stack_name
  endpoint_id        = var.endpoint_id
  deployment_type    = "standalone"
  method             = "string"
  stack_file_content = var.stack_file_content

  env {
    name  = "DOCKER_REGISTRY"
    value = var.docker_registry
  }

  env {
    name  = "FORCE_UPDATE"
    value = var.force_update != "" ? var.force_update : "none"
  }
}
