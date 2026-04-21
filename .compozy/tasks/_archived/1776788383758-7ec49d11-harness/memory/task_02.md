# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace prepend/append-only startup prompt composition with daemon-owned section descriptors and selector evaluation against resolved startup policy.
- Move `agh-network` startup content out of the inline session startup overlay path and into explicit selected startup sections.

## Important Decisions
- Keep `session.PromptProvider` unchanged and layer policy-aware selection around it in `internal/daemon`.
- Add an optional startup-aware assembler seam so `session.Manager` can pass durable startup context without replacing the existing assembler contract.
- Stop using the daemon-owned startup overlay for the default boot path; startup selection should happen before final prompt concatenation.
- Keep the generic `StartupPromptOverlay` session option for explicit callers/tests, but daemon boot now leaves it unset so section selection is the one authoritative default startup path.
- Preserve previous effective ordering semantics by rendering memory before the base prompt, skills after the base prompt, and `agh-network` last as an explicit append section.

## Learnings
- Existing resolver output already exposes `IncludeSections`, so selector predicates can stay aligned with the daemon-owned harness policy vocabulary from task 01 without widening the provider contract.
- Default startup section budgets can stay permissive while tests cover trim and omit behavior with synthetic descriptors; no new config knob was needed for this task.
- Existing daemon boot tests expected the deprecated startup overlay injection and needed to be updated to assert a startup-aware prompt assembler instead.

## Files / Surfaces
- `internal/daemon/composed_assembler.go`
- `internal/daemon/composed_assembler_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/harness_context.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/prompt_sections.go`
- `internal/daemon/section_selector.go`
- `internal/session/manager_helpers.go`
- `internal/session/manager_integration_test.go`
- `internal/session/manager_network_skill.go` (removed)
- `internal/session/manager_start.go`
- `internal/session/manager_test.go`
- `internal/session/prompt_overlay.go`

## Errors / Corrections
- Fixed daemon compile failures after removing the overlay helper by restoring the `context` import needed by the prompt augmenter path and re-adding the `strings` import needed by resolver tests.
- Fixed a stale daemon boot unit expectation that still required `StartupPromptOverlay`; the boot contract now asserts a startup-aware prompt assembler and a nil default overlay.

## Ready for Next Run
- Implementation is complete, verified, and committed locally as `be9ebb6c` (`feat: add startup prompt section selection`).
- Post-commit worktree still contains unrelated/tracking-only edits in `.agents/skills/compozy/references/config-reference.md`, `.compozy/tasks/harness/_tasks.md`, `.compozy/tasks/harness/task_01.md`, `.compozy/tasks/harness/task_02.md`, `web/AGENTS.md`, `web/CLAUDE.md`, plus untracked `.compozy/tasks/harness/_meta.md` and `.compozy/tasks/harness/memory/`.
- Verification run:
  - fresh pre-commit rerun: `make verify`
  - `go test ./internal/daemon ./internal/session -count=1`
  - `go test -tags integration ./internal/daemon ./internal/session -count=1`
  - `go test ./internal/daemon -coverprofile=/tmp/task02-daemon.cover.out -count=1`
  - `go vet ./...`
  - `make verify`
