# Extension Registry — Regression Suite

## Suite Overview

This regression suite validates that the extension registry feature does not break existing functionality and that all registry operations work correctly across updates.

---

## Suite Tiers

### Smoke Suite (15-30 min) — Run Per Build

| Order | Test ID | Description | Priority |
|-------|---------|-------------|----------|
| 1 | SMOKE-001 | `make verify` passes (fmt, lint, test, build) | P0 |
| 2 | SMOKE-002 | `agh extension search` returns results from ClawHub | P0 |
| 3 | SMOKE-003 | `agh extension install` from GitHub completes | P0 |
| 4 | SMOKE-004 | `agh extension remove` cleans filesystem and DB | P0 |
| 5 | SMOKE-005 | `agh skill search` works after migration | P0 |

**Gate:** If any smoke test fails, stop regression. Investigate before proceeding.

### Targeted Regression (30-60 min) — Per Change

| Order | Test ID | Description | Priority |
|-------|---------|-------------|----------|
| 1 | TC-REG-001 | Existing skill install flow unchanged after migration | P1 |
| 2 | TC-REG-002 | Existing skill search flow unchanged after migration | P1 |
| 3 | TC-REG-003 | Existing extension local install still works | P1 |
| 4 | TC-REG-004 | Database migrations preserve existing extension data | P1 |
| 5 | TC-REG-005 | Config loading with new marketplace fields backward-compatible | P1 |
| 6 | TC-REG-006 | Extension capability ceiling enforced for marketplace installs | P1 |

### Full Regression (2-4 hours) — Weekly / Pre-Release

All smoke + targeted tests, plus:

| Order | Test ID | Description | Priority |
|-------|---------|-------------|----------|
| 7 | TC-FUNC-001–008 | Full MultiRegistry functional suite | P0-P1 |
| 8 | TC-FUNC-009–016 | Full Installer functional suite | P0-P1 |
| 9 | TC-FUNC-017–026 | Full Extension CLI suite | P0-P1 |
| 10 | TC-FUNC-027–030 | Skill CLI migration suite | P1 |
| 11 | TC-INT-001–010 | All integration tests | P0-P1 |
| 12 | TC-SEC-001–008 | Full security suite | P0 |

---

## Execution Strategy

### Order

1. **Smoke** — If any fail, STOP. Fix before continuing.
2. **P0 Functional + Security** — Critical path validation.
3. **P1 Functional + Integration** — Feature completeness.
4. **P2 Edge cases** — Boundary conditions and rare scenarios.
5. **Exploratory** — Ad-hoc testing of unusual combinations.

### Pass / Fail Criteria

| Verdict | Criteria |
|---------|----------|
| **PASS** | All P0 pass, 90%+ P1 pass, no Critical/High bugs open |
| **FAIL** | Any P0 fails, Critical bug discovered, security vulnerability, data loss |
| **CONDITIONAL** | P1 failures with documented workarounds and fix plan |

---

## Maintenance

- **Add new regression tests** when: bugs are fixed, new registry sources are added, CLI flags change.
- **Retire tests** when: features are removed, tests become redundant with automated coverage.
- **Review quarterly**: Remove flaky tests, update expected outputs, align with current API contracts.
