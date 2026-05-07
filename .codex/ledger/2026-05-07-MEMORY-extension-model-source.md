Goal (incl. success criteria):

- Implement provider-model-catalog Task 08: extension `model.source` manifest/service contract, `models/list` AGH-to-extension calls, Host API `models/list|refresh|status`, capability gating/ceilings, validation/persistence via daemon-owned model catalog, required tests, tracking updates, and one local commit after clean verification.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory files before edits and before finish.
- Must follow `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- Must read AGENTS/CLAUDE guidance, provider-model-catalog PRD docs, `_techspec.md`, `_tasks.md`, task_08, and ADRs before implementation.
- No destructive git commands without explicit user permission.
- `make verify` is required before completion and before/after commit.
- Conversation in Brazilian Portuguese; code/docs/artifacts in English.

Key decisions:

- Reuse Task 07's daemon catalog service and contract payloads; Host API model methods must return daemon-owned projections/status, not raw extension rows.
- Extension rows are only input to `internal/modelcatalog`; daemon validation, persistence, merge policy, and public projections remain authoritative.
- Extension source IDs are derived through `modelcatalog.SourceKindExtensionID`, producing `extension:<slug>` and rejecting names/rows that fail the TechSpec source-id rule.
- Generated TypeScript SDK contracts are included because `make codegen-check` required regeneration after Host API/SDK contract changes.

State:

- Task complete in local commit `fef35196 feat: add extension model source contract`; post-commit `make verify` passed.

Done:

- Read RTK.
- Scanned `.codex/ledger/` for cross-agent awareness and read relevant provider-model-catalog ledgers/memories.
- Read workflow shared memory and task_08 memory.
- Loaded required workflow, Go, and test skill entrypoints plus canonical references.
- Read root AGENTS/CLAUDE, `internal/CLAUDE.md`, Task 08, `_tasks.md`, TechSpec extension/source sections, and ADR-001..003.
- Added `model.source`, `models/list`, Host API `models/list|refresh|status`, capability mappings, manifest/source-id validation, extension source adapter, daemon wiring, and generated contract updates.
- Added unit/integration coverage for manifest validation, capability/marketplace ceilings, Host API daemon projection/status, validated persistence, malformed rows, denied source, subprocess success, and stale fallback.
- Focused validation passed: test convention helper on new/conformed tests; `go test ./internal/extension/...`; `go test ./internal/extension/... -race`; `go test ./internal/modelcatalog ./internal/daemon`; `make codegen-check`; `make lint`; focused coverage for new handlers/adapters >=80%.
- Full `make verify` passed before task tracking completion.
- Updated `.compozy/tasks/provider-model-catalog/task_08.md`, `_tasks.md`, task memory, and shared workflow memory for Task 08 completion.
- Created local commit `fef35196 feat: add extension model source contract`.
- Post-commit `make verify` passed.

Now:

- Final response.

Next:

- None for Task 08.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_08.md`
- `.compozy/tasks/provider-model-catalog/task_08.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `internal/extension/{protocol,contract,host_api*,model_source*,capability*,manifest*,manager_model_source_test.go,tool_runtime.go}`
- `internal/modelcatalog/source_id.go`
- `internal/daemon/{boot.go,daemon.go,model_catalog.go}`
- `sdk/typescript/src/generated/contracts.ts`
- Tracking updated but not for commit: `.compozy/tasks/provider-model-catalog/{task_08.md,_tasks.md,memory/}`
