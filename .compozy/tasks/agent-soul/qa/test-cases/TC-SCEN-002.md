## TC-SCEN-002: Heartbeat Policy, Session Health, And Advisory Wake

**Priority:** P0
**Type:** Real Scenario
**Status:** Passed
**Estimated Time:** 45 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Behavioral Scenario Charter

- Startup situation: an `ops` agent needs advisory wake/reentry guidance for launch monitoring.
- Operator intent: author `HEARTBEAT.md`, inspect wake eligibility, read session health, and request a manual advisory wake without creating task work.
- Expected business outcome: the operator can understand whether the session is wake-eligible and why a wake was sent or skipped.
- AGH surfaces used: CLI, HTTP API, session health/status/inspect, wake status/audit, runtime persistence.
- Real provider/LLM expectation: execute a provider-backed wake only when credentials and an attachable idle session are available; otherwise run a dry-run or skipped wake boundary and document the blocker.

---

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
|-------------|------|-------------------|-----------------|
| Operator | AGH admin | Requests status/wake through public surfaces | CLI/API JSON |
| ops | Launch operations agent | Exposes wake policy and session health | `HEARTBEAT.md`, status, health, wake event |

---

### Preconditions

- [ ] Bootstrap manifest exists and isolated runtime/provider env is available.
- [ ] Daemon/API readiness is confirmed.
- [ ] An `ops` agent definition exists in the scenario workspace.
- [ ] A real session exists if provider credentials are reachable; otherwise the session boundary is documented.

---

### Journey Steps

1. **Validate and write Heartbeat policy**
   - Surface: CLI
   - Input: `agh agent heartbeat validate ops --file <scenario>/HEARTBEAT.md --workspace <workspace> --json`; `agh agent heartbeat write ops --file <scenario>/HEARTBEAT.md --workspace <workspace> --json`
   - **Expected:** Policy is valid, persisted through managed authoring, and response includes digest/revision metadata.

2. **Inspect policy and status**
   - Surface: CLI and HTTP API
   - Input: `agh agent heartbeat inspect ops --workspace <workspace> --json`; `agh agent heartbeat status ops --workspace <workspace> --json`; HTTP equivalents.
   - **Expected:** Inspect/status include policy digest, config digest, bounded preferences, diagnostics, and no task/queue ownership fields.

3. **Read session health and inspect correlation**
   - Surface: CLI/API
   - Input: `agh session health <session-id> --json`; `agh session inspect <session-id> --include-wake-events --json`
   - **Expected:** Session health reports closed enum state/health/eligibility and inspect correlates policy/wake state without raw claim tokens or provider secrets.

4. **Request manual advisory wake**
   - Surface: CLI/API
   - Input: `agh agent heartbeat wake ops --session <session-id> --dry-run --json` or non-dry-run when provider-backed wake is reachable.
   - **Expected:** Result is `sent`, `skipped`, or `dry_run` with a closed reason. Wake audit is recorded when supported. No task run is created or claimed.

5. **Disruption probe: ineligible or missing session**
   - Surface: CLI/API
   - Input: Wake against a missing or stopped session.
   - **Expected:** Deterministic reason such as `session_not_found`, `session_unhealthy`, or `session_not_attachable`; operator-readable diagnostics explain the next action.

---

### Required Evidence

- CLI command and output: `qa/evidence/TC-SCEN-002-cli.log`
- API responses: `qa/evidence/TC-SCEN-002-api.json`
- Produced artifact: scenario `HEARTBEAT.md`
- Session health/status/inspect JSON
- Wake decision/audit JSON
- Provider-backed wake evidence or exact blocked boundary

---

### Pass Criteria

- Heartbeat authoring, status, health, and wake boundaries are operator-readable and deterministic.
- Heartbeat remains advisory: no task queue, task run, task lease, or scheduler authority is created by the wake flow.
