#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for example in "${repo_root}/examples/terraform/local" "${repo_root}/examples/opentofu/local"; do
  test -f "${example}/main.tf" || { echo "Missing provider example: ${example}" >&2; exit 1; }
  grep -q 'repo_credential_ref' "${example}/main.tf"
  grep -q 'repo_url' "${example}/main.tf"
  grep -q 'repo_ref' "${example}/main.tf"
done
grep -q 'KESTRIX_PROVIDER_API_TOKEN' "${repo_root}/README.md"
grep -q 'kestrix_project' "${repo_root}/README.md"
if command -v terraform >/dev/null 2>&1; then terraform fmt -check "${repo_root}/examples/terraform"; fi
if command -v tofu >/dev/null 2>&1; then tofu fmt -check "${repo_root}/examples/opentofu"; fi
if command -v terraform >/dev/null 2>&1; then
  "${repo_root}/scripts/protocol.sh"
fi
echo "Provider documentation and examples are valid."
