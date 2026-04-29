# Tool Registry — Full Regression Suite

- **Lane:** Full (2-4 hours)
- **Frequency:** Weekly and pre-release
- **Coverage:** All P0 + all P1 + ≥ 80% of P2 + sampling of P3
- **Pass criteria:** 100% P0 + 100% P1 + ≥ 95% P2 + no critical defect open

## Execution Order

1. Smoke (`smoke-regression.md`) end-to-end.
2. Targeted P1 cases for all surfaces (treat the full suite as if every surface was touched).
3. P2 edge / boundary / error-mapping cases:
   - TC-FUNC-005, TC-FUNC-009..010, TC-FUNC-014..015, TC-FUNC-017
   - TC-FUNC-019..020, TC-FUNC-024..026, TC-FUNC-029
   - TC-FUNC-034, TC-FUNC-036..038, TC-FUNC-041..042, TC-FUNC-046..049
   - TC-INT-002..004, TC-INT-006, TC-INT-008, TC-INT-010, TC-INT-014..015
   - TC-PERF-001..003
   - TC-UI-002..006
   - TC-FUNC-057..058
4. Security/redaction (`security-redaction-regression.md`) end-to-end.
5. P3 sampling: docs copy review (canonical IDs in examples), web visual spot checks across 375/768/1280 viewports.
6. `make verify`, `make test-e2e-runtime`, `make test-e2e-web`, and `make codegen-check`.

## Coverage Matrix by Concern

| Concern | Cases |
|---------|-------|
| Canonical `ToolID` & collisions | TC-FUNC-001..006, TC-FUNC-016..017 |
| Config lifecycle | TC-FUNC-007..015 |
| Registry indexing & toolsets | TC-FUNC-007..010, TC-FUNC-024 |
| Effective policy & ACP ceiling | TC-FUNC-011..015, TC-FUNC-027..030 |
| Dispatch & hooks & budgets | TC-FUNC-027..034, TC-INT-005..006 |
| Native tools | TC-FUNC-018..026, TC-INT-005 |
| TypeScript extension-host | TC-FUNC-040..042, TC-INT-007..008 |
| Go SDK extension-host | TC-FUNC-043..044, TC-INT-009..010 |
| MCP call-through | TC-FUNC-035..039, TC-INT-011..012 |
| Hosted AGH MCP | TC-FUNC-045..049, TC-INT-013..015 |
| HTTP/UDS APIs | TC-FUNC-050..052, TC-INT-001..004 |
| CLI | TC-FUNC-053..056 |
| Web diagnostics | TC-UI-001..006 |
| Docs & generated refs | TC-FUNC-057..058 |
| Security & redaction | TC-SEC-001..014 |
| Concurrency / perf | TC-PERF-001..003 |

## Pass / Fail / Conditional

- **PASS:** Coverage targets met, no critical defect open, `make verify` clean.
- **FAIL:** Any P0/P1 fails OR < 95% P2 pass OR critical defect open OR sentinel leak.
- **CONDITIONAL:** P2 misses with documented workaround and `BUG-NNN.md` open at severity ≤ Medium.

## Outputs

- `qa/logs/full/<TC-ID>.log`
- `qa/traces/full/<TC-ID>/`
- `qa/screenshots/full/<TC-UI-ID>/<viewport>/<state>.png`
- `qa/issues/BUG-NNN.md` per defect
- Final `qa/verification-report.md` written by Task 16
