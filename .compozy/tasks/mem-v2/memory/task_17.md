# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Hard-cut the AGH CLI memory surface to the approved Memory v2 Slice 1 verbs, selectors, deterministic errors, and structured output contracts.
- Task references checked: `_techspec.md` `CLI verbs`, `Agent Manageability Plan`, `Greenfield Delete Targets`, `Development Sequencing` step 27, ADR-009, ADR-011, and ADR-012.

## Important Decisions

- `agh memory read` and `agh memory consolidate` are removed from the command tree and generated CLI reference; canonical commands are now `agh memory show` and `agh memory dream trigger`.
- CLI memory commands use the public Memory v2 DTOs and final route family through `internal/cli/client.go`; no command path calls legacy `GET /api/memory/search`, `PUT /api/memory/:filename`, or `POST /api/memory/consolidate`.
- `--scope agent` requires both `--agent` and `--agent-tier`; invalid selector combinations return deterministic `memory.scope.*` errors suitable for agent automation.
- Memory command output supports `-o json`, `-o jsonl`, `-o toon`, and human output through shared output bundles; JSON preserves public contract envelopes rather than legacy flat payloads.
- Public copy and generated CLI docs were co-shipped for the hard cut because stale `read/consolidate` docs would make the CLI surface untruthful before task 24.

## Learnings

- `make cli-docs` regenerates the memory command subtree from Cobra, but the CLI reference landing card at `packages/site/content/runtime/cli-reference/index.mdx` still needed a direct copy update.
- Existing daemon memory e2e fixtures still assumed positional `agh memory write <filename>` and `GET /api/memory/search?`; those were updated to Memory v2 shapes where touched.
- Scanner policy can reject fixture text that looks like debugging/session noise, such as "regression gates"; tests should use durable-memory language when exercising controller-backed writes.
- Do not run Mage-backed targets in parallel. A parallel `make bun-typecheck` run failed with the known `mage_output_file.go` race; sequential rerun passed.

## Files / Surfaces

- CLI command tree and output: `internal/cli/memory.go`, `internal/cli/format.go`, `internal/cli/memory_test.go`, `internal/cli/helpers_test.go`.
- CLI daemon client: `internal/cli/client.go`, `internal/cli/client_test.go`.
- Contract selector fix for CLI bodies: `internal/api/contract/memory.go`, `internal/api/core/memory.go`.
- Agent/operator guidance: `internal/memory/assembler.go`, bundled `agh-memory-guide`, runtime memory docs, CLI reference docs, and landing copy/tests.
- Generated artifacts refreshed: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`, and `packages/site/content/runtime/cli-reference/memory/**`.

## Errors / Corrections

- Initial `make lint` failed on `lll` and `goconst` in the new memory CLI surface; line breaks and existing boolean string constants fixed it.
- First full `make verify` failed because the landing test expected old copy (`Memory at ~/.agh/memory/*.md`); updated the test to `Memory as scoped Markdown`.
- Targeted daemon integration uncovered an unresolved behavior gap: controller-backed writes are not visible to search before an explicit reindex in the current e2e path. Full `make verify` does not run this integration lane, so the risk is carried forward instead of hidden.

## Ready for Next Run

- Focused validation passed: `go test ./internal/cli -count=1`, `go test -race ./internal/cli -count=1`, and `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/memory ./internal/tools/builtin -count=1`.
- Site validation passed after copy update: `cd packages/site && bun run test components/landing/__tests__/landing.test.tsx`.
- Codegen/docs validation passed: `make codegen`, `make codegen-check`, `make cli-docs`, `make bun-typecheck` sequential, `make lint`, and `git diff --check`.
- Full pre-tracking `make verify` passed with Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8354 tests`, and package boundaries `OK`.
- Next task after state update should be `task_18` (Native Tools and Extension Host Memory Surfaces).
