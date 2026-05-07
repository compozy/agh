# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Task 13 real-scenario QA for the provider model catalog using Task 12 QA artifacts as the execution contract.
- Success requires a fresh isolated QA lab, daemon-served CLI/API/UDS/OpenAI/Host API checks, browser workflows, `make test-e2e-runtime`, `make test-e2e-web`, final `make verify`, `qa/verification-report.md`, and BUG/regression fixes for any reproduced defects.

## Important Decisions
- Use a fresh `agh-qa-bootstrap` lab for this independent QA pass; do not reuse older provider-model-catalog labs.
- Treat live-provider validation as opt-in/boundary evidence unless reachable credentials and provider prerequisites are proven during the run.
- Do not touch unrelated dirty worktree changes; stage only Task 13-owned deliverables/fixes if automatic commit becomes valid.

## Learnings
- Initial worktree is dirty before Task 13 starts, including unrelated tracked web/packages UI changes and many untracked prior task artifacts.
- `agh-qa-bootstrap` without a generic playbook is appropriate for this PRD-specific QA task because Task 12 already defines TC-SCEN provider-catalog journeys; the lab charter must be filled from those TC-SCEN cases before running the auditor.
- TC-FUNC-015/TC-REG-002 produced a generated artifact drift signal. Filed BUG-001; final reruns of `make codegen`, `make cli-docs`, and `make codegen-check` were idempotent, and final self-review showed no remaining generated diff.
- TC-SEC-002 reproduced a critical contract/runtime mismatch: `/api/openai/v1/models?provider_id=codex` returns HTTP 200 catalog data with no `Authorization` header and with `Authorization: Bearer bad-token`. Filed BUG-002. Root-cause scan shows no generic HTTP `/api/*` bearer-auth middleware or `HTTPConfig` token authority exists today; only loopback/CORS paths produce OpenAI-shaped errors.
- `make test-e2e-runtime` initially failed at integration-test compile time because daemon ACP helper agents did not implement the `acp-go-sdk` v0.12.2 `Agent` interface (`CloseSession` and related required methods). Filed BUG-003 and updated the affected integration helper agents with valid no-op lifecycle/config methods.
- `make test-e2e-web` initially failed in `session-provider-override.spec.ts` because the test still treated `session-create-provider-select` as a native `<select>` after the UI moved to `ProviderCommandSelect`. Filed BUG-004 and updated the E2E to assert/select through the shipped command-popover control, refresh the catalog, assert the empty workspace-only catalog state, and continue through manual model entry.
- The browser E2E refresh proved a product/API gap: workspace-scoped provider `models.curated` metadata is not projected into provider-scoped daemon catalog APIs. Filed BUG-005 as open follow-up because fixing it needs workspace-aware catalog contract design.
- Final `make verify` initially failed on a `goconst` finding in `internal/cli/config.go`; added `configDefaultKey` for the provider model default path and reran `make lint` plus `make verify` successfully.
- Real-scenario audit produced blocker C9 because the scenario contract requires one provider-backed live session and the isolated lab had no live provider credentials. This is recorded as BLOCKED, not PASS.
- Isolated daemon was stopped cleanly at the end; final status evidence is `qa/logs/daemon-status-final.json`.

## Files / Surfaces
- Planned Task 13-owned artifacts: `qa/verification-report.md`, possible `qa/issues/BUG-NNN.md`, task tracking files, this task memory, and any regression/code files required by reproduced bugs.
- Fresh bootstrap manifest: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`.
- Lab root: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab`; runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`; base URL: `http://127.0.0.1:62444`.
- BUG-001 artifact: `.compozy/tasks/provider-model-catalog/qa/issues/BUG-001-generated-contracts-cli-reference-drift.md`.
- BUG-002 artifact: `.compozy/tasks/provider-model-catalog/qa/issues/BUG-002-openai-models-auth-not-enforced.md`.
- BUG-003 artifact: `.compozy/tasks/provider-model-catalog/qa/issues/BUG-003-runtime-e2e-acp-test-agents-missing-close-session.md`.
- BUG-004 artifact: `.compozy/tasks/provider-model-catalog/qa/issues/BUG-004-web-e2e-provider-select-uses-stale-native-select-contract.md`.
- BUG-005 artifact: `.compozy/tasks/provider-model-catalog/qa/issues/BUG-005-workspace-provider-models-not-projected-into-session-catalog.md`.
- Verification report: `.compozy/tasks/provider-model-catalog/qa/verification-report.md`.
- Real-scenario audit report: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/qa-audit-report.md`.

## Errors / Corrections
- Smoke focused Go gate initially failed with widespread `context deadline exceeded` while bootstrapping fresh `globaldb` schemas under `go test -race` without a `-parallel` cap. Narrow reproductions for `./internal/store/globaldb` and `./internal/extension` passed with `-parallel=4`, matching the repository's `make test` race contract. Rerunning the focused gate with `-parallel=4` is the root-cause correction for the QA execution command, not a product-code workaround.

## Completion State
- Task 13 is not complete and should remain `pending`.
- Completed evidence: isolated bootstrap, planned gates, daemon-served CLI/HTTP/UDS/Host API checks, browser fallback/manual checks, `make test-e2e-runtime`, `make test-e2e-web`, generated artifact idempotence, final `make verify`, verification report, and real-scenario audit.
- Cleared/fixed findings: BUG-001 has no remaining generated diff; BUG-003 and BUG-004 are fixed in working tree.
- Open blockers: BUG-002, BUG-005, and missing live provider-backed ACP session proof.
- No automatic commit was created because the task is still blocked despite green `make verify`.

## Ready for Next Run
- Branch at start: `fix-migrations`; HEAD short SHA at start: `2debf0cf`.
