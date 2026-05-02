## TC-SCEN-003: Soul-Influenced Session Context And Agent Output

**Priority:** P0
**Type:** Real Scenario
**Status:** Passed With Provider Output Observation
**Estimated Time:** 45 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Behavioral Scenario Charter

- Startup situation: a launch-review startup wants the `reviewer` agent to produce an evidence-first launch risk note using the persona authored in `SOUL.md`.
- Operator intent: author a distinctive Soul, start or inspect a real session, prompt the agent to produce a launch readiness artifact, and verify that runtime context and agent-visible output reflect the persona without claiming task ownership or leaking hidden state.
- Expected business outcome: the operator can trust that authored persona context changes real agent behavior when a provider is reachable, and can see the exact provider boundary when live LLM execution is unavailable.
- AGH surfaces used: CLI, HTTP API, `/agent/context`, session lifecycle, provider-backed session when reachable, runtime persistence, and produced artifact inspection.
- Real provider/LLM expectation: use the isolated provider home from the bootstrap manifest. If no live provider credential/tool is reachable, record the exact command and error boundary, then validate session Soul snapshot/projection through reachable AGH runtime surfaces only.
- Blocked live-provider boundary, if any: to be filled during execution.

---

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
|-------------|------|-------------------|-----------------|
| Operator | AGH admin | Uses public CLI/API surfaces and inspects persisted context | CLI/API transcript |
| reviewer | Launch reviewer | Produces a concise evidence-first risk note shaped by `SOUL.md` | Provider transcript or documented provider boundary plus AGH context state |

---

### Preconditions

- [ ] Bootstrap manifest exists and isolated runtime/provider env is available.
- [ ] Daemon/API readiness is confirmed.
- [ ] `reviewer` agent exists in the scenario workspace with a valid `AGENT.md`.
- [ ] Valid `SOUL.md` exists for `reviewer` through managed authoring.
- [ ] Provider-backed execution is reachable, or the exact credential/tool boundary is documented.

---

### Journey Steps

1. **Author a distinctive Soul persona**
   - Surface: CLI
   - Input: `agh agent soul write reviewer --file <scenario>/reviewer/SOUL.md --workspace <workspace> --json`
   - **Expected:** Managed write succeeds, digest is recorded, and the persona includes a verifiable behavioral instruction such as "use evidence-first risk bullets and avoid hype."

2. **Start or inspect a reviewer session**
   - Surface: CLI
   - Input: `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME" agh session new --agent reviewer --workspace <workspace> --json`
   - **Expected:** A session is created with the active Soul digest/snapshot. If provider startup fails due to missing credentials or tool setup, the exact error is captured and the remaining steps use runtime context surfaces only.

3. **Verify `/agent/context` or session inspect includes Soul projection**
   - Surface: CLI/API
   - Input: `agh agent context --session <session-id> --json` or the matching HTTP context endpoint.
   - **Expected:** Context includes a compact Soul projection or provenance matching the inspect digest, with bounded body/projection and redacted relative source path.

4. **Prompt the agent for a launch readiness note when live provider is reachable**
   - Surface: provider-backed AGH session
   - Input: Ask the reviewer to produce a launch readiness note for a realistic release checklist artifact.
   - **Expected:** Agent output is not an echo; it follows the Soul persona, produces a coherent artifact or message, and does not claim task ownership or expose hidden tokens.

5. **Use the produced artifact or context evidence in a later operator check**
   - Surface: CLI/API and file inspection
   - Input: Inspect the artifact/message/context state created by the session.
   - **Expected:** The operator can connect the output to the session, agent role, workspace, and Soul digest. If live provider was blocked, the report clearly marks runtime projection evidence as not live LLM proof.

---

### Required Evidence

- CLI command and output: `qa/evidence/TC-SCEN-003-session-context.log`
- API/context response: `qa/evidence/TC-SCEN-003-agent-context.json`
- Live provider transcript or blocked provider command/error: `qa/evidence/TC-SCEN-003-provider-boundary.log`
- Produced artifact path and content summary when provider is reachable.
- Persistence/session evidence showing Soul digest/projection correlation.

---

### Pass Criteria

- Live provider output reflects the authored Soul persona when provider execution is reachable.
- If live provider execution is blocked, the exact provider/tool/credential boundary is documented and no mock or fake response is counted as LLM proof.
- AGH context/session state shows the Soul digest or projection through public surfaces.
- No raw claim token, absolute source path, provider credential, or full hidden prompt transcript appears in operator-facing output.

### Failure Criteria

- The feature is declared behaviorally validated using only acpmock, smoke, CRUD, unit/integration, or page-render evidence.
- Agent output ignores the authored persona when a live provider is reachable.
- Context output leaks absolute paths, raw tokens, secrets, or unbounded hidden prompt text.
