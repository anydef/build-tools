output "server_uuid" {
  description = "UUID of the created HAProxy server"
  value       = restapi_object.haproxy_server.id
}

output "backend_uuid" {
  description = "UUID of the created HAProxy backend"
  value       = restapi_object.haproxy_backend.id
}

output "backend_name" {
  description = "Name of the created HAProxy backend"
  value       = local.backend_name
}

output "dns_override_uuid" {
  description = "UUID of the created Unbound host override"
  value       = restapi_object.dns_host_override.id
}

output "local_mapfile_uuid" {
  description = "UUID of the resolved LOCAL HAProxy mapfile"
  value       = local.local_mapfile_uuid
}

output "public_mapfile_uuid" {
  description = "UUID of the resolved PUBLIC HAProxy mapfile"
  value       = local.public_mapfile_uuid
}

output "service_url" {
  description = "The full URL of the service"
  value       = "https://${local.subdomain}.${var.domain}"
}
