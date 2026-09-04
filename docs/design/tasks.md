## 1. Provider module and protocol

- [x] 1.1 Create the provider Go package and `cmd/terraform-provider-kestrix` protocol entry point.
- [x] 1.2 Pin Terraform Plugin Framework dependencies and add provider metadata/version injection.
- [x] 1.3 Add provider schema and configuration for endpoint, API token, TLS, CA file, and timeout.
- [x] 1.4 Add provider configuration precedence, validation diagnostics, context cancellation, and secret redaction.
- [x] 1.5 Add Makefile targets for provider format, unit tests, protocol tests, and local build.
- [x] 1.6 Verify the provider binary starts and negotiates successfully with Terraform and OpenTofu protocol clients.

## 2. Kestrix API client

- [x] 2.1 Implement a provider-owned HTTP client with base URL normalization, auth headers, timeout, and TLS transport.
- [x] 2.2 Implement typed project request/response models matching the current Kestrix project API.
- [x] 2.3 Implement create, read, update, and delete operations with status/error mapping and redacted diagnostics.
- [x] 2.4 Add `httptest.Server` coverage for success responses, validation errors, 404, 409, 5xx, timeout, and malformed JSON.
- [x] 2.5 Verify client tests pass without PostgreSQL, Docker, or external network access.

## 3. Project resource and state

- [x] 3.1 Implement the `kestrix_project` resource schema with required remote URL/ref and supported optional fields.
- [x] 3.2 Implement Create and Read with server-normalized state and stable project identity.
- [x] 3.3 Implement Update for mutable fields and explicit replacement behavior for immutable fields.
- [x] 3.4 Implement Delete according to the resolved Kestrix project retention/delete contract and fail explicitly if unsupported.
- [x] 3.5 Implement import by project name or stable API ID and verify all non-secret attributes are populated.
- [x] 3.6 Handle remote 404 by removing the resource from state and add drift tests for remote changes.
- [x] 3.7 Reject token, password, private-key, and other plaintext repository credential attributes.
- [x] 3.8 Add resource tests for plan validation, lifecycle, import, drift, replacement, and secret redaction.

## 4. Acceptance environment

- [x] 4.1 Add a disposable provider acceptance stack based on the existing production Compose deployment and isolated ports/project name.
- [x] 4.2 Add test setup that creates a mode-0600 master key and uses the protected `_test` database identity.
- [ ] 4.3 Add acceptance coverage for apply, refresh, update, import, drift, not-found recreation, and destroy behavior.
- [x] 4.4 Add failure diagnostics and deterministic cleanup for Compose resources, volumes, temporary files, and credentials.
- [x] 4.5 Add Makefile acceptance target that is opt-in when Docker is unavailable and fails clearly when explicitly requested.

## 5. Documentation and examples

- [x] 5.1 Replace the early-development provider document with implementation-backed Terraform and OpenTofu installation instructions.
- [x] 5.2 Document provider configuration, environment variables, TLS, timeouts, resource attributes, import, and state security.
- [x] 5.3 Add working Terraform and OpenTofu examples for local Kestrix and deployed Kestrix endpoints.
- [x] 5.4 Add documentation validation that checks examples and generated provider schema/reference output.

## 6. Build, release, and registry deployment

- [x] 6.1 Add CI jobs for provider format, unit/protocol tests, acceptance tests, and cross-platform compilation.
- [x] 6.2 Add reproducible release packaging for Linux, macOS, and Windows on amd64 and arm64.
- [x] 6.3 Generate `SHA256SUMS`, provenance metadata, and signed artifacts using protected CI credentials.
- [x] 6.4 Add tag-driven GitLab release creation that publishes the exact artifacts validated by CI.
- [x] 6.5 Add protected/manual registry publication using the validated release artifacts without rebuilding.
- [x] 6.6 Document development installation, release promotion, rollback, and registry verification.
- [ ] 6.7 Verify the complete release pipeline with a non-production version tag and both Terraform and OpenTofu initialization.

## 7. Final verification

- [x] 7.1 Run `make fmt-check`, `make vet`, provider unit/protocol tests, and the existing backend test suite.
- [x] 7.2 Run provider acceptance tests against the disposable Docker deployment.
- [x] 7.3 Run Terraform and OpenTofu example init/plan/apply/import/destroy checks.
- [x] 7.4 Run `openspec validate --strict` and review generated documentation and release artifacts for secrets.
