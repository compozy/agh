# Tool Registry — Smoke Regression Suite

- **Lane:** Smoke (15-30 min)
- **Frequency:** Daily and per-build
- **Coverage:** P0 critical safety/dispatch/redaction paths only
- **Stop rule:** If any smoke case fails, halt downstream regression lanes and open a `BUG-NNN.md`. Do not proceed to targeted/full/security suites.
- **Pass criteria:** 100% of cases listed below must pass.

## Execution Order

Negative-first: deny / unauthorized / conflicted / approval-required cases run before the happy path so silent-allow regressions surface first.

| # | Case ID | Title | Priority | Trace |
|---|---------|-------|----------|-------|
| 1 | TC-SEC-001 | `deny-all` blocks every executable backend at dispatch time | P0 | T03/T04, ADR-005, Inv 1, 4 |
| 2 | TC-SEC-009 | Hosted MCP rejects bind without UDS peer + AGH binary validation | P0 | T10, ADR-002, Inv 16, 21 |
| 3 | TC-SEC-005 | Remote MCP `Authorization` header never crosses `internal/tools` boundary | P0 | T09, ADR-010, Inv 12, 20 |
| 4 | TC-FUNC-016 | Canonical-ID collision keeps both tools operator-visible and session-hidden | P0 | T03/T07, ADR-007, Inv 7 |
| 5 | TC-FUNC-001 | Canonical `ToolID` validator accepts MVP examples and rejects forbidden forms | P0 | T01, ADR-007 |
| 6 | TC-INT-001 | `Registry.Call` is the only execution path; CLI/HTTP/UDS/hosted MCP all enter it | P0 | T04/T11/T10, Inv 1, 2 |
| 7 | TC-FUNC-021 | `agh__skill_view` returns real skills content with budget truncation metadata | P0 | T05, ADR-004 |
| 8 | TC-INT-013 | Hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools` | P0 | T10/T11, Inv 13 |
| 9 | TC-INT-007 | TypeScript extension publishes executable read-only tool through registry | P0 | T07, ADR-001/008 |
| 10 | TC-INT-009 | Go SDK extension publishes executable read-only tool through registry | P0 | T08, ADR-009 |
| 11 | TC-INT-011 | Local stdio MCP fixture lists and calls a tool through `MCPCallExecutor` | P0 | T09, ADR-010/011 |
| 12 | TC-FUNC-031 | Result limiter truncates oversized outputs identically across CLI/HTTP/UDS/MCP | P0 | T04, Inv 11 |
| 13 | TC-SEC-011 | CLI/HTTP/UDS approval token is single-use and bound to tool/session/workspace/input | P0 | T11/T12, ADR-005, Inv 27 |
| 14 | TC-INT-016 | `make verify` passes on fresh lab | P0 | All tasks |

## Pre-Conditions

- Fresh `AGH_HOME` from `agh-qa-bootstrap`.
- Daemon started with `[tools].enabled = true`, `[tools.hosted_mcp].enabled = true`, default `[tools.policy].external_default = "disabled"`.
- TypeScript and Go test extensions installed, `tool.provider` capability granted.
- Local stdio MCP fixture configured under `[mcp_servers.smoke_stdio]`.
- ACP test runtime available (`internal/testutil/acpmock`) for hosted MCP smoke.

## Pass / Fail / Conditional

- **PASS:** All 14 cases pass with no redaction sentinel detected.
- **FAIL:** Any P0 case fails, any sentinel leaked, or `make verify` non-zero.
- **CONDITIONAL:** Not allowed for the smoke lane.

## Outputs

For each case, append to `qa/logs/smoke/<TC-ID>.log`:

- Command(s) executed.
- Exit code.
- Redacted JSON payload.
- Timestamp.

Capture screenshots only for cases that intersect Task 13 surfaces (none of the 14 here are UI-only; all are backend/CLI/HTTP/UDS/MCP).
