---
status: completed
title: Environment, Extension, and Release Hardening
type: infra
complexity: high
dependencies:
  - task_08
---

# Task 09: Environment, Extension, and Release Hardening

## Overview

Finish the selected environment, extension, and release hardening items after setup lifecycle behavior is stable. This task sanitizes and repairs `.env` handling, adds extension `requires_env` validation, and updates release packaging targets while preserving signing, checksums, and SBOM expectations.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and task_08 outputs before changing release or setup scripts
- DO NOT commit generated secrets, local `.env` contents, or private release artifacts
- DO NOT weaken signing, checksum, or SBOM behavior while adding packaging targets
- `.env` repair must be explicit, bounded, and safe for user-owned files
- Extension environment validation must report missing requirements without leaking values
</critical>

<requirements>
- MUST sanitize `.env` generation and repair flows so malformed or unsafe entries are handled predictably
- MUST add extension `requires_env` metadata parsing and validation
- MUST expose missing extension environment requirements through CLI/API surfaces where appropriate
- MUST update GoReleaser packaging targets needed by Hermes hardening without removing existing release guarantees
- MUST add tests or dry-run validation for environment repair, extension requirements, and release config integrity
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 9.1 Audit current `.env`, extension manifest, and release packaging behavior against the TechSpec
- [x] 9.2 Implement `.env` sanitization and repair with clear diagnostics and safe write behavior
- [x] 9.3 Add `requires_env` support to extension manifests, validation, and operator-visible status
- [x] 9.4 Update release packaging targets while preserving signing, checksums, and SBOM behavior
- [x] 9.5 Add focused tests and release dry-run checks for the changed environment and packaging paths
- [x] 9.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Keep environment repair conservative: parse structured lines, report unsupported content, and avoid overwriting user intent without a clear command path. Extension `requires_env` should become part of manifest validation and status reporting. Release changes should be validated through configuration checks rather than relying on manual inspection.

### Relevant Files
- `internal/config/config.go` - environment-related config validation
- `internal/config/bootstrap.go` - `.env` generation and repair entry points
- `internal/extension/manifest.go` - extension manifest schema and `requires_env`
- `internal/cli/extension.go` - extension validation and status output
- `.goreleaser.yml` - packaging targets, checksums, signing, and SBOM settings
- `.github/workflows/release.yml` - release workflow integration
- `scripts/` - release or packaging helper scripts

### Dependent Files
- `internal/config/*_test.go` - `.env` parsing, sanitization, and repair tests
- `internal/extension/*_test.go` - manifest `requires_env` validation tests
- `internal/cli/*extension*_test.go` - CLI output for missing environment requirements
- `packages/site/` - install, environment, extension, and release docs
- `web/src/` - extension settings/status display if environment requirements surface in the app
- `.compozy/tasks/hermes/task_10.md` - QA plan must include environment and release checks

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - includes environment, extension, and release hardening in the selected scope

## Deliverables
- Safe `.env` sanitization and repair behavior
- Extension `requires_env` manifest support and validation
- Operator-visible missing environment requirement diagnostics
- Updated release packaging config with signing, checksums, and SBOM preserved
- Tests and release dry-run validation evidence
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] `.env` sanitizer preserves valid entries and rejects or repairs malformed entries predictably
  - [x] Extension manifest validation detects missing `requires_env` values without exposing secret values
  - [x] CLI diagnostics identify missing requirements and affected extensions clearly
  - [x] Release config parsing confirms expected targets, checksums, signing, and SBOM entries
- Integration tests:
  - [x] Setup or config flow repairs `.env` in a temp home without touching unrelated files
  - [x] Extension install/status reports missing environment requirements consistently
  - [x] Release dry-run or config check succeeds with the updated packaging matrix
- Test coverage target: >=80%
- All tests must pass

### Verification Evidence

- `go test ./internal/config ./internal/extension ./internal/cli ./internal/api/contract ./internal/api/core ./internal/daemon ./internal/settings`
- `go test ./internal/config -run TestGoReleaserConfigPreservesTrustArtifactsAndPackageTargets -count=1`
- `bun run typecheck` in `web/`
- `bunx vitest run src/routes/_app/settings/-hooks-extensions.test.tsx src/hooks/routes/use-settings-hooks-extensions-page.test.tsx src/systems/settings/adapters/settings-api.test.ts` in `web/`
- `make web-lint`
- `make verify` passed after lint fixes: web lint/typecheck/test/build, Go fmt/lint, race-enabled Go tests (`DONE 5851 tests in 55.156s`), build, and package boundaries.
- `go run github.com/goreleaser/goreleaser/v2@v2.15.3 check` was attempted for local dry-run/config validation but GoReleaser OSS rejected the repository's existing Pro configuration; CI remains configured to run GoReleaser Pro dry-run, and the added Go release-config test is the local integrity gate.

## Success Criteria
- Environment setup is safer and repairable
- Extensions declare required environment variables and expose missing requirements clearly
- Release packaging changes keep existing trust artifacts intact
- Affected infra, CLI, web, and docs tests pass
