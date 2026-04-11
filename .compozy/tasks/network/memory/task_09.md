# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add bundled `agh-network` skill content and inject it into session startup/resume prompts only when the session has non-empty `Space` metadata, with tests covering discovery, alignment, and resume behavior.

## Important Decisions
- Use the PRD/tech spec as the approved design baseline for this run instead of reopening design discussion.
- Load the injected guidance from the embedded bundled-skill asset directly rather than widening the session `SkillRegistry` interface for a one-off bundled lookup.
- Inject the bundled content after the normal startup prompt assembly/hook flow returns and before `driver.Start()`, so create and resume share one path and the existing prompt-provider contract stays unchanged.
- Keep CLI-surface validation in tests by comparing the bundled skill text against the real `agh network` Cobra command tree and send-command flags instead of duplicating the command list in test fixtures.

## Learnings
- The current startup path in `internal/session/manager_start.go` already centralizes both create and resume flows through `startSession`, so one injection point can cover both behaviors.
- `internal/network/delivery.go` currently renders inbound messages as `<network-message trust="untrusted">` with XML-escaped previews and base64 canonical JSON bodies; the skill text needs to match that exact wrapper contract and the allowlisted `agh network` command set from task 08.
- The bundled skill content can stay aligned with the real CLI surface by asserting against the Cobra command tree and send-command flags in tests, rather than hardcoding a duplicate command list.
- `make verify` passes cleanly with the task 09 changes on the committed state, and the touched-package unit coverage is `internal/skills/bundled` 86.7% and `internal/session` 81.9%.

## Files / Surfaces
- `internal/skills/bundled/skills/agh-network/SKILL.md`
- `internal/skills/bundled/content.go`
- `internal/skills/bundled/embed.go`
- `internal/skills/bundled/bundled_test.go`
- `internal/session/{manager_start.go,manager_helpers.go,manager_test.go,manager_integration_test.go}`
- `internal/cli/network.go`
- `internal/network/delivery.go`

## Errors / Corrections
- `go test -tags integration ./internal/session` still fails outside task 09 scope in `TestManagerIntegrationResumeClassifiesCrashAndActivates`, which expects a crash-classified resumed stop reason. Task 09's targeted integration test and `make verify` still pass.

## Ready for Next Run
- Task 09 code is implemented and committed as `b4ea1c2` (`feat: inject bundled network skill`): bundled `agh-network` content exists, startup/resume injection is gated on non-empty `Space`, and unit/integration tests cover discovery, CLI alignment, no-space behavior, and resume reinjection.
- Final evidence: `go test ./internal/skills/bundled`, `go test ./internal/session`, `go test -tags integration ./internal/session -run TestManagerIntegrationResumeWithSpaceReinjectsBundledNetworkSkillBeforeACPStart`, and post-commit `make verify` all passed.
