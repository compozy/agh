Goal (incl. success criteria):

- Execute provider-model-catalog Task 1 end-to-end: hard-cut old provider model config fields, add nested model config/catalog/discovery shape, update contracts/generated web types/minimal consumers, tests, verification, tracking, and one local commit.

Constraints/Assumptions:

- Must use RTK for shell commands.
- Must use workflow memory files before editing and before finishing.
- Must use cy-workflow-memory, cy-execute-task, cy-final-verify; Go/test/contract/web skills as applicable.
- No destructive git commands without explicit user permission.
- `make verify` is the blocking completion gate; automatic commit only after clean verification and self-review.
- Artifacts/code/docs in English; conversation in Brazilian Portuguese.

Key decisions:

- Ledger created for this agent session at `.codex/ledger/2026-05-07-MEMORY-provider-config-hard-cut.md`.
- Preserve ACP session `SupportedModels`/`supported_models` where it represents active session capabilities; hard cut targets provider config/settings/session provider option payload fields.
- Existing modified worktree files are treated as user/branch state; do not restore or clean.

State:

- Task 01 complete with local commit and post-commit verification.

Done:

- Read RTK instructions.
- Read workflow shared memory and Task 1 memory.
- Read required workflow, Go/test/contract/web skills and repository/PRD/ADR guidance.
- Captured pre-change residue signal: old provider config fields remain across config/settings/API/generated/web.
- Implemented nested provider model config/catalog source/discovery config in `internal/config`.
- Updated repo-root `config.toml` to `[providers.<id>.models] default = ...`.
- `rtk go test ./internal/config`: 604 passed.
- Updated settings/API/CLI/workspace/session/situation residues for nested `Models`.
- `rtk go test ./internal/settings ./internal/api/core ./internal/api/contract ./internal/cli ./internal/workspace ./internal/session ./internal/situation`: 2150 passed.
- Updated minimal web settings/session/workspace consumers and fixtures away from old provider flat fields.
- Ran `rtk make codegen`, `rtk make codegen-check`, `rtk make bun-typecheck`, focused web settings/session tests, backend expanded focused tests, and full `rtk make verify`.
- Fixed first full-verify residue in `internal/testutil/e2e/config_seed_test.go` and lint cleanup issues.
- Self-review found `ProviderModelsConfig.Curated` explicit empty slice handling; fixed `providerModelsConfigIsZero` to distinguish nil from empty and added coverage.
- Fresh `rtk go test ./internal/config`: 622 passed.
- Fresh `rtk make verify`: exit 0. Non-blocking warnings only: Node `NO_COLOR`/`FORCE_COLOR`, Vite chunk-size, macOS linker `-bind_at_load`.
- Updated workflow memory and task tracking.
- Created local commit `0ff846d4 refactor: hard cut provider model config`.
- Post-commit `rtk make verify`: exit 0 with the same non-blocking warning classes.

Now:

- Final response.

Next:

- None for Task 01.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- Workflow memory: `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`, `.compozy/tasks/provider-model-catalog/memory/task_01.md`.
- Task docs: `.compozy/tasks/provider-model-catalog/task_01.md`, `_tasks.md`, `_techspec.md`, ADRs.
