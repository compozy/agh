# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Task 17 end-to-end: consume Agent Soul/Heartbeat QA artifacts, run smoke/P0/P1 operator flows first, document and fix confirmed defects, persist QA evidence under `.compozy/tasks/agent-soul/qa/`, and finish only after `make verify` passes.

## Important Decisions
- Task 16 dependency correction: `task_16.md` is marked completed, but the required `qa/test-plans/` and `qa/test-cases/` artifacts were absent and `_tasks.md` still listed Task 16 as pending. Reconstruct the missing QA report artifacts in the expected directories before executing Task 17 so QA execution has traceable cases.
- Use a fresh isolated QA lab for this run: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-164846-213590-lab`. Mirror bootstrap manifest/env into the workflow QA root while keeping the lab's isolated `AGH_HOME`, HTTP port, UDS path, and provider home.

## Learnings
- Existing QA directory only contained peer-review artifacts before this run; no executable smoke/P0/P1 test cases were present.
- Agent Soul/Heartbeat MVP QA is CLI/API/runtime-centered; Web evidence is generated-contract and guard-test evidence because no Web editor is in scope.
- Baseline `make verify` passed before scenario execution; the log is `.compozy/tasks/agent-soul/qa/evidence/baseline-make-verify.log`.
- P0 TC-SCEN-001 found a CLI boundary defect: `agh agent soul write` rejects omitted `--expected-digest` for first create before the daemon/service can apply the TechSpec create semantics. Heartbeat write has the same CLI pattern while the service supports missing-file creation with an empty expected digest.
- P0 TC-SCEN-002 found a Heartbeat dry-run response defect: dry-run does not persist wake state/events, but the operator response includes generated wake/prompt identifiers for artifacts that do not exist.
- P0 TC-SCEN-002 found a missing-session wake defect: daemon composition passed `session.ErrSessionNotFound` through to Heartbeat wake instead of normalizing it to the Heartbeat missing-health sentinel that produces a closed `session_not_found` decision.
- Focused test-shape helper documented by the skill is not at `scripts/check-test-conventions.py`; the repo-local helper path is `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`.
- Focused race regressions now pass for the three confirmed defects:
  - `CGO_ENABLED=1 go test -race ./internal/cli -run 'TestAgent(Soul|Heartbeat)Commands|TestCLIAgentAuthoredContextIntegration' -count=1`
  - `CGO_ENABLED=1 go test -race ./internal/heartbeat -run TestManagedWakeServiceDecision -count=1`
  - `CGO_ENABLED=1 go test -race ./internal/daemon -run TestHeartbeatWakeHealthReader -count=1`
- Final pre-verify P0 reruns passed through public CLI/API surfaces against the isolated daemon: Soul inspect/history/stale-CAS, Heartbeat inspect/status/session-health/session-inspect/dry-run wake/missing-session wake.
- Full `make verify` exited 0 after fixes. Evidence: `.compozy/tasks/agent-soul/qa/evidence/final-make-verify.log`. The report records non-fatal inherited environment/toolchain warnings separately from lint/test errors.
- Final QA report is `.compozy/tasks/agent-soul/qa/verification-report.md`.

## Files / Surfaces
- `.compozy/tasks/agent-soul/qa/`
- `.compozy/tasks/agent-soul/qa/test-plans/agent-authored-context-test-plan.md`
- `.compozy/tasks/agent-soul/qa/test-plans/agent-authored-context-regression-suite.md`
- `.compozy/tasks/agent-soul/qa/test-cases/SMOKE-001.md`
- `.compozy/tasks/agent-soul/qa/test-cases/TC-SCEN-001.md`
- `.compozy/tasks/agent-soul/qa/test-cases/TC-SCEN-002.md`
- `.compozy/tasks/agent-soul/qa/test-cases/TC-REG-001.md`
- `.compozy/tasks/agent-soul/qa/test-cases/TC-REG-002.md`
- `.compozy/tasks/agent-soul/qa/test-cases/TC-REG-003.md`
- `.compozy/tasks/agent-soul/qa/bootstrap-manifest.json`
- `.compozy/tasks/agent-soul/qa/bootstrap.env`
- `.compozy/tasks/agent-soul/qa/evidence/baseline-make-verify.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-soul-write.json`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-001-cli-regression-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-002-heartbeat-dry-run-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-003-heartbeat-missing-session-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-final-soul-inspect-cli.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-final-soul-inspect-api.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-final-soul-stale-cas.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-final-soul-history.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-inspect-cli.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-status-cli.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-status-api.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-session-health.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-session-inspect.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-dry-run.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-missing-session.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-missing-session-api.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-missing-session-api.status`
- `.compozy/tasks/agent-soul/qa/evidence/final-make-verify.log`
- `.compozy/tasks/agent-soul/qa/verification-report.md`
- `.compozy/tasks/agent-soul/qa/issues/BUG-001-cli-create-requires-cas.md`
- `.compozy/tasks/agent-soul/qa/issues/BUG-002-heartbeat-dry-run-returns-synthetic-ids.md`
- `.compozy/tasks/agent-soul/qa/issues/BUG-003-heartbeat-missing-session-not-closed-decision.md`
- `internal/cli/authored_context.go`
- `internal/cli/authored_context_test.go`
- `internal/heartbeat/wake.go`
- `internal/heartbeat/wake_test.go`
- `internal/daemon/authored_context_runtime.go`
- `internal/daemon/heartbeat_wake_runtime_test.go`
- `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-164846-213590-lab`

## Errors / Corrections
- Correcting missing QA planning artifacts as a prerequisite to execution instead of skipping the Task 17 requirement to run generated smoke and P0/P1 regression cases.
- BUG-001: CLI write commands forced flag presence for create operations. Planned correction is to preserve trimming and explicit flag handling for update/delete/rollback while allowing write commands to pass an empty expected digest when the flag is omitted.
- BUG-002: Heartbeat dry-run decisions should evaluate gates without exposing event/prompt ids for non-persisted artifacts.
- BUG-003: Heartbeat wake should map daemon session-missing health reads to a closed `session_not_found` decision without changing direct session health API behavior.
- BUG-001 correction implemented in `internal/cli/authored_context.go` with CLI regression coverage.
- BUG-002 correction implemented in `internal/heartbeat/wake.go` with wake-service regression coverage.
- BUG-003 correction implemented in `internal/daemon/authored_context_runtime.go` with daemon boundary regression coverage.

## Ready for Next Run
- Next step is tracking update, self-review, one local commit, and post-commit `make verify`.
