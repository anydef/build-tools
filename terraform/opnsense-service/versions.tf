terraform {
  required_version = ">= 1.0"

  required_providers {
    restapi = {
      source  = "Mastercard/restapi"
      version = "~> 3.0"
    }
    external = {
      source  = "hashicorp/external"
      version = "~> 2.0"
    }
  }
}
