# =============================================================================
# cargo.mk — shared targets for cross-compiling Rust binaries and publishing
# crates to a Gitea cargo registry.
# =============================================================================
# Include from a project Makefile after setting:
#
#   CARGO_TARGET   (required, e.g. arm-unknown-linux-musleabi)
#   PRIMARY_BIN    (required — primary binary name)
#
# Optional:
#   SECONDARY_BIN  additional binary names included in the tarball (space-sep)
#   PKG_VERSION    defaults to scraped Cargo.toml [package].version
#   CARGO          defaults to `cross` (local); CI sets CARGO=cargo
#   ENV_FILE       defaults to $(BUILD_CONTEXT)/.env.tpl
#   BUILD_CONTEXT  defaults to $(CURDIR)
#
# Local targets:   build, package, publish
# CI    targets:   ci-build, ci-package, ci-publish
#                  (invoked by .gitea/workflows/release.yaml)
#
# The release upload to Gitea is intentionally NOT here — workflows do that
# directly with akkuman/gitea-release-action so we don't pull `tea`/`curl`/`jq`
# into every Rust project.
# =============================================================================

# Resolve BUILD_TOOLS_DIR to the directory containing this file when not set.
BUILD_TOOLS_DIR ?= $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

BUILD_CONTEXT ?= $(CURDIR)
ENV_FILE      ?= $(BUILD_CONTEXT)/.env.tpl
CARGO         ?= cross
PKG_VERSION   ?= $(shell awk -F\" '/^version/ {print $$2; exit}' $(BUILD_CONTEXT)/Cargo.toml)

RELEASE_DIR    := $(BUILD_CONTEXT)/target/$(CARGO_TARGET)/release
DIST_DIR       := $(BUILD_CONTEXT)/dist
ARTIFACT_NAME  := $(PRIMARY_BIN)-$(PKG_VERSION)-$(CARGO_TARGET).tar.gz

# Sourced before recipes that need secrets. CI sets _OP_LOADED=1 to skip.
LOAD_ENV = if [ -z "$$_OP_LOADED" ] && [ -f "$(ENV_FILE)" ]; then \
		source $(BUILD_TOOLS_DIR)/load-env-tpl.sh $(ENV_FILE); \
	fi;

# Gitea's cargo registry expects `Authorization: Bearer <token>`.
# `cargo publish` sends the token in CARGO_REGISTRIES_GITEA_TOKEN verbatim,
# so we prefix it if it isn't already.
BEARERIZE_CARGO_TOKEN = \
	case "$$CARGO_REGISTRIES_GITEA_TOKEN" in \
		Bearer\ *) ;; \
		*) export CARGO_REGISTRIES_GITEA_TOKEN="Bearer $$CARGO_REGISTRIES_GITEA_TOKEN" ;; \
	esac;

.PHONY: build package publish ci-build ci-package ci-publish cargo-help

cargo-help:
	@echo "Local cargo targets:"
	@echo "  build      cross-compile release binaries for $(CARGO_TARGET) (CARGO=$(CARGO))"
	@echo "  package    tar release binaries into $(DIST_DIR)/"
	@echo "  publish    publish crate to the Gitea cargo registry"
	@echo ""
	@echo "CI targets (invoked by .gitea/workflows/release.yaml):"
	@echo "  ci-build, ci-package, ci-publish"

## Cross-compile release binaries
build:
	$(CARGO) build --target $(CARGO_TARGET) --release

## Tar release binaries for distribution / release upload
package: build
	mkdir -p $(DIST_DIR)
	tar -C $(RELEASE_DIR) -czf $(DIST_DIR)/$(ARTIFACT_NAME) $(PRIMARY_BIN) $(SECONDARY_BIN)
	@echo "Packaged $(DIST_DIR)/$(ARTIFACT_NAME)"

## Publish the crate to the Gitea cargo registry.
publish:
	@$(LOAD_ENV) \
	if [ -z "$$CARGO_REGISTRIES_GITEA_TOKEN" ]; then \
		echo "CARGO_REGISTRIES_GITEA_TOKEN is not set"; exit 1; \
	fi; \
	$(BEARERIZE_CARGO_TOKEN) \
	cargo publish --registry gitea --allow-dirty

# -----------------------------------------------------------------------------
# CI variants — bypass `cross` (the runner image already has the toolchain).
# `ci-publish` tolerates "already exists" so version bumps drive publishes.
# -----------------------------------------------------------------------------

ci-build:
	$(MAKE) CARGO=cargo build

ci-package:
	$(MAKE) CARGO=cargo package

ci-publish:
	@$(LOAD_ENV) \
	if [ -z "$$CARGO_REGISTRIES_GITEA_TOKEN" ]; then \
		echo "CARGO_REGISTRIES_GITEA_TOKEN is not set"; exit 1; \
	fi; \
	$(BEARERIZE_CARGO_TOKEN) \
	out=$$(cargo publish --registry gitea --allow-dirty 2>&1); rc=$$?; \
	echo "$$out"; \
	if [ $$rc -ne 0 ] && ! echo "$$out" | grep -qiE "already (exists|uploaded)"; then \
		exit $$rc; \
	fi
