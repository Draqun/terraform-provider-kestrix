#!/usr/bin/env bash
set -Eeuo pipefail

# Opt-in acceptance tests: drive the full resource lifecycle against a running
# Kestrix API. A plan-only smoke test cannot catch state-consistency failures,
# which only surface once a real apply writes the server's answer into state.
#
#   KESTRIX_PROVIDER_ACC=1                 required opt-in
#   KESTRIX_PROVIDER_ENDPOINT=http://...   required, a reachable Kestrix API
#   KESTRIX_PROVIDER_ACC_REPO_URL=...      optional; defaults to a local fixture
#                                          repository, which requires Kestrix to
#                                          run on this host
#   KESTRIX_PROVIDER_API_TOKEN=...         optional bearer token

if [[ "${KESTRIX_PROVIDER_ACC:-0}" != "1" ]]; then
  echo "Provider acceptance tests skipped (set KESTRIX_PROVIDER_ACC=1 to enable)."
  exit 0
fi

endpoint="${KESTRIX_PROVIDER_ENDPOINT:-}"
[[ -n "${endpoint}" ]] || { echo "Acceptance tests require KESTRIX_PROVIDER_ENDPOINT." >&2; exit 1; }
for tool in curl git; do
  command -v "${tool}" >/dev/null 2>&1 || { echo "Acceptance tests require ${tool}." >&2; exit 1; }
done
command -v terraform >/dev/null 2>&1 || command -v tofu >/dev/null 2>&1 || {
  echo "Acceptance tests require terraform or tofu." >&2; exit 1; }

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
provider_bin="${repo_root}/bin/terraform-provider-kestrix"
test -x "${provider_bin}" || { echo "Build the provider first: make build" >&2; exit 1; }

curl -fsS "${endpoint}/api/health" >/dev/null || { echo "Kestrix is not healthy at ${endpoint}." >&2; exit 1; }

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kestrix-provider-acc.XXXXXX")"
projects=()
cleanup() {
  local status=$?
  set +e
  for project in "${projects[@]:-}"; do
    [[ -n "${project}" ]] && curl -fsS -X DELETE "${endpoint}/api/projects/${project}" >/dev/null 2>&1
  done
  rm -rf "${temp_dir}"
  exit "${status}"
}
trap cleanup EXIT INT TERM

# A project is only accepted once Kestrix can clone the repository and parse its
# pipeline, so the default fixture is a real git repository holding one.
repo_url="${KESTRIX_PROVIDER_ACC_REPO_URL:-}"
if [[ -z "${repo_url}" ]]; then
  fixture="${temp_dir}/fixture-repo"
  mkdir -p "${fixture}/ci"
  cat > "${fixture}/ci/pipeline.tf" <<'EOF'
project "acceptance" {
  default_image = "alpine:3.20"

  ci_repo "acceptance" {
    path = "."
    kind = "subpath"
  }

  ci_task "noop" {
    trigger = "repo.acceptance.changed"
    execute {
      repo   = "acceptance"
      script = "true"
    }
  }
}
EOF
  git -C "${fixture}" init -q -b main
  git -C "${fixture}" -c user.email=acc@kestrix.test -c user.name=acceptance add -A
  git -C "${fixture}" -c user.email=acc@kestrix.test -c user.name=acceptance commit -qm "acceptance fixture"
  repo_url="file://${fixture}"
fi

cli_config="${temp_dir}/terraform.rc"
printf '%s\n' \
  'provider_installation {' \
  '  dev_overrides {' \
  '    "registry.terraform.io/kestrix/kestrix" = "'"${repo_root}/bin"'"' \
  '    "registry.opentofu.org/kestrix/kestrix" = "'"${repo_root}/bin"'"' \
  '  }' \
  '  direct {}' \
  '}' > "${cli_config}"
export TF_CLI_CONFIG_FILE="${cli_config}"
export TF_IN_AUTOMATION=1

api_status() {
  curl -s -o /dev/null -w '%{http_code}' "${endpoint}/api/projects/$1"
}

run_lifecycle() {
  local cli="$1"
  local project="acc-${cli}-$$"
  local work="${temp_dir}/${cli}"
  projects+=("${project}")
  mkdir -p "${work}"

  cat > "${work}/main.tf" <<EOF
terraform {
  required_providers {
    kestrix = { source = "kestrix/kestrix" }
  }
}

provider "kestrix" { endpoint = "${endpoint}" }

resource "kestrix_project" "acc" {
  name          = "${project}"
  repo_url      = "${repo_url}"
  repo_ref      = var.repo_ref
  pipeline_dir  = "ci"
  pipeline_file = "pipeline.tf"
}

variable "repo_ref" {
  type    = string
  default = "main"
}
EOF
  if [[ "${cli}" == "tofu" ]]; then
    printf '%s\n' \
      'provider "registry.opentofu.org/kestrix/kestrix" {' \
      '  version = "0.0.0"' \
      '}' > "${work}/.terraform.lock.hcl"
  fi

  echo "── ${cli}: apply"
  "${cli}" -chdir="${work}" apply -auto-approve -no-color >/dev/null
  [[ "$(api_status "${project}")" == "200" ]] || { echo "${cli}: project was not created" >&2; return 1; }

  echo "── ${cli}: plan is empty after apply"
  "${cli}" -chdir="${work}" plan -no-color -detailed-exitcode >/dev/null

  echo "── ${cli}: update"
  git -C "${fixture:-${temp_dir}}" branch -q release main 2>/dev/null || true
  "${cli}" -chdir="${work}" apply -auto-approve -no-color -var repo_ref=release >/dev/null
  curl -fsS "${endpoint}/api/projects/${project}" | grep -q '"repo_ref":"release"' \
    || { echo "${cli}: update did not reach the API" >&2; return 1; }

  echo "── ${cli}: out-of-band drift is detected"
  curl -fsS -X PUT "${endpoint}/api/projects/${project}" -H 'Content-Type: application/json' \
    -d "{\"name\":\"${project}\",\"repo_url\":\"${repo_url}\",\"repo_ref\":\"main\",\"pipeline_dir\":\"ci\",\"pipeline_file\":\"pipeline.tf\"}" >/dev/null
  if "${cli}" -chdir="${work}" plan -no-color -detailed-exitcode -var repo_ref=release >/dev/null; then
    echo "${cli}: drift was not detected" >&2
    return 1
  fi

  echo "── ${cli}: import restores full state"
  "${cli}" -chdir="${work}" state rm kestrix_project.acc -no-color >/dev/null
  "${cli}" -chdir="${work}" import -no-color kestrix_project.acc "${project}" >/dev/null
  "${cli}" -chdir="${work}" plan -no-color -detailed-exitcode >/dev/null

  echo "── ${cli}: a remotely deleted project is planned for recreation"
  curl -fsS -X DELETE "${endpoint}/api/projects/${project}" >/dev/null
  if "${cli}" -chdir="${work}" plan -no-color -detailed-exitcode >/dev/null; then
    echo "${cli}: deletion was not detected" >&2
    return 1
  fi
  "${cli}" -chdir="${work}" apply -auto-approve -no-color >/dev/null

  echo "── ${cli}: destroy"
  "${cli}" -chdir="${work}" destroy -auto-approve -no-color >/dev/null
  [[ "$(api_status "${project}")" == "404" ]] || { echo "${cli}: project outlived destroy" >&2; return 1; }

  echo "✓ ${cli} lifecycle passed"
}

for cli in terraform tofu; do
  if command -v "${cli}" >/dev/null 2>&1; then run_lifecycle "${cli}"; fi
done
echo "Provider acceptance lifecycle passed."
