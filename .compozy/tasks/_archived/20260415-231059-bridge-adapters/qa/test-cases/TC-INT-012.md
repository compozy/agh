## TC-INT-012: Conformance Matrix Multi-Provider Validation

**Priority:** P2
**Type:** Integration
**Systems:** extensiontest.Harness, extensiontest.ProviderConformanceSummary, extensiontest.BuildConformanceMatrix, extensiontest.ValidateConformanceMatrix, extensiontest.ScriptedPromptDriver, bridgesdk.Runtime, extension.Manager, bridges.Broker, bridges.RoutingKey, bridges.BridgeStatus
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-15

---

### Objective

Validate the conformance harness against at least 2 distinct provider implementations (e.g., GitHub + WhatsApp or Telegram + Slack). The harness runs each provider through a scripted prompt driver, verifies marker file completion for the 6 conformance targets (handshake, ownership, state, delivery, ingest, shutdown), aggregates results into a `ProviderConformanceSummary` per provider, and builds a `ConformanceMatrix` that is validated against the required coverage targets: multi-instance, restart-recovery, auth-degradation, dm-policy, and rate-limit-recovery.

### Preconditions

- [ ] At least 2 provider extension binaries are built and available (e.g., `github-provider`, `whatsapp-provider`)
- [ ] Extension directories contain valid manifests with `bridge_provider` blocks
- [ ] Mock platform API servers are configured for each provider
- [ ] Test listen addresses reserved via `reserveIntegrationListenAddr`
- [ ] ScriptedPromptDriver configured with `agent_message` + `done` events for delivery validation
- [ ] Environment variables for provider-specific configuration (API URLs, webhook secrets, test keys) are set

### Test Steps

1. **Build provider binaries**
   - Input: `go build` each provider extension from its source directory
   - **Expected:** Binaries compile without errors

2. **Configure harness for Provider A (e.g., GitHub) with multi-instance**
   - Input: `extensiontest.NewHarness(t, HarnessConfig{Platform: "github", ManagedInstances: [{ID: "brg-gh-pat"}, {ID: "brg-gh-app"}], Driver: ScriptedPromptDriver, ExtraEnv: {listen_addr, api_base}})`
   - **Expected:** Harness initializes; provider binary spawns as subprocess

3. **Run Provider A conformance scenario: multi-instance ownership**
   - Input: Wait for both instances to reach `status=ready`; send distinct webhook events for each instance; verify ingest routes to different sessions
   - **Expected:** Marker files created: handshake, ownership (multi-instance), ingest; `ProviderConformanceSummary` for Provider A records multi_instance=passed

4. **Run Provider A conformance scenario: delivery through scripted driver**
   - Input: Trigger a delivery for one instance via the ScriptedPromptDriver's `agent_message` + `done` events
   - **Expected:** Provider sends messages via mock API; delivery ack received with `remote_message_id`; marker file created: delivery

5. **Shutdown Provider A gracefully**
   - Input: Send `shutdown` JSON-RPC request via harness
   - **Expected:** Provider acknowledges shutdown; marker file: shutdown; subprocess exits cleanly

6. **Configure harness for Provider B (e.g., WhatsApp) with dm-policy and auth-degradation**
   - Input: `extensiontest.NewHarness(t, HarnessConfig{Platform: "whatsapp", ManagedInstances: [{ID: "brg-wa-1", dm_policy: "pairing"}], ...})`
   - **Expected:** Harness initializes; provider binary spawns

7. **Run Provider B conformance scenario: auth degradation**
   - Input: Provider reports `auth_failed` via `bridges/instances/report_state`; then recovers to `ready`
   - **Expected:** Instance transitions through `auth_required` -> `ready`; marker for state validation; summary records auth_degradation=passed

8. **Run Provider B conformance scenario: rate-limit recovery**
   - Input: Provider reports `rate_limited` degradation; recovers after simulated backoff
   - **Expected:** Instance transitions through `degraded` -> `ready`; summary records rate_limit_recovery=passed

9. **Run Provider B conformance scenario: dm-policy enforcement**
   - Input: Send a DM from an unrecognized sender to the `pairing` policy instance
   - **Expected:** Provider enforces pairing flow or rejects; marker for dm-policy; summary records dm_policy=passed

10. **Build conformance matrix from both providers**
    - Input: `extensiontest.BuildConformanceMatrix(summaryA, summaryB)`
    - **Expected:** Matrix has entries for at least 2 distinct platforms; each platform's conformance targets are tracked

11. **Validate the conformance matrix against required targets**
    - Input: `extensiontest.ValidateConformanceMatrix(matrix, CoverageTargetMultiInstance, CoverageTargetRestartRecovery, CoverageTargetAuthDegradation, CoverageTargetDMPolicy, CoverageTargetRateLimitRecovery)`
    - **Expected:** Validation passes; all 5 coverage targets have at least 1 provider passing each

12. **Verify matrix aggregation correctness**
    - Input: Inspect `matrix` entries per platform
    - **Expected:** Each platform entry lists the provider name, the conformance targets it covers, and pass/fail status for each. No target is duplicated. At least one provider covers each required target across the matrix

### Data Validation

| Field                           | Source Value                   | Transformed Value                        | Status |
| ------------------------------- | ------------------------------ | ---------------------------------------- | ------ |
| Provider A platform             | `github`                       | ConformanceMatrix entry key = `github`   |        |
| Provider B platform             | `whatsapp`                     | ConformanceMatrix entry key = `whatsapp` |        |
| CoverageTargetMultiInstance     | Provider A multi-instance test | Passed by github provider                |        |
| CoverageTargetRestartRecovery   | Provider A or B restart test   | Passed by at least 1 provider            |        |
| CoverageTargetAuthDegradation   | Provider B auth_failed cycle   | Passed by whatsapp provider              |        |
| CoverageTargetDMPolicy          | Provider B pairing flow        | Passed by whatsapp provider              |        |
| CoverageTargetRateLimitRecovery | Provider B rate_limited cycle  | Passed by whatsapp provider              |        |
| Matrix length                   | 2 provider summaries           | len(matrix) >= 2                         |        |

### Error Scenarios

- [ ] Provider binary fails to build: test fails at step 1 with build error
- [ ] Provider fails to reach `ready` within timeout: harness times out, conformance marker for handshake missing
- [ ] One coverage target has no provider passing it: `ValidateConformanceMatrix` returns error identifying the uncovered target
- [ ] Duplicate provider platform in matrix: `BuildConformanceMatrix` aggregates by platform name
- [ ] Provider crashes during conformance scenario: harness captures the failure, marks the corresponding target as failed
- [ ] Mock API server unreachable: provider reports degradation, delivery marker may fail
- [ ] Marker file not created within expected window: harness reports timeout for that specific conformance phase

### Related Test Cases

- TC-INT-001 (provider launch validated per-provider here)
- TC-INT-002 (webhook ingest validated per-provider here)
- TC-INT-004 (delivery validated per-provider here)
- TC-INT-005 (restart recovery is one of the coverage targets)
- TC-INT-006 (auth degradation is one of the coverage targets)
