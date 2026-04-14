# Case Execution Matrix

Generated: 2026-04-14

Summary:
- PASS: 68
- FAIL: 1
- FAIL case: `TC-SEC-003` (`.compozy/tasks/core-tasks/issues/BUG-001.md`)

Common evidence:
- Unit/transport logs: `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/logs/case-suite-unit.json`, `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/logs/case-suite-unit.log`
- Integration logs: `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/logs/case-suite-integration.json`, `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/logs/case-suite-integration.log`
- Live CLI/API suite: `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/live/live-summary.json`
- Live security suite: `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/security/tc-sec-006-007-summary.json`, `tc-sec-003-*.txt/json`, `tc-sec-006-db-tables.txt`, `tc-sec-006-post-health.json`
- Performance suite: `.compozy/tasks/core-tasks/runtime/full-suite-20260414-160548/perf/perf-summary.json`

## Smoke
- `SMOKE-001` PASS — isolated daemon booted and exposed healthy task/observe surfaces. Evidence: `live/observe-health.http.json`, live daemon on `127.0.0.1:50512`.
- `SMOKE-002` PASS — global task creation via HTTP. Evidence: `live/http-create-primary.http.json`, `live/live-summary.json`.
- `SMOKE-003` PASS — task listing via HTTP. Evidence: `live/http-list-global.http.json`, `live/live-summary.json`.
- `SMOKE-004` PASS — task detail via HTTP with child/dependency expansion. Evidence: `live/http-get-primary.http.json`, `live/live-summary.json`.
- `SMOKE-005` PASS — task title update via PATCH. Evidence: `live/http-patch-primary.http.json`, `live/live-summary.json`.
- `SMOKE-006` PASS — enqueue and claim task run. Evidence: `live/cli-enqueue-run.stdout.json`, `live/cli-claim-run.stdout.json`, `live/live-summary.json`.
- `SMOKE-007` PASS — start and complete task run. Evidence: `live/cli-start-run.stdout.json`, `live/cli-complete-run.stdout.json`, `live/live-summary.json`.
- `SMOKE-008` PASS — cancel task tree and active runs. Evidence: `live/cli-cancel-root.stdout.json`, `live/cli-get-cancel-child.stdout.json`, `live/cli-get-cancel-grandchild.stdout.json`, `live/live-summary.json`.
- `SMOKE-009` PASS — CLI task list returns results. Evidence: `live/cli-list-workspace-scope.stdout.json`, `live/cli-list-ready.stdout.json`, `live/live-summary.json`.
- `SMOKE-010` PASS — observe projections return task metrics. Evidence: `live/observe-health.http.json`, `perf/perf-summary.json` (`TC-PERF-006`), `internal/observe` tests in `case-suite-unit.json`.

## Functional
- `TC-FUNC-001` PASS — `internal/task`: `TestManagerCreateTaskUsesTrustedActorContext`, integration `TestTaskManagerCreateTaskPersistsAgentSessionIdentity`.
- `TC-FUNC-002` PASS — workspace-scoped task creation covered by integration `TestTaskManagerChildAndDependencyFlowsPersistAudit` and live `live/cli-create-workspace-task.stdout.json`.
- `TC-FUNC-003` PASS — invalid scope binding rejected by `TestValidateScopeBinding/global_with_workspace`.
- `TC-FUNC-004` PASS — mutable field updates covered by `TestManagerUpdateTaskAllowsMutableOwnershipAndChannelFields`.
- `TC-FUNC-005` PASS — immutable field rejection covered by `TestValidateImmutableTaskFields/scope_immutable`, `workspace_id_immutable`, `parent_task_id_immutable`, `created_by_immutable`, `origin_immutable`.
- `TC-FUNC-006` PASS — valid child creation covered by `TestManagerCreateChildTaskEnforcesParentRulesAndEmitsAudit/global_parent_allows_workspace_child_and_emits_parent_event`.
- `TC-FUNC-007` PASS — max-depth creation covered by `TestGraphLimitGuards/depth_at_limit` and `perf/perf-summary.json` (`TC-PERF-003`).
- `TC-FUNC-008` PASS — direct-child limit covered by `TestGraphLimitGuards/direct_child_count_over_limit` and `perf/perf-summary.json` (`TC-PERF-003`).
- `TC-FUNC-009` PASS — valid dependency edge covered by `TestManagerAddAndRemoveDependencyReconcileStatusAndEvents` and integration `TestTaskManagerChildAndDependencyFlowsPersistAudit`.
- `TC-FUNC-010` PASS — self-dependency rejection covered by `TestDomainValidationHelpers/task_dependency_self_dependency`.
- `TC-FUNC-011` PASS — cycle rejection covered by `perf/perf-summary.json` (`TC-PERF-002` cycle and chain-cycle checks).
- `TC-FUNC-012` PASS — dependency-count limit covered by `TestGraphLimitGuards/dependency_count_over_limit` and `perf/perf-summary.json` (`TC-PERF-002`).
- `TC-FUNC-013` PASS — dependency removal/status reconciliation covered by `TestManagerAddAndRemoveDependencyReconcileStatusAndEvents`.
- `TC-FUNC-014` PASS — enqueue on ready task covered by integration `TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage` and live `live/cli-enqueue-run.stdout.json`.
- `TC-FUNC-015` PASS — claim queued run covered by integration `TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage` and live `live/cli-claim-run.stdout.json`.
- `TC-FUNC-016` PASS — start claimed run/dedicated session covered by integration `TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage`, daemon `TestTaskSessionBridgeStartTaskSessionUsesDedicatedSystemSessions`, and live `live/cli-start-run.stdout.json`.
- `TC-FUNC-017` PASS — complete running run covered by integration `TestTaskManagerRunLifecyclePersistsAndReconcilesAgainstStorage` and live `live/cli-complete-run.stdout.json`.
- `TC-FUNC-018` PASS — fail running run covered by `TestManagerBlockedExecutionAndFailureGuardrails` and `TestManagerGetTaskAndFailRunGuardrails`.
- `TC-FUNC-019` PASS — invalid transition rejection covered by `TestManagerRunLifecycleRejectsInvalidTransitions`.
- `TC-FUNC-020` PASS — attach-session and second-attach rejection covered by `TestManagerAttachRunSessionAndRetryLatestRunOutcome` and live `live/cli-attach-session-first.stdout.json`, `live/cli-attach-session-second.stderr.txt`.
- `TC-FUNC-021` PASS — idempotent enqueue semantics covered by `TestManagerNonHumanIdempotencyAndExecutionGuards` and `TestManagerNetworkPeerEnqueueRunUsesOriginScopedIdempotency`.
- `TC-FUNC-022` PASS — cancel queued task runs covered by `TestManagerCancelTaskPropagatesAcrossTree` and integration `TestTaskManagerCancelTaskTreePersistsCancellationAudit`.
- `TC-FUNC-023` PASS — cooperative then forced stop on running child runs covered by `TestManagerCancelTaskPropagatesAcrossTree`, integration `TestTaskManagerCancelTaskTreePersistsCancellationAudit`, and `perf/perf-summary.json` (`TC-PERF-004`).
- `TC-FUNC-024` PASS — cancel propagation to grandchildren covered by `TestManagerCancelTaskPropagatesAcrossTree`, integration `TestTaskManagerCancelTaskTreePersistsCancellationAudit`, and `perf/perf-summary.json` (`TC-PERF-004`).
- `TC-FUNC-025` PASS — cancel terminal task rejection covered by `TestManagerBlockedExecutionAndFailureGuardrails` (`CancelTask(completedTask)`).
- `TC-FUNC-026` PASS — oversized task metadata rejected by `TestPayloadSizeGuards/metadata_over_limit` and live `security/tc-sec-006-007-summary.json`.
- `TC-FUNC-027` PASS — oversized run result rejected by `TestPayloadSizeGuards/result_over_limit` and API mapping `TestStatusForTaskError/payload_too_large`.
- `TC-FUNC-028` PASS — oversized task-event payload rejected by `TestPayloadSizeGuards/payload_over_limit`.
- `TC-FUNC-029` PASS — orphaned claimed run re-queued on boot covered by `TestManagerRecoverRunOnBoot/claimed_run_requeues_and_records_recovery_event` and daemon `TestPlanTaskRunRecoveryClassifiesClaimedStartingRunning/claimed_without_session_requeues`.
- `TC-FUNC-030` PASS — orphaned running run marked failed on boot covered by `TestManagerRecoverRunOnBoot/running_run_fails_closed_when_the_attached_session_is_not_live` and daemon `TestPlanTaskRunRecoveryClassifiesClaimedStartingRunning/running_with_missing_session_fails`.

## Integration
- `TC-INT-001` PASS — HTTP create with server-derived identity. Evidence: `live/http-create-primary.http.json`, `TestHTTPTaskRoutesRoundTrip`.
- `TC-INT-002` PASS — HTTP list with filters. Evidence: `live/http-list-global.http.json`, `TestHTTPTaskRoutesRoundTrip`.
- `TC-INT-003` PASS — HTTP get detail payload. Evidence: `live/http-get-primary.http.json`, `TestHTTPTaskRoutesRoundTrip`.
- `TC-INT-004` PASS — immutable PATCH rejected with 400. Evidence: `live/http-patch-immutable.http.json`, `live/http-patch-immutable-identity.http.json`.
- `TC-INT-005` PASS — HTTP/UDS task-route parity covered by `TestHTTPTaskRoutesRoundTrip`, `TestHTTPTaskRunLifecycleRoutesRoundTrip`, `TestUDSTaskRoutesRoundTrip`, and `TestUDSTaskRunLifecycleRoutesRoundTrip`.
- `TC-INT-006` PASS — CLI task create via daemon API. Evidence: `live/cli-create-workspace-task.stdout.json`, `TestCLITaskCreateListGetIntegration`.
- `TC-INT-007` PASS — CLI filtered task list. Evidence: `live/cli-list-workspace-ready.stdout.json`, `live/cli-list-owner.stdout.json`, `TestCLITaskCreateListGetIntegration`.
- `TC-INT-008` PASS — CLI task cancel propagates to children. Evidence: `live/cli-cancel-root.stdout.json`, `live/cli-get-cancel-child.stdout.json`, `live/cli-get-cancel-grandchild.stdout.json`.
- `TC-INT-009` PASS — start run binds dedicated session. Evidence: `live/cli-start-run.stdout.json`, daemon `TestTaskSessionBridgeStartTaskSessionUsesDedicatedSystemSessions`, `TestBootWiresTaskRuntimeWithDedicatedSessionBridge`.
- `TC-INT-010` PASS — attach-session binds once and rejects second bind. Evidence: `live/cli-attach-session-first.stdout.json`, `live/cli-attach-session-second.stderr.txt`, `TestManagerAttachRunSessionAndRetryLatestRunOutcome`.
- `TC-INT-011` PASS — automation direct task creation with automation actor/origin covered by automation integration `TestManagerIntegrationDirectTaskBackedJobDelegatesIntoTaskDomain`.
- `TC-INT-012` PASS — automation-linked agent session origin covered by automation integration `TestManagerIntegrationAutomationSessionCanCreateTaskWithAutomationOrigin`.
- `TC-INT-013` PASS — extension task creation and capability checks covered by `TestHostAPIHandlerTasksCreateUsesTrustedExtensionIdentity` and `TestHostAPIHandlerTaskOperationsRequireCapabilities/ShouldDenyCreate`.
- `TC-INT-014` PASS — network peer task creation with channel binding covered by network `TestCreateTaskFromPeerUsesServerDerivedIdentityAndAcceptedAudit`.
- `TC-INT-015` PASS — stale-channel write rejection covered by network `TestUpdateTaskFromPeerAllowsOnlyStaleChannelRepair/rejects_unrelated_writes_while_stale_channel_remains` and task `TestManagerStartRunRejectsStaleRunChannelWithoutMutation`.

## Security
- `TC-SEC-001` PASS — server-derived `created_by` ignores client payload. Evidence: `live/http-create-primary.http.json`, `live/live-summary.json`.
- `TC-SEC-002` PASS — server-derived `origin` ignores client payload. Evidence: `live/http-create-primary.http.json`, `live/live-summary.json`.
- `TC-SEC-003` FAIL — unauthenticated HTTP task requests were accepted (`200/201`) instead of rejected. Evidence: `security/tc-sec-003-get-headers.txt`, `security/tc-sec-003-post-headers.txt`, `security/tc-sec-003-get-body.json`, `security/tc-sec-003-post-body.json`, issue `.compozy/tasks/core-tasks/issues/BUG-001.md`.
- `TC-SEC-004` PASS — extension without `task.write` denied task creation via `TestHostAPIHandlerTaskOperationsRequireCapabilities/ShouldDenyCreate` and capability tests in `internal/extension`.
- `TC-SEC-005` PASS — channel-mismatch write rejected via `TestEnqueueRunFromPeerRejectsChannelMismatchAndAudits` and `TestUpdateTaskFromPeerAllowsOnlyStaleChannelRepair/rejects_unrelated_writes_while_stale_channel_remains`.
- `TC-SEC-006` PASS — SQL injection strings stored/validated literally with database intact. Evidence: `security/tc-sec-006-007-summary.json`, `security/tc-sec-006-db-tables.txt`, `security/tc-sec-006-post-health.json`.
- `TC-SEC-007` PASS — oversized metadata rejected with `413` and boundary accepted. Evidence: `security/tc-sec-006-007-summary.json`, task guards `TestPayloadSizeGuards/metadata_over_limit`, `result_over_limit`, `payload_over_limit`, and API mapping `TestStatusForTaskError/payload_too_large`.
- `TC-SEC-008` PASS — read-denied access rejected by `TestManagerGetAndListTasksRequireReadAuthorityAndBuildView`, integration `TestTaskManagerGetTaskRequiresReadAuthorityIntegration`, and API mapping `TestStatusForTaskError/permission_denied`.

## Performance
- `TC-PERF-001` PASS — 1000 sequential creates within thresholds. Evidence: `perf/perf-summary.json`.
- `TC-PERF-002` PASS — dependency fill, limit rejection, and cycle detection within thresholds. Evidence: `perf/perf-summary.json`.
- `TC-PERF-003` PASS — hierarchy depth and child fan-out within thresholds. Evidence: `perf/perf-summary.json`.
- `TC-PERF-004` PASS — cancellation propagation across 100 descendants within thresholds. Evidence: `perf/perf-summary.json`.
- `TC-PERF-005` PASS — composite `ListTasks` queries on 10K tasks within thresholds. Evidence: `perf/perf-summary.json`.
- `TC-PERF-006` PASS — observe summary/metrics/stuck-work queries on 10K tasks + 50K runs within thresholds after the `observe.Health()` snapshot reuse fix. Evidence: `perf/perf-summary.json`, `internal/observe/tasks_health_optimization_test.go`.
