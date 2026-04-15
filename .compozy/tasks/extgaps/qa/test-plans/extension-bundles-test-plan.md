# Test Plan: Extension Bundles and Activation Runtime

## Executive Summary

This test plan covers the **Extension Bundles and Activation Runtime** feature — a new subsystem allowing extensions to declare static bundle specifications that operators activate through APIs, with the daemon owning activation, persistence, and reconciliation.

**Objectives:**
- Validate bundle spec loading, validation, and catalog discovery
- Verify activation lifecycle (create, preview, update, deactivate)
- Confirm resource materialization (jobs, triggers, bridges) and inventory tracking
- Test network channel binding and effective default resolution
- Ensure extension lifecycle guards (disable/uninstall blocking)
- Validate reconciliation correctness across boot, reload, and state changes
- Verify API contract compliance for all 8 bundle endpoints

**Key Risks:**
- Reconciliation ordering during boot could leave orphaned or stale resources
- Primary channel claim conflicts could deadlock multiple activations
- Rollback on reconciliation failure may leave partial state
- Bridge materialization depends on cross-extension loading which can fail silently

---

## Scope

### In-Scope
- Bundle spec declaration in extension manifests (.toml and .json)
- Bundle loading, validation, and catalog API
- Activation lifecycle via HTTP and UDS APIs
- Resource materialization: automation jobs, triggers, bridge instances
- Inventory tracking per activation
- Network channel declaration and primary channel binding
- Effective default channel computation (runtime vs config)
- Extension disable/uninstall guards when bundles are active
- Reconciliation on activate/update/deactivate/boot/reload
- Transactional rollback on reconciliation failure
- SQLite persistence (bundle_activations, bundle_activation_inventory tables)
- Stable ID generation for deterministic resource identity
- Scope handling (global vs workspace)

### Out-of-Scope
- CLI workflow for bundle activation (not implemented)
- Bundle webhook triggers (explicitly unsupported)
- Web UI for bundle management (separate delivery)
- Performance benchmarking under production load

---

## Test Strategy

| Layer | Approach | Tool |
|-------|----------|------|
| Unit | Table-driven subtests with t.Parallel() | `go test -race` |
| Service | Mock store + automation syncer + extension lister | `go test -race` |
| API | HTTP handler tests with gin test mode | `go test -race` |
| Store | Real SQLite via t.TempDir() | `go test -race` |
| Integration | Real daemon boot with test extensions | `go test -tags integration` |
| Contract | OpenAPI spec vs route inventory | `make verify` |

---

## Environment Requirements

- Go 1.23+ with `-race` flag
- macOS / Linux
- SQLite 3.x (via go-sqlite3)
- `make verify` must pass (fmt, lint, test, build)

---

## Entry Criteria

- All 8 build steps from the PRD are implemented
- `make build` succeeds
- `make lint` reports zero issues
- OpenAPI spec and TypeScript contracts regenerated

---

## Exit Criteria

- All P0 test cases pass
- 90%+ P1 test cases pass
- No critical or high-severity bugs remain open
- `make verify` passes clean
- 80%+ code coverage per package (bundles, extension/bundle, globaldb bundles, api/core bundles)

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Reconciliation leaves orphaned resources after partial failure | Medium | High | TC-FUNC-012, TC-FUNC-013: verify rollback and cleanup |
| Primary channel conflict not detected during concurrent activations | Low | High | TC-FUNC-010: concurrent claim test |
| Extension load failure silently skips bundles in catalog | Medium | Medium | TC-FUNC-001: verify catalog completeness |
| Scope mismatch between activation and materialized resources | Low | High | TC-FUNC-007, TC-FUNC-008: scope propagation tests |
| Stable ID collision for different inputs | Very Low | Critical | TC-FUNC-014: hash collision boundary test |
| SQLite schema migration breaks existing activations | Low | High | TC-INT-001: migration idempotency test |
| Bridge materialization fails when provider extension is absent | Medium | Medium | TC-FUNC-015: cross-extension bridge test |
| Webhook trigger bypass | Low | Medium | TC-SEC-001: webhook rejection test |

---

## Timeline and Deliverables

| Phase | Deliverable | Test Cases |
|-------|-------------|------------|
| 1 - Core | Bundle validation, model, store | TC-FUNC-001 to TC-FUNC-006 |
| 2 - Service | Activation lifecycle, reconciliation | TC-FUNC-007 to TC-FUNC-015 |
| 3 - API | HTTP/UDS handler coverage | TC-INT-001 to TC-INT-010 |
| 4 - Regression | Full regression suite | SMOKE-001 to SMOKE-005 |
| 5 - Edge Cases | Security, boundary conditions | TC-SEC-001 to TC-SEC-003 |

---

## Test Case Index

See individual TC-*.md files in `../test-cases/` for full test case details.

### Functional (TC-FUNC-*)
- TC-FUNC-001: Bundle catalog lists all available bundles from installed extensions
- TC-FUNC-002: Bundle spec validation rejects invalid manifests
- TC-FUNC-003: Activation preview returns materialized resources without persisting
- TC-FUNC-004: Activation creates and persists all resources with inventory
- TC-FUNC-005: Activation update modifies primary channel binding
- TC-FUNC-006: Deactivation removes activation and cleans up resources
- TC-FUNC-007: Global-scope activation propagates scope to all resources
- TC-FUNC-008: Workspace-scope activation resolves workspace and scopes resources
- TC-FUNC-009: Primary channel binding sets effective default channel
- TC-FUNC-010: Second primary channel claim returns 409 conflict
- TC-FUNC-011: Network settings returns configured and effective defaults
- TC-FUNC-012: Failed reconciliation rolls back activation creation
- TC-FUNC-013: Failed reconciliation rolls back activation update
- TC-FUNC-014: Stable ID generation is deterministic and collision-resistant
- TC-FUNC-015: Bridge materialization resolves platform from provider extension

### Integration (TC-INT-*)
- TC-INT-001: HTTP POST /api/bundles/activations creates activation
- TC-INT-002: HTTP GET /api/bundles/catalog returns extension bundles
- TC-INT-003: HTTP PATCH /api/bundles/activations/:id updates binding
- TC-INT-004: HTTP DELETE /api/bundles/activations/:id deactivates
- TC-INT-005: HTTP GET /api/bundles/network/settings returns channel state
- TC-INT-006: Extension disable blocked when bundles active (HTTP 409)
- TC-INT-007: Bundle reconciliation runs during daemon boot
- TC-INT-008: Extension reload triggers bundle reconciliation
- TC-INT-009: UDS endpoints mirror HTTP behavior
- TC-INT-010: All HTTP error codes match StatusForBundleError mapping

### Security (TC-SEC-*)
- TC-SEC-001: Webhook trigger event type rejected in bundle specs
- TC-SEC-002: SQL injection prevented in store layer queries
- TC-SEC-003: Path traversal prevented in bundle file loading

### Smoke (SMOKE-*)
- SMOKE-001: Catalog → Preview → Activate → List → Get → Deactivate
- SMOKE-002: Activate with primary channel → verify effective default → deactivate → verify fallback
- SMOKE-003: Extension with bundles → disable blocked → deactivate bundles → disable succeeds
- SMOKE-004: Bundle with jobs + triggers + bridges → all materialized in inventory
- SMOKE-005: make verify passes with all bundle code
