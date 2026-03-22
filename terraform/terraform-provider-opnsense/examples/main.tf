terraform {
  required_providers {
    opnsense = {
      source = "registry.terraform.io/anydef/opnsense"
    }
  }
}

provider "opnsense" {
  url        = var.opnsense_url
  api_key    = var.opnsense_api_key
  api_secret = var.opnsense_api_secret
  insecure   = true
}

variable "opnsense_url" {
  type    = string
  default = "https://192.168.1.1"
}

variable "opnsense_api_key" {
  type      = string
  sensitive = true
}

variable "opnsense_api_secret" {
  type      = string
  sensitive = true
}

# ---------------------------------------------------------------------------
# Example: Full service setup for "myapp" running at 192.168.100.10:8080
# This is equivalent to the opnsense-service Terraform module.
# ---------------------------------------------------------------------------

locals {
  service_name = "MYAPP"
  address      = "192.168.100.10"
  port         = "8080"
  domain       = "lab.anydef.de"
  subdomain    = "myapp"
  fqdn         = "${local.subdomain}.${local.domain}"
  dns_server   = "192.168.1.1" # OPNsense LAN IP where HAProxy listens
  frontend_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" # UUID of your HTTPS frontend
}

# 1. Create the real server
resource "opnsense_haproxy_server" "myapp" {
  name        = "${local.service_name}_server"
  description = "[terraform] Server for myapp at ${local.address}:${local.port}"
  address     = local.address
  port        = local.port
  ssl         = "0"
  ssl_verify  = "0"
}

# 2. Create the backend pool, linking to the server
resource "opnsense_haproxy_backend" "myapp" {
  name              = "${local.service_name}_backend"
  description       = "[terraform] Backend pool for myapp (${local.fqdn})"
  mode              = "http"
  algorithm         = "source"
  linked_servers    = opnsense_haproxy_server.myapp.id
  http2_enabled     = "0"
  persistence       = "sticktable"
  stickiness_pattern = "sourceipv4"
  stickiness_expire = "30m"
  stickiness_size   = "50k"
  tuning_httpreuse  = "safe"
}

# 3. Create the ACL (match requests for the FQDN)
resource "opnsense_haproxy_acl" "myapp" {
  name          = "${local.service_name}_host_acl"
  description   = "[terraform] Match requests for ${local.fqdn}"
  expression    = "hdr"
  value         = local.fqdn
  negate        = "0"
  case_sensitive = "0"
}

# 4. Create the action (route matching requests to the backend)
resource "opnsense_haproxy_action" "myapp" {
  name        = "${local.service_name}_rule"
  description = "[terraform] Route ${local.fqdn} to ${local.service_name}_backend"
  type        = "use_backend"
  test_type   = "if"
  linked_acls = opnsense_haproxy_acl.myapp.id
  operator    = "and"
  use_backend = opnsense_haproxy_backend.myapp.id
}

# 5. Link the action to the HTTPS frontend (prepend before catch-all rules)
resource "opnsense_haproxy_frontend_action" "myapp" {
  frontend_id = local.frontend_id
  action_id   = opnsense_haproxy_action.myapp.id
  prepend     = true
}

# 6. Create DNS host override so the FQDN resolves to OPNsense
resource "opnsense_unbound_host_override" "myapp" {
  hostname = local.subdomain
  domain   = local.domain
  server   = local.dns_server
  rr       = "A"
}

# 7. Reconfigure HAProxy to apply all changes
resource "opnsense_haproxy_reconfigure" "apply" {
  depends_on = [
    opnsense_haproxy_server.myapp,
    opnsense_haproxy_backend.myapp,
    opnsense_haproxy_acl.myapp,
    opnsense_haproxy_action.myapp,
    opnsense_haproxy_frontend_action.myapp,
  ]
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "server_uuid" {
  value = opnsense_haproxy_server.myapp.id
}

output "backend_uuid" {
  value = opnsense_haproxy_backend.myapp.id
}

output "acl_uuid" {
  value = opnsense_haproxy_acl.myapp.id
}

output "action_uuid" {
  value = opnsense_haproxy_action.myapp.id
}

output "dns_override_uuid" {
  value = opnsense_unbound_host_override.myapp.id
}

output "service_url" {
  value = "https://${local.fqdn}"
}
