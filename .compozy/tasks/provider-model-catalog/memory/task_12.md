# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build release-grade QA artifacts for the provider model catalog program (Tasks 01-11) under `.compozy/tasks/provider-model-catalog/qa/`.
- Cover every TechSpec safety invariant, ADR decision, public surface, and failure mode with concrete test cases that Task 13 can execute without inventing scenarios.

## Important Decisions
- Treated Task 11 regression coverage as baseline evidence; TC-FUNC/INT cases reuse those invariants instead of rediscovering parity/redaction/concurrency behavior.
- Real-scenario coverage split into TC-SCEN-001 (operator + web) and TC-SCEN-002 (agent + CLI/HTTP/UDS/Host API) so that artifacts produced in TC-SCEN-001 (the curated `manual-gpt` row) are reused in TC-SCEN-002, satisfying auditor C10.
- `MODELCATALOG_LIVE=1` annex documented in test plan + verification report template; default runs use stub HTTP servers and fake subprocesses, opt-in only for real-provider boundaries.
- `make verify` deliberately deferred to Task 13. Task 12 is documentation-only; running the gate here would fail on pre-existing unrelated worktree modifications that predate this task.

## Learnings
- HTTP and UDS register native catalog routes via `/api/providers/*catalog_path` dispatcher (not standard Gin params); UDS deliberately omits `/api/openai/v1/models`. TC-INT-002, TC-INT-003, TC-INT-004 anchor on those exact routes.
- Docs vitest is `packages/site/lib/__tests__/provider-model-catalog-docs.test.ts`; TC-REG-002 references it via `bun run test -- provider-model-catalog-docs`.
- Provider models CLI surface lives in `internal/cli/provider_models.go` + `internal/cli/client_provider_models.go`; structured JSON output is the agent-manageability contract (TC-INT-003).
- Migration registry is at v23 (after v22 `memv2_memory_events`); TC-INT-001 enforces append-only invariant + reopen-after-restart parity.

## Files / Surfaces
- Created `qa/test-plans/00-coverage-matrix.md` (mapping invariants/ADRs/tasks/surfaces to TC IDs).
- Created `qa/test-plans/provider-model-catalog-test-plan.md` (master plan, charter, env, exit criteria, verification commands).
- Created `qa/test-plans/provider-model-catalog-regression.md` (tiered execution suite, P0/P1 gating).
- Created 33 test-case files under `qa/test-cases/` (SMOKE-001; TC-FUNC-001..015; TC-INT-001..006; TC-PERF-001..002; TC-SEC-001..002; TC-UI-001..003; TC-REG-001..002; TC-SCEN-001..002).
- Created `qa/issues/BUG-NNN-template.md` (Task 13 bug template).
- Created `qa/verification-report-template.md` (Task 13 close-out template).
- Did not modify production code, generated contracts, web sources, docs, or migration registry.

## Errors / Corrections
- None.

## Ready for Next Run
- Task 13 should bootstrap an isolated lab via `agh-qa-bootstrap`, execute the tiered suite in `provider-model-catalog-regression.md`, file BUGs from the template, and close out by renaming `verification-report-template.md` to `verification-report.md` populated with command evidence.
