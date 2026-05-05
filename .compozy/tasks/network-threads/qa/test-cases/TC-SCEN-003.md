## TC-SCEN-003: Summarize Back To Public Thread - Operator Sees Outcome Without Work Leakage

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** E2E / behavior-first

---

### Behavioral Scenario Charter

- Startup situation: Public thread and direct-room review from TC-SCEN-001 and TC-SCEN-002 exist.
- Operator intent: Bring the outcome of restricted review back to the public thread as an understandable summary.
- Expected business outcome: The public audience receives a concise result without exposing direct-room transcript details or reusing the direct-room `work_id`.
- AGH surfaces used: CLI send/messages, API thread/direct routes, Web thread route, runtime audit evidence.
- Real provider/LLM expectation: Reviewer or requester agent writes a coherent public summary when reachable.
- Blocked live-provider boundary, if any: To be filled by `qa-execution`.

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
| --- | --- | --- | --- |
| Operator | Outcome verifier | Confirms public summary and private detail separation. | CLI/API/Web evidence. |
| Reviewer agent | Summary author | Posts a public summary in the original thread. | Thread message and live transcript. |
| Requester agent | Consumer | Can act on the summary without direct-room transcript access. | Follow-up thread reply or artifact. |

### Preconditions

- [ ] TC-SCEN-002 produced a direct-room review or equivalent direct-room state.
- [ ] Original public `thread_id` and direct-room `direct_id` are known.
- [ ] `trace_id` and `causation_id` from the handoff are recorded.

### Journey Steps

1. **Agent or operator posts a public summary**
   - Surface: CLI/API/provider-backed session
   - Input: `agh network send --session "$AGH_SESSION_ID" --channel builders --surface thread --thread thread_launch_review --kind say --reply-to "$HANDOFF_MESSAGE_ID" --trace-id trace_launch_review --causation-id "$DIRECT_MESSAGE_ID" --body '{"text":"Summary: migration review passed with one cleanup follow-up.","intent":"summarize-back"}' -o json`
   - **Expected:** Message is `surface:"thread"`, contains the public `thread_id`, and does not reuse the direct-room `work_id`.

2. **Operator verifies public visibility**
   - Surface: CLI/API/Web
   - Input: List thread messages and open `/network/builders/threads/thread_launch_review`.
   - **Expected:** Summary appears in the public thread and can be understood by the operator.

3. **Operator verifies direct-room detail remains isolated**
   - Surface: CLI/API
   - Input: Compare the public summary message to direct-room messages for `$DIRECT_ID`.
   - **Expected:** Direct-room transcript details remain in direct-room queries only.

4. **Operator verifies correlation**
   - Surface: runtime audit/work lookup
   - Input: Inspect persisted message fields for `reply_to`, `trace_id`, and `causation_id`.
   - **Expected:** Summary links back to the handoff via correlation fields without binding itself to the direct-room work.

5. **Disruption probe**
   - Probe: Try to post a public summary with the direct-room `work_id`.
   - **Expected:** The runtime rejects cross-container continuation or records a bug if it accepts the invalid continuation.

### Required Evidence

- Public summary CLI/API payload.
- Browser screenshot of public thread summary.
- Direct-room messages showing restricted detail remains separate.
- Runtime/audit correlation evidence.
- Live agent/LLM transcript or blocked provider boundary.

### Behavioral Evidence

- Operator journey: direct-room result summarized back into the original public thread.
- Live agent/LLM behavior: reviewer/requester summary authoring or exact blocked provider boundary.
- Artifacts produced and used: public summary message, direct-room transcript comparison, correlation fields.
- Cross-surface assertions: CLI/API/Web/runtime agree on public summary state and direct-room isolation.
- Disruption probes: cross-container `work_id` reuse is rejected.

### Pass Criteria

- Public thread contains the summary and not the restricted transcript.
- Direct-room `work_id` is not reused in the public thread.
- Correlation fields preserve handoff traceability.

### Failure Criteria

- Summary requires access to direct-room details to be understood.
- Direct-room `work_id` crosses into the public thread.
- Web shows stale or misleading conversation state.
