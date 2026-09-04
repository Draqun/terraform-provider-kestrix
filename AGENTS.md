# AGENTS.md — terraform-provider-kestrix

## What this is
The Terraform/OpenTofu provider for [Kestrix](https://gitlab.com/kestrix/kestrix).
It lives in its own repository because the Terraform Registry requires a
dedicated `terraform-provider-<name>` repository with its own `v*` release tags.

## Tech stack
- Go 1.25, `github.com/hashicorp/terraform-plugin-framework` v1.19
- No dependency on the Kestrix module: the provider talks to the public HTTP API

## Layout
| Path | Role |
|------|------|
| `main.go` | Plugin entry point (`providerserver.Serve`) |
| `internal/provider/provider.go` | Provider schema, configuration precedence |
| `internal/provider/project_resource.go` | `kestrix_project` resource |
| `internal/provider/client.go` | Kestrix HTTP API client |
| `scripts/` | protocol, docs, coverage, acceptance, release |
| `examples/` | Terraform and OpenTofu examples |

## Commands (Makefile is source of truth)
```bash
make build       # bin/terraform-provider-kestrix
make coverage    # unit tests behind a 100% statement gate
make protocol    # negotiate with terraform and tofu
make docs        # validate examples and README
make quality     # linters + everything above
make acceptance  # opt-in full lifecycle, needs a running Kestrix
make release VERSION=v0.1.0
```

## Conventions
- Conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `test:`
- **100% statement coverage** on `internal/provider` is enforced by CI
- `gofmt`, `go vet`, and `staticcheck` must pass; staticcheck is pinned in the
  Makefile because newer releases raise their minimum Go version
- **Makefile = source of truth** — never run tools manually
- Prefer `jj` (jujutsu) for VCS if available, otherwise `git`

## Facts agents miss
- **A plan-only test proves nothing.** State-consistency failures appear only on
  a real `apply`. `pipeline_dir`, `pipeline_file`, `default_branch`, and
  `gitlab_project_id` must stay `Optional + Computed` because Kestrix assigns
  defaults on create.
- **State may never hold an unknown value** — `optionalString` maps an unknown
  fallback to null.
- The acceptance script builds a **local fixture git repository** by default, so
  it only works when Kestrix runs on the same host. Point
  `KESTRIX_PROVIDER_ACC_REPO_URL` at a reachable repository otherwise.
- Kestrix validates a project on create by cloning the repository and parsing
  its pipeline file, so acceptance repositories need a real `ci/pipeline.tf`.
