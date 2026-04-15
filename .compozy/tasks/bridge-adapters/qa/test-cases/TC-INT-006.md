## TC-INT-006: Auth Degradation and Recovery Cycle

**Priority:** P1
**Type:** Integration
**Systems:** bridgesdk.HostAPIClient (bridges/instances/report_state), bridges.BridgeStatus, bridges.BridgeDegradation, bridges.BridgeDegradationReason, bridges.ValidateInstanceStateTransition, extension.HostAPI, bridgesdk.ClassifiedError, store/globaldb
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective
Validate the full auth degradation and recovery lifecycle: a provider detects an authentication failure, reports `auth_failed` degradation through `bridges/instances/report_state`, the daemon transitions the instance to `auth_required` status with degradation metadata, health metrics update, and later when the provider resolves auth and reports `ready`, the daemon clears the degradation and transitions back to `ready`.

### Preconditions
- [ ] Provider runtime initialized with 1 bridge instance (`brg-auth-1`, scope=global, platform=slack, status=ready, enabled=true)
- [ ] Instance has valid bound secrets for its API token
- [ ] globaldb bridge_instances row for `brg-auth-1` has `status=ready`, `degradation=NULL`
- [ ] Health check is configured with a 30s interval

### Test Steps
1. **Verify initial healthy state**
   - Input: Call `session.HostAPI().GetBridgeInstance(ctx, "brg-auth-1")`
   - **Expected:** Instance has `status=ready`, `enabled=true`, `degradation=nil`

2. **Provider detects auth failure and reports degradation**
   - Input: Provider calls `session.HostAPI().ReportBridgeInstanceState(ctx, BridgesInstancesReportStateParams{BridgeInstanceID: "brg-auth-1", Status: BridgeStatusAuthRequired, Degradation: &BridgeDegradation{Reason: BridgeDegradationReasonAuthFailed, Message: "OAuth token expired"}})`
   - **Expected:** Returns updated `BridgeInstance` with `status=auth_required`, `degradation.reason=auth_failed`, `degradation.message="OAuth token expired"`

3. **Verify state transition is persisted in globaldb**
   - Input: Query globaldb bridge_instances for `id=brg-auth-1`
   - **Expected:** Row has `status=auth_required`, `enabled=true` (remains enabled), degradation JSON with `reason=auth_failed`

4. **Verify lifecycle state transition rules**
   - Input: Call `ValidateInstanceStateTransition(current{enabled:true, status:ready}, nextEnabled:true, nextStatus:auth_required)`
   - **Expected:** No error (ready -> auth_required is valid)
   - Input: Call `ValidateInstanceStateTransition(current{enabled:true, status:auth_required}, nextEnabled:true, nextStatus:error)`
   - **Expected:** No error (auth_required -> error is valid)

5. **Verify instance is unavailable for ingest during auth_required**
   - Input: Attempt to ingest a message targeting `brg-auth-1`
   - **Expected:** Host API returns `ErrBridgeInstanceUnavailable` or similar rejection

6. **Provider resolves auth and reports healthy**
   - Input: Provider calls `session.HostAPI().ReportBridgeInstanceState(ctx, BridgesInstancesReportStateParams{BridgeInstanceID: "brg-auth-1", Status: BridgeStatusReady, ClearDegradation: true})`
   - **Expected:** Returns updated `BridgeInstance` with `status=ready`, `degradation=nil`

7. **Verify recovery is persisted**
   - Input: Query globaldb bridge_instances for `id=brg-auth-1`
   - **Expected:** Row has `status=ready`, degradation is NULL or empty

8. **Verify instance accepts ingest after recovery**
   - Input: Ingest a message targeting `brg-auth-1`
   - **Expected:** Ingest succeeds, returns valid `BridgesMessagesIngestResult`

### Data Validation
| Field | Source Value | Transformed Value | Status |
|-------|------------|-------------------|--------|
| BridgesInstancesReportStateParams.Status | `auth_required` | BridgeInstance.Status = BridgeStatusAuthRequired | |
| BridgeDegradation.Reason | `auth_failed` | BridgeDegradationReasonAuthFailed | |
| BridgeDegradation.Message | `OAuth token expired` | Stored as degradation.message | |
| ClearDegradation=true | Recovery signal | BridgeInstance.Degradation = nil | |
| BridgeInstance.Enabled | `true` | Unchanged through degradation cycle | |

### Error Scenarios
- [ ] Reporting `auth_required` on a `disabled` instance: `ValidateInstanceStateTransition` rejects (disabled can only go to starting)
- [ ] Reporting `ready` without `ClearDegradation=true` when degradation exists: instance stays degraded but status transitions
- [ ] Reporting degradation with `reason=""`: `BridgeDegradation.Validate()` returns error (reason is required)
- [ ] Reporting degradation with status=ready: `BridgeInstance.Validate()` rejects (degradation requires degraded/auth_required/error status)
- [ ] Reporting unsupported degradation reason: `BridgeDegradationReason.Validate()` returns error
- [ ] Invalid transition: ready -> disabled without setting enabled=false: `ValidateInstanceStateTransition` returns `ErrInvalidBridgeStateTransition`
- [ ] Provider reports rate_limited degradation: instance transitions to `degraded` (not `auth_required`)

### Related Test Cases
- TC-INT-001 (initial instance setup and launch)
- TC-INT-002 (ingest flow requires healthy instance)
- TC-INT-012 (conformance harness validates auth-degradation coverage target)
