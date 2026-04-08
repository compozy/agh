# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add provenance-sidecar helpers for marketplace skills in `internal/skills`, with SHA-256 hashing, `.agh-meta.json` read/write/detection/verification, and unit coverage for the task_06 acceptance list.

## Important Decisions
- Hashing uses raw `SKILL.md` file bytes rather than parsed body text so load-time tamper verification matches the install-time sidecar digest.
- `VerifyHash` returns a `HashMismatchError` carrying `ExpectedHash` and `ActualHash` for downstream registry logging in task_07.
- Sidecar writes use `json.MarshalIndent(..., "", "  ")` plus a trailing newline to keep `.agh-meta.json` human-readable and stable across writes.

## Learnings
- Baseline gap: `internal/skills/provenance.go` is not present yet; only the `Provenance` type and clone behavior from task_01 exist.
- ADR-004 requires storing a SHA-256 hash at install time and recomputing it on every load to detect tampering.
- The existing `internal/skills` test helpers (`writeSkillFile`) were sufficient for provenance tests, so no extra fixtures or production hooks were needed.

## Files / Surfaces
- `internal/skills/types.go`
- `internal/skills/registry.go`
- `internal/skills/loader.go`
- `internal/skills/registry_test.go`
- `internal/skills/loader_test.go`
- `internal/skills/verify_test.go`
- `internal/skills/provenance.go`
- `internal/skills/provenance_test.go`

## Errors / Corrections
- None.

## Ready for Next Run
- Task complete. Verification evidence: `go test ./internal/skills -count=1`, `go test ./internal/skills -cover -count=1` (`81.8%`), and `make verify` all passed after the provenance implementation landed.
