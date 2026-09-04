## Context

Kestrix exposes project registration through its HTTP API, including required remote `repo_url` and `repo_ref`, optional `repo_credential_ref`, pipeline location, and GitLab integration fields. The repository has a CLI and API client patterns but no provider module. The provider must be usable by both Terraform and OpenTofu without coupling Kestrix to either CLI's runtime.

The delivery path spans provider code, API contract tests, disposable Kestrix deployment, and registry-style release artifacts:

```text
Terraform/OpenTofu
        |
        v
  Kestrix provider  --->  Kestrix HTTP API  ---> PostgreSQL
        |                         |
        +--> state/import         +--> remote source + credential_ref

Git tag -> CI build/test -> platform binaries -> checksums/signature
        -> GitLab release -> registry publication metadata
```

## Goals / Non-Goals

**Goals:**

- Ship one provider module with protocol compatibility for Terraform and OpenTofu.
- Manage the lifecycle of remote Kestrix projects declaratively and idempotently.
- Keep authentication and repository credential values out of Terraform/OpenTofu state and logs.
- Make local development, acceptance testing, release, and installation reproducible.

**Non-Goals:**

- A hosted registry service owned by Kestrix.
- Managing credential contents, pipeline runs, webhooks, or Docker resources as provider resources.
- Replacing the existing Kestrix API or CLI.

## Decisions

### Provider framework and module layout

Use the Go Terraform Plugin Framework and its protocol v6-compatible server entry point. OpenTofu can consume the same provider protocol and schema. Keep provider code in a separate `provider/` Go package with a small `cmd/terraform-provider-kestrix/` executable; do not import UI or HTTP handlers into it. Use the existing module's Go version and pin framework dependencies.

Alternatives considered: the legacy SDK would reduce initial boilerplate but has weaker typed plan/state behavior; separate Terraform and OpenTofu providers would duplicate maintenance and diverge in protocol behavior.

### API client and configuration

Implement a provider-owned HTTP client with endpoint, API token, TLS verification, CA file, and timeout settings. Configuration precedence is provider block, environment variables, then documented defaults. The token is marked sensitive and is never copied into resource state or diagnostic text. Every request uses context cancellation and bounded timeouts.

### Resource model

Implement `kestrix_project` with the API's project fields: `name`, `repo_url`, `repo_ref`, `repo_credential_ref`, `pipeline_dir`, `pipeline_file`, `gitlab_project_id`, `gitlab_base_url`, `default_branch`, `webhook_secret` handling, and compatible optional fields. `repo_url` and `repo_ref` are required. Use create/update/delete endpoints where available; if deletion is intentionally archival or unsupported, return a clear diagnostic and document the lifecycle behavior rather than silently pretending success.

Use stable project name or server ID as the Terraform ID according to the API contract, and implement import as `kestrix_project.<name> <project-name>`. Read after create/update to persist server-normalized values and detect drift. Credential fields are references only; plaintext credential material is not accepted by this resource.

### Testing

Unit tests use an `httptest.Server` for request, schema, diagnostics, state, timeout, and error behavior. Protocol tests validate provider metadata and schemas. Acceptance tests start the existing Docker deployment with a unique Compose project and test database, set the provider endpoint and master-key path, apply/read/update/import/destroy a project, and clean up with diagnostics on failure. Acceptance tests are opt-in and never use a developer database.

### Release and deployment

CI runs formatting, unit tests, protocol tests, acceptance tests, cross-platform builds, checksum generation, and signature verification. Release tags produce binaries for Linux, macOS, and Windows on amd64 and arm64, plus `SHA256SUMS` and a signed provenance/attestation. A GitLab release stores immutable artifacts. Registry publication is a separate protected job driven by the same tag and artifacts, so failed publication cannot rebuild different binaries. Documentation and examples are published alongside the release.

### Security

Provider API tokens come from environment variables or a sensitive provider argument and are redacted by diagnostics. Repository credentials are represented only by `repo_credential_ref`; the provider must reject fields named token/password/private key and must not log request bodies. TLS verification is enabled by default. Acceptance fixtures use disposable secrets and isolated databases.

## Risks / Trade-offs

- **API/provider schema drift** → contract tests run against the handler layer and acceptance tests run against the production API image.
- **OpenTofu and Terraform protocol differences** → use the shared protocol implementation and test both CLIs in CI where available.
- **Destroy semantics may conflict with project retention policy** → resolve and document API delete behavior before implementation; fail explicitly if deletion is unsupported.
- **Registry credentials could leak in CI** → keep publication in a protected, manual/tag-only job and use masked variables.
- **Acceptance tests are slower and Docker-dependent** → keep them opt-in locally, run them in a dedicated CI job, and retain compose logs as artifacts.

## Migration Plan

1. Add the provider module, API client, schemas, tests, docs, and local Makefile targets without changing existing API behavior.
2. Add the provider acceptance environment and run it against the current Docker deployment.
3. Publish an initial `v0.1.0` provider release from a signed tag; document local filesystem installation for development.
4. Enable protected registry publication after artifact and acceptance jobs pass.
5. Roll back by removing the provider release from the registry channel; existing Kestrix projects and API deployments remain unchanged.

## Open Questions

- Does the project API guarantee a stable numeric ID, or should the provider use project name as its immutable ID?
- Should project deletion delete the Kestrix registration or transition it to an archived state?
- Which registry namespace and signing identity will be used for the first public release?
