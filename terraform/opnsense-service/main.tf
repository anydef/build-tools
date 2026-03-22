locals {
  subdomain    = var.subdomain != "" ? var.subdomain : var.service_name
  name_upper   = upper(var.service_name)
  server_name  = "${local.name_upper}_server"
  backend_name = "${local.name_upper}_backend"
  port_str     = tostring(var.port)
  curl_auth    = "${var.opnsense_api_key}:${var.opnsense_api_secret}"

  # Per-service rule naming
  acl_name    = "${local.name_upper}_host_acl"
  action_name = "${local.name_upper}_rule"
  fqdn        = "${local.subdomain}.${var.domain}"

  # Mapfile approach (legacy, when use_direct_rules=false)
  domain_prefix       = upper(split(".", var.domain)[0])
  local_mapfile_name  = "${local.domain_prefix}_LOCAL_SUBDOMAINS_mapfile"
  public_mapfile_name = "${local.domain_prefix}_PUBLIC_SUBDOMAINS_mapfile"
  local_mapfile_uuid  = var.use_direct_rules ? "" : data.external.mapfile_lookup[0].result.local_uuid
  public_mapfile_uuid = var.use_direct_rules ? "" : data.external.mapfile_lookup[0].result.public_uuid
  mapfile_key         = local.subdomain
}

# =============================================================================
# Single API call: create/update all HAProxy resources
# =============================================================================
# Fetches the full settings ONCE, then creates or updates server, backend,
# and optionally ACL + action. Returns all UUIDs.
# On subsequent runs where resources already exist, uses per-resource GET/SET
# endpoints (fast, small responses) instead of re-fetching the full settings.
# =============================================================================

data "external" "haproxy_setup" {
  program = ["/bin/bash", "-c", <<-EOT
    set -e
    AUTH="${local.curl_auth}"
    URL="${var.opnsense_url}"
    USE_DIRECT="${var.use_direct_rules}"
    PUBLIC="${var.public}"
    SUBNET_ACL_NAME="${var.local_subnet_acl_name}"
    FRONTEND_NAME="${var.https_frontend_name}"

    SERVER_PAYLOAD='${jsonencode({
      server = {
        enabled     = "1"
        name        = local.server_name
        description = "[terraform] Server for ${var.service_name} at ${var.address}:${local.port_str}"
        address     = var.address
        port        = local.port_str
        ssl         = var.ssl
        sslVerify   = "0"
      }
    })}'

    BACKEND_BASE='${jsonencode({
      backend = {
        enabled                 = "1"
        name                    = local.backend_name
        description             = "[terraform] Backend pool for ${var.service_name} (${local.fqdn})"
        mode                    = "http"
        algorithm               = "source"
        http2Enabled            = var.http2_enabled
        ba_advertised_protocols = var.http2_enabled == "1" ? "h2,http11" : "http11"
        persistence             = "sticktable"
        stickiness_pattern      = "sourceipv4"
        stickiness_expire       = "30m"
        stickiness_size         = "50k"
        tuning_httpreuse        = "safe"
        healthCheckEnabled      = var.health_check_enabled
        healthCheck             = var.health_check
      }
    })}'

    ACL_PAYLOAD='${jsonencode({
      acl = {
        name          = local.acl_name
        description   = "[terraform] Match requests for ${local.fqdn}"
        expression    = "hdr"
        hdr           = local.fqdn
        negate        = "0"
        caseSensitive = "0"
      }
    })}'

    # --- Timing helper ---
    SVC_NAME="${var.service_name}"
    LOGFILE="/tmp/haproxy_setup_$SVC_NAME.log"
    T_START=$(date +%s)
    log_timing() {
      local ELAPSED=$(( $(date +%s) - T_START ))
      echo "[$SVC_NAME] +$${ELAPSED}s $1" | tee -a "$LOGFILE" >&2
    }
    : > "$LOGFILE"
    log_timing "start"

    # --- Helper: find or create a resource ---
    # Usage: find_or_create <type> <name> <add_path> <set_path> <payload> <jq_path>
    # Returns UUID on stdout
    # On plan/refresh: finds existing by name (read-only from cached settings).
    # Only creates via API if the resource doesn't exist yet.
    # Updates are handled separately by terraform_data provisioners.
    find_or_create() {
      local TYPE="$1" NAME="$2" ADD_PATH="$3" SET_PATH="$4" PAYLOAD="$5" JQ_PATH="$6"

      log_timing "find $TYPE '$NAME'..."
      # Try to find by name in the cached settings
      local UUID=$(echo "$SETTINGS" | jq -r --arg name "$NAME" \
        "$JQ_PATH | to_entries[] | select(.value.name == \$name) | .key" | head -1)

      if [ -n "$UUID" ] && [ "$UUID" != "null" ]; then
        log_timing "found $TYPE '$NAME' ($UUID)"
        echo "$UUID"
      else
        # Create new (only on first run)
        log_timing "create $TYPE '$NAME'..."
        local RESPONSE=$(curl -s -k -u "$AUTH" -X POST "$URL$ADD_PATH" \
          -H "Content-Type: application/json" -d "$PAYLOAD")
        local NEW_UUID=$(echo "$RESPONSE" | jq -r '.uuid')
        if [ -z "$NEW_UUID" ] || [ "$NEW_UUID" = "null" ]; then
          echo "Failed to create $TYPE '$NAME': $RESPONSE" >&2
          exit 1
        fi
        log_timing "create $TYPE '$NAME' done ($NEW_UUID)"
        echo "$NEW_UUID"
      fi
    }

    # --- Fetch settings (or use pre-fetched) ---
    # Cache settings to a temp file — shared across parallel module instances
    # within the same terraform run. Avoids hammering the OPNsense API.
    CACHE="/tmp/.haproxy_settings_cache.json"
    CACHE_MAX_AGE=60
    NOW=$(date +%s)
    if [ -f "$CACHE" ]; then
      FILE_AGE=$(stat -c %Y "$CACHE" 2>/dev/null || stat -f %m "$CACHE" 2>/dev/null || echo 0)
      AGE=$(( NOW - FILE_AGE ))
    else
      AGE=999
    fi
    if [ "$AGE" -lt "$CACHE_MAX_AGE" ]; then
      log_timing "reading settings from cache"
      SETTINGS=$(cat "$CACHE")
      log_timing "cache read done"
    else
      log_timing "fetching settings from API..."
      SETTINGS=$(curl -s -k -u "$AUTH" "$URL/api/haproxy/settings/get")
      echo "$SETTINGS" > "$CACHE"
      log_timing "API fetch done"
    fi

    # --- Server ---
    SERVER_UUID=$(find_or_create "server" "${local.server_name}" \
      "/api/haproxy/settings/addServer" \
      "/api/haproxy/settings/setServer" \
      "$SERVER_PAYLOAD" \
      ".haproxy.servers.server")

    # --- Backend (inject linkedServers) ---
    BACKEND_PAYLOAD=$(echo "$BACKEND_BASE" | jq --arg srv "$SERVER_UUID" '.backend.linkedServers = $srv')
    BACKEND_UUID=$(find_or_create "backend" "${local.backend_name}" \
      "/api/haproxy/settings/addBackend" \
      "/api/haproxy/settings/setBackend" \
      "$BACKEND_PAYLOAD" \
      ".haproxy.backends.backend")

    # --- ACL + Action (only for direct rules) ---
    ACL_UUID=""
    ACTION_UUID=""
    FRONTEND_UUID=""
    if [ "$USE_DIRECT" = "true" ]; then
      log_timing "looking up subnet ACL and frontend by name..."
      # Look up subnet ACL and frontend UUIDs by name
      SUBNET_ACL_UUID=$(echo "$SETTINGS" | jq -r --arg name "$SUBNET_ACL_NAME" \
        '.haproxy.acls.acl | to_entries[] | select(.value.name == $name) | .key' | head -1)
      FRONTEND_UUID=$(echo "$SETTINGS" | jq -r --arg name "$FRONTEND_NAME" \
        '.haproxy.frontends.frontend | to_entries[] | select(.value.name == $name) | .key' | head -1)

      if [ -z "$FRONTEND_UUID" ] || [ "$FRONTEND_UUID" = "null" ]; then
        echo "Frontend '$FRONTEND_NAME' not found" >&2
        exit 1
      fi

      ACL_UUID=$(find_or_create "acl" "${local.acl_name}" \
        "/api/haproxy/settings/addAcl" \
        "/api/haproxy/settings/setAcl" \
        "$ACL_PAYLOAD" \
        ".haproxy.acls.acl")

      # Build linkedAcls
      if [ -n "$SUBNET_ACL_UUID" ] && [ "$SUBNET_ACL_UUID" != "null" ] && [ "$PUBLIC" = "false" ]; then
        LINKED_ACLS="$ACL_UUID,$SUBNET_ACL_UUID"
      else
        LINKED_ACLS="$ACL_UUID"
      fi

      ACTION_PAYLOAD=$(jq -n \
        --arg name "${local.action_name}" \
        --arg desc "[terraform] Route ${local.fqdn} to ${local.backend_name}" \
        --arg acls "$LINKED_ACLS" \
        --arg backend "$BACKEND_UUID" \
        '{action: {name: $name, description: $desc, type: "use_backend", testType: "if", linkedAcls: $acls, operator: "and", use_backend: $backend}}')

      ACTION_UUID=$(find_or_create "action" "${local.action_name}" \
        "/api/haproxy/settings/addAction" \
        "/api/haproxy/settings/setAction" \
        "$ACTION_PAYLOAD" \
        ".haproxy.actions.action")
    fi

    log_timing "done"
    jq -n \
      --arg server_uuid "$SERVER_UUID" \
      --arg backend_uuid "$BACKEND_UUID" \
      --arg acl_uuid "$ACL_UUID" \
      --arg action_uuid "$ACTION_UUID" \
      --arg frontend_uuid "$FRONTEND_UUID" \
      '{"server_uuid":$server_uuid,"backend_uuid":$backend_uuid,"acl_uuid":$acl_uuid,"action_uuid":$action_uuid,"frontend_uuid":$frontend_uuid}'
  EOT
  ]
}

# =============================================================================
# Destroy provisioners — one terraform_data per resource for cleanup
# =============================================================================

resource "terraform_data" "haproxy_server" {
  input = {
    uuid         = data.external.haproxy_setup.result.server_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "curl -s -k -u '${self.input.curl_auth}' -X POST '${self.input.opnsense_url}/api/haproxy/settings/delServer/${self.input.uuid}' -H 'Content-Type: application/json' -d '{}'"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "terraform_data" "haproxy_backend" {
  input = {
    uuid         = data.external.haproxy_setup.result.backend_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "curl -s -k -u '${self.input.curl_auth}' -X POST '${self.input.opnsense_url}/api/haproxy/settings/delBackend/${self.input.uuid}' -H 'Content-Type: application/json' -d '{}'"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "terraform_data" "haproxy_acl" {
  count = var.use_direct_rules ? 1 : 0

  input = {
    uuid         = data.external.haproxy_setup.result.acl_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "curl -s -k -u '${self.input.curl_auth}' -X POST '${self.input.opnsense_url}/api/haproxy/settings/delAcl/${self.input.uuid}' -H 'Content-Type: application/json' -d '{}'"
    interpreter = ["/bin/bash", "-c"]
  }
}

resource "terraform_data" "haproxy_action" {
  count = var.use_direct_rules ? 1 : 0

  input = {
    uuid         = data.external.haproxy_setup.result.action_uuid
    opnsense_url = var.opnsense_url
    curl_auth    = local.curl_auth
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "curl -s -k -u '${self.input.curl_auth}' -X POST '${self.input.opnsense_url}/api/haproxy/settings/delAction/${self.input.uuid}' -H 'Content-Type: application/json' -d '{}'"
    interpreter = ["/bin/bash", "-c"]
  }
}

# =============================================================================
# Link action to HTTPS frontend (per-service rules only)
# =============================================================================

resource "terraform_data" "frontend_link" {
  count = var.use_direct_rules ? 1 : 0

  input = {
    action_uuid   = data.external.haproxy_setup.result.action_uuid
    frontend_uuid = data.external.haproxy_setup.result.frontend_uuid
    opnsense_url  = var.opnsense_url
    curl_auth     = local.curl_auth
  }

  provisioner "local-exec" {
    command     = <<-EOT
      set -e
      FRONTEND_UUID="${data.external.haproxy_setup.result.frontend_uuid}"
      FRONTEND=$(curl -s -k -u "${local.curl_auth}" \
        "${var.opnsense_url}/api/haproxy/settings/getFrontend/$FRONTEND_UUID")
      CURRENT=$(echo "$FRONTEND" | jq -r '[.frontend.linkedActions | to_entries[] | select(.value.selected == 1) | .key] | join(",")')
      ACTION_UUID="${data.external.haproxy_setup.result.action_uuid}"

      if echo ",$CURRENT," | grep -q ",$ACTION_UUID,"; then
        echo "Action already linked to frontend"
      else
        # Prepend: per-service rules must come before catch-all mapfile rules
        NEW_ACTIONS="$ACTION_UUID,$CURRENT"
        NEW_ACTIONS=$(echo "$NEW_ACTIONS" | sed 's/,$//')
        curl -s -k -u "${local.curl_auth}" \
          -X POST "${var.opnsense_url}/api/haproxy/settings/setFrontend/$FRONTEND_UUID" \
          -H "Content-Type: application/json" \
          -d "$(jq -n --arg actions "$NEW_ACTIONS" '{"frontend": {"linkedActions": $actions}}')" > /dev/null
      fi
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  provisioner "local-exec" {
    when        = destroy
    command     = <<-EOT
      set -e
      FRONTEND=$(curl -s -k -u "${self.input.curl_auth}" \
        "${self.input.opnsense_url}/api/haproxy/settings/getFrontend/${self.input.frontend_uuid}")
      CURRENT=$(echo "$FRONTEND" | jq -r '[.frontend.linkedActions | to_entries[] | select(.value.selected == 1) | .key] | join(",")')
      ACTION_UUID="${self.input.action_uuid}"
      NEW_ACTIONS=$(echo "$CURRENT" | tr ',' '\n' | grep -v "^$ACTION_UUID$" | paste -sd,)
      curl -s -k -u "${self.input.curl_auth}" \
        -X POST "${self.input.opnsense_url}/api/haproxy/settings/setFrontend/${self.input.frontend_uuid}" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg actions "$NEW_ACTIONS" '{"frontend": {"linkedActions": $actions}}')" > /dev/null
    EOT
    interpreter = ["/bin/bash", "-c"]
  }

  depends_on = [terraform_data.haproxy_action]
}

# =============================================================================
# Mapfile entries (legacy, when use_direct_rules=false)
# =============================================================================

data "external" "mapfile_lookup" {
  count = var.use_direct_rules ? 0 : 1
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

resource "terraform_data" "local_mapfile_entry" {
  count = var.use_direct_rules ? 0 : 1
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

  depends_on = [terraform_data.haproxy_backend]
}

resource "terraform_data" "public_mapfile_entry" {
  count = !var.use_direct_rules && var.public ? 1 : 0

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

  depends_on = [terraform_data.haproxy_backend]
}

# =============================================================================
# Unbound DNS Host Override
# =============================================================================

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

# =============================================================================
# Reconfigure services to apply changes
# =============================================================================

resource "terraform_data" "haproxy_reconfigure" {
  count = var.skip_reconfigure ? 0 : 1

  input = var.use_direct_rules ? {
    server_id      = data.external.haproxy_setup.result.server_uuid
    backend_id     = data.external.haproxy_setup.result.backend_uuid
    frontend_link  = terraform_data.frontend_link[0].id
    local_mapfile  = ""
    public_mapfile = ""
  } : {
    server_id      = data.external.haproxy_setup.result.server_uuid
    backend_id     = data.external.haproxy_setup.result.backend_uuid
    frontend_link  = ""
    local_mapfile  = terraform_data.local_mapfile_entry[0].id
    public_mapfile = var.public ? terraform_data.public_mapfile_entry[0].id : ""
  }

  provisioner "local-exec" {
    command     = <<-EOT
      set -e
      # Restart HAProxy to apply config changes.
      curl -s -k -u '${local.curl_auth}' -X POST '${var.opnsense_url}/api/haproxy/service/restart'
    EOT
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
