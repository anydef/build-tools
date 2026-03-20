variable "service_name" {
  description = "Service name, used for naming HAProxy server/backend (e.g., 'resawod')"
  type        = string
}

variable "address" {
  description = "Service IP address (e.g., '192.168.100.10')"
  type        = string
}

variable "port" {
  description = "Service port"
  type        = number
}

variable "domain" {
  description = "Domain for DNS host override (e.g., 'lab.anydef.de')"
  type        = string
}

variable "dns_server" {
  description = "IP that DNS should resolve to (OPNsense LAN IP where HAProxy listens)"
  type        = string
  default     = "192.168.1.1"
}

variable "subdomain" {
  description = "Subdomain key for mapfile entry and DNS hostname. Defaults to service_name."
  type        = string
  default     = ""
}

variable "public" {
  description = "Whether this service should be publicly accessible. If true, adds entry to both LOCAL and PUBLIC mapfiles. If false, only LOCAL."
  type        = bool
  default     = false
}

variable "ssl" {
  description = "Whether backend server uses SSL"
  type        = string
  default     = "0"
}

variable "http2_enabled" {
  description = "Enable HTTP/2 on the backend ('0' = disabled, '1' = enabled). Disabled by default — most backends (e.g. nginx on plain HTTP) don't support h2c, causing sporadic 503s."
  type        = string
  default     = "0"
}

variable "health_check_enabled" {
  description = "Enable health checking on the backend ('0' = disabled, '1' = enabled). Disabled by default to avoid false 503s on macvlan services."
  type        = string
  default     = "0"
}

variable "health_check" {
  description = "Health check type (e.g., '', 'HTTP', 'TCP'). Empty string means none."
  type        = string
  default     = ""
}

variable "use_direct_rules" {
  description = "Use per-service ACL+action rules instead of shared mapfiles. Avoids read-modify-write race conditions on shared mapfiles."
  type        = bool
  default     = true
}

variable "local_subnet_acl_uuid" {
  description = "UUID of the LOCAL_SUBDOMAINS_SUBNETS_condition ACL (src 192.168.0.0/20). Required when use_direct_rules=true."
  type        = string
  default     = ""
}

variable "https_frontend_uuid" {
  description = "UUID of the 1_HTTPS_frontend to link actions to. Required when use_direct_rules=true."
  type        = string
  default     = ""
}

variable "opnsense_url" {
  description = "OPNsense base URL (e.g., 'https://192.168.1.1')"
  type        = string
}

variable "opnsense_api_key" {
  description = "OPNsense API key"
  type        = string
  sensitive   = true
}

variable "opnsense_api_secret" {
  description = "OPNsense API secret"
  type        = string
  sensitive   = true
}
