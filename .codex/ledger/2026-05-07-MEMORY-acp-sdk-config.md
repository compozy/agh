Goal (incl. success criteria):

- Execute provider-model-catalog Task 06: upgrade ACP SDK to v0.12.2, capture ACP session config options, prefer set_config_option for model/reasoning when present, preserve legacy set_model fallback, expose named contract payload, add required tests, run verification, update tracking, and create one local commit if clean.

Constraints/Assumptions:

- Use RTK prefix for shell commands.
- Do not run destructive git commands.
- Use cy-workflow-memory before code edits; shared memory and task_06 memory have been read.
- Backend Go work requires AGH Go/test conventions and internal/CLAUDE.md.
- Contract changes require codegen co-ship.
- Conversation in Brazilian Portuguese; artifacts in English.

Key decisions:

- Active ACP session `configOptions` are session-scoped truth and must remain separate from the pre-session model catalog.
- Use `session/set_config_option` for model when a conservative model select option exists; legacy `session/set_model` is fallback only when config options are absent and legacy model state supports the requested model.
- Use `session/set_config_option` for reasoning effort only when a conservative reasoning select option exists and includes the requested value; never synthesize reasoning levels from catalog metadata.

State:

- Task 06 implementation is complete and committed in `cc1e31b6` (`feat: upgrade acp session config options`); pre-commit and post-commit `make verify` pass.

Done:

- Read RTK rule.
- Read root AGENTS.md/CLAUDE.md, internal/CLAUDE.md, workflow memory files, and required skill entry points.
- Read PRD docs, ADRs, workflow memory, relevant provider-model-catalog ledgers, ACP SDK v0.6.3/v0.12.2 source, Zed references, Harnss reference, and current AGH ACP/session/contract code.
- Captured pre-change focused test signal: `rtk go test ./internal/acp ./internal/session -run 'TestStartSetsPreferredSessionModelWhenProvided|TestStartResumeUsesLoadSession|TestCreateOpensStoreRegistersSessionAndActivates' -count=1` passed.

Now:

- Final response.

Next:

- None for Task 06.

Open questions (UNCONFIRMED if needed):

- None currently.

Working set (files/ids/commands):

- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_06.md`
- `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md`
- `go.mod`, `go.sum`
- `internal/acp/*`
- `internal/session/*`
- `internal/api/contract/*`
- `internal/api/core/*`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- Commit: `cc1e31b6` (`feat: upgrade acp session config options`)
- Verification: `rtk go test ./internal/acp ./internal/session ./internal/api/contract ./internal/api/core -count=1` passed 1454 tests; `rtk make verify` passed pre-commit and post-commit.
