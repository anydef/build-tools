terraform {
  required_providers {
    onepassword = {
      source  = "1Password/onepassword"
      version = "2.1.2"
    }
    postgresql = {
      source  = "cyrilgdn/postgresql"
      version = "1.26.0"
    }
  }
}
data "onepassword_vault" "target" {
  name = var.op_vault_name
}

resource "random_password" "password" {
  length  = 32
  special = false
}

resource "postgresql_role" "role" {
  name     = var.db_name
  login    = true
  password = random_password.password.result
}

resource "postgresql_database" "database" {
  name  = var.db_name
  owner = postgresql_role.role.name

  depends_on = [postgresql_role.role]
}

resource "postgresql_schema" "schema" {
  name     = var.db_name
  database = postgresql_database.database.name
  owner    = postgresql_role.role.name

  depends_on = [postgresql_database.database]
}

locals {
  database_url = "postgres://${postgresql_role.role.name}:${random_password.password.result}@${var.pg_host}:${var.pg_port}/${postgresql_database.database.name}?search_path=${postgresql_schema.schema.name}&sslmode=disable"
}

resource "onepassword_item" "postgres" {
  vault    = data.onepassword_vault.target.uuid
  title    = "${var.db_name}-database"
  category = "database"
  database = postgresql_database.database.name
  hostname = var.pg_host
  port = var.pg_port
  type = "postgresql"
  username = postgresql_role.role.name
  password = random_password.password.result

  section {
    label = "Database"

    field {
      label = "host"
      type  = "STRING"
      value = var.pg_host
    }

    field {
      label = "port"
      type  = "STRING"
      value = tostring(var.pg_port)
    }

    field {
      label = "database"
      type  = "STRING"
      value = postgresql_database.database.name
    }

    field {
      label = "schema"
      type  = "STRING"
      value = postgresql_schema.schema.name
    }

    field {
      label = "url"
      type  = "CONCEALED"
      value = local.database_url
    }
  }

  depends_on = [postgresql_schema.schema]
}
