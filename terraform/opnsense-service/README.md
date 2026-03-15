# opnsense-service

Reusable Terraform module that registers a service in OPNsense HAProxy and Unbound DNS.

## What it does

For each service, the module:

1. **HAProxy Server** — registers the backend server (IP + port)
2. **HAProxy Backend** — creates a backend linked to the server
3. **Mapfile entry** — adds a `subdomain BACKEND_name` entry to the domain's LOCAL mapfile (and PUBLIC if `public = true`)
4. **Unbound DNS override** — creates a host override so `subdomain.domain` resolves to the HAProxy IP
5. **Reconfigures** both HAProxy and Unbound to apply changes

## Prerequisites

- OPNsense with HAProxy and Unbound plugins
- Domain infrastructure already set up (mapfiles, ACLs, actions, frontend linking). See [`lab_domain_proxy/`](../../../lab_domain_proxy/) for the `lab.anydef.de` setup.
- `restapi` provider configured in the calling module (see [Provider Configuration](#provider-configuration))
- `curl` and `jq` available on the machine running Terraform

## Usage

```hcl
module "opnsense_service" {
  source = "./modules/opnsense-service"

  service_name        = "resawod"
  address             = "192.168.100.10"
  port                = 3009
  domain              = "lab.anydef.de"
  opnsense_url        = var.opnsense_url
  opnsense_api_key    = var.opnsense_api_key
  opnsense_api_secret = var.opnsense_api_secret
}
```

### Public service (accessible from the internet)

```hcl
module "opnsense_service" {
  source = "./modules/opnsense-service"

  service_name        = "myapp"
  address             = "192.168.100.20"
  port                = 8080
  domain              = "lab.anydef.de"
  public              = true
  opnsense_url        = var.opnsense_url
  opnsense_api_key    = var.opnsense_api_key
  opnsense_api_secret = var.opnsense_api_secret
}
```

### Custom subdomain

```hcl
module "opnsense_service" {
  source = "./modules/opnsense-service"

  service_name        = "my-long-service-name"
  subdomain           = "myapp"              # maps to myapp.lab.anydef.de
  address             = "192.168.100.30"
  port                = 3000
  domain              = "lab.anydef.de"
  opnsense_url        = var.opnsense_url
  opnsense_api_key    = var.opnsense_api_key
  opnsense_api_secret = var.opnsense_api_secret
}
```

## Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| `service_name` | Service name, used for HAProxy server/backend naming | `string` | - | yes |
| `address` | Service IP address | `string` | - | yes |
| `port` | Service port | `number` | - | yes |
| `domain` | Domain for DNS and mapfile lookup (e.g., `lab.anydef.de`) | `string` | - | yes |
| `opnsense_url` | OPNsense base URL | `string` | - | yes |
| `opnsense_api_key` | OPNsense API key | `string` | - | yes |
| `opnsense_api_secret` | OPNsense API secret | `string` | - | yes |
| `subdomain` | Subdomain key. Defaults to `service_name` | `string` | `""` | no |
| `public` | If true, adds entry to both LOCAL and PUBLIC mapfiles | `bool` | `false` | no |
| `dns_server` | IP that DNS resolves to (HAProxy listener) | `string` | `192.168.1.1` | no |
| `ssl` | Whether backend server uses SSL | `string` | `"0"` | no |

## Outputs

| Name | Description |
|------|-------------|
| `server_uuid` | UUID of the HAProxy server |
| `backend_uuid` | UUID of the HAProxy backend |
| `backend_name` | Name of the HAProxy backend (e.g., `RESAWOD_backend`) |
| `dns_override_uuid` | UUID of the Unbound host override |
| `local_mapfile_uuid` | UUID of the LOCAL mapfile |
| `public_mapfile_uuid` | UUID of the PUBLIC mapfile |
| `service_url` | Full URL of the service (e.g., `https://resawod.lab.anydef.de`) |

## Provider Configuration

The calling module must configure the `restapi` provider:

```hcl
terraform {
  required_providers {
    restapi = {
      source  = "Mastercard/restapi"
      version = "~> 3.0"
    }
  }
}

provider "restapi" {
  uri      = var.opnsense_url
  username = var.opnsense_api_key
  password = var.opnsense_api_secret
  insecure = true

  create_method         = "POST"
  update_method         = "POST"
  destroy_method        = "POST"
  read_method           = "GET"
  id_attribute          = "uuid"
  write_returns_object  = false
  create_returns_object = true
}
```

## How it works

### Mapfile naming convention

The module derives mapfile names from the domain:

| Domain | LOCAL mapfile | PUBLIC mapfile |
|--------|-------------|----------------|
| `lab.anydef.de` | `LAB_LOCAL_SUBDOMAINS_mapfile` | `LAB_PUBLIC_SUBDOMAINS_mapfile` |
| `anydef.de` | ANYDEF_LOCAL_SUBDOMAINS_mapfile | ANYDEF_PUBLIC_SUBDOMAINS_mapfile |

### Mapfile key

The key is the bare subdomain (e.g., `resawod`). HAProxy's `map_dom` function strips domain parts from the right when looking up `resawod.lab.anydef.de`, eventually matching the bare key.

A domain-specific ACL (`hdr_end .lab.anydef.de`) on the HAProxy action ensures only requests for `*.lab.anydef.de` hit the LAB mapfiles, preventing conflicts with other domains.

### Mapfile read-modify-write

The mapfile is a shared resource. The module reads the current content via `curl`, removes any existing entry for its subdomain, appends the new entry, and writes it back. On `terraform destroy`, the entry is removed.

### Reconfigure

After all changes, the module calls `serviceReconfigure` on both HAProxy and Unbound to apply the configuration.
