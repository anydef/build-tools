variable "pg_host" {
  description = "PostgreSQL host"
  type        = string
}

variable "pg_port" {
  description = "PostgreSQL port"
  type        = number
  default     = 5432
}

variable "db_name" {
  description = "Name of the db"
  type = string
}
variable "op_vault_name" {
  description = "1Password vault name where the DB credentials will be stored"
  type        = string
  default     = "HomeLab"
}
