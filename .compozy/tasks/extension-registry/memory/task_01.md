# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create `internal/registry/` foundation by extracting archive/path/version/move helpers from `internal/cli/skill_marketplace.go`, adding decompression/file-count guards, and defining shared registry types/interfaces for later tasks.
- Keep `skill_marketplace` integration behavior intact and finish with clean verification.

## Important Decisions
- Add dedicated `internal/registry` unit tests for the moved helpers/types instead of depending only on CLI-local helper coverage.
- Keep the current task scoped to shared foundations only; do not start adapter or installer work from later tasks.
- Keep thin compatibility wrappers in `internal/cli/skill_marketplace.go` while moving the real implementations into `internal/registry/`.
- Make `registry.VersionIsNewer` reject invalid version strings instead of falling back to lexical comparison.

## Learnings
- The pre-change baseline is still the legacy shape: no `internal/registry` package exists, helper logic lives in `internal/cli/skill_marketplace.go`, and helper tests live in `internal/cli/skill_test.go`.
- The current `versionIsNewer` logic still falls back to lexical comparison for invalid versions; task 01 requires invalid version strings to return `false` without panicking.
- `observe.BridgeSource` now requires `DeliveryMetrics()`, so the CLI integration bridge stub must implement that method for marketplace integration tests to compile.

## Files / Surfaces
- `internal/registry/extract.go`
- `internal/registry/version.go`
- `internal/registry/types.go`
- `internal/registry/source.go`
- `internal/registry/extract_test.go`
- `internal/registry/version_test.go`
- `internal/registry/source_test.go`
- `internal/cli/skill_marketplace.go`
- `internal/cli/skill_marketplace_integration_test.go`
- `internal/cli/cli_integration_test.go`

## Errors / Corrections
- `go test -tags integration ./internal/cli ...` initially failed because `integrationBridgeService` no longer satisfied `observe.BridgeSource`; added the missing `DeliveryMetrics()` method to the test stub.
- `make verify` initially failed on deprecated `tar.TypeRegA`; replaced it with `0`/`tar.TypeReg` handling and re-ran full verification.

## Ready for Next Run
- Task complete. `internal/registry` now exists with shared extraction/version/types/source foundations, registry package coverage is 81.6%, the required marketplace integration tests pass unchanged, and `make verify` is green.
