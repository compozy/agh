Goal (incl. success criteria):

- Execute provider-model-catalog Task 11 end-to-end: cross-surface regression hardening across config, catalog, HTTP, UDS, CLI, Host API, web fixtures/types/docs, redaction, refresh concurrency/timeout, tracking updates, verification, and one local commit.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory before editing code and before finishing.
- Must follow `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, repository Go/test/debug/concurrency skills, and AGH instructions.
- Must read AGENTS/CLAUDE guidance, PRD docs, `_techspec.md`, `_tasks.md`, Task 11, and ADRs before implementation.
- No destructive git commands without explicit user permission.
- `make verify` is required before completion and before/after commit.
- Conversation in Brazilian Portuguese; code/docs/artifacts in English.

Key decisions:

- Ledger created for this agent session at `.codex/ledger/2026-05-07-MEMORY-cross-surface-hardening.md`.

State:

- Complete; final response pending.

Done:

- Read RTK instructions.
- Scanned `.codex/ledger/` and read relevant provider-model-catalog ledgers for prior hard-cut/surface/extension tasks.
- Loaded `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify` skill entrypoints.
- Read workflow shared memory and Task 11 memory.
- Read root/internal/web/site AGENTS/CLAUDE guidance, required Go/test/debug/concurrency skills, Task 11, `_tasks.md`, `_techspec.md`, and ADR-001..003.
- Recorded Task 11 objective and initial known blocker in workflow task memory.
- Re-read ledger/workflow memory after context compaction.
- Captured baseline hard-cut scan and deterministic daemon refresh failure.
- Audited redaction/conversion, model catalog refresh coalescing, daemon refresh lifetime, API/CLI/Host API, and web fixture surfaces.
- Patched daemon refresh timeout to use duration-based detached context deadline.
- Added defensive redaction at API and Host API model catalog projection boundaries.
- Added hard-cut residue guard, redaction tests, refresh concurrency/SQLite contention tests, transport/OpenAI/CLI parity checks, ACP mock config option fixture support, and web E2E fixture/model override assertions.
- Narrow Go and Vitest checks for modified surfaces pass after fixing guard build-artifact exclusions and concurrency-test timing.
- Focused Task 11 gates pass: targeted Go suite, `make bun-typecheck`, `make bun-test`, `make web-build`, and `make codegen-check`.
- Full `make verify` passes before tracking/commit.
- `internal/modelcatalog` targeted coverage is 80.8% total statements.
- Workflow shared and task memory updated with resolved daemon refresh risk and Task 11 verification handoff.
- Task 11 tracking file and master `_tasks.md` status updated to completed.
- Local commit created: `7566e79d test: harden provider model catalog regressions`.
- Post-commit `make verify` passes.

Now:

- Send final summary and verification report.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-07-MEMORY-cross-surface-hardening.md`
- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_11.md`
- `.compozy/tasks/provider-model-catalog/task_11.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `.compozy/tasks/provider-model-catalog/_techspec.md`
- `.compozy/tasks/provider-model-catalog/adrs/`
- `internal/daemon/model_catalog.go`
- `internal/modelcatalog/redact.go`
- `internal/api/core/model_catalog*.go`
- `internal/extension/host_api_models.go`
