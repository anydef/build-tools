# Terraform Provider for OPNsense

A native Terraform provider for managing OPNsense HAProxy and Unbound resources via the OPNsense API. Built with the Terraform Plugin Framework.

## Motivation

The previous approach used `data.external` with shell scripts to manage OPNsense resources. This had several problems:

- **Slow**: Each module fetched the full HAProxy settings (~700KB JSON) via the OPNsense API, taking ~4s per call. With 6 modules, the PHP API serialized concurrent requests behind a session lock, causing ~135s timeouts.
- **No diff detection**: Resources were always updated on every run, even when nothing changed.
- **`depends_on` ignored**: Terraform evaluates `data.external` during plan, ignoring `depends_on` ordering.
- **Array unmarshal bug**: The `restapi` provider (Mastercard/restapi) couldn't handle OPNsense's polymorphic API responses where fields could be arrays or maps.
- **Stale processes**: HAProxy reloads left zombie processes with outdated configs, causing intermittent 503s.

This provider solves all of these by implementing proper CRUD operations with Terraform state management.

## Status

**Scaffolded and compiles.** Not yet tested against a live OPNsense instance.

## Resources

| Resource | Description |
|---|---|
| `opnsense_haproxy_server` | HAProxy real server (address + port) |
| `opnsense_haproxy_backend` | HAProxy backend pool (links to servers) |
| `opnsense_haproxy_acl` | HAProxy ACL / condition (host match, src match, etc.) |
| `opnsense_haproxy_action` | HAProxy action / rule (use_backend with ACL conditions) |
| `opnsense_haproxy_frontend_action` | Link an action to a frontend (with prepend support for rule ordering) |
| `opnsense_haproxy_reconfigure` | Trigger HAProxy reconfigure/restart |
| `opnsense_unbound_host_override` | Unbound DNS host override (auto-reconfigures Unbound) |

## Usage

```hcl
terraform {
  required_providers {
    opnsense = {
      source = "registry.terraform.io/anydef/opnsense"
    }
  }
}

provider "opnsense" {
  url        = "https://192.168.1.1"
  api_key    = var.opnsense_api_key
  api_secret = var.opnsense_api_secret
  insecure   = true
}

resource "opnsense_haproxy_server" "grafana" {
  name        = "GRAFANA_server"
  description = "[terraform] Server for grafana at 192.168.100.14:3000"
  address     = "192.168.100.14"
  port        = "3000"
}

resource "opnsense_haproxy_backend" "grafana" {
  name           = "GRAFANA_backend"
  description    = "[terraform] Backend pool for grafana (grafana.lab.anydef.de)"
  linked_servers = opnsense_haproxy_server.grafana.id
}

resource "opnsense_haproxy_acl" "grafana" {
  name       = "GRAFANA_host_acl"
  expression = "hdr"
  value      = "grafana.lab.anydef.de"
}

resource "opnsense_haproxy_action" "grafana" {
  name        = "GRAFANA_rule"
  type        = "use_backend"
  test_type   = "if"
  linked_acls = opnsense_haproxy_acl.grafana.id
  use_backend = opnsense_haproxy_backend.grafana.id
}

resource "opnsense_haproxy_frontend_action" "grafana" {
  frontend_id = "15949679-3870-49d9-b796-eabe9a7265d4"
  action_id   = opnsense_haproxy_action.grafana.id
  prepend     = true
}

resource "opnsense_unbound_host_override" "grafana" {
  hostname = "grafana"
  domain   = "lab.anydef.de"
  server   = "192.168.1.1"
}

resource "opnsense_haproxy_reconfigure" "apply" {
  depends_on = [
    opnsense_haproxy_frontend_action.grafana,
  ]
}
```

See `examples/main.tf` for a complete service setup.

## Development

```bash
# Build
go build -o terraform-provider-opnsense

# Install locally for testing
go build -o ~/.terraform.d/plugins/registry.terraform.io/anydef/opnsense/0.1.0/linux_amd64/terraform-provider-opnsense

# Or use dev_overrides in ~/.terraformrc
cat >> ~/.terraformrc << EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/anydef/opnsense" = "/home/anydef/repo/anydef/build-tools/terraform/terraform-provider-opnsense"
  }
  direct {}
}
EOF
```

## Architecture

```
internal/
├── provider/
│   └── provider.go          # Provider config (url, api_key, api_secret, insecure)
└── resources/
    ├── client.go             # OPNsense HTTP client + response parsing helpers
    ├── haproxy_server.go     # CRUD for HAProxy servers
    ├── haproxy_backend.go    # CRUD for HAProxy backends
    ├── haproxy_acl.go        # CRUD for HAProxy ACLs
    ├── haproxy_action.go     # CRUD for HAProxy actions
    ├── haproxy_frontend_action.go  # Join resource: link action to frontend
    ├── haproxy_reconfigure.go      # Trigger HAProxy restart
    └── unbound_host_override.go    # CRUD for Unbound DNS overrides
```

### API Response Handling

OPNsense's API returns polymorphic responses — the same field can be a string on write but a `{uuid: {value, selected}}` map on read. The `client.go` helpers handle this:

- `extractStringField(data, key)` — extracts a string regardless of whether the value is a plain string or a selection map
- `extractSelectedUUIDs(data, key)` — extracts selected UUIDs from a `{uuid: {value, selected: 1}}` map

## TODO

### Before First Use
- [ ] Test all resources against a live OPNsense instance
- [ ] Verify Read handles all OPNsense API response formats (the array fields that broke `restapi`)
- [ ] Test import for existing resources (`terraform import`)
- [ ] Test destroy for all resources (correct deletion order)
- [ ] Verify `frontend_action` prepend ordering works correctly

### Provider Improvements
- [ ] Add data sources for looking up existing resources by name (e.g., `data.opnsense_haproxy_frontend` to find frontend UUID by name instead of hardcoding)
- [ ] Add `opnsense_haproxy_mapfile` resource (for legacy mapfile approach)
- [ ] Add `opnsense_haproxy_frontend` data source to look up frontend UUID by name
- [ ] Add `opnsense_haproxy_acl` data source to look up shared ACLs (e.g., `LOCAL_SUBDOMAINS_SUBNETS_condition`)
- [ ] Support multiple linked_servers and linked_acls (currently comma-separated strings)
- [ ] Add validation for enum fields (expression types, action types, etc.)

### Build & Release
- [ ] Set up GoReleaser for building cross-platform binaries
- [ ] Publish to a private Terraform registry (or use GitHub releases + `source` URL)
- [ ] Add CI pipeline for running `go test` and `go vet`
- [ ] Write acceptance tests using a test OPNsense instance

### Migration from Shell Scripts
- [ ] Import existing OPNsense resources into Terraform state managed by this provider
- [ ] Update the `opnsense-service` module to use this provider instead of `data.external` + `terraform_data`
- [ ] Remove `restapi` provider dependency from consumers
- [ ] Update `add-service-domain` skill in homelab repo

### Stretch Goals
- [ ] Support more OPNsense subsystems (firewall rules, VPN, etc.)
- [ ] Add Unbound domain override resource
- [ ] Publish to the public Terraform Registry
