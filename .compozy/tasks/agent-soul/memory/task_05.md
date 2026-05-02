# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_05 as a backend-only foundation for optional `HEARTBEAT.md`: config authority, strict parser/resolver, deterministic digests/provenance, active-hours/cadence preferences within config bounds, prompt/status projections, and redacted diagnostics.
- Out of scope for this task: migration v13, session health service, scheduler integration, wake service, task lease changes, network greet changes, API/CLI/UDS surfaces.

## Important Decisions
- Follow the existing `internal/soul` resolver shape for missing files, diagnostics, safe workspace-relative paths, digesting, and compact/read projections, but keep Heartbeat in a separate `internal/heartbeat` package.
- Use `_techspec_heartbeat.md` and ADR-011 as the field-level source of truth for `[agents.heartbeat]` defaults: enabled=true, max_body_bytes=32768, context_projection_bytes=4096, min_interval=5m, default_interval=30m, wake_cooldown=1m, max_wakes_per_cycle=25, active_session_only=true, allow_active_hours_preferences=true, wake_event_retention=168h, session_health_stale_after=2m, session_health_hook_min_interval=1m.
- Keep task_05 to resolver/config only. No migration v13, wake tables, session health service, scheduler gate, synthetic prompting, task lease mutation, network greet mutation, API, CLI, UDS, or Host API surface was added.

## Learnings
- Baseline before implementation: no `internal/heartbeat` package exists, and `rg` only finds heartbeat references in session supervision/network/task lease contexts.
- `internal/heartbeat` resolves missing `HEARTBEAT.md` as optional inactive valid policy, while invalid present content fails closed with redacted diagnostics.
- Heartbeat policy/config digests include the resolved `[agents.heartbeat]` subset, so future persistence/wake tasks can detect semantic config drift.
- `rg` scope check after implementation found no `internal/heartbeat` dependency on scheduler/task/session/network packages; only config keys and forbidden-term diagnostics mention those authorities.

## Files / Surfaces
- Production surfaces: `internal/config/config.go`, `internal/config/merge.go`, `internal/config/tool_surface.go`, `internal/heartbeat/heartbeat.go`.
- Tests: `internal/config/config_test.go`, `internal/config/tool_surface_test.go`, `internal/heartbeat/heartbeat_test.go`.

## Errors / Corrections
- Focused checks passed: `go test ./internal/config ./internal/heartbeat -count=1`; `go test -race ./internal/config ./internal/heartbeat -count=1`; `go test ./internal/heartbeat -cover -count=1` with 84.8%; Heartbeat test-shape helper; `golangci-lint run ./internal/config ./internal/heartbeat`.
- Full pre-commit `make verify` passed with `DONE 7550 tests in 109.370s` and `OK: all package boundaries respected`.
- Fresh pre-commit `make verify` after tracking/memory updates passed with `DONE 7550 tests in 11.086s` and `OK: all package boundaries respected`.
- Local implementation commit created: `40df005c feat: add heartbeat policy resolver foundation`.
- Post-commit `make verify` passed with `DONE 7550 tests in 14.052s` and `OK: all package boundaries respected`.
- Final focused coverage recheck passed: `go test ./internal/heartbeat -cover -count=1` reported 84.8%.
- Final full-gate recheck passed: `make verify` reported `DONE 7550 tests in 12.888s` and `OK: all package boundaries respected`.

## Ready for Next Run
- Task 05 implementation, tracking, local commit, and post-commit verification are complete.
- Task 06 can consume `internal/heartbeat.ResolvedPolicy`, `ConfigProvenanceFor`, `Resolve`, and `Parse` for persisted snapshots, but must add migration v13/session-health/wake-audit storage itself.
