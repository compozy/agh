## TC-SCEN-001: Categorized Agent — Sidebar Tree → Session-Create Picker → Live Session

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06
**Execution Class:** E2E / behavior-first (Playwright + provider-backed session)

---

### Behavioral Scenario Charter

- Startup situation: Fresh QA lab with one categorized agent (`category_path: ["Marketing", "Sales"]`) and one root-level agent. Daemon, HTTP, UDS, native tools, and Web all wired against the same isolated `AGH_HOME`.
- Operator intent: Discover the categorized agent in the sidebar tree, open it, start a new session via the grouped session-create command picker, and verify the same agent state (including `category_path`) across CLI, API/UDS, native tool, and Web.
- Expected business outcome: The operator can group agents into a hierarchy purely via metadata and run sessions normally regardless of category. Cross-surface state agrees byte-for-byte.
- AGH surfaces used: Web sidebar tree, Web session-create dialog, Web `/agents/:name` route, CLI (`agh agent list/info -o json`), HTTP (`/api/agents`, `/api/workspaces/:id`), UDS (`agh agent info`), native tool (`agh__workspace_describe`), provider-backed session.
- Real provider/LLM expectation: Start a real provider-backed AGH session for the categorized agent and observe at least one coherent reply.
- Blocked live-provider boundary, if any: To be filled by `qa-execution`. If blocked, name the exact provider, credential, binary, or account boundary and continue all local surfaces.
- Scenario contract minimums covered: categorized + root-level agents, four surfaces (CLI / HTTP / UDS / Web / native), task-tree (session start), live provider OR explicit boundary, artifact reuse (sidebar agent → session → live transcript), disruption probe (daemon restart preserves grouping), required `make verify` + `make test-e2e-runtime` + `make test-e2e-web`.

---

### Actors and Agent Roles

| Actor / Agent | Role | Expected Behavior | Evidence Source |
| --- | --- | --- | --- |
| Operator | Scenario driver | Picks categorized agent in sidebar, starts session, inspects all surfaces. | CLI transcript, browser screenshot, HTTP / UDS responses, native tool response. |
| Categorized agent (`Marketing/Sales`) | Provider-backed peer | Starts a session and produces a coherent reply. | Session events, transcript or blocked-provider boundary. |
| Reviewer / observer agent | Cross-surface verifier | Calls `agh__workspace_describe` and `agh agent info -o json` to confirm `category_path` parity. | Native tool transcript, JSON evidence. |

---

### Preconditions

- [ ] Bootstrap manifest exists; isolated runtime/provider env is reachable.
- [ ] Daemon, HTTP, UDS, and Web readiness confirmed (smoke check passes).
- [ ] Provider-backed agent session for the categorized agent is reachable, OR the exact credential/tool boundary is documented.
- [ ] Scenario workspace contains a categorized AGENT.md (`["Marketing", "Sales"]`) and a root-level AGENT.md.
- [ ] SMOKE-001 passed.

---

### Journey Steps

1. **Operator confirms category metadata at the source.**
   - Surface: filesystem + CLI.
   - Input: `cat <workspace>/.../AGENT.md | rg "category_path"` and `agh agent info <categorized-agent> -o json | jq .agent.category_path`.
   - **Expected:** Both return `["Marketing", "Sales"]`.

2. **Operator confirms parity across HTTP, UDS, and native tool.**
   - Surface: HTTP + UDS + native tool.
   - Input: `curl -sS $BASE/api/agents/<categorized-agent> | jq .agent.category_path`, `agh agent info <categorized-agent> -o json | jq .agent.category_path`, and a request to the native `agh__workspace_describe` tool from a session.
   - **Expected:** All three return `["Marketing", "Sales"]` byte-for-byte.

3. **Operator opens the Web sidebar.**
   - Surface: Web (browser).
   - Input: Navigate to `/`.
   - **Expected:** Sidebar shows folder `Marketing` containing folder `Sales` containing the categorized agent leaf. The root-level agent renders at the top level alongside the folder. Screenshot saved to `qa/screenshots/sidebar-tree.png`.

4. **Operator routes to the categorized agent.**
   - Surface: Web.
   - Input: Click the categorized leaf.
   - **Expected:** URL becomes `/agents/<categorized-agent>`. The agent page renders with provider + category metadata. Both ancestor folders are expanded.

5. **Operator opens the session-create dialog.**
   - Surface: Web.
   - Input: Click "New session" → open the agent picker (`session-create-agent-select`).
   - **Expected:** The popover renders the grouped `Command` list. The group heading `Marketing / Sales` is visible (`agent-command-group-category:Marketing/Sales`). The categorized agent is inside that group.

6. **Operator starts a provider-backed session.**
   - Surface: Web → daemon.
   - Input: Pick the categorized agent and submit.
   - **Expected:** A real provider-backed session starts (or the exact blocked boundary is recorded). The session appears in `agh session list -o json` with a `session_id`. Screenshot of the session view saved to `qa/screenshots/session-running.png`.

7. **Cross-surface inspection of the running session.**
   - Surface: CLI + HTTP + Web.
   - Input: `agh session info <session_id> -o json`, `curl $BASE/api/sessions/<session_id>`, and the Web session view.
   - **Expected:** All three agree on the session's agent name and provider, and the agent's `category_path` matches.

8. **Disruption probe — daemon restart.**
   - Probe: Stop the daemon (`pkill agh` or the lab-managed equivalent) and start it again with the same isolated `AGH_HOME`.
   - **Expected:** After restart, `/api/agents` still exposes the same `category_path`; the sidebar tree still groups the categorized agent under `Marketing/Sales`; the session is either resumed or shown as terminated with persisted history.

9. **Disruption probe — invalid `category_path`.**
   - Probe: Edit AGENT.md to set `category_path: [""]` and `make web-test` style sanity check via `agh agent info -o json`.
   - **Expected:** Daemon emits an `agent_diagnostic` with the blank-segment message; the agent appears in `/api/agents` with `category_path: null` (or the diagnostic flow); the sidebar still renders without crashing. Restore the original AGENT.md before continuing.

---

### Required Evidence

- CLI command and JSON outputs at every step.
- HTTP request/response transcripts.
- UDS request/response transcripts.
- Native tool call payload and response.
- Browser screenshots: sidebar tree, agent page, session-create grouped popover, running session.
- Live agent/LLM transcript OR `qa/provider-attempt.json` recording the exact blocked provider boundary.
- Daemon restart log proving persistence.

---

### Behavioral Evidence

- Operator journey: full end-to-end discovery → routing → session start → cross-surface verification.
- Live agent / LLM behavior: at least one coherent provider reply OR explicit blocked-provider boundary.
- Artifacts produced and used: AGENT.md → `/api/agents` payload → sidebar leaf → session → session info JSON. Each artifact is referenced by a later step.
- Cross-surface assertions: same `category_path` on the same `agent.name` across CLI, HTTP, UDS, native tool, and Web.
- Disruption probes: daemon restart preserves grouping; invalid segment is rejected.

---

### Audit Coverage

- C4: operator + categorized agent + observer.
- C5: CLI + HTTP + UDS + native tool + Web.
- C6: session start as the task tree.
- C8: `category_path` parity across all five surfaces.
- C9: live provider session OR documented boundary.
- C10: AGENT.md authored once, reused by every surface.
- C11: daemon restart + invalid segment probes.
- C14: `make verify`, `make test-e2e-runtime`, `make test-e2e-web -- agent-categories`, plus the manual run captured in this case.

---

### Pass Criteria

- The operator goal is achieved, evidence is captured, and any blocked provider boundary is documented in `qa/provider-attempt.json`.
- All four surfaces agree on `category_path`.
- Both disruption probes complete with the expected behavior.

---

### Failure Criteria

- The test relies only on smoke, CRUD, mock, fake-provider, or page-render evidence.
- `category_path` differs between any two surfaces.
- A live provider boundary is missing but not documented.
- Sidebar tree crashes when an invalid segment exists in disk state.
