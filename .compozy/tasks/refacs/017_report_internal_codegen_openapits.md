# Iteration 017 Report: `internal/codegen/openapits`

## Scope

- Package: `github.com/pedronauck/agh/internal/codegen/openapits`
- Deterministic order index: 17 from `rtk go list ./internal/...`
- Next package: `github.com/pedronauck/agh/internal/codegen/sdkts`
- Analysis modes: `$refactoring-analysis`, `$extreme-software-optimization`, `$systematic-debugging`, `$no-workarounds`
- Subagents: read-only refactoring explorer completed; read-only performance explorer was launched but did not return before timeout and was closed. Performance evidence below is from local measurements in the parent run.

## Baseline

Commands run before implementation:

```bash
rtk go test ./internal/codegen/openapits -count=1
rtk golangci-lint run ./internal/codegen/openapits
rtk proxy go test ./internal/codegen/openapits -cover -count=1
rtk go test -tags integration ./internal/codegen/openapits -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/openapits -count=1
rtk proxy go test ./internal/codegen/openapits -run '^$' -bench . -benchmem -count=1
rtk make codegen-check
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/openapits/generate_test.go
```

Observed baseline:

- Package tests: passed (`15 passed in 1 packages`).
- Package lint: no issues.
- Package coverage: `79.5% of statements`, below the AGH 80% package floor.
- Integration-tag package tests: passed.
- Race package tests: passed.
- Package had no benchmark functions.
- `make codegen-check`: passed.
- Test-shape checker failed because existing subtests used names like `ShouldGenerate...` without the required `Should ...` spacing.

## Findings

### P1: `Generate` could rewrite the final artifact before the full pipeline succeeded

`Generate` wrote `openapi-typescript` output directly to `artifact.OutputPath`, then ran `oxfmt` on the same final file. If formatting failed after generation, a checked-in generated TypeScript file could be left partially rewritten even though the overall generation command failed.

Implemented:

- Added a private `generateWithRunner` pipeline.
- Generate now writes to a temporary file in the output directory.
- `oxfmt` formats the temporary file.
- The final output path is published only after both external commands succeed.
- Temporary files are removed on success and failure.
- Public `Generate(ctx, Artifact)` and `Check(ctx, Artifact)` signatures stayed unchanged.

Behavior proof:

- `TestGenerate/Should publish output only after formatting succeeds` proves an existing final output remains unchanged when formatting fails.
- `TestGenerate/Should publish formatted output after the pipeline succeeds` proves the final file receives the formatted temp output and no temp files remain.
- `make codegen-check` passed after the change.

### P1: artifact paths were raw unvalidated strings

`Artifact` is the public contract between `magefile.go` and this package. Before this iteration, empty spec/output paths or matching spec/output paths failed late through filesystem or external CLI errors. A matching spec/output path could also clobber the source OpenAPI document.

Implemented:

- Added `ErrInvalidArtifact`.
- Added `Artifact.validate`.
- `Generate` and `Check` now reject empty `SpecPath`, empty `OutputPath`, and matching clean spec/output paths before invoking external tools or creating temporary output.

Behavior proof:

- `TestGenerate/Should reject invalid artifacts before running generators`.
- `TestCheck/Should reject invalid artifacts before regenerating output`.

### P1: command cancellation lost the caller's context error

`runCommand` used `exec.CommandContext`, but when a context was already canceled or canceled during command execution, the returned error could surface as a raw process error instead of preserving `context.Canceled` or `context.DeadlineExceeded` for `errors.Is` callers.

Implemented:

- `runCommand` now checks `ctx.Err()` after `cmd.Run()` fails and wraps the context error when present.

Behavior proof:

- `TestRunCommand/Should preserve context cancellation` asserts `errors.Is(err, context.Canceled)`.

### P2: external command execution was tightly coupled to high-level generation behavior

The old implementation made it expensive to cover generation/format failure behavior without forcing real `bunx` failures.

Implemented:

- Added a private `commandRunner` interface and `execRunner`.
- Kept the exported API unchanged.
- Used the private runner only to test failure and publish semantics around the real external-command boundary.
- Kept the existing real `Generate`/`Check` tests as integration-like proof against actual `openapi-typescript` and `oxfmt`.

### P2: package-local tests were not gate-clean

The package had passing tests, but AGH's test-shape checker rejected subtest names, and coverage was just below the package floor.

Implemented:

- Renamed subtests from `ShouldX...` to `Should ...`.
- Added focused validation, transactional publish, and cancellation tests.

Final coverage:

- `81.4% of statements`.

## Performance Analysis

No performance optimization was implemented.

Observed measurements:

```bash
rtk proxy hyperfine --warmup 1 --runs 5 'go test ./internal/codegen/openapits -count=1'
rtk proxy hyperfine --warmup 1 --runs 5 'make codegen-check'
```

Observed results:

- Package tests after added coverage: about `1.001 s +/- 0.044 s`.
- `make codegen-check`: about `1.512 s +/- 0.153 s`.

Opportunity matrix:

| Opportunity | Evidence | Impact | Confidence | Effort | Score | Decision |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| Avoid an extra temp-file read/write in `Generate` | Final implementation uses temp file plus `os.Rename`; no extra read/write remains. | 1 | 4 | 1 | 4.0 | Already handled as part of transactional correctness |
| Replace external `bunx` invocations with in-process generation | Runtime is dominated by external `openapi-typescript`/`oxfmt`; changing this would require owning third-party codegen behavior. | 2 | 1 | 5 | 0.4 | Defer |
| Parallelize artifact checks in `magefile.go` | `make codegen-check` is ~1.5s; changing orchestration is outside this package boundary. | 1 | 2 | 3 | 0.67 | Defer |

Decision:

- The only implemented performance-relevant change was the temp-file publish path using rename after formatting. It was justified by correctness and worktree hygiene, not by a user-facing latency target.
- No additional optimization reached the implementation threshold for this package iteration.

## Deferred

- Extract shared drift-check utilities between `internal/codegen/openapits` and `cmd/agh-codegen`. The helpers overlap, but sharing them crosses this package boundary and should wait for a broader codegen cleanup.
- Change `magefile.go` to run browser OpenAPI artifact checks in parallel. The measured `make codegen-check` runtime is not high enough to justify broadening this iteration.
- Replace `bunx openapi-typescript` or `bunx oxfmt` with in-process libraries. That would change the trust boundary and must be a separate design decision.

## Behavior Proof

- Public API compatibility: `Generate(ctx, Artifact)` and `Check(ctx, Artifact)` signatures are unchanged.
- Generated artifact drift: `make codegen-check` passes.
- Final-file preservation: tests prove format failure does not mutate the final output.
- Cleanup: tests prove successful and failed temp outputs are removed.
- Error matching: tests prove invalid artifacts and context cancellation are matchable with `errors.Is`.
- An exploratory `make codegen` reformatted `openapi/agh.json`; that generated formatting-only diff was not kept because it is outside this package's refactor scope and `make codegen-check` remains the final drift gate for this iteration.

## Files Changed

- `internal/codegen/openapits/generate.go`
- `internal/codegen/openapits/generate_test.go`

## Validation

Final validation commands:

```bash
rtk go test ./internal/codegen/openapits -count=1
rtk golangci-lint run ./internal/codegen/openapits
rtk proxy go test ./internal/codegen/openapits -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/openapits -count=1
rtk go test -tags integration ./internal/codegen/openapits -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/openapits/generate_test.go
rtk make codegen-check
rtk go test ./internal/codegen/openapits ./cmd/agh-codegen -count=1
rtk golangci-lint run ./internal/codegen/openapits ./cmd/agh-codegen
rtk proxy go test -tags mage . -count=1
rtk make verify
```

Observed results:

- Package tests: `23 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `81.4% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `23 passed in 1 packages`.
- Test-shape check: passed for `internal/codegen/openapits/generate_test.go`.
- `make codegen-check`: passed.
- Direct dependent codegen package set: `64 passed in 2 packages`.
- Direct dependent lint set: no issues.
- Mage-tag root tests: passed.
- `make verify`: passed.
