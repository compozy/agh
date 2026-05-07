# Iteration 016 Report: `internal/cli/docpost`

## Scope

- Package: `github.com/pedronauck/agh/internal/cli/docpost`
- Deterministic order index: 16 from `rtk go list ./internal/...`
- Next package: `github.com/pedronauck/agh/internal/codegen/openapits`
- Analysis modes: `$refactoring-analysis`, `$extreme-software-optimization`, `$systematic-debugging`, `$no-workarounds`
- Subagents: read-only refactoring explorer and read-only performance explorer

## Baseline

Commands run before implementation:

```bash
rtk go test ./internal/cli/docpost -count=1
rtk golangci-lint run ./internal/cli/docpost
rtk proxy go test ./internal/cli/docpost -cover -count=1
rtk go test -tags integration ./internal/cli/docpost -count=1
rtk proxy go test ./internal/cli/docpost -run '^$' -bench . -benchmem -count=1
rtk proxy go test ./internal/cli/docpost -run '^TestProcess' -count=100 -cpuprofile=/tmp/docpost-016-process.cpu -memprofile=/tmp/docpost-016-process.mem -memprofilerate=1
```

Observed baseline:

- Package tests: passed (`61 passed in 1 packages`).
- Package lint: no issues.
- Package coverage: `88.2% of statements`.
- Integration-tag package tests: passed (`61 passed in 1 packages`).
- Package has no benchmark functions.
- Profiling `TestProcess` showed this package is dominated by filesystem/syscall work from reading, writing, walking, and removing generated docs. No package-local CPU or allocation hotspot justified a performance rewrite in this iteration.

## Findings

### P1: source filename parsing accepted ambiguous `agh*` files

`readInputs` accepted any markdown filename whose base started with `agh`. That meant a file like `aghost.md` had no command segments and planned the same root output path as `agh.md`.

Implemented:

- Added strict command source parsing through `commandSegments`.
- Accepted only `agh.md` or filenames beginning with `agh_`.
- Validated every command segment with `^[A-Za-z0-9-]+$`, rejecting empty segments such as `agh__list.md`.
- Added `readInput` to isolate per-file parsing and reading.
- Added `validateOutputPaths` before any output write so generated path collisions fail before partial output is emitted.

Behavior proof:

- `TestProcessInputRefacs/Should reject ambiguous agh-prefixed source filenames`
- `TestProcessInputRefacs/Should reject invalid empty command segments`
- `TestProcessInputRefacs/Should reject duplicate planned output paths`

### P1: command naming and target path rules were spread across raw string operations

The package repeated command-name, target-URL, and output-path conversions through direct `strings.ReplaceAll`, `strings.Join`, and ad hoc root checks. `buildTargetMap` also carried a dead `hasChildren` parameter.

Implemented:

- Added `input.isRoot`, `input.commandName`, `input.targetURL`, and `input.outputPath`.
- Kept `baseNameToCommand` as the single command display-name conversion.
- Removed the dead `hasChildren` parameter from `buildTargetMap`.
- Updated subcommand rendering, direct-child sorting, and link target generation to use the `input` methods.

Behavior proof:

- Existing target-map tests continue to pass with the simplified signature.
- The generated CLI reference remains deterministic across two consecutive `agh doc` runs.

### P1: link rewriting mutated markdown code examples

`rewriteLinks` and `remapLinks` used regex replacement over the whole document. Command links shown in fenced code blocks or inline code spans could be rewritten as live documentation links, changing literal examples.

Implemented:

- Added `transformMarkdownOutsideCode` and `transformInlineText`.
- Applied cross-link stripping and target remapping only outside fenced code blocks and inline code spans.
- Preserved literal examples inside code while keeping normal prose links functional.

Behavior proof:

- `TestLinkRewriteRefacs/Should preserve command links inside code regions`
- `TestLinkRewriteRefacs/Should remap only non-code command links`

### P2: generated CLI reference needed explicit regeneration validation

After the generator changes, `make cli-docs` needed to be rerun to prove the CLI reference output still materializes correctly. The first regeneration exposed generator-owned formatting churn, and the repo formatter in `make verify` normalized the generated docs back to the tracked formatting.

Implemented:

- Re-ran `make cli-docs`.
- Ran `make site-build` to validate the site can build with the regenerated docs.
- Confirmed there is no final persistent diff under `packages/site/content/runtime/cli-reference`.

## Performance Analysis

The performance explorer and local profiling did not identify an implement-now optimization.

Observed evidence:

- Full `agh doc` generation in a temporary output directory averaged about `98 ms` in the read-only audit.
- Package-local markdown transforms measured in the low tens of microseconds per repeated test batch.
- `Process` profiles were dominated by filesystem operations: `os.WriteFile`, `os.ReadFile`, `os.RemoveAll`, `filepath.WalkDir`, and syscall frames.
- Local memory profiling showed small package-local allocation contributors relative to the filesystem-heavy workflow.

Decision:

- No performance-specific production change was made.
- The implemented changes are correctness and maintainability refactors with behavior tests.

## Deferred

- Split `docpost.go` into focused files for input parsing, output planning, markdown transforms, and meta generation. This is valuable, but not required to fix the actionable correctness issues.
- Replace regex-based markdown transforms with a parser-backed transform if the CLI reference begins carrying more complex markdown constructs. The current implementation handles the common fenced-code and inline-code cases without taking on a new parser dependency.
- Preserve generator source metadata for richer diagnostics around output collisions. The current collision error reports both colliding filenames and the planned output path.
- Optimize filesystem writes only if a future profile shows CLI docs generation on a user-facing hot path. Current evidence says the workflow is build/docs tooling, not runtime-critical.

## Behavior Proof

- Invalid source filenames now fail before reading/writing generated output.
- Output path collisions now fail before partial writes.
- Inter-command links in prose still rewrite to absolute `/runtime/cli-reference/...` URLs.
- Links inside fenced code blocks and inline code spans remain literal examples.
- Existing generated output layout remains the same: root command is `agh.mdx`, command families use nested `index.mdx` pages, and subdirectory `meta.json` files are regenerated.
- Running generated docs twice with the same binary produced no diff between output directories.

## Files Changed

- `internal/cli/docpost/docpost.go`
- `internal/cli/docpost/docpost_test.go`
- `internal/cli/docpost/docpost_refac_test.go`

## Validation

Final validation commands:

```bash
rtk go test ./internal/cli/docpost -count=1
rtk golangci-lint run ./internal/cli/docpost
rtk proxy go test ./internal/cli/docpost -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/cli/docpost -count=1
rtk go test -tags integration ./internal/cli/docpost -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/docpost/docpost_refac_test.go
rtk go test ./internal/cli/docpost ./internal/cli -run 'Test(NewDocCommand|ProcessInputRefacs|LinkRewriteRefacs|BuildTargetMap|RemapLinks|RewriteLinks)' -count=1
rtk make cli-docs
rtk proxy go build -o /tmp/agh-docpost-016 ./cmd/agh
rtk make site-build
rtk make verify
```

Observed results:

- Package tests: `68 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `89.5% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `68 passed in 1 packages`.
- New test-shape check: passed for `internal/cli/docpost/docpost_refac_test.go`.
- Direct dependent/focused command tests: `24 passed in 2 packages`.
- Generated CLI docs: regenerated successfully, with no final persistent diff under `packages/site/content/runtime/cli-reference`.
- Determinism check: two consecutive `agh doc` outputs from the same binary had no diff.
- Site build: passed.
- `make verify`: passed.
