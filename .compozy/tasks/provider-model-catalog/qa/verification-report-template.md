# Provider Model Catalog - Verification Report Template

> Task 13 must rename this file to `verification-report.md` and fill every section before reporting completion.

## Run Metadata

- **Date:** YYYY-MM-DD
- **Operator:** <name / agent>
- **Branch:** <git branch>
- **Commit:** <SHA>
- **Bootstrap manifest path:** `.compozy/tasks/provider-model-catalog/qa/lab/bootstrap-manifest.json`
- **Lab root:** <from manifest>
- **Runtime home (`AGH_HOME`):** <from manifest>
- **Daemon ports:** <http>, <uds-socket>, <metrics>
- **`AGH_WEB_API_PROXY_TARGET`:** <derived>
- **tmux-bridge socket:** <derived>

## Smoke Readiness

| Step | Command | Result | Notes |
|------|---------|--------|-------|
| 1 | `make build` | <pass/fail> | |
| 2 | `make codegen-check` | | |
| 3 | `make bun-typecheck && make bun-test` | | |
| 4 | Focused Go gates | | |
| 5 | `agh daemon start --foreground` | | |

## Test Case Results

| TC | Title | Priority | Result | Notes / BUG-IDs |
|----|-------|----------|--------|-----------------|
| TC-FUNC-001 | Provider config hard cut | P0 | | |
| TC-FUNC-002 | Curated validation rules | P1 | | |
| TC-FUNC-003 | Builtin source priority 10 | P1 | | |
| TC-FUNC-004 | Merge determinism | P0 | | |
| TC-FUNC-005 | `models.dev` TTL/disable/aliases | P1 | | |
| TC-FUNC-006 | Stale fallback | P1 | | |
| TC-FUNC-007 | Partial vs all-source failure | P0 | | |
| TC-FUNC-008 | Live provider timeout | P1 | | |
| TC-FUNC-009 | No ACP session calls from discovery | P1 | | |
| TC-FUNC-010 | ACP `set_config_option` precedence | P0 | | |
| TC-FUNC-011 | Extension manifest validation | P1 | | |
| TC-FUNC-012 | Extension capability denial | P1 | | |
| TC-FUNC-013 | Source error redaction | P0 | | |
| TC-FUNC-014 | Detached refresh deadline | P1 | | |
| TC-FUNC-015 | Codegen + docs drift | P1 | | |
| TC-INT-001 | Migration v23 fresh + reopen | P0 | | |
| TC-INT-002 | HTTP/UDS handler payloads | P0 | | |
| TC-INT-003 | Canonical JSON parity | P0 | | |
| TC-INT-004 | OpenAI projection HTTP-only | P0 | | |
| TC-INT-005 | Extension success/denial | P0 | | |
| TC-INT-006 | ACP SDK upgrade flows | P0 | | |
| TC-PERF-001 | Refresh concurrency + coalesce | P0 | | |
| TC-PERF-002 | Detached refresh + shutdown join | P0 | | |
| TC-SEC-001 | Secret redaction across surfaces | P0 | | |
| TC-SEC-002 | OpenAI auth + envelope | P0 | | |
| TC-UI-001 | Settings source status + refresh | P1 | | |
| TC-UI-002 | Manual entry + curated edit | P1 | | |
| TC-UI-003 | New session ACP override | P1 | | |
| TC-REG-001 | Hard-cut residue scan | P1 | | |
| TC-REG-002 | Generated docs + CLI sync | P1 | | |
| TC-SCEN-001 | Operator real journey | P0 | | |
| TC-SCEN-002 | Agent real journey | P0 | | |

## Verification Commands Executed

For each command record verbatim invocation, exit code, and duration. Attach full logs under `qa/lab/logs/`.

```bash
# Example
make codegen-check       # exit 0, 12s
make bun-test            # exit 0, 1m24s
go test -race ./internal/modelcatalog/...  # exit 0, 2m11s
make test-e2e-runtime    # exit 0, 4m02s
make test-e2e-web        # exit 0, 6m18s
make verify              # exit 0, 14m37s
```

## Filed Bugs

| BUG | Severity | TC | Status | Fix Commit |
|-----|----------|----|--------|------------|
| | | | | |

## Live-Provider Annex (Optional)

If `MODELCATALOG_LIVE=1` was set, document the real-provider boundary here:

- Provider exercised: <provider_id>
- Credential source: <bootstrap manifest field>
- Endpoints hit: `models.dev/api.json` (real), `<provider list endpoint>` (real)
- Result: <pass/fail + paragraph>

If not run, state explicitly: "Live-provider annex not executed; default run uses stub HTTP servers and fake subprocesses."

## Residual Risk

- <bullet list of unresolved findings + recommended follow-up tasks>

## Final Verification

- `make verify` exit code: <0 / non-zero>
- Duration: <minutes>
- Log path: `qa/lab/logs/make-verify.log`

## Sign-Off

- Reporter: <name>
- Date: <YYYY-MM-DD>
- Decision: PASS | FAIL | CONDITIONAL (with documented workaround)
