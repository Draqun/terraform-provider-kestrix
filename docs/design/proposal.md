## Why

Kestrix currently documents a Terraform provider but ships no provider implementation, so project registration still requires direct HTTP calls or the UI. OpenTofu and Terraform users need a supported, idempotent infrastructure-as-code interface with a reproducible build, release, registry publication, and deployment process.

## What Changes

- Add a Go provider compatible with both Terraform Plugin Framework and OpenTofu.
- Implement a `kestrix_project` resource for remote project registration, update, import, refresh, and deletion semantics.
- Add provider configuration for endpoint, authentication, TLS, and request timeouts without storing secrets in state.
- Add acceptance-test infrastructure using a disposable Kestrix API/database stack.
- Add provider documentation, examples, generated schemas, and local validation commands.
- Add CI jobs for unit, protocol, acceptance, build, signing, and release artifacts.
- Publish versioned provider binaries and checksums for Terraform/OpenTofu registry installation.

## Capabilities

### New Capabilities

- `terraform-provider`: OpenTofu/Terraform provider contract, project resource, state/import behavior, testing, packaging, and release deployment.

### Modified Capabilities

- None. The provider consumes the existing project API contract; changes to that API will be proposed separately if implementation reveals a missing requirement.

## Impact

Affected areas include a new provider module/package, project API client, generated provider documentation, CI/release configuration, registry metadata, acceptance-test infrastructure, and operator documentation. The provider must work against local, Docker, and deployed Kestrix endpoints and must preserve the existing encrypted credential-reference model.

## Non-goals

- Managing repository credentials or exposing credential plaintext through Terraform state.
- Managing pipelines, runs, webhooks, Docker infrastructure, or GitLab projects beyond fields already supported by `kestrix_project`.
- Building a hosted Terraform/OpenTofu registry service.
