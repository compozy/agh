# Tools Refac Real-Scenario QA Verification Report

**qa-output-path:** `.compozy/tasks/tools-refac`
**Status:** PASS
**Created:** 2026-04-30
**Last Updated:** 2026-04-30
**Scenario:** `tools-refac-real-scenario-20260430-074748-514234`

## QA Bootstrap

```qa-bootstrap
manifest: /Users/pedronauck/dev/qa-labs/agh-tools-refac-real-scenario-20260430-074748-514234-lab/qa-artifacts/qa/bootstrap-manifest.json
repo_manifest_copy: .compozy/tasks/tools-refac/qa/bootstrap-manifest.json
lab_root: /Users/pedronauck/dev/qa-labs/agh-tools-refac-real-scenario-20260430-074748-514234-lab
runtime_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime
base_url: http://127.0.0.1:57781
uds_path: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime/aghd.sock
tmux_bridge_socket: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime/tmux-bridge.sock
provider_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/provider
provider_codex_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/provider/.codex
web_api_proxy_target: http://127.0.0.1:57781
```

The lab was fresh, not reused. `agh-qa-bootstrap` allocated short runtime/provider homes because the direct lab-local UDS paths would exceed the macOS Unix socket path limit.

## Scope Executed

Task 13 consumed the saved QA dossier under:

- `.compozy/tasks/tools-refac/qa/test-plans/`
- `.compozy/tasks/tools-refac/qa/test-cases/`

Execution covered the required real daemon-served surfaces: CLI, HTTP, UDS, session tool projections, hosted MCP, built-in tool calls, autonomy lease flows, policy/approval denial, config mutation boundaries, hooks, automation, extensions, MCP auth status, generated contract checks, site/docs build, and runtime E2E.

## Scenario Evidence

| Lane | Result | Evidence |
|---|---:|---|
| Bootstrap and daemon coordinates | PASS | `qa/bootstrap-manifest.json`, `qa/bootstrap.env`, `qa/logs/bootstrap/daemon-status-cli.json`, `qa/logs/bootstrap/daemon-status-http.json`, `qa/logs/bootstrap/daemon-status-uds.json` |
| CLI/HTTP/UDS tool discovery parity | PASS | `qa/logs/live-runtime/http-tools-list.json`, `qa/logs/live-runtime/uds-tools-list.json`, `qa/logs/live-runtime/http-session-tools-list.json`, `qa/logs/live-runtime/uds-session-tools-list.json`, `qa/logs/live-runtime/final-http-session-tools.json`, `qa/logs/live-runtime/final-uds-session-tools.json` |
| Tool search/info/invoke parity | PASS | `qa/logs/live-runtime/http-tool-invoke-search.json`, `qa/logs/live-runtime/uds-tool-invoke-search.json`, `qa/logs/live-runtime/tool-info-search-cli.json`, `qa/logs/live-runtime/tool-invoke-search-cli.json` |
| Hosted MCP transport | PASS | `qa/logs/live-runtime/final-hosted-mcp-transcript.json`, `qa/logs/live-runtime/final-hosted-mcp-stderr.log` |
| Autonomy hard cut and lease flow | PASS | `qa/logs/live-runtime/task-next-session-cli-default-channel.json`, `qa/logs/live-runtime/task-heartbeat-session-cli-default-channel.json`, `qa/logs/live-runtime/task-complete-session-cli-default-channel.json`, `qa/logs/live-runtime/task-get-default-channel-after-complete.json`, `qa/logs/live-runtime/task-complete-session-cli-raw-token-denied.*` |
| Config mutation and redaction boundary | PASS | `qa/logs/live-runtime/tool-config-set-automation.json`, `qa/logs/live-runtime/config-get-automation-cli.json`, `qa/logs/live-runtime/tool-config-set-secret-denied.*`, `qa/logs/live-runtime/mcp-auth-config-secret-env-redacted.json` |
| Hooks, automation, extensions | PASS | `qa/logs/live-runtime/tool-info-hooks-list.json`, `qa/logs/live-runtime/tool-invoke-hooks-list.json`, `qa/logs/live-runtime/tool-info-automation-jobs-list-after-fix.json`, `qa/logs/live-runtime/tool-invoke-automation-jobs-list-after-fix.json`, `qa/logs/live-runtime/tool-info-extensions-list-after-fix.json`, `qa/logs/live-runtime/tool-invoke-extensions-list-after-fix.json` |
| MCP auth status | PASS | `qa/logs/live-runtime/mcp-auth-status-cli.json`, `qa/logs/live-runtime/mcp-auth-status-tool-info-after-fix.json`, `qa/logs/live-runtime/mcp-auth-status-tool-after-fix.json` |
| Runtime E2E | PASS | `qa/logs/final-test-e2e-runtime.log` |
| Codegen and generated contract | PASS | `qa/logs/final-codegen-check.log`; final `make verify` also ran codegen-check through workspace typecheck |
| Site/docs build | PASS | `qa/logs/final-site-build.log`; final `make verify` also ran web build/typecheck/test lanes |
| Focused coverage | PASS | `qa/logs/focused-go-coverage.log`: `internal/mcp` 80.6%, `internal/tools` 80.8% |
| Monorepo gate | PASS | `qa/logs/final-make-verify.log` |

## Defects Captured And Fixed

| ID | Status | Root Cause | Fix Evidence |
|---|---:|---|---|
| `BUG-001` | Fixed | Hosted MCP same-binary validation compared differently cased macOS paths as different executables. | `internal/mcp/hosted.go`, `internal/mcp/hosted_test.go`, `go test ./internal/mcp`, `qa/logs/live-runtime/hosted-mcp-transcript-after-fix.json` |
| `BUG-002` | Fixed | One auth-blocked remote MCP source made the whole registry unavailable, including builtin auth-status tools. | `internal/tools/mcp.go`, `internal/tools/mcp_test.go`, `go test ./internal/tools`, `qa/logs/live-runtime/mcp-auth-status-tool-after-fix.json` |
| `BUG-003` | Fixed | Automation native tools captured a nil manager before daemon automation boot completed. | `internal/daemon/native_tools.go`, `internal/daemon/native_automation_tools.go`, focused daemon tests, `qa/logs/live-runtime/tool-invoke-automation-jobs-list-after-fix.json` |
| `BUG-004` | Fixed | Runtime E2E tests counted hosted MCP `session_new` lifecycle diagnostics as prompt diagnostics. | `internal/testutil/acpmock/diagnostics.go`, daemon integration tests |
| `BUG-005` | Fixed | UDS/HTTP observe parity expected only one turn augmenter, while current runtime emits durable-memory and situation augmenters. | `internal/api/udsapi/transport_parity_integration_test.go` |
| `BUG-006` | Fixed | Runtime harness artifact test required exact unaugmented echo text even though the runtime now dispatches augmented context plus the user prompt. | `internal/testutil/e2e/runtime_harness_integration_test.go` |

Issue records are stored under `.compozy/tasks/tools-refac/qa/issues/BUG-001.md` through `BUG-006.md`.

## Verification Commands

| Command | Result | Evidence Summary |
|---|---:|---|
| `make test-e2e-runtime` | PASS | `internal/daemon`: 22 tests; `internal/api/httpapi`: 8 tests; `internal/api/udsapi`: 14 tests; `internal/testutil/e2e`: 6 tests |
| `go test ./internal/mcp` | PASS | Hosted MCP same-executable regression passed |
| `go test ./internal/tools` | PASS | MCP auth-blocked source isolation regression passed |
| `go test ./internal/daemon -run 'TestDaemonNativeAutomationTools|TestDaemonNativeTools'` | PASS | Native automation manager lookup passed |
| `go test -race -count=1 -tags integration ./internal/daemon -run 'TestDaemonE2EMemoryRecallUsesCatalogSynthesisWithoutMutatingStoredUserMessage|TestDaemonE2EMockAgentsRemainIsolated'` | PASS | Prompt diagnostics regressions passed |
| `go test -race -count=1 -tags integration ./internal/api/udsapi -run TestUDSTransportObserveHarnessLifecycleParityMatchesHTTP` | PASS | Observe parity regression passed |
| `go test -race -count=1 -tags integration ./internal/testutil/e2e -run TestStartRuntimeHarnessCapturesTranscriptAndEventsArtifacts` | PASS | Artifact assertion regression passed |
| `go test ./internal/mcp ./internal/tools -cover` | PASS | `internal/mcp` 80.6%, `internal/tools` 80.8% |
| `make codegen-check` | PASS | No generated contract drift |
| `make site-build` | PASS | Site build completed |
| `make verify` | PASS | `bun-lint`: 0 warnings/0 errors; `bun-test`: 258 files, 1845 tests; Go tests: 7097 tests; package boundaries respected |

## Notes

- `make verify` emitted the known Node warning that `NO_COLOR` is ignored when `FORCE_COLOR` is set. This was also present in baseline logs and did not produce lint/test/build failures.
- Web build emitted the existing Vite large chunk advisory. It did not fail the build.
- The live daemon evidence used the isolated QA runtime, not the default user AGH home.

## Final Verdict

PASS. The saved QA dossier was executed against a fresh isolated daemon-served runtime, six reproduced defects were recorded and fixed at root cause, focused coverage met the >=80% requirement for the touched tool packages, runtime E2E passed, downstream codegen/site checks passed, and the full monorepo `make verify` gate passed.
