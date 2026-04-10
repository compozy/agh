# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `internal/extension/capability.go` with `ExtensionSource`, `CapabilityChecker`, and typed `ErrCapabilityDenied`.
- Enforce the source-tier ceiling before manifest requests, then expose generic `Check` and Host API `CheckHostAPI` guards.
- Deliver table-driven tests covering all required source tiers, wildcard behavior, denial data, and Host API dual-gate enforcement.

## Important Decisions
- Mirror `internal/skills.SkillSource` with a local ordinal enum only; no extra JSON/string helpers are needed for task 04.
- Keep scope to generic capability enforcement and Host API method-to-family mapping from `_protocol.md`; hook-event capability mapping is a later integration concern.
- Treat marketplace defaults as the ADR baseline allowlist (`session.read`, `tool.read`, `observe.read`) plus the protocol-needed read families (`memory.read`, `skills.read`), while still denying `permission.*`, `session.write`, and `memory.write` unless explicitly allowlisted.
- Apply marketplace action ceilings from the Host API method-to-security map so the read-only action list stays aligned with the protocol instead of duplicating separate handwritten allowlists.

## Learnings
- `internal/subprocess/handshake.go` already carries `source_tier`, `granted_actions`, and `granted_security`, so this task only needs to provide the computation and enforcement layer.
- `internal/extension/` currently contains only manifest parsing code, making the absence of the checker a direct pre-change signal.
- Denial payloads stay simple for task 04: `ErrCapabilityDenied.Data` always carries `method`, `required`, and `granted`, while task 07 remains responsible for translating that into JSON-RPC responses.

## Files / Surfaces
- `internal/extension/manifest.go`
- `internal/subprocess/handshake.go`
- `internal/skills/types.go`
- `internal/hooks/permission.go`
- Added: `internal/extension/capability.go`, `internal/extension/capability_test.go`

## Errors / Corrections
- Initial skill-file read used the wrong `cy-workflow-memory` path; corrected to `/Users/pedronauck/Dev/projects/agh/.agents/skills/cy-workflow-memory/SKILL.md`.
- No code corrections were needed after focused package tests; the first full implementation passed `go test ./internal/extension`, `go test ./internal/extension -coverprofile=...`, and `make verify`.

## Ready for Next Run
- Verified evidence:
- `go test ./internal/extension -count=1`
- `go test ./internal/extension -coverprofile=/tmp/internal-extension-task04.cover.out -covermode=count -count=1` => `81.7%`
- `make verify`
- Next run should only need final handoff context unless follow-up review feedback appears.
