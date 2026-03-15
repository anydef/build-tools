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
