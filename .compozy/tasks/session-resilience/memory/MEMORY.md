# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 02 is complete: session stop paths now classify a canonical `StopReason` from explicit `StopCause` signals inside `finalizeStopped()`, persist the result to `SessionMeta`, and include `stop_reason` in `session_stopped` event payloads.

## Shared Decisions

- Non-user stop initiators should use cause-aware stop entrypoints instead of inferring intent from `ctx.Err()` or raw process exits. `Manager.Stop()` remains the user path; daemon shutdown uses an explicit cause-aware stop path so watcher races do not overwrite shutdown intent.

## Shared Learnings

- `store.SessionMeta` and `store.SessionInfo` can no longer be treated as identical structs once stop metadata starts landing. Use explicit field mapping at package boundaries instead of direct struct conversion so later stop-reason propagation work does not break unrelated packages.
- Process-exit classification must only synthesize `CauseCompleted` / `CauseProcessExited` when no explicit stop was already requested; otherwise user/shutdown stop reasons are lost during watcher races.

## Open Risks

- Task 03 still needs to propagate stop fields through `store.SessionInfo`, global DB persistence, and API/query layers. Until then, stop metadata is only present in `SessionMeta` and the in-memory `session.SessionInfo`.

## Handoffs

- Task 03 can rely on `SessionMeta.stop_reason`, in-memory `session.SessionInfo.StopReason`, and stored `session_stopped` event payloads already being classified for user stop, crash, and daemon shutdown flows.
