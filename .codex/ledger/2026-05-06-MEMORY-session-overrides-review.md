Goal (incl. success criteria):

- Implement `.codex/plans/session-runtime-overrides-hardening.md` end-to-end.
- Success: backend/session/config/settings/CLI/web/docs/codegen changes complete, tests added, targeted verification and `make verify` pass, and completion audit maps each plan requirement to evidence.
  Constraints/Assumptions:
- Must use RTK for shell commands.
- Must not run destructive git commands.
- Current requested work is production implementation.
- Conversation can be BR-PT; artifacts are English.
- Explicit skills requested: architectural-analysis, no-workarounds, golang-pro, testing-anti-patterns, extreme-software-optimization.
- Mandatory repo skills also active as applicable: agh-code-guidelines, agh-test-conventions, agh-contract-codegen-coship, React/Vitest/web design/docs skills.
  Key decisions:
- Preserve unrelated dirty worktree changes; edit only plan-related files.
- Treat `supported_models` as advisory; enforce reasoning provider support in backend.
  State:
- Complete; final verification passed.
  Done:
- Loaded RTK.
- Scanned ledger directory for cross-agent awareness.
- Loaded internal/CLAUDE.md and AGH Go/security review skills.
- Traced CreateSession override plumbing through contract, HTTP/UDS core handler, CLI, manager startup, provider config, extension host API, persistence, docs, and Codex ACP package behavior.
- Identified blockers around actual reasoning_effort propagation, validation boundary mismatch, incomplete extension/docs surfaces, and missing provider support enforcement.
- Wrote accepted plan to `.codex/plans/session-runtime-overrides-hardening.md`.
- Added plan metadata with accepted status, execution state, provenance, and primary surfaces.
- Loaded newly requested implementation skills and domain skills.
- Added session read/persistence fields for model and reasoning_effort.
- Added provider tri-state supported_models/supports_reasoning_effort through config overlay, settings payloads, settings service, workspace config cloning, and CLI config-set classification.
- Added manager-level runtime override validation so non-HTTP callers cannot bypass provider/reasoning checks.
- Added focused Go coverage for session runtime override persistence/rejection, provider config tri-state merge/load validation, settings provider payloads, and CLI config-set path classification.
- Focused Go verification passed: `go test ./internal/config ./internal/settings ./internal/api/core ./internal/session ./internal/cli ./internal/workspace -count=1` (2582 passed in 6 packages).
- Updated session create dialog layout/model/reasoning controls and settings providers editor/cards for supported models + reasoning support.
- Added extension Host API `sessions/create` params for model and reasoning_effort.
- Regenerated OpenAPI/web generated types and CLI docs.
- Updated runtime provider/config docs.
- Focused verification passed after web/docs/extension work:
  - `go test ./internal/config ./internal/settings ./internal/api/core ./internal/session ./internal/cli ./internal/workspace ./internal/extension ./internal/extension/contract -count=1` (3120 passed in 8 packages).
  - `bunx vitest run` for session dialog/hook and settings provider route/hook tests (41 passed in 4 files).
- Fixed Go lint copy warnings from heavier `ProviderItem` by switching conversion/clone/find loops to pointer/indexing.
- Added final transport/settings guard tests for invalid runtime overrides and explicit empty/false settings payloads.
- Final `make verify` passed. It emitted the existing Vite chunk-size warning and macOS linker warning, with exit code 0.
  Now:
- Final response and goal closure.
  Next:
- None.
  Open questions (UNCONFIRMED if needed):
- None.
  Working set (files/ids/commands):
- .codex/ledger/2026-05-06-MEMORY-session-overrides-review.md
- .codex/plans/session-runtime-overrides-hardening.md
- internal/api/contract/contract.go
- internal/api/core/\*
- internal/session/\*
- internal/config/provider.go
- internal/cli/session.go
- internal/extension/contract/host_api.go
- internal/extension/host_api.go
- packages/site/content/runtime/cli-reference/session/new.mdx
- packages/site/content/runtime/core/configuration/config-toml.mdx
- packages/site/content/runtime/core/agents/providers.mdx
