# Kestrix Terraform/OpenTofu provider

Manages remote [Kestrix](https://gitlab.com/kestrix/kestrix) project
registrations. It works with both Terraform and OpenTofu, because both speak the
Terraform provider protocol.

## Installation and configuration

Published releases use the address `registry.terraform.io/kestrix/kestrix`.
For local development, build `bin/terraform-provider-kestrix` with `make build`
and point the CLI at it with a development override:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kestrix/kestrix" = "/path/to/this/repo/bin"
    "registry.opentofu.org/kestrix/kestrix" = "/path/to/this/repo/bin"
  }
  direct {}
}
```

```hcl
provider "kestrix" {
  endpoint        = "http://localhost:8080"
  api_token       = var.kestrix_api_token
  ca_file         = "/etc/kestrix/ca.pem"   # optional
  timeout_seconds = 30                      # optional
  insecure        = false                   # optional; avoid in production
}
```

Every setting can also be supplied through `KESTRIX_PROVIDER_ENDPOINT`,
`KESTRIX_PROVIDER_API_TOKEN`, `KESTRIX_PROVIDER_CA_FILE`,
`KESTRIX_PROVIDER_TIMEOUT_SECONDS`, and `KESTRIX_PROVIDER_INSECURE`.
Provider arguments take precedence over environment variables. TLS verification
is enabled by default. API tokens are sensitive and never reach resource state
or error messages.

## `kestrix_project`

`name`, `repo_url`, and `repo_ref` are required. `repo_credential_ref` is only
the reference returned by Kestrix when a credential is stored; the provider
never accepts or resolves Git tokens, passwords, or private keys.

Optional attributes are `pipeline_dir`, `pipeline_file`, `gitlab_project_id`,
`default_branch`, and sensitive `webhook_secret`.

`pipeline_dir`, `pipeline_file`, `default_branch`, and `gitlab_project_id` are
also computed: Kestrix fills them in when the configuration leaves them unset
(`ci`, `pipeline.tf`, `main`, and `0` respectively), and Terraform records the
value the server returned.

```hcl
resource "kestrix_project" "app" {
  name                = "app"
  repo_url            = "https://gitlab.com/acme/app.git"
  repo_ref            = "main"
  repo_credential_ref = var.repo_credential_ref
  pipeline_dir        = "ci"
  pipeline_file       = "pipeline.tf"
}
```

Changing `name` replaces the resource. Import an existing project by its Kestrix
project name:

```bash
terraform import kestrix_project.app app
# or: tofu import kestrix_project.app app
```

Protect the Terraform/OpenTofu state backend: it holds project configuration and
the sensitive webhook secret. Store repository credentials in Kestrix Settings
and pass only their reference to this provider.

## Development

```bash
make build        # bin/terraform-provider-kestrix
make coverage     # unit tests behind a 100% statement gate
make protocol     # start the plugin and negotiate with terraform and tofu
make docs         # validate examples and this document
make quality      # everything above plus linters
```

`make acceptance` is opt-in and drives the full resource lifecycle — apply,
empty follow-up plan, update, out-of-band drift, import, recreation after a
remote delete, and destroy — against a running Kestrix:

```bash
KESTRIX_PROVIDER_ACC=1 KESTRIX_PROVIDER_ENDPOINT=http://127.0.0.1:8080 make acceptance
```

It builds its own fixture repository by default, which requires Kestrix to run
on the same host. Point `KESTRIX_PROVIDER_ACC_REPO_URL` at a repository the
Kestrix deployment can clone to test against a remote deployment.

## Releases

`make release VERSION=v0.1.0` builds Linux, macOS, and Windows amd64/arm64
archives, `SHA256SUMS`, and provenance metadata under `dist/provider/`. CI signs
those exact artifacts with protected credentials, publishes the GitLab release,
and only then runs the protected/manual registry publication job, so a failed
publication is retried without rebuilding.
