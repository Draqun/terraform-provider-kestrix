#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
provider_bin="${repo_root}/bin/terraform-provider-kestrix"
test -x "${provider_bin}" || { echo "Build the provider first: make build" >&2; exit 1; }
temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kestrix-provider-protocol.XXXXXX")"
trap 'rm -rf "${temp_dir}"' EXIT
printf '%s\n' \
  'provider_installation {' \
  '  dev_overrides {' \
  '    "registry.terraform.io/kestrix/kestrix" = "'"${repo_root}/bin"'"' \
  '    "registry.opentofu.org/kestrix/kestrix" = "'"${repo_root}/bin"'"' \
  '  }' \
  '  direct {}' \
  '}' > "${temp_dir}/terraform.rc"
printf '%s\n' \
  'terraform {' \
  '  required_providers {' \
  '    kestrix = { source = "kestrix/kestrix", version = "0.0.0" }' \
  '  }' \
  '}' \
  'provider "kestrix" {}' > "${temp_dir}/main.tf"
cat > "${temp_dir}/.terraform.lock.hcl" <<EOF
provider "registry.opentofu.org/kestrix/kestrix" {
  version     = "0.0.0"
  constraints = "0.0.0"
}
provider "registry.terraform.io/kestrix/kestrix" {
  version     = "0.0.0"
  constraints = "0.0.0"
}
EOF

verify_cli() {
  local cli="$1"
  local label="$2"
  if ! command -v "${cli}" >/dev/null 2>&1; then
    echo "${label} not installed; protocol verification skipped."
    return 0
  fi
  TF_CLI_CONFIG_FILE="${temp_dir}/terraform.rc" "${cli}" -chdir="${temp_dir}" providers schema -json >/dev/null
  echo "${label} provider protocol negotiation passed."
}

verify_cli terraform Terraform
verify_cli tofu OpenTofu
