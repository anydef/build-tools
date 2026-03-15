locals {
  subdomain    = var.subdomain != "" ? var.subdomain : var.service_name
  name_upper   = upper(var.service_name)
  server_name  = "${local.name_upper}_server"
  backend_name = "${local.name_upper}_backend"
  port_str     = tostring(var.port)
  curl_auth    = "${var.opnsense_api_key}:${var.opnsense_api_secret}"

  # Derive mapfile names from domain: lab.anydef.de -> LAB_LOCAL_SUBDOMAINS_mapfile
  domain_prefix       = upper(split(".", var.domain)[0])
  local_mapfile_name  = "${local.domain_prefix}_LOCAL_SUBDOMAINS_mapfile"
  public_mapfile_name = "${local.domain_prefix}_PUBLIC_SUBDOMAINS_mapfile"
  local_mapfile_uuid  = data.external.mapfile_lookup.result.local_uuid
  public_mapfile_uuid = data.external.mapfile_lookup.result.public_uuid

  # Bare subdomain as mapfile key — map_dom strips domain parts from right
  mapfile_key = local.subdomain
}

# -----------------------------------------------------------------------------
# Look up mapfile UUIDs by name
# -----------------------------------------------------------------------------
data "external" "mapfile_lookup" {
  program = ["/bin/bash", "-c", <<-EOT
    set -e
    SETTINGS=$(curl -s -k -u "${local.curl_auth}" \
      "${var.opnsense_url}/api/haproxy/settings/get")

    LOCAL_UUID=$(echo "$SETTINGS" | jq -r --arg name "${local.local_mapfile_name}" '
      .haproxy.mapfiles.mapfile
      | to_entries[]
      | select(.value.name == $name)
      | .key')
    if [ -z "$LOCAL_UUID" ]; then
      echo "mapfile '${local.local_mapfile_name}' not found" >&2
      exit 1
    fi

    PUBLIC_UUID=$(echo "$SETTINGS" | jq -r --arg name "${local.public_mapfile_name}" '
      .haproxy.mapfiles.mapfile
      | to_entries[]
      | select(.value.name == $name)
      | .key')
    if [ -z "$PUBLIC_UUID" ]; then
      echo "mapfile '${local.public_mapfile_name}' not found" >&2
      exit 1
    fi

    jq -n --arg local_uuid "$LOCAL_UUID" --arg public_uuid "$PUBLIC_UUID" \
      '{"local_uuid": $local_uuid, "public_uuid": $public_uuid}'
  EOT
  ]
}

# -----------------------------------------------------------------------------
# HAProxy Server
# -----------------------------------------------------------------------------
resource "restapi_object" "haproxy_server" {
  path         = "/api/haproxy/settings/addServer"
  read_path    = "/api/haproxy/settings/getServer/{id}"
  update_path  = "/api/haproxy/settings/setServer/{id}"
  destroy_path = "/api/haproxy/settings/delServer/{id}"
  id_attribute = "uuid"

  data = jsonencode({
    server = {
      enabled   = "1"
      name      = local.server_name
      address   = var.address
      port      = local.port_str
      ssl       = var.ssl
      sslVerify = "0"
    }
  })

  ignore_server_additions = true
}

# -----------------------------------------------------------------------------
# HAProxy Backend
# -----------------------------------------------------------------------------
resource "restapi_object" "haproxy_backend" {
  path         = "/api/haproxy/settings/addBackend"
  read_path    = "/api/haproxy/settings/getBackend/{id}"
  update_path  = "/api/haproxy/settings/setBackend/{id}"
  destroy_path = "/api/haproxy/settings/delBackend/{id}"
  id_attribute = "uuid"

  data = jsonencode({
    backend = {
      enabled                 = "1"
      name                    = local.backend_name
      mode                    = "http"
      algorithm               = "source"
      linkedServers           = restapi_object.haproxy_server.id
      http2Enabled            = "1"
      ba_advertised_protocols = "h2,http11"
      persistence             = "sticktable"
      stickiness_pattern      = "sourceipv4"
      stickiness_expire       = "30m"
      stickiness_size         = "50k"
      tuning_httpreuse        = "safe"
    }
  })

  ignore_server_additions = true
}

# -----------------------------------------------------------------------------
# HAProxy Mapfile entries (read-modify-write)
# -----------------------------------------------------------------------------

# LOCAL mapfile — always updated
resource "terraform_data" "local_mapfile_entry" {
  input = {
    mapfile_key  = local.mapfile_key
    backend_name = local.backend_name
    mapfile_uuid = local.local_mapfile_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    command     = <<-EOT
      set -e
      MAPFILE=$(curl -s -k -u "${local.curl_auth}" \
        "${var.opnsense_url}/api/haproxy/settings/getMapfile/${local.local_mapfile_uuid}")
      CURRENT=$(echo "$MAPFILE" | jq -r '.mapfile.content // ""')
      NAME=$(echo "$MAPFILE" | jq -r '.mapfile.name')
      DESC=$(echo "$MAPFILE" | jq -r '.mapfile.description // ""')
      FILTERED=$(echo "$CURRENT" | grep -v "^${local.mapfile_key} " || true)
      NEW_CONTENT=$(printf '%s\n%s %s' "$FILTERED" "${local.mapfile_key}" "${local.backend_name}" | sed '/^$/d')
      curl -s -k -u "${local.curl_auth}" \
        -X POST "${var.opnsense_url}/api/haproxy/settings/setMapfile/${local.local_mapfile_uuid}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg name "$NAME" --arg desc "$DESC" --arg content "$NEW_CONTENT" \
          '{"mapfile": {"name": $name, "description": $desc, "content": $content}}')"
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  provisioner "local-exec" {
    when        = destroy
    command     = <<-EOT
      set -e
      MAPFILE=$(curl -s -k -u "${self.input.curl_auth}" \
        "${self.input.opnsense_url}/api/haproxy/settings/getMapfile/${self.input.mapfile_uuid}")
      CURRENT=$(echo "$MAPFILE" | jq -r '.mapfile.content // ""')
      NAME=$(echo "$MAPFILE" | jq -r '.mapfile.name')
      DESC=$(echo "$MAPFILE" | jq -r '.mapfile.description // ""')
      NEW_CONTENT=$(echo "$CURRENT" | grep -v "^${self.input.mapfile_key} " || true)
      curl -s -k -u "${self.input.curl_auth}" \
        -X POST "${self.input.opnsense_url}/api/haproxy/settings/setMapfile/${self.input.mapfile_uuid}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg name "$NAME" --arg desc "$DESC" --arg content "$NEW_CONTENT" \
          '{"mapfile": {"name": $name, "description": $desc, "content": $content}}')"
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  depends_on = [restapi_object.haproxy_backend]
}

# PUBLIC mapfile — only updated when var.public is true
resource "terraform_data" "public_mapfile_entry" {
  count = var.public ? 1 : 0

  input = {
    mapfile_key  = local.mapfile_key
    backend_name = local.backend_name
    mapfile_uuid = local.public_mapfile_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    command     = <<-EOT
      set -e
      MAPFILE=$(curl -s -k -u "${local.curl_auth}" \
        "${var.opnsense_url}/api/haproxy/settings/getMapfile/${local.public_mapfile_uuid}")
      CURRENT=$(echo "$MAPFILE" | jq -r '.mapfile.content // ""')
      NAME=$(echo "$MAPFILE" | jq -r '.mapfile.name')
      DESC=$(echo "$MAPFILE" | jq -r '.mapfile.description // ""')
      FILTERED=$(echo "$CURRENT" | grep -v "^${local.mapfile_key} " || true)
      NEW_CONTENT=$(printf '%s\n%s %s' "$FILTERED" "${local.mapfile_key}" "${local.backend_name}" | sed '/^$/d')
      curl -s -k -u "${local.curl_auth}" \
        -X POST "${var.opnsense_url}/api/haproxy/settings/setMapfile/${local.public_mapfile_uuid}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg name "$NAME" --arg desc "$DESC" --arg content "$NEW_CONTENT" \
          '{"mapfile": {"name": $name, "description": $desc, "content": $content}}')"
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  provisioner "local-exec" {
    when        = destroy
    command     = <<-EOT
      set -e
      MAPFILE=$(curl -s -k -u "${self.input.curl_auth}" \
        "${self.input.opnsense_url}/api/haproxy/settings/getMapfile/${self.input.mapfile_uuid}")
      CURRENT=$(echo "$MAPFILE" | jq -r '.mapfile.content // ""')
      NAME=$(echo "$MAPFILE" | jq -r '.mapfile.name')
      DESC=$(echo "$MAPFILE" | jq -r '.mapfile.description // ""')
      NEW_CONTENT=$(echo "$CURRENT" | grep -v "^${self.input.mapfile_key} " || true)
      curl -s -k -u "${self.input.curl_auth}" \
        -X POST "${self.input.opnsense_url}/api/haproxy/settings/setMapfile/${self.input.mapfile_uuid}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg name "$NAME" --arg desc "$DESC" --arg content "$NEW_CONTENT" \
          '{"mapfile": {"name": $name, "description": $desc, "content": $content}}')"
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  depends_on = [restapi_object.haproxy_backend]
}

# -----------------------------------------------------------------------------
# Unbound DNS Host Override
# -----------------------------------------------------------------------------
resource "restapi_object" "dns_host_override" {
  path         = "/api/unbound/settings/addHostOverride"
  read_path    = "/api/unbound/settings/getHostOverride/{id}"
  update_path  = "/api/unbound/settings/setHostOverride/{id}"
  destroy_path = "/api/unbound/settings/delHostOverride/{id}"
  id_attribute = "uuid"

  data = jsonencode({
    host = {
      enabled  = "1"
      hostname = local.subdomain
      domain   = var.domain
      rr       = "A"
      server   = var.dns_server
    }
  })

  ignore_server_additions = true
}

# -----------------------------------------------------------------------------
# Reconfigure services to apply changes
# -----------------------------------------------------------------------------
resource "terraform_data" "haproxy_reconfigure" {
  input = {
    server_id     = restapi_object.haproxy_server.id
    backend_id    = restapi_object.haproxy_backend.id
    local_mapfile = terraform_data.local_mapfile_entry.id
    public_mapfile = var.public ? terraform_data.public_mapfile_entry[0].id : ""
  }

  provisioner "local-exec" {
    command     = "curl -s -k -u '${local.curl_auth}' -X POST '${var.opnsense_url}/api/haproxy/service/reconfigure'"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "terraform_data" "unbound_reconfigure" {
  input = {
    dns_id = restapi_object.dns_host_override.id
  }

  provisioner "local-exec" {
    command     = "curl -s -k -u '${local.curl_auth}' -X POST '${var.opnsense_url}/api/unbound/service/reconfigure'"
    interpreter = ["/bin/bash", "-c"]
  }
}
