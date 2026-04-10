# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 02 is complete: session stop paths now classify a canonical `StopReason` from explicit `StopCause` signals inside `finalizeStopped()`, persist the result to `SessionMeta`, and include `stop_reason` in `session_stopped` event payloads.
- Task 03 is complete: global session rows now persist `stop_reason` / `stop_detail`, observer stop updates and reconciliation write those fields into SQLite, and session list/detail API payloads expose the session-level stop metadata.
- Task 04 is complete: `Resume()` now repairs stale meta state before agent startup, persists crash classification back to `meta.json`, validates resume infrastructure with aggregated diagnostics, and end-to-end tests cover stop reason propagation through stop and resume flows.

## Shared Decisions

- Non-user stop initiators should use cause-aware stop entrypoints instead of inferring intent from `ctx.Err()` or raw process exits. `Manager.Stop()` remains the user path; daemon shutdown uses an explicit cause-aware stop path so watcher races do not overwrite shutdown intent.
- Resume repair must run before ACP startup while preserving the existing pre/post resume hook boundaries, so resume hooks see already-classified stop metadata and a validated infrastructure baseline.

## Shared Learnings

- `store.SessionMeta` and `store.SessionInfo` can no longer be treated as identical structs once stop metadata starts landing. Use explicit field mapping at package boundaries instead of direct struct conversion so later stop-reason propagation work does not break unrelated packages.
- Process-exit classification must only synthesize `CauseCompleted` / `CauseProcessExited` when no explicit stop was already requested; otherwise user/shutdown stop reasons are lost during watcher races.
- Resume validation should collect all independent infrastructure failures in one attempt instead of failing fast, otherwise operators lose the full repair diagnosis.
- Real ACP stop integration tests need an additional wait for final stopped state before resume/assertions; `Stop()` returning does not guarantee watcher finalization has persisted the last stop metadata yet.

## Open Risks

- No additional cross-task risks recorded for the session-resilience workflow after task 04.

## Handoffs

- Task 03 can rely on `SessionMeta.stop_reason`, in-memory `session.SessionInfo.StopReason`, and stored `session_stopped` event payloads already being classified for user stop, crash, and daemon shutdown flows.
- Task 04 can now rely on `sessions.stop_reason` / `sessions.stop_detail` in the global DB and on HTTP/UDS session payloads carrying the same session-level stop metadata.
- Future resume or hook work can rely on `Resume()` repairing stale stop metadata before hook dispatch and on `[session.limits].timeout` being available through config parsing and merge overlays.
