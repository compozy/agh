# Agent Authored Context Regression Suite

## Execution Order

1. Smoke readiness first.
2. P0 behavioral journeys.
3. P0 technical regressions.
4. P1 parity and generated-consumer checks.
5. Final verification gate and post-gate reruns.

## Smoke Suite

| ID | Priority | Purpose | Evidence |
|----|----------|---------|----------|
| SMOKE-001 | P0 | Prove the repo and isolated runtime can start deeper QA | Contract discovery, bootstrap manifest, daemon status, generated contract check |

Smoke is readiness-only. Passing smoke does not prove Agent Soul/Heartbeat behavior.

## P0 Behavioral Journeys

| ID | Priority | Journey | Must Pass |
|----|----------|---------|-----------|
| TC-SCEN-001 | P0 | Managed Soul authoring, inspect, stale-CAS disruption, history, HTTP read parity | Yes |
| TC-SCEN-002 | P0 | Heartbeat policy authoring, status with session health, wake decision/audit boundary | Yes |
| TC-SCEN-003 | P0 | Soul-influenced session context and provider-backed or explicitly blocked agent output | Yes |
| TC-REG-001 | P0 | Invalid authored content fails closed without Soul/Heartbeat cross-feature bleed | Yes |

## P1 Regression Checks

| ID | Priority | Regression | Must Pass |
|----|----------|------------|-----------|
| TC-REG-002 | P1 | HTTP/CLI CAS contract and unsupported `If-Match` rejection | Yes unless API daemon is blocked |
| TC-REG-003 | P1 | Generated Web/SDK/docs consumers remain truthful and no fake Web editor exists | Yes |
| TC-REG-004 | P1 | Revision rollback/delete, daemon restart recovery, and redacted output edge cases | Yes |

## Pass/Fail Criteria

PASS:

- All P0 cases pass with evidence.
- P1 cases pass or are blocked by exact environment prerequisites.
- Live provider/LLM validation is executed or the missing credential/tool boundary is recorded.
- No Critical or High bug remains open.
- `make verify` passes after the last code change.

FAIL:

- Any P0 journey fails without a fixed root cause.
- A mutation path bypasses managed authoring.
- Heartbeat wake creates task ownership, task runs, or lease side effects.
- Session health is unavailable or unreadable for created sessions without a documented environment blocker.
- Final `make verify` fails.

## Evidence Layout

- Command transcripts: `qa/evidence/*.log`
- JSON responses: `qa/evidence/*.json`
- Runtime/bootstrap data: `qa/bootstrap-manifest.json`, `qa/bootstrap.env`
- Bugs: `qa/issues/BUG-*.md`
- Final report: `qa/verification-report.md`
