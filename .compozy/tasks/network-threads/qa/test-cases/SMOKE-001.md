## SMOKE-001: Network Threads QA Readiness

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** Smoke readiness only

---

### Objective

Confirm that the repository, generated artifacts, docs guardrails, and E2E harness commands are ready for behavior-first QA. This case is entry criteria only and must not be reported as release-grade evidence by itself.

### Preconditions

- [ ] Worktree state is recorded before QA execution.
- [ ] `.compozy/tasks/network-threads/qa/test-plans/network-threads-test-plan.md` exists.
- [ ] `.compozy/tasks/network-threads/qa/test-cases/` contains the P0/P1 cases.
- [ ] QA bootstrap will use a fresh lab unless a same-session healthy manifest is explicitly reused.

### Test Steps

1. **Inspect current workflow status**
   - Input: `sed -n '1,220p' .compozy/tasks/network-threads/state.yaml`
   - **Expected:** `task_18` and `task_19` state is clear; completed implementation tasks are not treated as pending implementation work.

2. **Run or schedule baseline gate discovery**
   - Input: `make verify`
   - **Expected:** Command is the canonical full gate for final completion. If it fails, QA execution records the first failing stage and does not continue as if release readiness exists.

3. **Confirm targeted harness commands exist**
   - Input: `make test-e2e-runtime` and `make test-e2e-web`
   - **Expected:** Both commands are available and are listed in the regression suite for targeted behavior validation.

4. **Scan active docs/contracts for legacy examples**
   - Input: repo-defined docs/tests or `rg` scans for active `interaction_id`, `kind:"direct"`, `--interaction-id`, and old send flags outside archived/provenance paths.
   - **Expected:** Active docs, generated contracts, CLI examples, prompts, fixtures, and tests do not teach old protocol behavior except in explicit negative assertions.

5. **Prepare browser validation path**
   - Input: QA bootstrap manifest or discovered dev-server command.
   - **Expected:** `AGH_WEB_API_PROXY_TARGET` is available before `make web-dev` when the daemon runs on an isolated port; browser-use is primary and `agent-browser` is fallback.

### Behavioral Evidence

- Operator journey: readiness gate before behavior-first testing.
- Live agent/LLM behavior: not applicable; this is smoke readiness only.
- Artifacts produced and used: QA plan and test cases.
- Cross-surface assertions: not final proof; only checks that surfaces can be exercised.

### Disruption Probes

- If a command is missing or stale, QA execution must stop, file a bug, and fix the root cause before P0 journeys.
- If active docs/contracts still show legacy fields, QA execution must file a hard-cut regression.

### Related Test Cases

- TC-SCEN-001
- TC-SCEN-002
- TC-SCEN-003
- TC-INT-001

