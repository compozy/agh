# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Execute Task 01: extract the Memory v2 shared contract into `internal/memory/contract`, hard-delete `internal/memory/types.go`, update direct callers, add focused contract/boundary tests, verify, update task tracking, and create one local commit.
- Success criteria: no old-package shim/alias remains, Slice 1 lexical contract excludes embedding/vector fields, affected packages compile against `contract`, >=80% focused contract package coverage where applicable, `make verify` passes before completion.

## Important Decisions

- Implementation repo is `/Users/pedronauck/dev/compozy/agh3`; `/Users/pedronauck/dev/compozy/looper` was the initial shell cwd but does not contain the Memory v2 implementation surface.
- `brainstorming` is not applied as an implementation gate because the task is executing the approved mem-v2 TechSpec/ADRs rather than designing a new feature interactively.
- `contract.Header` hard-cuts to canonical YAML `agent` for the agent name; JSON/API payloads still expose `agent_name`.
- `contract.Header.Provenance` is optional (`*Provenance`) to keep current list/header responses valid until later tasks populate provenance.

## Learnings

- Root/internal AGH guidance requires hard cuts over compatibility, no destructive git commands, `make verify` as the monorepo gate, and AGH-specific Go/test skills before production/test Go edits.
- Required AGH skills `agh-code-guidelines`, `agh-test-conventions`, and `agh-contract-codegen-coship` exist under the AGH repo `.agents/skills/` even though they are not in the session's advertised skill list.
- After the hard cut, focused and full verification passed: `go test ./internal/memory/contract -cover` reported 98.8% coverage; `go test ./internal/memory/... ./internal/api/... ./internal/cli/... ./internal/extension/...` passed; `env -u NO_COLOR make verify` passed with 8081 Go tests and package-boundary checks.
- `make verify` emits existing non-fatal Vite chunk-size and macOS linker warnings, but the command exits 0 and reports frontend lint `0 warnings/0 errors`, Go lint `0 issues`, and `OK: all package boundaries respected`.
- Local code commit created: `4d372f70 refactor: extract memory contract package`. Post-commit `env -u NO_COLOR make verify` passed after a transient site metadata timeout was rerun directly and then the full gate was rerun successfully.

## Files / Surfaces

- Initial dirty state before implementation: untracked `.compozy/tasks/mem-v2/memory/` directory containing workflow memory files.
- Touched surfaces: `internal/memory/contract/**`, `internal/memory/{store,catalog,document,assembler,recall}*.go`, `internal/api/**`, `internal/cli/**`, `internal/extension/**`, `internal/codegen/sdkts/generate.go`, `internal/daemon/**`, `openapi/agh.json`, `sdk/typescript/src/generated/contracts.ts`, `web/src/generated/agh-openapi.d.ts`, minimal `web/src/systems/knowledge/**` contract-consumer updates, and task tracking files.

## Errors / Corrections

- Corrected initial workdir assumption: task code is in `agh3`, not `looper`.
- A broad type-prefix replacement briefly touched field selectors; corrected before validation and used compile tests to confirm direct callers.
- Self-review removed an initial legacy YAML `agent_name` bridge from `contract.Header` because mem-v2 is greenfield and the TechSpec canonical field is `agent`.
- `make verify` initially failed after exposing `agent` scope/provenance through generated types; fixed root cause by making provenance optional and updating the narrow web knowledge scope assumptions.

## Ready for Next Run

- Task 01 implementation, verification, self-review, memory update, tracking update, local code commit, and post-commit verification are complete.
- Tracking/memory artifacts are intentionally left uncommitted unless repo policy changes: `.compozy/tasks/mem-v2/_tasks.md`, `.compozy/tasks/mem-v2/task_01.md`, `.compozy/tasks/mem-v2/memory/`, and `.codex/ledger/2026-05-05-MEMORY-memory-contract.md`.
