# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace `internal/skills` hook-owned types and runner with `internal/hooks.HookDecl` declarations, migrate skill loader parsing to the dotted taxonomy, update fixtures/tests, and leave the task with clean verification.

## Important Decisions
- Treat task 07 as a hard cut-over: legacy `on_*` skill hook events are rejected with replacement guidance instead of being silently remapped.
- Remove the `internal/hooks` notifier adapter because switching `internal/skills` to import `internal/hooks` exposed a package cycle through `internal/session`.
- Keep the daemon compiling in task 07 by constructing transient `hooks.Hooks` dispatchers inside `internal/daemon` until task 09 owns the full notifier-fanout replacement.

## Learnings
- The skill loader needs a post-parse normalization pass after source and provenance assignment so each `hooks.HookDecl` carries stable `Source` and `SkillSource` values.
- Strict YAML decoding with `KnownFields(true)` works for the new hook schema and still leaves room for descriptive legacy-event errors before the declaration hits `hooks.ValidateHookDecl`.

## Files / Surfaces
- `internal/skills/types.go`
- `internal/skills/loader.go`
- `internal/skills/registry.go`
- `internal/skills/hook_decl.go`
- `internal/skills/loader_test.go`
- `internal/skills/registry_test.go`
- `internal/skills/testdata/loader/*`
- `internal/daemon/boot.go`
- `internal/daemon/notifier.go`
- `internal/daemon/notifier_test.go`
- `internal/daemon/notifier_integration_test.go`
- `internal/hooks/hooks_test.go`
- `internal/hooks/agent_event.go`

## Errors / Corrections
- The first cut introduced an import cycle: `internal/session -> internal/skills -> internal/hooks -> internal/session`. Fixed by removing the base-package notifier bridge and updating hooks tests to call the typed dispatchers directly.
- `make verify` exposed an unused `durationValue` helper left behind in the loader rewrite; removing it restored a clean lint/build pass.

## Ready for Next Run
- Verification is clean:
  - `go test ./internal/hooks ./internal/skills ./internal/daemon -count=1`
  - `go test -tags integration ./internal/daemon -run 'TestNotifierFanout' -count=1`
  - `go test -coverprofile=/tmp/internal-skills.cover ./internal/skills -count=1` -> `81.3%`
  - `make verify`
- Task 09 should replace the temporary daemon-side bridge helpers with the final hooks-platform notifier wiring.
