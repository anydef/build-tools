output "server_uuid" {
  description = "UUID of the created HAProxy server"
  value       = data.external.haproxy_server.result.uuid
}

output "backend_uuid" {
  description = "UUID of the created HAProxy backend"
  value       = data.external.haproxy_backend.result.uuid
}

output "backend_name" {
  description = "Name of the created HAProxy backend"
  value       = local.backend_name
}

output "dns_override_uuid" {
  description = "UUID of the created Unbound host override"
  value       = restapi_object.dns_host_override.id
}

output "acl_uuid" {
  description = "UUID of the per-service ACL (only when use_direct_rules=true)"
  value       = var.use_direct_rules ? data.external.haproxy_acl[0].result.uuid : ""
}

output "action_uuid" {
  description = "UUID of the per-service action (only when use_direct_rules=true)"
  value       = var.use_direct_rules ? data.external.haproxy_action[0].result.uuid : ""
}

output "local_mapfile_uuid" {
  description = "UUID of the resolved LOCAL HAProxy mapfile (only when use_direct_rules=false)"
  value       = var.use_direct_rules ? "" : local.local_mapfile_uuid
}

output "public_mapfile_uuid" {
  description = "UUID of the resolved PUBLIC HAProxy mapfile (only when use_direct_rules=false)"
  value       = var.use_direct_rules ? "" : local.public_mapfile_uuid
}

output "service_url" {
  description = "The full URL of the service"
  value       = "https://${local.subdomain}.${var.domain}"
}
