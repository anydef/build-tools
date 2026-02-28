output "stack_id" {
  description = "ID of the deployed Portainer stack"
  value       = portainer_stack.this.id
}

output "stack_name" {
  description = "Name of the deployed stack"
  value       = portainer_stack.this.name
}
