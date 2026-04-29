# Tool Registry — Targeted Regression Suite

- **Lane:** Targeted (30-60 min)
- **Frequency:** Per change touching `internal/tools`, `internal/extension`, `internal/mcp`, `internal/api/*`, `internal/cli`, `web/src/systems/tools/**`, or `packages/site/content/runtime/**`
- **Coverage:** All P0 cases from smoke + P1 cases for the surfaces touched
- **Pass criteria:** 100% P0 + ≥ 90% P1

## Inclusion Map by Touched Surface

| Touched surface | Required P1 cases (in addition to smoke P0) |
|-----------------|---------------------------------------------|
| `internal/tools/registry*` / `policy*` / `projection*` | TC-FUNC-002..017, TC-INT-002..006, TC-PERF-001..003 |
| `internal/tools/dispatch*` / `result*` | TC-FUNC-027..033, TC-INT-005, TC-PERF-001 |
| `internal/tools/builtin_*` | TC-FUNC-018..026 |
| `internal/extension` + `sdk/typescript` | TC-INT-007/008, TC-FUNC-040..042, TC-SEC-014 |
| `sdk/go` | TC-INT-009/010, TC-FUNC-043..044 |
| `internal/mcp` (call-through) | TC-INT-011/012, TC-FUNC-035..039, TC-SEC-005..008 |
| `internal/mcp` (hosted) | TC-INT-013/014/015, TC-FUNC-045..049, TC-SEC-009/010 |
| `internal/api/contract` + `internal/api/core` + `internal/api/httpapi` + `internal/api/udsapi` | TC-INT-001/004, TC-FUNC-050..052, TC-SEC-011..013 |
| `internal/cli` | TC-FUNC-053..056 |
| `web/src/systems/tools/**` | TC-UI-001..006 |
| `packages/site` | TC-FUNC-057..058 |

## Execution Order

1. Run all smoke (P0) cases first.
2. Run P1 cases in the order listed above, negative-first within each group.
3. Stop on any P0 failure; otherwise continue and tally P1.

## Pass / Fail / Conditional

- **PASS:** All P0 pass and ≥ 90% of P1 pass.
- **FAIL:** Any P0 fails OR < 90% of P1 pass OR any redaction sentinel leak.
- **CONDITIONAL:** P1 failures with documented workaround, fix plan, and `BUG-NNN.md` open with severity ≤ High.

## Outputs

- `qa/logs/targeted/<TC-ID>.log`
- `qa/traces/targeted/<TC-ID>/` (Playwright traces for UI cases)
- `qa/screenshots/targeted/<TC-UI-ID>/<viewport>/<state>.png`
- `qa/issues/BUG-NNN.md` for any reproduced defect

## Out of Scope

- Full multi-extension stress.
- Full MCP transport matrix (covered in `full-regression`).
- Long-running concurrency stress beyond TC-PERF-001..003.
