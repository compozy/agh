# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement ACP/session lifecycle hardening so provider startup/protocol/prompt/process failures are typed, persisted, observable, redacted, and recoverable through CLI/API/SSE/read models.
- Complete downstream `web/` generated/client type and `packages/site` documentation changes caused by the contract additions.

## Important Decisions

- Canonical lifecycle failure type lives in `internal/store` as `FailureKind` plus `SessionFailure`.
- ACP and session code classify failures at source with wrapped errors; read paths project stored failure diagnostics rather than parsing error strings.
- Startup failures after a session directory/event store is opened preserve stopped metadata instead of deleting the session, so operators can inspect the failure later.
- Crash bundles are JSON under `${AGH_HOME}/logs/crash-bundles`, bounded/redacted before write, owner-only permissions, and referenced from `SessionFailure.CrashBundlePath`.
- Observe health extends Task 02 typed health with `health.failures` and `health.agent_probes`; historical failures degrade `health.failures.status`, while probe failures degrade top-level health.
- Probe results parse raw commands internally but expose redacted bounded command/error text.

## Learnings

- The web app derives daemon/session DTOs from generated OpenAPI, but `web/src/systems/session/types.ts` keeps a manual `AgentEventPayload` for AI SDK message data and needed an explicit optional `failure`.
- Health payload changes require updating home/daemon fixtures and type tests in addition to generated `web/src/generated/agh-openapi.d.ts` and `sdk/typescript/src/generated/contracts.ts`.
- Existing site docs already have focused lifecycle, event streaming, and `agh observe health` pages; the Task 03 docs fit there without adding a new docs route.

## Files / Surfaces

- Backend: `internal/store/*`, `internal/store/globaldb/*`, `internal/acp/*`, `internal/session/*`, `internal/observe/*`, `internal/api/contract/*`, `internal/api/core/*`, `internal/cli/*`, `internal/daemon/*`, `internal/transcript/*`, `internal/diagnostics/*`.
- Generated/client: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`.
- Web: `web/src/systems/session/types.ts`, `web/src/systems/daemon/*`, `web/src/hooks/routes/use-home-page.test.tsx`.
- Site docs: `packages/site/content/runtime/core/sessions/lifecycle.mdx`, `packages/site/content/runtime/core/sessions/events.mdx`, `packages/site/content/runtime/cli-reference/observe/health.mdx`.

## Errors / Corrections

- Initial probe implementation exposed configured command text verbatim; corrected to expose redacted bounded command/error while still parsing the raw command internally.
- TypeScript health fixtures initially missed required `health.failures`; corrected fixtures and removed readonly fixture incompatibility in `use-daemon-health.test.ts`.
- Full `make verify` exposed a real timeout lifecycle bug: `timeout_cancel_grace` was reused as the entire forced-stop operation deadline after cooperative prompt cancel, so slow race-enabled metadata/hooks could expire the stop context before the driver stop call. Corrected production code to keep `timeout_cancel_grace` as cancel grace only and use a separate bounded forced-stop deadline.

## Ready for Next Run

- Targeted Go packages pass: `go test ./internal/acp ./internal/session ./internal/store ./internal/store/globaldb ./internal/observe ./internal/api/core ./internal/api/contract ./internal/cli ./internal/daemon`.
- Generated contracts pass `make codegen-check`.
- Impacted web typecheck/tests pass: `bun run --cwd web typecheck:raw` and `bun run --cwd web test:raw src/systems/daemon/types.test.ts src/systems/daemon/adapters/daemon-api.test.ts src/systems/daemon/hooks/use-daemon-health.test.ts src/hooks/routes/use-home-page.test.tsx`.
- Timeout regression checks pass: `go test ./internal/session -run TestPromptActivitySupervisorTimeoutCancelsThenStopsSession -count=20 -race` and `go test ./internal/session -run TestPromptActivity -count=1 -race`.
- Full gate passes: `make verify` exited 0 after all changes; Go test phase reported `DONE 5790 tests`, session package passed after the forced-stop deadline fix, and package boundaries were respected.
- Task tracking has been updated: `task_03.md` subtasks/tests checked and `_tasks.md` Task 03 status set to `completed`.
- Local commit created: `b01f4963 feat: harden acp session lifecycle`.
- Post-commit full gate passes: `make verify` exited 0; output included `Found 0 warnings and 0 errors`, `0 issues.`, `DONE 5790 tests in 5.640s`, and `OK: all package boundaries respected`.
- Remaining: none for Task 03.
