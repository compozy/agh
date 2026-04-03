# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/plugins` with embedded assets for Claude, Codex, OpenCode, and Pi, plus tests and tracking updates for task_01.
- Pre-change signal on 2026-04-01: `internal/plugins` directories exist, but the task-owned files are missing, so the task is not implemented in the current workspace.

## Important Decisions
- Use the task file + techspec as the design baseline; do not reopen brainstorming.
- Claude hook JSON will include `AGH_AGENT_NAME` gating and `AGH_BIN` fallback to satisfy the explicit task requirements even though the illustrative techspec snippet is shorter.

## Learnings
- The workspace already contains unrelated changes in `.compozy/tasks/global-plugins/_meta.md`; avoid touching them.
- `go test -count=1 -cover ./internal/plugins` passed with 100.0% coverage.
- `make verify` passed cleanly after the task-owned changes.

## Files / Surfaces
- `internal/plugins/embed.go`
- `internal/plugins/embed_test.go`
- `internal/plugins/claude/.claude-plugin/plugin.json`
- `internal/plugins/claude/hooks/hooks.json`
- `internal/plugins/codex/hooks.json`
- `internal/plugins/codex/agh-forwarder.sh`
- `internal/plugins/opencode/agh-hook.ts`
- `internal/plugins/pi/agh-hook.ts`
- `.compozy/tasks/global-plugins/task_01.md`
- `.compozy/tasks/global-plugins/_tasks.md`

## Errors / Corrections
- Corrected stale continuity assumptions from an earlier session: the plugin package is absent in the current tree and must be rebuilt from scratch.

## Ready for Next Run
- Task-local implementation and verification are complete. If a follow-up run is needed, inspect task_04 install/uninstall behavior against the embedded asset package and preserve executable permission for the Codex forwarder script.
