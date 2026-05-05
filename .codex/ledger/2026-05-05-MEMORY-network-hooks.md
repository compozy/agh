# Goal (incl. success criteria):

- Implement `.compozy/tasks/network-threads/task_07.md`: network hook taxonomy/payloads, post-commit best-effort dispatch, daemon bridge wiring, aggregate status counters, low-cardinality metrics, structured log/audit field coverage, tests, tracking updates, clean `make verify`, and one local commit.
- Success criteria: all six required `network.*` hook events are cataloged and async-only; dispatch happens only after a committed non-duplicate conversation write; hook failures do not roll back network writes; status exposes aggregate counters and approved metric labels; touched package tests plus full `make verify` pass before completion and again around the commit.

# Constraints/Assumptions:

- Do not run destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- Existing dirty worktree entries are user/other-agent work; do not revert or stage them unless task-scoped and intentionally updated.
- Must use workflow memory before edits and before finish; update `.compozy/tasks/network-threads/memory/task_07.md` as task-local state changes.
- Required installed skills: `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`; Go/test/fix work also uses `golang-pro`, `testing-anti-patterns`, `no-workarounds`, and `systematic-debugging`.
- Requested AGH-specific skills (`agh-code-guidelines`, `agh-test-conventions`, `deadlock-finder-and-fixer`, `nats`) were not found in the visible skill roots; repo guidance and installed fallback skills apply.
- Network hooks are observation-only: no sync hooks, deny hooks, replay log, table tailer, or routing/persistence authority.

# Key decisions:

- Use `internal/network.Manager.writeConversationMessage` as the post-commit hook/stat call site because Task 06 routes outbound/inbound committed conversation writes through this helper.
- Dispatch only when `store.NetworkConversationStore.WriteConversationMessage` returns success and `Duplicate == false`.
- Keep high-cardinality identifiers in hook payloads and logs/audit fields, but not metric labels.
- Status counters remain runtime/manager-facing for Task 07; public API status payload changes are left to Task 08 unless needed to compile.

# State:

- complete_final_handoff_ready

# Done:

- Read workflow shared/current memory, relevant network ledgers, root/internal guidance, task_07, `_tasks.md`, `_techspec.md`, `_design.md`, and ADR-001/002/003.
- Loaded required and domain skills; recorded unavailable AGH-specific skills.
- Captured pre-change signal: code had no `network.*` hook events or named task_07 metric counters.
- Inspected hook taxonomy/payload/matcher/introspection/dispatch files and network manager/stats/delivery write surfaces.
- Added async-only network hook taxonomy, payload aliases, observation patch, matcher support, introspection descriptors, and dispatch methods.
- Added `network.HookDispatcher`, manager injection, daemon hooks bridge methods, and daemon boot wiring through `hooksNotifier`.
- Wired post-commit observation from `Manager.writeConversationMessage` after non-duplicate durable store success.
- Added runtime aggregate counters, low-cardinality `MetricSample` snapshots, delivery queue depth samples, direct resolve totals, and structured log fields for surface/container/work correlation.
- Added focused tests for hook catalog/introspection, matchers, payload redaction, async network dispatch, post-commit timing, failure isolation, duplicate suppression, metric labels, and queue depth labels.
- Targeted validation passed:
  - `go test ./internal/hooks -count=1`
  - `go test ./internal/network -count=1`
  - `go test ./internal/daemon -count=1`
  - `go test ./internal/api/core ./internal/store/... -count=1`
  - `go test ./internal/hooks -cover -count=1` -> `76.9%` package-wide; new network dispatch/matcher functions are covered, but the broad pre-existing hooks package remains below the task target.
  - `go test ./internal/network -cover -count=1` -> `80.3%`
- First `make verify` failed at codegen with `unknown hook contract type "NetworkThreadOpenedPayload"`; diagnosed root cause as the extension SDK hook-contract registry missing the new network descriptor payload names.
- Added network hook payload and patch names to `internal/extension/contract/sdk.go` and extended the hook-contract registry test for network descriptors.
- Later `make verify` reached Go lint and failed because direct network matcher fields pushed `hooks.HookDecl` over gocritic's heavy-copy threshold. Reworked the matcher model to embedded pointer `NetworkMatcher` and `CompactionMatcher` structs, keeping flat generated matcher fields while bringing targeted lint back to `0 issues`.
- Regenerated SDK/OpenAPI artifacts and normalized `openapi/agh.json` with the repo's compact-array JSON style; `make codegen-check` passes.
- Full `make verify` passed after implementation and codegen/lint corrections.
- Self-review found partial matcher exposure: core `hooks.HookMatcher` had network fields, but config/extension manifest parsing plus CLI/settings display paths did not. Added those fields and focused tests.
- Focused post-fix validation passed: `go test ./internal/hooks ./internal/network ./internal/daemon ./internal/config ./internal/settings ./internal/cli ./internal/extension ./internal/extension/contract -count=1`.
- A follow-up `make verify` failed lint because the new config/manifest matcher fields made hook declaration configs heavy to copy and made `hookMatcherMap` too complex; fixed by switching affected loops/functions to pointer/index access and extracting small settings matcher helpers. `make lint` now reports `0 issues`.
- Full post-self-review `make verify` passed after the lint correction with `0 issues`, `DONE 8177 tests`, and `OK: all package boundaries respected`.
- Updated task_07 checkboxes/status and inserted the Task 07 completed row into the current compact `_tasks.md` table.
- Fresh pre-commit `make verify` after tracking updates passed with frontend lint `Found 0 warnings and 0 errors`, Vitest `330 passed` / `2092 passed`, Go lint `0 issues`, `DONE 8177 tests`, and `OK: all package boundaries respected`.
- Reviewed staged scope: only hook/network/daemon/config/settings/contract/generated implementation files were staged; `.compozy` tracking/memory and unrelated QA deletions stayed unstaged.
- Created local commit `797a8ad8 feat: add network observation hooks`, then amended it to `4fffcc1d feat: add network observation hooks` after the final coverage hardening.
- Promoted Task 07 durable context into shared workflow memory.
- Post-commit `make verify` passed with frontend lint `Found 0 warnings and 0 errors`, Vitest `330 passed` / `2092 passed`, Go lint `0 issues`, `DONE 8177 tests`, and `OK: all package boundaries respected`.
- Rechecked touched-package coverage and found `internal/hooks` still at `77.1%`; added focused tests for dispatch event emitter context, turn/session/correlation extraction, hook type validation, and task/spawn async clone paths.
- Targeted `go test ./internal/hooks -cover -count=1` now passes with `82.2%` coverage.
- Full pre-amend `make verify` passed after the coverage tests with frontend lint `Found 0 warnings and 0 errors`, Go lint `0 issues`, `DONE 8242 tests`, and `OK: all package boundaries respected`.
- Final post-amend `make verify` passed for commit `4fffcc1d` with frontend lint `Found 0 warnings and 0 errors`, Go lint `0 issues`, `DONE 8242 tests`, and `OK: all package boundaries respected`.
- Updated task_07 validation evidence and workflow memory with the final commit hash and verification result.

# Now:

- Final response.

# Next:

- None.

# Open questions (UNCONFIRMED if needed):

- None.

# Working set (files/ids/commands):

- PRD/task: `.compozy/tasks/network-threads/task_07.md`, `.compozy/tasks/network-threads/_tasks.md`
- Workflow memory: `.compozy/tasks/network-threads/memory/MEMORY.md`, `.compozy/tasks/network-threads/memory/task_07.md`
- Hook surfaces: `internal/hooks/events.go`, `payloads.go`, `types.go`, `matcher.go`, `dispatch.go`, `introspection.go`, `async_clone.go`, `dispatch_events.go`
- Runtime surfaces: `internal/network/manager.go`, `stats.go`, `delivery.go`, `audit.go`, new hook dispatcher file if needed
- Daemon bridge: `internal/daemon/hooks_bridge.go`, `internal/daemon/boot.go`
- Extension SDK contracts: `internal/extension/contract/sdk.go`, `internal/extension/contract/sdk_test.go`
- Matcher integration surfaces: `internal/config/hooks.go`, `internal/config/tool_surface.go`, `internal/extension/manifest.go`, `internal/extension/manager.go`, `internal/settings/collections.go`, `internal/cli/hooks.go`
