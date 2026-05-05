# Goal (incl. success criteria):

- Implement `.compozy/tasks/network-threads/task_02.md`: hard-cut the `internal/network` runtime envelope from `interaction_id`/`kind:"direct"` to `surface`, `thread_id`, `direct_id`, and `work_id`.
- Success criteria: active `internal/network` cannot construct or validate `interaction_id`, `kind:"direct"`, `KindDirect`, `DirectBody`, or `Envelope.InteractionID`; container/work validation tests and RFC 004 canonicalization tests pass; `make verify` passes before completion; task memory/tracking updated; one local commit created only after clean verification and self-review.

# Constraints/Assumptions:

- Do not run destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- Existing dirty worktree entries are user/other-agent work; do not revert or clobber them.
- Scope is task_02 only: runtime envelope/validation/lifecycle symbols in `internal/network`; persistence, public contracts, API/web/docs are later tasks unless needed to compile.
- No compatibility aliases/readers: legacy `interaction_id` and `kind:"direct"` must fail closed.
- Requested skills `nats`, `agh-code-guidelines`, and `agh-test-conventions` are not installed in this session; repo docs and available Go/testing skills are the fallback.

# Key decisions:

- Treat `_techspec.md` and ADR-002/ADR-003 as source of truth for `work_id`, `surface`, and kind/surface separation.
- Do not implement store-backed thread/direct/work creation in this task; later tasks own persistence and API/web surfaces.
- Add only the pure direct-room identity helper from the TechSpec because runtime direct-surface sends need deterministic `direct_id` construction without persistence.
- Keep public API/CLI `interaction_id` DTO compatibility outside `internal/network` where later task_08 owns the public hard cut; map those existing fields into `internal/network.WorkID` only where required to compile.

# State:

- complete

# Done:

- Read workflow shared memory and task memory for task_02.
- Scanned relevant cross-agent ledgers for network/thread/hard-cut context.
- Loaded skills: `cy-workflow-memory`, `cy-execute-task`, `golang-pro`, `testing-anti-patterns`, `no-workarounds`, `systematic-debugging`, and `cy-final-verify`.
- Read root `AGENTS.md`, root `CLAUDE.md`, `internal/AGENTS.md`, `internal/CLAUDE.md`, task_02, `_tasks.md`, `_techspec.md`, `_design.md`, and ADR-001/002/003.
- Confirmed task_01 is complete in `_tasks.md` and shared memory says active RFC/glossary docs now use `surface`/container IDs/`work_id`.
- Implemented the `internal/network` envelope hard cut, validator invariants, work lifecycle rename, router dispatch updates, delivery prompt metadata, store-entry `WorkID` compile fallout, and API/core mapping needed to feed the new internal runtime.
- Added/updated tests for container symmetry, greet/whois conversation-field rejection, `receipt`/`trace` work requirements, legacy `interaction_id`/`kind:"direct"` fail-closed behavior, raw claim-token rejection, RFC 004 nullable-field canonicalization, lifecycle, router, delivery, and direct-room identity.
- Targeted checks passed: `go test ./internal/network -count=1`, `go test ./internal/network -cover -count=1` (`81.9%`), and `go test ./internal/api/udsapi ./internal/...`.
- First full `make verify` reached Go lint and failed on gocritic/lll issues in `internal/network/audit.go`, `router.go`, and `lifecycle.go`; fixed the one-case type switch, large `receiveState` copies, and trace-transition line wrapping.
- Fresh `make verify` then passed with `DONE 8097 tests` and `OK: all package boundaries respected`.
- Self-review tightened TechSpec alignment by requiring `work_id` for `kind:"capability"` and adding a focused validator test before rerunning verification.
- Final targeted checks after the capability correction passed: `go test ./internal/network -count=1` and `go test ./internal/network -cover -count=1` (`81.9%`).
- Final full `make verify` passed with `0 issues.`, `DONE 8098 tests`, and `OK: all package boundaries respected`.
- Updated task_02 tracking, master `_tasks.md`, task memory, and shared workflow memory.
- Fresh pre-commit `make verify` after tracking/memory updates passed with `0 issues.`, `DONE 8098 tests`, and `OK: all package boundaries respected`.
- Created local commit `cc6194c3 feat: hard cut network wire model`.
- Post-commit `make verify` passed with `0 issues.`, `DONE 8098 tests`, and `OK: all package boundaries respected`.

# Now:

- Final report.

# Next:

- None.

# Open questions (UNCONFIRMED if needed):

- None.

# Working set (files/ids/commands):

- PRD dir: `.compozy/tasks/network-threads/`
- Workflow memory: `.compozy/tasks/network-threads/memory/MEMORY.md`
- Task memory: `.compozy/tasks/network-threads/memory/task_02.md`
- Main code/test surfaces: `internal/network/envelope.go`, `internal/network/validate.go`, `internal/network/lifecycle.go`, `internal/network/router.go`, `internal/network/delivery.go`, `internal/network/audit.go`, `internal/network/*_test.go`, plus narrow compile fallout in API/core, ACP meta, situation preview, and store network-message types.
