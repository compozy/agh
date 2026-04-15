# Regression Suite: Extension Bundles

## Smoke Suite (15 min, run per-build)

| Order | Test Case | Critical Path |
|-------|-----------|---------------|
| 1 | SMOKE-005 | make verify passes |
| 2 | SMOKE-001 | Full activation lifecycle |
| 3 | SMOKE-002 | Primary channel binding lifecycle |
| 4 | SMOKE-003 | Extension lifecycle guard |
| 5 | SMOKE-004 | All resource types materialized |

**Pass criteria:** All 5 pass. If any fail, STOP — do not proceed to targeted suite.

---

## Targeted Suite (30 min, run per-change to bundles/)

### P0 — Must pass

| Test Case | Area |
|-----------|------|
| TC-FUNC-001 | Catalog discovery |
| TC-FUNC-002 | Bundle spec validation |
| TC-FUNC-004 | Activation with persistence |
| TC-FUNC-006 | Deactivation and cleanup |
| TC-FUNC-009 | Primary channel binding |
| TC-FUNC-010 | Channel conflict detection |
| TC-FUNC-012 | Rollback on failed reconciliation |
| TC-INT-001 | HTTP activation endpoint |
| TC-INT-004 | HTTP deactivation endpoint |
| TC-INT-006 | Extension disable guard |
| TC-INT-010 | Error code mapping |
| TC-SEC-001 | Webhook rejection |
| TC-SEC-002 | SQL injection prevention |

### P1 — Must pass for release

| Test Case | Area |
|-----------|------|
| TC-FUNC-003 | Preview without persistence |
| TC-FUNC-005 | Update activation binding |
| TC-FUNC-007 | Global scope propagation |
| TC-FUNC-008 | Workspace scope propagation |
| TC-FUNC-011 | Network settings completeness |
| TC-FUNC-013 | Update/deactivate rollback |
| TC-FUNC-014 | Stable ID determinism |
| TC-FUNC-015 | Bridge platform resolution |
| TC-INT-002 | HTTP catalog endpoint |
| TC-INT-003 | HTTP update endpoint |
| TC-INT-005 | HTTP network settings endpoint |
| TC-SEC-003 | Path traversal prevention |

---

## Full Suite (1 hour, run weekly/pre-release)

All TC-FUNC-*, TC-INT-*, TC-SEC-*, and SMOKE-* test cases.

---

## Execution Order

1. **Smoke** — if any fail, stop
2. **P0 functional** → P0 integration → P0 security
3. **P1 functional** → P1 integration → P1 security
4. **Exploratory** — ad-hoc testing around new/changed code

## Pass/Fail Criteria

| Result | Criteria |
|--------|----------|
| **PASS** | All P0 pass, 90%+ P1 pass, no critical/high bugs open |
| **FAIL** | Any P0 fails, any critical bug discovered, data loss scenario |
| **CONDITIONAL** | P1 failures with documented workarounds, fix plan in place |

## Current Test Coverage Assessment

| Package | Existing Tests | Estimated Coverage | Status |
|---------|---------------|-------------------|--------|
| `internal/bundles` | 2 tests (service_test.go) | ~15% | BELOW THRESHOLD |
| `internal/extension` (bundle.go) | 0 tests | 0% | CRITICAL GAP |
| `internal/store/globaldb` (bundles) | 0 tests | 0% | CRITICAL GAP |
| `internal/api/core` (bundles) | 0 handler tests | 0% | CRITICAL GAP |
| `internal/api/httpapi` (bundle routes) | 1 route check | ~5% | CRITICAL GAP |
| `internal/api/udsapi` (bundle routes) | 1 route check | ~5% | CRITICAL GAP |
| `internal/extension/registry` (guards) | 1 test | ~30% | BELOW THRESHOLD |

**Overall bundle feature coverage: ~8%** (estimated)
**Required: 80% per package**
