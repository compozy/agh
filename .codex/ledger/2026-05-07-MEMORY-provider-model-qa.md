Goal (incl. success criteria):

- Execute provider-model-catalog Task 13 real-scenario QA end-to-end.
- Success requires isolated QA bootstrap, planned QA gates/scenarios, browser/runtime E2E evidence, reproduced BUG files plus regression fixes if any, `qa/verification-report.md`, clean `make verify`, self-review, tracking updates, and one local commit if there are commit-worthy Task 13 changes.

Constraints/Assumptions:

- Use RTK for shell commands.
- Do not run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`).
- Worktree is dirty with unrelated web/packages/ui files and many untracked existing artifacts; do not touch or stage unrelated changes.
- Use workflow memory files before edits and before finish.
- Conversation in BR-PT; artifacts in English.
- Task requires `agh-qa-bootstrap`, `real-scenario-qa`, `qa-execution`, `agh-worktree-isolation`, `cy-execute-task`, `cy-workflow-memory`, and `cy-final-verify`.

Key decisions:

- Fresh QA lab is required for this independent QA pass.
- Task 12 QA artifacts are the execution contract; Task 13 should not invent new scope beyond defects/follow-ups.

State:

- Task 13 QA execution is blocked, not complete.
- Final `make verify` passed after implemented fixes, but release-grade QA remains blocked by BUG-002, BUG-005, and missing live provider-backed session evidence.

Done:

- Read RTK, workflow shared memory, task_13 memory, root AGENTS/CLAUDE guidance, internal/web guidance, required skill docs, task_13/task_12/\_tasks, ADRs, QA plans, and QA test cases.
- Scanned relevant provider-model-catalog ledgers for cross-agent awareness.
- Captured initial worktree state and branch (`fix-migrations`, HEAD `2debf0cf`).
- Created fresh QA lab with `agh-qa-bootstrap`:
  - manifest: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`
  - lab root: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab`
  - `AGH_HOME`: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`
  - base URL: `http://127.0.0.1:62444`
  - UDS: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/aghd.sock`
  - tmux socket: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/tmux-bridge.sock`
- Ran initial artifact/focused gates: `make build`, `make codegen-check`, `make bun-typecheck`, `make bun-test`, focused Go race gate with `-parallel=4`, `make codegen`, `make cli-docs`, and provider-model-catalog docs vitest passed.
- Filed `.compozy/tasks/provider-model-catalog/qa/issues/BUG-001-generated-contracts-cli-reference-drift.md` after TC-FUNC-015/TC-REG-002 produced a generated artifact drift signal; final generator reruns are idempotent and no generated diff remains.
- Started the isolated daemon, seeded `providers.codex.models.curated` with `manual-gpt`, restarted the daemon, refreshed config source, and proved HTTP/UDS native list parity for the same catalog state.
- Filed `.compozy/tasks/provider-model-catalog/qa/issues/BUG-002-openai-models-auth-not-enforced.md` after TC-SEC-002 reproduced missing auth enforcement: no-auth and bad-bearer calls to `/api/openai/v1/models?provider_id=codex` returned HTTP 200 catalog data.
- Filed/fixed BUG-003 for runtime E2E ACP helper interface drift; `make test-e2e-runtime` now passes.
- Filed/fixed BUG-004 for stale web E2E provider selector assumptions; `make test-e2e-web` now passes.
- Filed BUG-005 for workspace-scoped provider model metadata not being projected into daemon-backed session catalog APIs; it remains open.
- Wrote `.compozy/tasks/provider-model-catalog/qa/verification-report.md`.
- Ran final `make verify`: first attempt failed on `internal/cli/config.go` `goconst`; added `configDefaultKey`, `make lint` passed, and `make verify` rerun passed.
- Ran real-scenario audit; it failed with expected blocker C9 because no live provider-backed session evidence exists.
- Updated Task 13 tracking as partial: 13.1-13.4 and 13.6 checked, 13.5 open; master `_tasks.md` remains pending.
- Stopped the isolated daemon; final status reports `stopped` with 0 active sessions.

Now:

- Perform final self-review/status inspection and report the blocked QA outcome without creating a commit.

Next:

- Follow-up work should resolve BUG-002 and BUG-005 or revise the accepted contracts, then rerun Task 13 evidence including live-provider-backed session proof when credentials are available.

Open questions (UNCONFIRMED if needed):

- BUG-002 fix path is unresolved: current code has no generic HTTP API bearer-token authority, so a complete fix likely needs a dedicated auth design decision.
- BUG-005 fix path is unresolved: workspace-aware catalog projection requires API/contract/web/source-identity design.
- Live-provider credentials were not available in this isolated lab.

Working set (files/ids/commands):

- PRD dir: `.compozy/tasks/provider-model-catalog`
- Task file: `.compozy/tasks/provider-model-catalog/task_13.md`
- Master tasks: `.compozy/tasks/provider-model-catalog/_tasks.md`
- Workflow memory: `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`, `.compozy/tasks/provider-model-catalog/memory/task_13.md`
- QA report target: `.compozy/tasks/provider-model-catalog/qa/verification-report.md`
- QA audit report: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/qa-audit-report.md`
- Final verify log: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/final-make-verify-rerun.log`
