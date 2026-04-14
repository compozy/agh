# Extension Registry — Test Plan

## Executive Summary

This test plan covers the extension registry feature for AGH, which provides remote discovery, installation, removal, and update capabilities for extensions and skills via multiple registry backends (ClawHub, GitHub Releases). The feature introduces a `RegistrySource` interface, a `MultiRegistry` aggregator, a domain-agnostic `Installer` pipeline, and CLI commands under `agh extension` and `agh skill` namespaces.

### Objectives

1. Validate correct behavior of registry search, info, download, install, remove, and update flows for both extensions and skills.
2. Verify security controls: path traversal prevention, decompression bomb limits, prompt injection scanning, capability ceiling enforcement.
3. Confirm multi-source aggregation with priority-based deduplication and concurrent querying.
4. Ensure CLI UX matches spec: correct flags, error messages, and output formatting.
5. Validate database schema changes (3 new nullable columns) and metadata persistence.

### Key Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Malicious tar.gz with path traversal | Medium | Critical | `CleanArchiveEntryPath()` + `PathWithinRoot()` tests |
| Decompression bomb exhausting disk | Low | Critical | `io.LimitReader` + counting writer with 500MB limit |
| GitHub rate limiting during CI | Medium | High | Mock HTTP servers in tests; optional `GITHUB_TOKEN` |
| Prompt injection in extension manifests | Medium | High | Content verification rules in installer |
| Concurrent installs corrupting state | Low | High | SQLite UNIQUE constraint + temp dir isolation |
| Registry source returning malformed data | Medium | Medium | Error handling and graceful degradation tests |

---

## Scope

### In-Scope

- `internal/registry/` — types, interfaces, MultiRegistry, Installer, extraction, versioning
- `internal/registry/clawhub/` — ClawHub adapter
- `internal/registry/github/` — GitHub Releases adapter
- `internal/cli/extension.go` — Extension CLI commands (search, install, remove, update)
- `internal/cli/extension_marketplace.go` — Extension marketplace helpers
- `internal/cli/skill_commands.go` — Migrated skill CLI commands
- `internal/cli/skill_marketplace.go` — Migrated skill marketplace helpers
- `internal/extension/registry.go` — Registry metadata fields
- `internal/store/globaldb/global_db.go` — Schema additions
- `internal/config/config.go` — Marketplace configuration

### Out-of-Scope

- Daemon hot-reload on install (Phase 2)
- AGH Registry backend (future)
- Cryptographic signature verification (Phase 2)
- Web UI for registry browsing
- `.old_project/` directory

---

## Test Strategy

### Approach

| Layer | Method | Tools |
|-------|--------|-------|
| Unit | Table-driven Go tests with `-race` | `go test`, mock interfaces |
| Integration | Real SQLite + mock HTTP servers | `go test -tags integration` |
| CLI E2E | Binary execution with captured output | `go build` + subprocess |
| Security | Crafted malicious archives + injection payloads | Custom test fixtures |
| Regression | Automated suite covering existing skill flows | `make verify` |

### Test Data Strategy

- **Archives**: Generated via `mustTarGz()` helper for deterministic tar.gz creation
- **HTTP Responses**: `httptest.Server` with canned JSON responses for GitHub/ClawHub APIs
- **Database**: `t.TempDir()` for isolated SQLite instances per test
- **Manifests**: Template-generated `extension.toml` and `SKILL.md` with varying content

---

## Environment Requirements

| Component | Requirement |
|-----------|-------------|
| OS | macOS (darwin), Linux |
| Go | 1.22+ (matches go.mod) |
| SQLite | Bundled via go-sqlite3 |
| Network | Not required (mock HTTP servers for unit/integration) |
| Disk | 100MB free for temp archives and test databases |
| GitHub Token | Optional; required only for live API integration tests |

---

## Entry Criteria

1. All code for extension registry tasks 01-05 is merged to the `ext-registry` branch.
2. `make build` succeeds without errors.
3. `make lint` passes with zero warnings.
4. Existing unit tests pass (`make test`).
5. Test fixtures and helpers are available.

## Exit Criteria

1. All P0 test cases pass.
2. 90%+ of P1 test cases pass.
3. No Critical or High severity bugs remain open.
4. Code coverage >= 80% per package in `internal/registry/`, `internal/registry/clawhub/`, `internal/registry/github/`.
5. `make verify` passes cleanly.
6. Security test cases for path traversal, decompression bombs, and prompt injection all pass.

---

## Test Case Summary

| Category | ID Range | Count | Priority |
|----------|----------|-------|----------|
| Functional — MultiRegistry | TC-FUNC-001 to TC-FUNC-008 | 8 | P0-P1 |
| Functional — Installer | TC-FUNC-009 to TC-FUNC-016 | 8 | P0-P1 |
| Functional — CLI Extension | TC-FUNC-017 to TC-FUNC-026 | 10 | P0-P1 |
| Functional — CLI Skill Migration | TC-FUNC-027 to TC-FUNC-030 | 4 | P1 |
| Integration — ClawHub | TC-INT-001 to TC-INT-004 | 4 | P1 |
| Integration — GitHub | TC-INT-005 to TC-INT-010 | 6 | P0-P1 |
| Security | TC-SEC-001 to TC-SEC-008 | 8 | P0 |
| Regression | TC-REG-001 to TC-REG-006 | 6 | P1 |
| Smoke | SMOKE-001 to SMOKE-005 | 5 | P0 |

**Total: 59 test cases**

---

## Timeline and Deliverables

| Phase | Deliverable |
|-------|-------------|
| Planning | This test plan + all test case files |
| Unit Testing | Go test files with table-driven tests |
| Integration Testing | Integration test files with `//go:build integration` |
| Security Testing | Malicious archive and injection payload tests |
| Regression | Full `make verify` pass + CLI flow verification |
| Report | Verification report with pass/fail summary |
