# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Produce the release-grade QA dossier for `tools-refac` covering tasks 01-11 across tool / CLI / HTTP / UDS / hosted MCP / codegen / site / web surfaces. The dossier must let task_13 execute without re-scoping the feature.

## Important Decisions

- Adopted the Hermes QA dossier shape: 5 plan files under `qa/test-plans/` plus per-domain TCs under `qa/test-cases/` (`TC-FUNC-*`, `TC-INT-*`, `TC-SEC-*`, `TC-AUT-*`, `TC-REG-*`, `TC-UI-*`, plus a `TC-AUDIT-001` self-check).
- Pinned `qa-output-path=.compozy/tasks/tools-refac` so task_13 runs against this root.
- Treated the autonomy hard cut and the redaction sweep as smoke-lane P0 stop conditions in the regression suite.
- Centralized cross-channel raw-`claim_token` / MCP secret grep procedure in `tools-refac-redaction-suite.md` so every TC that touches autonomy, MCP auth, hooks, automation, extensions, or config can reference it without duplicating the procedure.
- Encoded codegen + CLI docs + site build + web Vitest verification in a dedicated dossier (`tools-refac-codegen-and-docs.md`) referenced by TC-REG-001..005 and TC-UI-001.
- Recorded `make bun-typecheck` + `make bun-test` as the canonical web/site gates (matching the Verify pipeline) rather than `make web-*` workspace-only commands.
- Treated `claim_token_hash` as the only allowed observability metadata that survives the redaction sweep — every grep step lists it as the explicit exception.

## Learnings

- The Hermes regression-suite template (smoke → targeted → full) is the right shape for a contract-heavy/agent-manageability redesign. Web UI screenshots only land in TC-UI-001 because the redesign is contract-heavy and not UI-led.
- The traceability matrix doubles as the "unit-test" requirement audit for QA-only tasks: every implementation task gets a scenario + a regression hot spot, and the audit table is what TC-AUDIT-001 verifies.
- The redaction suite must enumerate channels (CLI human/JSON, HTTP/UDS, hosted MCP frames, SSE, daemon log, observe events, memory, generated OpenAPI/TS, site docs) explicitly; "grep across logs" is not specific enough.
- The hosted MCP bind safety case (TC-SEC-006) requires a test seam to mint nonces and to spawn a foreign-UID/foreign-binary peer; without that, task_13 cannot prove fail-closed behavior.

## Files / Surfaces

Produced under `.compozy/tasks/tools-refac/qa/`:

- `test-plans/tools-refac-test-plan.md`
- `test-plans/tools-refac-regression.md`
- `test-plans/tools-refac-traceability.md`
- `test-plans/tools-refac-codegen-and-docs.md`
- `test-plans/tools-refac-redaction-suite.md`
- `test-cases/TC-FUNC-001.md` … `TC-FUNC-008.md`
- `test-cases/TC-INT-001.md` … `TC-INT-006.md`
- `test-cases/TC-SEC-001.md` … `TC-SEC-006.md`
- `test-cases/TC-AUT-001.md` … `TC-AUT-006.md`
- `test-cases/TC-REG-001.md` … `TC-REG-005.md`
- `test-cases/TC-UI-001.md`
- `test-cases/TC-AUDIT-001.md`
- `verification-report.md` (scaffold, reserved for task_13)

Tracking updates:

- `task_12.md` flipped to `status: completed`, subtasks/tests checkboxes flipped to `[x]`.
- `_tasks.md` row 12 set to `completed`.

## Errors / Corrections

- The shared workflow memory cited a previously committed `qa/` evidence set (commit `8c5d78d7`); that commit is not present in the current branch and the `qa/` directory was empty under `issues/`, `logs/`, `screenshots/`. Treated the shared memory as historical hint and produced the dossier from scratch under the same `qa-output-path`.

## Ready for Next Run

- Task_13 (Real-Scenario QA Execution) can consume this dossier directly. Pass `qa-output-path=.compozy/tasks/tools-refac` to `qa-execution`.
- Capture bootstrap manifest under `qa/logs/<TC-ID>/` and reuse only across consecutive cases of the same active QA pass.
- Begin with the smoke lane (TC-FUNC-001 → TC-INT-003 → TC-AUT-001 → TC-SEC-001 → TC-SEC-002 → TC-FUNC-004 → TC-REG-001) before any targeted lane.
