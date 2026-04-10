# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement hook run persistence, HTTP introspection endpoints, and telemetry/logging/metrics for task 12.

## Important Decisions
- Treat the existing hooks PRD/techspec as the approved design baseline and keep scope focused on persistence/introspection rather than expanding runtime hook families.
- Use the per-session SQLite store for hook run history and let the observer own read/query access for the HTTP endpoints.
- Persist hook run audits in a dedicated `hook_runs` table inside each session DB and query them through observer-owned store openers so `/api/hooks/runs` can inspect historical sessions without a live recorder.
- Reuse active session dispatch context as the preferred hook run writer; fall back to the observer sink only when the pipeline is outside a session-managed recorder path.
- Gate `PatchApplied` persistence by event family unless debug logging is enabled: always keep audit patches for `permission.*`, `prompt.*`, `tool.*`, and `input.*`, omit them for other families by default.

## Learnings
- The current hook runtime has no telemetry sink or registry introspection surface yet; the observer API is still limited to global event summaries and health.
- Active session hook dispatch can reuse the existing recorder path if hook telemetry is passed through context instead of opening a second writer by default.
- `session.pre_create` remains best-effort for persistent telemetry because the per-session DB may not exist yet; the observer intentionally skips writes when the session DB path has not been created.
- The hooks runtime can expose catalog and taxonomy introspection without leaking internal ordering logic into HTTP handlers by centralizing that translation inside `internal/hooks`.

## Files / Surfaces
- `internal/hooks`
- `internal/store/sessiondb`
- `internal/observe`
- `internal/api/httpapi`
- `internal/api/contract`
- `internal/daemon`
- `internal/session`
- `internal/api/core`
- `internal/api/testutil`
- `internal/store/types.go`

## Errors / Corrections
- Adjusted session-managed dispatch paths to carry hook recorder context so task 12 writes use the existing session DB writer instead of opening duplicate handles.
- Added observer and HTTP test seams for hook catalog, run, and events queries because the previous observer interface only covered event summaries and health.

## Ready for Next Run
- Verification evidence:
- `go test ./internal/hooks ./internal/store/sessiondb ./internal/observe ./internal/api/httpapi ./internal/daemon ./internal/session -count=1`
- `go test -tags integration ./internal/api/httpapi ./internal/hooks -count=1`
- `go test -cover ./internal/hooks ./internal/store/sessiondb ./internal/observe ./internal/api/httpapi -count=1` with `internal/hooks 82.7%`, `internal/store/sessiondb 82.4%`, `internal/observe 81.4%`, and `internal/api/httpapi 81.4%`
- `make verify` before commit
- `git commit -m "feat: add hooks observability api"` created `945db92`
- `make verify` on committed `HEAD`
- Follow-up risk to remember: if persistent auditing is ever required for `session.pre_create`, the session creation flow will need an earlier store allocation point instead of the current best-effort skip behavior.
