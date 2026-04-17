# Regression Suite: Shared Extensibility Resource Runtime

## Suite Overview

This regression suite covers the complete extensibility-parity migration. It is organized into four tiers for execution at different frequencies and confidence levels.

## Suite Tiers

### Smoke Suite (15-30 min, per-build)

| ID | Test Case | Priority | Package |
|----|-----------|----------|---------|
| SMOKE-001 | Resource CRUD round-trip | P0 | internal/resources |
| SMOKE-002 | Typed codec encode/decode round-trip | P0 | internal/resources |
| SMOKE-003 | Reconcile driver boot ordering | P0 | internal/resources |
| SMOKE-004 | Extension handshake includes resource grants | P0 | internal/extension |
| SMOKE-005 | UDS resource PUT/GET/DELETE | P0 | internal/api/udsapi |
| SMOKE-006 | Hook binding fires through resource-backed dispatch | P0 | internal/hooks |
| SMOKE-007 | Tool publication via resource snapshot | P0 | internal/tools |
| SMOKE-008 | Bundle activation creates owned resources | P0 | internal/bundles |

**Pass criteria**: ALL must pass. If any smoke test fails, stop execution.

### Targeted Suite (30-60 min, per-change)

Run the smoke suite plus test cases directly affected by the changed package:

| Changed Package | Additional Test Cases |
|----------------|----------------------|
| internal/resources | TC-FUNC-001 to TC-FUNC-010, TC-SEC-001 to TC-SEC-003 |
| internal/extension | TC-FUNC-011 to TC-FUNC-014, TC-SEC-004 to TC-SEC-006 |
| internal/api/udsapi | TC-FUNC-015 to TC-FUNC-016, TC-INT-001 to TC-INT-003 |
| internal/hooks | TC-FUNC-017 to TC-FUNC-018, TC-INT-004 to TC-INT-005 |
| internal/tools, internal/config/mcpjson | TC-FUNC-019 to TC-FUNC-020, TC-INT-006 to TC-INT-008 |
| internal/skills, internal/config/agent | TC-FUNC-021 to TC-FUNC-022, TC-INT-009 to TC-INT-010 |
| internal/automation | TC-FUNC-023 to TC-FUNC-024, TC-INT-011 to TC-INT-012 |
| internal/bridges | TC-FUNC-025 to TC-FUNC-026, TC-INT-013 to TC-INT-014 |
| internal/bundles | TC-FUNC-027 to TC-FUNC-030, TC-INT-015 to TC-INT-018 |

### Full Suite (2-4 hours, weekly/release)

All 66 test cases: SMOKE-* + TC-FUNC-* + TC-INT-* + TC-SEC-*

### Sanity Suite (10-15 min, after hotfix)

| ID | Test Case | Rationale |
|----|-----------|-----------|
| SMOKE-001 | Resource CRUD round-trip | Core persistence |
| SMOKE-005 | UDS resource PUT/GET/DELETE | API layer |
| TC-SEC-001 | Cross-source read denial | Security boundary |
| TC-SEC-005 | Stale nonce rejection | Session security |
| TC-FUNC-003 | CAS conflict on stale version | Data integrity |

## Execution Order

```
1. Smoke Suite (if ANY fails → STOP, report)
2. P0 tests (sorted by: TC-SEC → TC-FUNC → TC-INT)
3. P1 tests (sorted by: TC-SEC → TC-FUNC → TC-INT)
4. P2 tests
5. Exploratory testing (if time permits)
```

## Pass/Fail Criteria

| Verdict | Criteria |
|---------|----------|
| **PASS** | All P0 pass, >=90% P1 pass, no Critical/High bugs open, `make verify` clean |
| **FAIL** | Any P0 fails, any Critical bug discovered, any security vulnerability, any data loss scenario |
| **CONDITIONAL** | P1 failures with documented workarounds, fix plan in place, no security/data issues |

## Execution Commands

### Smoke

```bash
make verify
go test -race -count=1 ./internal/resources/... ./internal/extension/... ./internal/api/udsapi/... ./internal/hooks/... ./internal/tools/... ./internal/bundles/...
```

### Full with coverage

```bash
go test -race -cover -count=1 ./internal/resources/... ./internal/extension/... ./internal/api/... ./internal/hooks/... ./internal/tools/... ./internal/skills/... ./internal/automation/... ./internal/bridges/... ./internal/bundles/... ./internal/config/... ./internal/daemon/...
```

### Integration only

```bash
go test -race -tags integration -count=1 ./internal/...
```

### SDK

```bash
cd sdk/typescript && bun run test
```

## Coverage Gates

| Package | Minimum |
|---------|---------|
| internal/resources | 80% |
| internal/extension | 80% |
| internal/extension/surfaces | 80% |
| internal/config | 80% |
| internal/hooks | 80% |
| internal/tools | 80% |
| internal/skills | 80% |
| internal/automation | 80% |
| internal/bridges | 80% |
| internal/bundles | 80% |
| internal/api/udsapi | 80% |
| internal/api/core | 80% |
| internal/subprocess | 80% |

## Regression Triggers

Run the full suite when:

- Any resource runtime schema change
- Any codec or projector interface change
- Any reconcile driver behavior change
- Any extension protocol or handshake change
- Any authority, scope, or grant computation change
- Pre-release cut
- After rebasing onto main
