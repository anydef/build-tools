output "database_url" {
  description = "PostgreSQL connection URL"
  value       = local.database_url
  sensitive   = true
}

output "onepassword_item_uuid" {
  description = "UUID of the 1Password item holding Grafana DB credentials"
  value       = onepassword_item.postgres.uuid
}
