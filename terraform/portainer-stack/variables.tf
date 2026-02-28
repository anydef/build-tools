variable "stack_name" {
  description = "Name of the Portainer stack"
  type        = string
}

variable "endpoint_id" {
  description = "Portainer endpoint ID"
  type        = number
}

variable "stack_file_content" {
  description = "Docker Compose file content (pass via file() from the calling module)"
  type        = string
}

variable "docker_registry" {
  description = "Docker registry address"
  type        = string
}

variable "force_update" {
  description = "Set to a new value (e.g. timestamp) to force stack recreation"
  type        = string
  default     = ""
}
