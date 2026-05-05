## TC-REG-001: Network Threads Hard-Cut Regression - Legacy Fields Stay Rejected

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** Regression / contract guardrail

---

### Objective

Verify that active runtime, CLI, API/UDS, native tools, generated contracts, docs, prompts, and web fixtures use the final conversation model and reject legacy `interaction_id`, `kind:"direct"`, and old CLI flags.

### Preconditions

- [ ] Active docs and generated artifacts are present.
- [ ] Daemon can accept CLI/API/native-tool send attempts.
- [ ] Archived/provenance paths are excluded from active hard-cut scans.

### Test Steps

1. **Submit legacy JSON field through API**
   - Input: `POST /api/network/send` with `interaction_id`.
   - **Expected:** Request is rejected with deterministic validation; no message or work row is created.

2. **Submit legacy kind through CLI**
   - Input: `agh network send --session "$AGH_SESSION_ID" --channel builders --surface direct --direct "$DIRECT_ID" --kind direct --body '{"text":"legacy"}' -o json`
   - **Expected:** CLI rejects `--kind direct` and tells the operator to use `--surface direct` with a supported kind.

3. **Submit stale flags**
   - Input: Attempts with `--interaction-id`, `--thread-id`, `--direct-id`, or `--work-id`.
   - **Expected:** Stale flags are not accepted by the active CLI.

4. **Check native/hosted tool schema**
   - Input: Attempt `agh__network_send` with `interaction_id` or extra properties.
   - **Expected:** Tool schema rejects the payload and returns deterministic error state without raw-token leakage.

5. **Scan active docs/prompts/generated artifacts**
   - Input: Repo-defined docs tests or targeted `rg` scan over active docs, generated contract surfaces, bundled `agh-network` skill, and web fixtures.
   - **Expected:** Active surfaces do not teach `interaction_id`, `kind:"direct"`, `DirectBody`, `KindDirect`, or old send flags except as explicit negative assertions.

6. **Disruption probe**
   - Probe: Try a conversation-bearing `greet` or `whois` with `surface`, `thread_id`, `direct_id`, or `work_id`.
   - **Expected:** Runtime rejects the invalid envelope and creates no conversation row.

### Behavioral Evidence

- Operator journey: stale operational commands fail clearly rather than silently writing wrong state.
- Live agent/LLM behavior: optional; tool schema checks can be local.
- Artifacts produced and used: CLI/API/native-tool rejection output, docs scan output.
- Cross-surface assertions: rejection behavior is consistent across CLI/API/native tool ingress.

### Related Test Cases

- SMOKE-001
- TC-SCEN-001
- TC-SCEN-002

