## TC-SCEN-001: Managed Soul Authoring - Operator Can Trust Persona State

**Priority:** P0
**Type:** Real Scenario
**Status:** Passed
**Estimated Time:** 35 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Behavioral Scenario Charter

- Startup situation: a launch-review workspace needs a `reviewer` agent with explicit persona guidance.
- Operator intent: author, inspect, update, and audit `SOUL.md` through AGH public surfaces without direct mutation.
- Expected business outcome: the operator can see the active persona digest and revision history, stale writes fail safely, and HTTP/CLI read models agree.
- AGH surfaces used: CLI, HTTP API, runtime persistence, and generated DTOs.
- Real provider/LLM expectation: if provider credentials are available, start a reviewer session and verify the Soul snapshot is reflected in session/context state; otherwise record the provider boundary.

---

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
|-------------|------|-------------------|-----------------|
| Operator | AGH admin | Uses CLI/API only for managed authoring | CLI transcript and API JSON |
| reviewer | Launch reviewer | Carries review persona and conservative tone | `SOUL.md`, inspect payload, optional session state |

---

### Preconditions

- [ ] Bootstrap manifest exists and isolated runtime/provider env is available.
- [ ] Daemon/API readiness is confirmed.
- [ ] A `reviewer` agent definition exists in the scenario workspace.
- [ ] Provider-backed session is reachable or the exact provider/tool boundary is documented.

---

### Journey Steps

1. **Validate a proposed Soul body**
   - Surface: CLI
   - Input: `agh agent soul validate reviewer --file <scenario>/SOUL.md --workspace <workspace> --json`
   - **Expected:** JSON reports valid proposed Soul with deterministic fields and no absolute path leak.

2. **Write Soul through managed authoring**
   - Surface: CLI
   - Input: `agh agent soul write reviewer --file <scenario>/SOUL.md --workspace <workspace> --json`
   - **Expected:** `SOUL.md` is persisted, response includes digest/snapshot/revision metadata, and the body is not logged in evidence beyond the authored file.

3. **Inspect through CLI and HTTP**
   - Surface: CLI and HTTP API
   - Input: `agh agent soul inspect reviewer --workspace <workspace> --json`; `GET /api/agents/reviewer/soul?workspace=<workspace>`
   - **Expected:** CLI and API agree on `present`, `active`, `digest`, `source_path`, limits, frontmatter, and diagnostics.

4. **Execute stale CAS disruption probe**
   - Surface: CLI
   - Input: Re-run write/delete with an intentionally stale `--expected-digest sha256:stale`.
   - **Expected:** Mutation fails with deterministic `soul_conflict`, current file remains unchanged, and no partial revision is created.

5. **Read revision history**
   - Surface: CLI
   - Input: `agh agent soul history reviewer --limit 10 --workspace <workspace> --json`
   - **Expected:** History includes the managed write revision with actor/origin metadata and redacted relative source path.

6. **Optional live-provider check**
   - Surface: provider-backed AGH session
   - Input: Start or inspect a reviewer session when credentials are reachable.
   - **Expected:** Session Soul snapshot/provenance matches the inspect digest. If blocked, the final report names the exact missing provider/tool boundary.

---

### Required Evidence

- CLI command and output: `qa/evidence/TC-SCEN-001-cli.log`
- API response: `qa/evidence/TC-SCEN-001-api.json`
- Produced artifact: scenario `SOUL.md`
- Persistence/runtime evidence: inspect/history JSON
- Provider evidence or blocked boundary

---

### Pass Criteria

- Managed write, inspect, stale-CAS rejection, and history all behave as specified.
- CLI/API agree on digest and redacted source path.
- No direct file mutation is used for the mutation under test.
