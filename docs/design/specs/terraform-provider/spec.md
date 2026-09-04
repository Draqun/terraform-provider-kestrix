## ADDED Requirements

### Requirement: Provider is compatible with Terraform and OpenTofu
The project SHALL provide a versioned `kestrix` provider using the Terraform provider protocol supported by both Terraform and OpenTofu.

#### Scenario: Provider starts through the protocol
- **WHEN** Terraform or OpenTofu launches the provider during `init`
- **THEN** the provider SHALL complete protocol negotiation and expose its schema without requiring a Kestrix server connection

#### Scenario: Provider binaries are platform-specific
- **WHEN** a release is published
- **THEN** it SHALL contain checksummed binaries for the supported operating-system and architecture pairs with matching registry metadata

### Requirement: Provider configuration is explicit and secret-safe
The provider SHALL accept an endpoint, optional API token, TLS options, and request timeout through provider configuration and documented environment variables. API tokens SHALL be sensitive and SHALL NOT be persisted in resource state or diagnostic messages.

#### Scenario: Environment-based configuration
- **WHEN** the provider block omits a setting and its documented environment variable is present
- **THEN** the provider SHALL use that environment value

#### Scenario: Invalid endpoint or TLS configuration
- **WHEN** configuration contains an invalid endpoint, CA file, or timeout
- **THEN** provider configuration SHALL fail with an actionable diagnostic before any resource operation

### Requirement: Project resource manages remote Kestrix projects
The provider SHALL expose a `kestrix_project` resource for the supported project registration fields. `repo_url` and `repo_ref` SHALL be required, and `repo_credential_ref` SHALL contain only a stored credential reference.

#### Scenario: Create a remote project
- **WHEN** a valid `kestrix_project` is applied
- **THEN** the provider SHALL create the project through the Kestrix API and persist its stable identity in state

#### Scenario: Update mutable project settings
- **WHEN** a mutable project attribute changes
- **THEN** the provider SHALL update the existing Kestrix project and refresh state from the API response

#### Scenario: Missing remote source fields
- **WHEN** `repo_url` or `repo_ref` is omitted
- **THEN** planning or apply SHALL fail with a schema or validation diagnostic and SHALL NOT create a project

### Requirement: Resource refresh and import preserve drift semantics
The provider SHALL read the remote project during refresh and SHALL support importing an existing project by its documented identity. Server-normalized values SHALL be written back to state.

#### Scenario: Import an existing project
- **WHEN** a user imports `kestrix_project.example` using a valid project identity
- **THEN** the provider SHALL read the project and populate all non-secret state attributes

#### Scenario: Remote project disappears
- **WHEN** refresh receives a not-found response
- **THEN** the provider SHALL remove the resource from state so Terraform/OpenTofu can plan recreation

#### Scenario: Remote drift is detected
- **WHEN** a remote mutable field differs from configuration
- **THEN** refresh and planning SHALL expose the difference without silently overwriting it

### Requirement: Provider never handles plaintext repository credentials
The provider SHALL reject plaintext repository credential fields and SHALL send only `repo_credential_ref` when configuring a project source.

#### Scenario: Plaintext credential is supplied
- **WHEN** configuration contains a token, password, or private key field for a project
- **THEN** validation SHALL fail and the value SHALL NOT appear in a request, state snapshot, or diagnostic

#### Scenario: Credential reference is supplied
- **WHEN** configuration contains `repo_credential_ref`
- **THEN** the provider SHALL pass the reference to Kestrix without resolving or exposing its secret material

### Requirement: Provider behavior is covered by isolated tests
The project SHALL provide unit, protocol, and acceptance tests for provider configuration, resource lifecycle, import, drift, errors, and secret redaction. Acceptance tests SHALL use an isolated disposable Kestrix deployment and test database.

#### Scenario: Unit and protocol tests run without Docker
- **WHEN** the provider unit test target runs without Docker
- **THEN** it SHALL test the API client and schemas using local HTTP fixtures and SHALL pass or report only Docker-dependent tests as skipped

#### Scenario: Acceptance test cleanup
- **WHEN** an acceptance test passes, fails, or is interrupted
- **THEN** its temporary Compose project, volumes, credentials, and generated files SHALL be cleaned up and failure diagnostics SHALL be retained

### Requirement: Provider releases are reproducible and deployable
The project SHALL define Makefile and CI targets that format, test, build, checksum, sign, publish, and document provider releases. Publication SHALL be tag-driven and SHALL use the exact artifacts validated by CI.

#### Scenario: Release candidate validation
- **WHEN** a provider release tag is created
- **THEN** CI SHALL run unit, protocol, acceptance, cross-platform build, checksum, and signature verification jobs before publication

#### Scenario: Failed publication
- **WHEN** registry publication fails after artifact validation
- **THEN** the validated artifacts SHALL remain available from the GitLab release and a retry SHALL not rebuild them

#### Scenario: Local installation
- **WHEN** a developer runs the documented local installation flow
- **THEN** Terraform and OpenTofu SHALL be able to initialize the provider against a local Kestrix instance using the generated development artifact

### Requirement: Provider documentation matches the implementation
The project SHALL document provider configuration, all resource attributes, import, state security, examples, local development, acceptance testing, and release installation for both Terraform and OpenTofu.

#### Scenario: Documentation is generated and checked
- **WHEN** the documentation build and provider documentation test run
- **THEN** provider reference pages and examples SHALL be present and SHALL match the provider schema

#### Scenario: User follows the quick start
- **WHEN** a user configures the provider and applies the documented project example
- **THEN** the example SHALL create a valid remote Kestrix project without requiring plaintext repository credentials
