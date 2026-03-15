# OPNsense API Reference for Terraform Module

## Base URL
All endpoints follow: `https://<opnsense-host>/api/<module>/<controller>/<action>`

## Authentication
API key + secret via HTTP Basic Auth.

---

## HAProxy Plugin API

Base path: `/api/haproxy/settings/`

### Resources & CRUD Methods

| Resource   | Add                  | Get                  | Set                  | Del                  | Toggle                  |
|------------|----------------------|----------------------|----------------------|----------------------|-------------------------|
| Server     | settingsAddServer    | settingsGetServer    | settingsSetServer    | settingsDelServer    | settingsToggleServer    |
| Backend    | settingsAddBackend   | settingsGetBackend   | settingsSetBackend   | settingsDelBackend   | settingsToggleBackend   |
| Frontend   | settingsAddFrontend  | settingsGetFrontend  | settingsSetFrontend  | settingsDelFrontend  | settingsToggleFrontend  |
| ACL        | settingsAddAcl       | settingsGetAcl       | settingsSetAcl       | settingsDelAcl       | -                       |
| Action     | settingsAddAction    | settingsGetAction    | settingsSetAction    | settingsDelAction    | -                       |
| Mapfile    | settingsAddMapfile   | settingsGetMapfile   | settingsSetMapfile   | settingsDelMapfile   | -                       |
| Healthcheck| settingsAddHealthcheck| settingsGetHealthcheck| settingsSetHealthcheck| settingsDelHealthcheck| -                    |

Apply changes: `serviceReconfigure` (POST)
Test config: `serviceConfigtest` (GET)

### Server Schema
```json
{
  "server": {
    "enabled": "1",
    "name": "SONARR_server",
    "description": "",
    "address": "192.168.100.10",
    "port": "8989",
    "checkport": "",
    "mode": "active",
    "multiplexer_protocol": "unspecified",
    "type": "static",
    "serviceName": "",
    "number": "",
    "linkedResolver": "",
    "resolverOpts": "",
    "resolvePrefer": "",
    "ssl": "0",
    "sslSNI": "",
    "sslVerify": "0",
    "sslCA": [],
    "sslCRL": "",
    "sslClientCertificate": "",
    "maxConnections": "",
    "weight": "",
    "checkInterval": "",
    "checkDownInterval": "",
    "source": "",
    "advanced": "",
    "unix_socket": ""
  }
}
```

### Backend Schema
```json
{
  "backend": {
    "enabled": "1",
    "name": "SONARR_backend",
    "description": "",
    "mode": "http",
    "algorithm": "source",
    "random_draws": "2",
    "proxyProtocol": "",
    "linkedServers": "<server-uuid>",
    "linkedFcgi": "",
    "linkedResolver": "",
    "resolverOpts": "",
    "resolvePrefer": "",
    "source": "",
    "healthCheckEnabled": "1",
    "healthCheck": "",
    "healthCheckLogStatus": "0",
    "checkInterval": "",
    "checkDownInterval": "",
    "healthCheckFall": "",
    "healthCheckRise": "",
    "linkedMailer": "",
    "http2Enabled": "1",
    "http2Enabled_nontls": "0",
    "ba_advertised_protocols": "h2,http11",
    "forwardFor": "0",
    "forwardedHeader": "0",
    "persistence": "sticktable",
    "persistence_cookiemode": "piggyback",
    "persistence_cookiename": "SRVCOOKIE",
    "persistence_stripquotes": "1",
    "stickiness_pattern": "sourceipv4",
    "stickiness_expire": "30m",
    "stickiness_size": "50k",
    "basicAuthEnabled": "0",
    "tuning_timeoutConnect": "",
    "tuning_timeoutCheck": "",
    "tuning_timeoutServer": "",
    "tuning_retries": "",
    "customOptions": "",
    "tuning_defaultserver": "",
    "tuning_noport": "0",
    "tuning_httpreuse": "safe",
    "tuning_caching": "0",
    "linkedActions": "",
    "linkedErrorfiles": []
  }
}
```

### Mapfile Schema
```json
{
  "mapfile": {
    "name": "PUBLIC_SUBDOMAINS_mapfile",
    "description": "Public subdomains to backend mapping",
    "content": "home HOME_backend\nsonarr SONARR_backend\nradarr RADARR_backend"
  }
}
```

### ACL Schema
```json
{
  "acl": {
    "name": "find_acme_challenge",
    "description": "",
    "expression": "path_beg",
    "negate": "0",
    "caseSensitive": "0",
    "hdr_beg": "",
    "hdr_end": "",
    "hdr": "",
    "hdr_reg": "",
    "hdr_sub": "",
    "path_beg": "/.well-known/acme-challenge/",
    "path_end": "",
    "path": "",
    "path_reg": "",
    "path_dir": "",
    "path_sub": "",
    "url_param": "",
    "url_param_value": "",
    "ssl_c_verify_code": "",
    "ssl_c_ca_commonname": "",
    "ssl_hello_type": "x1",
    "src": "",
    "src_bytes_in_rate_comparison": "gt",
    "src_bytes_in_rate": "",
    "src_bytes_out_rate_comparison": "gt",
    "src_bytes_out_rate": ""
  }
}
```

### Frontend Schema (key fields)
```json
{
  "frontend": {
    "enabled": "1",
    "name": "1_HTTPS_frontend",
    "description": "",
    "bind": "127.4.4.3:443",
    "bindOptions": "",
    "mode": "http",
    "defaultBackend": "<backend-uuid>",
    "ssl_enabled": "1",
    "ssl_certificates": "<cert-uuid>",
    "ssl_default_certificate": "<cert-uuid>",
    "ssl_hstsEnabled": "1",
    "ssl_hstsMaxAge": "15768000",
    "ssl_minVersion": "TLSv1.2",
    "http2Enabled": "1",
    "http2Enabled_nontls": "0",
    "advertised_protocols": "h2,http11",
    "forwardFor": "0",
    "connectionBehaviour": "http-keep-alive",
    "linkedActions": "<action-uuid1>,<action-uuid2>",
    "linkedErrorfiles": []
  }
}
```

---

## Unbound DNS API

Base path: `/api/unbound/settings/`

### Host Override CRUD
| Action | Method |
|--------|--------|
| Add    | settingsAddHostOverride |
| Get    | settingsGetHostOverride |
| Set    | settingsSetHostOverride |
| Delete | settingsDelHostOverride |
| Toggle | settingsToggleHostOverride |

Apply changes: `serviceReconfigure` (POST)

### Host Override Schema
```json
{
  "host": {
    "enabled": "1",
    "hostname": "sonarr",
    "domain": "anydef.de",
    "rr": "A",
    "mxprio": "",
    "mx": "",
    "ttl": "",
    "server": "192.168.1.1",
    "txtdata": "",
    "description": ""
  }
}
```

### Host Alias CRUD
| Action | Method |
|--------|--------|
| Add    | settingsAddHostAlias |
| Get    | settingsGetHostAlias |
| Set    | settingsSetHostAlias |
| Delete | settingsDelHostAlias |
| Toggle | settingsToggleHostAlias |

---

## Current Configuration State

### HAProxy Servers (existing)
| Name            | Address          | Port  | UUID |
|-----------------|------------------|-------|------|
| acme_challenge  | 127.0.0.1        | 43580 | 26c6ec89-722e-4a65-90c1-b3f63d0441a4 |
| HOME_server     | 192.168.2.11     | 80    | e87dd8e4-2b3d-443b-ab4e-4c197b42895a |
| FRITZ_server    | 192.168.1.104    | 443   | 1f2edc03-9cc2-454c-aa90-4a752ba14e95 |
| GRAFANA_server  | 192.168.4.14     | 80    | 77625bd9-deb7-4e64-bdf5-8e516e183822 |
| OVERSEER_server | 192.168.1.234    | 5055  | 66920882-2221-4947-b656-c23cef22c07b |
| SONARR_server   | 192.168.1.234    | 8989  | 5b7effb9-074a-489d-9e7d-bc3afb707dbc |
| TOWER_server    | 192.168.1.234    | 443   | 082a5e5f-748a-4fd8-a8ba-2495067d52bc |
| TANDOOR_server  | 192.168.1.234    | 8154  | 6ec134ec-eaa8-431d-9b40-21b6af2165ba |
| RADARR_server   | 192.168.1.234    | 7878  | 8b67664a-ebff-4805-b12a-2f069dbfb627 |
| KELLNR_server   | 192.168.1.234    | 8000  | c1fcfa41-1a1d-4a52-a3f0-a2b756ed14ea |
| IMMICH_server   | 192.168.1.234    | 2283  | f4ec36e4-0ef8-4ccb-bfde-d0c4731b7b12 |

**Note:** Sonarr, Radarr, Overseer, Tandoor, Kellnr, Immich all point to `192.168.1.234` (Unraid/tower).
After migration to VLAN 25, these will change to `192.168.100.x` addresses.

### HAProxy Mapfiles (existing)
**PUBLIC_SUBDOMAINS_mapfile** (4874a974-d326-4725-888c-ff3811d19943):
```
home       HOME_backend
lab        HOME_backend
overseer   OVERSEER_backend
tandoor    TANDOOR_backend
kellnr     KELLNR_backend
immich     IMMICH_backend
```

**LOCAL_SUBDOMAINS_mapfile** (c0689e42-0a41-4f37-8e2c-c1d4f3c105ad):
```
home       HOME_backend
lab        HOME_backend
overseer   OVERSEER_backend
tandoor    TANDOOR_backend
fritz      FRITZ_backend
grafana    GRAFANA_backend
sonarr     SONARR_backend
radarr     RADARR_backend
tower      TOWER_backend
immich     IMMICH_backend
```

### HAProxy Frontends (existing)
| Name               | Bind           | Enabled | UUID |
|--------------------|----------------|---------|------|
| https (old)        | 0.0.0.0:443    | No      | a47abbd3-... |
| 0_SNI_frontend     | 0.0.0.0:80,443 | Yes     | 3062dd18-... |
| 1_HTTP_frontend    | 127.4.4.3:80   | Yes     | 35912271-... |
| 1_HTTPS_frontend   | 127.4.4.3:443  | Yes     | 15949679-... |

### Unbound Host Overrides (existing)
All enabled overrides point to `192.168.1.1` (OPNsense LAN IP / HAProxy):
| Hostname | Domain    | Server       | Enabled |
|----------|-----------|--------------|---------|
| home     | anydef.de | 192.168.1.1  | Yes     |
| fritz    | anydef.de | 192.168.1.1  | Yes     |
| grafana  | anydef.de | 192.168.1.1  | Yes     |
| sonarr   | anydef.de | 192.168.1.1  | Yes     |
| tower    | anydef.de | 192.168.1.1  | Yes     |
| tandoor  | anydef.de | 192.168.1.1  | Yes     |
| radarr   | anydef.de | 192.168.1.1  | Yes     |
| kellnr   | anydef.de | 192.168.1.1  | Yes     |
| immich   | anydef.de | 192.168.1.1  | Yes     |

Domain: **anydef.de**
All DNS overrides resolve `*.anydef.de` → `192.168.1.1` (OPNsense) → HAProxy routes by subdomain via mapfile.

---

## Network Context

### VLAN Layout
| VLAN | Name          | Subnet             | Gateway        | Interface |
|------|---------------|--------------------| ---------------|-----------|
| -    | LAN           | 192.168.0.0/20     | 192.168.1.1    | igc1      |
| 10   | ManagementLAN | 192.168.10.0/24    | 192.168.10.1   | igc1      |
| 20   | GeneralLAN    | 192.168.20.0/24    | 192.168.20.1   | igc1      |
| 25   | ServicesLAN   | 192.168.100.0/24   | 192.168.100.1  | igc1      |
| 30   | DMZ           | 192.168.30.0/24    | 192.168.30.1   | igc1      |
| 40   | IoT           | 192.168.40.0/24    | 192.168.40.1   | igc1      |
| 50   | Guest         | 192.168.50.0/24    | 192.168.50.1   | igc1      |

### Docker Network (Unraid)
- Type: macvlan, external
- Parent: eth0.25 (VLAN 25)
- Subnet: 192.168.100.0/24
- Gateway: 192.168.100.1
- Auto-assign range: 192.168.100.128/25
- Pinned IPs: 192.168.100.2-127 (managed in Docker Compose)

---

## Terraform Module Design Notes

### Workflow per service:
1. Create HAProxy **Server** (name, IP, port)
2. Create HAProxy **Backend** (link to server)
3. Update **Mapfile** content (add subdomain → backend mapping)
4. Create Unbound **Host Override** (subdomain.anydef.de → 192.168.1.1)
5. `serviceReconfigure` on both HAProxy and Unbound

### Important API behaviors:
- All `Add` methods return `{"uuid": "..."}` on success
- All `Set` methods require the UUID in the URL path
- Select fields accept comma-separated UUIDs for multi-select
- `serviceReconfigure` must be called after changes to apply
- Mapfile `content` is a newline-separated string of `key value` pairs
- Backend `linkedServers` references server UUIDs
- Frontend `linkedActions` references action UUIDs
- The OPNsense API uses the same REST pattern for all resources

### Existing Terraform provider:
- `browningluke/opnsense` — community provider, covers core resources
- May not cover HAProxy plugin — verify before building custom provider
- Alternative: use `restapi` generic provider with the schemas above
