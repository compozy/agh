# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Generate reusable QA planning artifacts under `.compozy/tasks/unified-capabilities/qa/` for task_10.
- Cover backend schema/digesting, `kind:"capability"` transfer + lifecycle, discovery/API contract alignment, `web/` peer-detail UX, and `packages/site` protocol/runtime docs.
- Do not execute the flows in this task; planning, priorities, traceability, and artifact layout only.

## Important Decisions

- Use `qa-output-path=.compozy/tasks/unified-capabilities` and keep all planning artifacts under the shared `qa/` subtree that task_10 will reuse.
- Create one feature-level test plan, one consolidated regression-suite definition, and a focused set of manual cases that map the changed seams rather than a generic smoke pack.
- Keep backend behavior as P0 and operator/public-surface behavior as P1: `TC-INT-001` through `TC-INT-004` are the hard gates, while `TC-UI-001`, `TC-REG-001`, and `TC-REG-002` complete the changed `web/` and `packages/site` surfaces.

## Learnings

- The repo currently has no existing `.compozy/tasks/unified-capabilities/qa/` tree, so task_09 must establish the artifact layout from scratch.
- The worktree contains unrelated web and task-file changes; this task must stay scoped to the unified-capabilities QA artifacts, workflow memory, and tracking files.
- Current backend/API/frontend/doc surfaces converge on typed capability payloads and `kind:"capability"`; the plan should target those typed surfaces, not raw `ext` blobs in API-visible payloads.
- The artifact set now exists and is centered on two plan docs plus seven manual cases:
  - `qa/test-plans/unified-capabilities-test-plan.md`
  - `qa/test-plans/unified-capabilities-regression.md`
  - `qa/test-cases/TC-INT-001.md` through `TC-INT-004.md`
  - `qa/test-cases/TC-UI-001.md`
  - `qa/test-cases/TC-REG-001.md`
  - `qa/test-cases/TC-REG-002.md`

## Files / Surfaces

- `.compozy/tasks/unified-capabilities/qa/test-plans/`
- `.compozy/tasks/unified-capabilities/qa/test-cases/`
- `.compozy/tasks/unified-capabilities/qa/issues/`
- `.compozy/tasks/unified-capabilities/qa/screenshots/`
- `.compozy/tasks/unified-capabilities/_techspec.md`
- `.compozy/tasks/unified-capabilities/task_01.md` through `task_10.md`
- `.compozy/tasks/unified-capabilities/adrs/adr-001.md` through `adr-003.md`
- `docs/rfcs/003_agh-network-v0.md`
- `docs/agents/capabilities.md`
- `internal/api/contract/contract.go`
- `web/src/systems/network/components/network-peer-detail-panel.tsx`
- `web/src/hooks/routes/use-network-page.ts`
- `web/src/systems/network/adapters/network-api.ts`
- `packages/site/content/protocol/{message-kinds,capability-discovery,examples,meta}.mdx`
- `packages/site/content/runtime/core/agents/{capabilities,definitions,meta}.mdx`

## Errors / Corrections

- `make verify` failed after the QA artifacts were written because `web/src/systems/session/hooks/use-session-actions.ts` has a TypeScript error at line 44 (`session.workspace_id` is `string | undefined` where `string` is required).
- That file is already modified in the worktree and is unrelated to task_09, so task tracking and commit steps are blocked until the user decides whether this unrelated web change should be fixed here.

## Ready for Next Run

- Artifact content is written and structural checks passed.
- Remaining work is blocked on the unrelated verification failure in `web/src/systems/session/hooks/use-session-actions.ts`; do not mark task_09 complete or commit until that repo gate is resolved and rerun.
