# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Funding configuration (`.github/FUNDING.yml`) for GitHub Sponsors and PayPal.
- Project status badges in `README.md` (CI pipeline, Terraform Registry, OpenTofu Registry, Go Report Card, License, Funding).
- Release automation tooling (`scripts/release.py`, `scripts/verify_version.py`, `scripts/generate_release_notes.py`) and Makefile targets (`release`, `release-push`, `release-dry-run`, `release-notes`).
- GitLab CI release jobs with automated release notes extraction and version verification.

## [0.1.0] - 2026-10-28

### Added
- Initial release of the Terraform and OpenTofu provider for Kestrix (`terraform-provider-kestrix`).
- `kestrix_project` resource:
  - Manage project definitions, repository configurations (`ci_repo`), and task DAG definitions (`ci_task`).
  - Supports dependency graphing (`depends_on`), conditional trigger expressions, runner image selection, and execution scripts.
- `kestrix_variable` resource:
  - Manage environment and pipeline variables scoped to projects or globally.
  - Supports masked and sensitive variable values with write-only protection.
- Data sources:
  - `kestrix_project`: inspect existing project configurations and repository mappings.
  - `kestrix_run`: query pipeline execution statuses, DAG states, and run results.
- Terraform Plugin Framework: built with Terraform Plugin Framework protocol v6, compatible with HashiCorp Terraform >= 1.0 and OpenTofu >= 1.6.
- Multi-platform support: automated cross-compilation for 11 OS/architecture targets (Linux, macOS, Windows, FreeBSD, OpenBSD on amd64 and arm64).
- GoReleaser packaging and GPG-signed manifests compatible with HashiCorp Terraform Registry (`Draqun/kestrix`) and OpenTofu Registry (`kestrix/kestrix`).
- Unit and acceptance test suites with 100% statement coverage enforcement.
- Complete documentation and usage examples for resources and data sources.
