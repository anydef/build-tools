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
    "name": "MYAPP_server",
    "description": "",
    "address": "192.168.100.10",
    "port": "8080",
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
    "name": "MYAPP_backend",
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
    "name": "EXAMPLE_LOCAL_SUBDOMAINS_mapfile",
    "description": "Subdomains to backend mapping",
    "content": "myapp MYAPP_backend\nanother ANOTHER_backend"
  }
}
```

### ACL Schema
```json
{
  "acl": {
    "name": "example_domain_condition",
    "description": "Match hosts ending with .example.domain.com",
    "expression": "hdr_end",
    "negate": "0",
    "caseSensitive": "0",
    "hdr_beg": "",
    "hdr_end": ".example.domain.com",
    "hdr": "",
    "hdr_reg": "",
    "hdr_sub": "",
    "path_beg": "",
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
    "name": "HTTPS_frontend",
    "description": "",
    "bind": "127.0.0.1:443",
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
    "hostname": "myapp",
    "domain": "example.com",
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

## Important API Behaviors

- All `Add` methods return `{"uuid": "...", "result": "saved"}` on success
- All `Add` methods return `{"result": "failed", "validations": {...}}` on validation error
- Mapfile `content` must not be empty (use a comment like `# managed by terraform` as placeholder)
- All `Set` methods require the UUID in the URL path
- Select fields accept comma-separated UUIDs for multi-select
- `serviceReconfigure` must be called after changes to apply
- Mapfile `content` is a newline-separated string of `key value` pairs
- Backend `linkedServers` references server UUIDs
- Frontend `linkedActions` references action UUIDs (order matters for rule evaluation)
- OPNsense uses POST for all write operations (add, set, del) — not standard REST PUT/DELETE
- GET endpoints reject request bodies — do not send Content-Type or data on reads

### HAProxy map_dom behavior

The `map_use_backend` action generates:
```
use_backend %[req.hdr(host),lower,map_dom(/path/to/mapfile)]
```

`map_dom` strips domain labels from the **right** of the input hostname and tries progressively shorter names until a match is found. For `myapp.sub.example.com`:
1. `myapp.sub.example.com` (exact match)
2. `myapp.sub.example`
3. `myapp.sub`
4. `myapp` (bare subdomain match)

This means a bare key like `myapp` in the mapfile will match `myapp.anything.example.com`. To avoid conflicts between domains (e.g., `*.example.com` vs `*.sub.example.com`), use separate mapfiles with domain-specific ACLs (`hdr_end`).

### Terraform provider notes

- Use `Mastercard/restapi` provider v3+ with `create_returns_object = true`
- Set `create_method`, `update_method`, `destroy_method` to `POST`
- Set `id_attribute = "uuid"`
- Use `ignore_server_additions = true` on resources to prevent drift from server-added fields