# Test Plan: Shared Extensibility Resource Runtime

## Executive Summary

This test plan covers the full implementation of the Shared Extensibility Resource Runtime as specified in the `extensibility-parity` TechSpec. The runtime replaces fragmented, domain-specific extensibility catalogs with a persisted shared resource runtime that becomes the authoritative desired-state control plane for 10 first-wave resource kinds across 12 implementation tasks.

### Objectives

- Validate that the canonical resource persistence kernel enforces authority, scope, ownership, and concurrency rules correctly
- Verify typed codecs, stores, and projector adapters keep raw JSON confined to the persistence boundary
- Confirm the reconcile driver honors single-flight, coalescing, topology ordering, degraded-circuit, and shutdown semantics
- Prove that each family migration (hooks, tools, MCP, agents, skills, automation, bridges, bundles) cleanly cuts over to resource-backed authority
- Verify extension protocol changes (grants, nonce, snapshot) enforce same-source isolation
- Validate UDS CRUD surface behavior and HTTP mutation gating
- Ensure no legacy authority paths remain active after each cutover phase

### Key Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Resource store becomes too generic, hiding domain invariants | Medium | High | Kind-specific codecs, typed stores, and projector adapters remain per-kind |
| Typed adapters drift from raw-store semantics | Medium | Medium | Thin adapters with contract tests alongside codec tests |
| Circular migration pressure between automation, bridges, bundles | Low | High | Migrate automation and bridges in parallel, bundles afterward |
| Projector failure semantics become ambiguous across domains | Medium | High | Full-snapshot idempotent reconcile, side-effect-free Build, atomic Apply |
| Protocol churn breaks extension fixtures and SDKs | Medium | Medium | Protocol, SDK, and fixture tests land in same phase as resource negotiation |
| Write-storm reconcile floods | Low | Medium | Single-flight per-kind scheduling with bounded coalescing |

## Scope

### In-Scope

- **Task 01**: Raw persistence kernel — CRUD, CAS, scope validation, source snapshots, authority stamping
- **Task 02**: Typed codecs, stores, projector adapters — typed boundary enforcement, raw JSON confinement
- **Task 03**: Reconcile driver — single-flight, coalescing, topology ordering, degraded-circuit, shutdown
- **Task 04**: Extension surface registry and resource grant config — surface legality, grant computation, operator policy
- **Task 05**: Extension resource protocol and SDK — handshake grants, nonce, snapshot, same-source reads
- **Task 06**: UDS resource CRUD APIs — operator CRUD, error mapping, HTTP gating
- **Task 07**: Hook binding migration — taxonomy-driven dispatch, tool.*/permission.* wiring, atomic swap
- **Task 08**: Tool and MCP server migration — static/dynamic publication, provide_tools removal
- **Task 09**: Agent and skill migration — definition cutover, reference resolution, provenance preservation
- **Task 10**: Automation definition migration — desired-state/operational-state split, projector safety
- **Task 11**: Bridge instance migration — external-state projector, runtime convergence, operational API preservation
- **Task 12**: Bundle activation fan-out — owner-indexed composition, allowlisted kinds, cycle-free reconcile

### Out-of-Scope

- HTTP `/api/resources` mutation routes (gated on operator auth middleware not yet present)
- Cross-source resource reads (denied in v1)
- Watch/event-stream resource notification (not in v1)
- Web UI resource management surfaces
- Extension direct `resources/put` and `resources/delete` (absent in v1)
- Performance benchmarking beyond basic write-storm coalescing validation

## Test Strategy

### Approach

Testing follows a layered strategy aligned with the TechSpec's development sequencing:

1. **Unit tests**: Per-package coverage for every codec, store, projector, and authority rule (>=80% coverage per package)
2. **Integration tests**: Real SQLite via `t.TempDir()`, real projectors, full-snapshot reconcile paths, extension subprocess fixtures
3. **Contract tests**: Typed boundary verification — no raw JSON leakage into domain code
4. **Smoke tests**: Critical-path validation of boot rebuild, CRUD, and family projection
5. **Security tests**: Authority boundary, scope enforcement, cross-source denial, nonce validation

### Test Execution Order

1. **Foundation layer** (Tasks 01-03): Persistence kernel + typed boundary + reconcile driver
2. **Extension protocol** (Tasks 04-05): Surface registry + grant config + protocol/SDK
3. **Tranche 1** (Tasks 06-08): UDS CRUD + hooks + tools/MCP
4. **Tranche 2** (Tasks 09-11): Agents/skills + automation + bridges
5. **Tranche 3** (Task 12): Bundle activation fan-out

## Environment Requirements

| Component | Requirement |
|-----------|-------------|
| Go version | 1.22+ (as specified in go.mod) |
| SQLite | Via go-sqlite3 CGO binding |
| Test flags | `-race` flag required on all runs |
| Coverage tool | `go test -cover` |
| Lint | golangci-lint (zero tolerance) |
| Build gate | `make verify` (fmt -> lint -> test -> build) |
| Node.js | For TypeScript SDK test execution (`bun run test`) |

## Entry Criteria

- [ ] All 12 tasks marked as completed in `_tasks.md`
- [ ] `make verify` passes on the ext-refac branch
- [ ] No unresolved merge conflicts
- [ ] All new packages have >=80% test coverage
- [ ] TypeScript SDK tests pass (`bun run test` in `sdk/typescript`)

## Exit Criteria

- [ ] All P0 test cases pass (100% required)
- [ ] All P1 test cases pass (>=90% required)
- [ ] No Critical or High severity bugs remain open
- [ ] `make verify` passes clean
- [ ] Coverage >=80% on every new or modified package
- [ ] No legacy authority path remains active for any migrated family

## Test Case Summary

| Type | Count | Priority Distribution |
|------|-------|-----------------------|
| Functional (TC-FUNC-*) | 30 | P0: 12, P1: 12, P2: 6 |
| Integration (TC-INT-*) | 18 | P0: 8, P1: 7, P2: 3 |
| Security (TC-SEC-*) | 10 | P0: 6, P1: 4 |
| Smoke (SMOKE-*) | 8 | P0: 8 |
| **Total** | **66** | **P0: 34, P1: 23, P2: 9** |

## Timeline and Deliverables

| Phase | Deliverable | Test Focus |
|-------|-------------|------------|
| Foundation | SMOKE-001 to SMOKE-003, TC-FUNC-001 to TC-FUNC-010 | Raw store, typed boundary, reconcile driver |
| Extension Protocol | TC-FUNC-011 to TC-FUNC-014, TC-SEC-001 to TC-SEC-004 | Grants, nonce, snapshot, same-source isolation |
| Tranche 1 | TC-FUNC-015 to TC-FUNC-020, TC-INT-001 to TC-INT-008 | UDS CRUD, hook migration, tool/MCP migration |
| Tranche 2 | TC-FUNC-021 to TC-FUNC-026, TC-INT-009 to TC-INT-014 | Agent/skill, automation, bridge migration |
| Tranche 3 | TC-FUNC-027 to TC-FUNC-030, TC-INT-015 to TC-INT-018 | Bundle activation fan-out, owner cleanup |
| Full Regression | All SMOKE + TC-SEC + selected TC-FUNC/TC-INT | End-to-end cutover validation |

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Incomplete legacy removal leaves dual authority | Medium | Critical | TC-FUNC tests explicitly verify no legacy path remains authoritative |
| Projector failure corrupts live runtime state | Low | Critical | TC-INT tests verify atomic swap and rollback preservation |
| Extension snapshot overwrites foreign-source records | Low | Critical | TC-SEC tests verify snapshot ownership conflict rejection |
| Reconcile driver leaks goroutines on shutdown | Low | High | TC-FUNC tests verify Close() drains within deadline |
| Bundle activation creates dependency cycle | Low | High | TC-INT tests verify cycle-free fan-out through store writes |
| Cross-source information leakage | Low | Critical | TC-SEC tests verify same-source read filtering |
