## TC-SCEN-001: Public Launch Thread Coordination - Operator Can Track Shared Work

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** E2E / behavior-first

---

### Behavioral Scenario Charter

- Startup situation: Fresh QA lab with a `builders` channel, at least two reachable peers, and Web/API/CLI surfaces pointed at the same daemon.
- Operator intent: Open or use a public thread for launch/review coordination and verify agents can see the same shared context.
- Expected business outcome: The operator can identify the public `thread_id`, see message history, inspect linked `work_id`, and compare the same state across CLI, API, Web, and runtime evidence.
- AGH surfaces used: CLI, HTTP or UDS API, Web `/network/:channel/threads`, runtime store/audit evidence, provider-backed session when reachable.
- Real provider/LLM expectation: A live agent should produce a relevant public-thread reply or work update. If unavailable, record the exact provider/tool boundary and keep harness evidence separate.
- Blocked live-provider boundary, if any: To be filled by `qa-execution`.

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
| --- | --- | --- | --- |
| Operator | Scenario driver | Starts the public coordination thread and verifies state. | CLI transcript, browser screenshot, API response. |
| Requester agent | Work initiator | Posts a launch/review request in the public thread. | Thread message payload and prompt wrapper. |
| Reviewer agent | Participant | Responds in the same public thread or accepts work with `work_id`. | Thread messages, work lookup, session transcript. |

### Preconditions

- [ ] Bootstrap manifest exists and isolated runtime/provider env is available.
- [ ] Daemon/API/Web readiness is confirmed.
- [ ] Provider-backed agent sessions are reachable, or the exact credential/tool boundary is documented.
- [ ] Scenario workspace has real startup directories and AGH configuration.
- [ ] SMOKE-001 passed or produced a fixed readiness issue.

### Journey Steps

1. **Operator opens a public thread**
   - Surface: CLI or API
   - Input: `agh network send --session "$AGH_SESSION_ID" --channel builders --surface thread --thread thread_launch_review --kind say --body '{"text":"Review the launch checklist.","intent":"review-request"}' -o json`
   - **Expected:** Response includes `surface:"thread"`, `thread_id:"thread_launch_review"`, no `direct_id`, and no `interaction_id`.

2. **Operator lists and shows the thread**
   - Surface: CLI
   - Input: `agh network threads list --channel builders -o json` and `agh network threads show --channel builders --thread thread_launch_review -o json`
   - **Expected:** The thread summary exists, message count is non-zero, and the summary is scoped to the `builders` channel.

3. **Agent performs meaningful shared work**
   - Surface: provider-backed AGH session when reachable
   - Input: Ask the reviewer agent to respond in the current public thread with an actionable review update.
   - **Expected:** The agent creates a coherent reply or work update in the same `thread_id`, with `work_id` only when lifecycle-bearing work is opened or continued.

4. **Operator compares API and Web state**
   - Surface: HTTP/UDS and browser
   - Input: Fetch `/api/network/channels/builders/threads/thread_launch_review/messages` or equivalent UDS route, then open `/network/builders/threads/thread_launch_review`.
   - **Expected:** CLI/API/Web show the same message IDs, same `thread_id`, and no direct-room messages.

5. **Disruption probe**
   - Probe: Restart the daemon or rerun the runtime harness, then query the thread again.
   - **Expected:** The public thread summary and messages remain persisted and understandable.

### Required Evidence

- CLI command and output.
- API request/response for thread messages.
- Browser URL and screenshot showing the public thread route.
- Live agent/LLM transcript or blocked provider boundary.
- Runtime persistence/audit evidence for the same `thread_id`.

### Behavioral Evidence

- Operator journey: public thread coordination from creation through cross-surface inspection.
- Live agent/LLM behavior: reviewer agent reply or exact blocked provider boundary.
- Artifacts produced and used: thread message payloads, browser screenshot, API/CLI transcripts.
- Cross-surface assertions: CLI/API/Web/runtime agree on `thread_id`, message IDs, and absence of direct-room messages.
- Disruption probes: daemon restart or harness rerun preserves thread state.

### Pass Criteria

- The operator goal is achieved or a product bug is filed.
- Agent behavior is coherent and remains in the public thread.
- CLI/API/Web/runtime state agree for the same `thread_id`.
- Smoke or harness checks are not counted as the final proof.

### Failure Criteria

- `interaction_id`, `kind:"direct"`, or direct-room payloads appear in the public thread flow.
- Web can only render a page without proving the operator can act on real thread state.
- Provider boundary is missing and not documented.
