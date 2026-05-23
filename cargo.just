# =============================================================================
# cargo.just — shared recipes for Rust cross-compile + Gitea cargo publish
# (mirror of cargo.mk)
# =============================================================================
# Import from a project justfile after setting:
#
#   cargo_target    (required, e.g. "arm-unknown-linux-musleabi")
#   primary_bin     (required — primary binary name)
#   build_tools_dir (required — caller provides; typically from bootstrap)
#
# Optional:
#   secondary_bin   (space-separated extra binary names to include in tarball)
#   build_context   (defaults to justfile_directory())
#   env_file        (defaults to build_context/.env.tpl)
#
# Locally CARGO defaults to `cross`; in CI set CARGO=cargo (or use ci-* recipes).
# =============================================================================

secondary_bin := ""

build_context := justfile_directory()
env_file      := build_context / ".env.tpl"
cargo         := env_var_or_default("CARGO", "cross")
pkg_version   := shell("grep -m1 '^version' " + build_context / "Cargo.toml" + " | cut -d'\"' -f2")

release_dir   := build_context / "target" / cargo_target / "release"
dist_dir      := build_context / "dist"
artifact_name := primary_bin + "-" + pkg_version + "-" + cargo_target + ".tar.gz"

# Cross-compile release binaries
build:
    {{cargo}} build --target {{cargo_target}} --release

# Tar release binaries for distribution / release upload
package: build
    mkdir -p {{dist_dir}}
    tar -C {{release_dir}} -czf {{dist_dir}}/{{artifact_name}} {{primary_bin}} {{secondary_bin}}
    @echo "Packaged {{dist_dir}}/{{artifact_name}}"

# Publish the crate to the Gitea cargo registry
[script('bash')]
publish:
    set -e
    if [ -z "${_OP_LOADED:-}" ] && [ -f "{{env_file}}" ]; then
        source {{build_tools_dir}}/load-env-tpl.sh {{env_file}}
    fi
    if [ -z "${CARGO_REGISTRIES_GITEA_TOKEN:-}" ]; then
        echo "CARGO_REGISTRIES_GITEA_TOKEN is not set"; exit 1
    fi
    case "$CARGO_REGISTRIES_GITEA_TOKEN" in
        Bearer\ *) ;;
        *) export CARGO_REGISTRIES_GITEA_TOKEN="Bearer $CARGO_REGISTRIES_GITEA_TOKEN" ;;
    esac
    cargo publish --registry gitea --allow-dirty

# -----------------------------------------------------------------------------
# CI variants — bypass `cross` (runner image already has the toolchain).
# `ci-publish` tolerates "already exists" so version bumps drive publishes.
# -----------------------------------------------------------------------------

ci-build:
    CARGO=cargo just build

ci-package:
    CARGO=cargo just package

[script('bash')]
ci-publish:
    set -e
    if [ -z "${_OP_LOADED:-}" ] && [ -f "{{env_file}}" ]; then
        source {{build_tools_dir}}/load-env-tpl.sh {{env_file}}
    fi
    if [ -z "${CARGO_REGISTRIES_GITEA_TOKEN:-}" ]; then
        echo "CARGO_REGISTRIES_GITEA_TOKEN is not set"; exit 1
    fi
    case "$CARGO_REGISTRIES_GITEA_TOKEN" in
        Bearer\ *) ;;
        *) export CARGO_REGISTRIES_GITEA_TOKEN="Bearer $CARGO_REGISTRIES_GITEA_TOKEN" ;;
    esac
    set +e
    out=$(cargo publish --registry gitea --allow-dirty 2>&1)
    rc=$?
    set -e
    echo "$out"
    if [ $rc -ne 0 ] && ! echo "$out" | grep -qiE "already (exists|uploaded)"; then
        exit $rc
    fi
