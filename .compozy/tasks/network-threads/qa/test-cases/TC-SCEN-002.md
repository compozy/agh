## TC-SCEN-002: Restricted Direct-Room Handoff - Operator Can Verify Isolated Review Work

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** E2E / behavior-first

---

### Behavioral Scenario Charter

- Startup situation: Public thread from TC-SCEN-001 exists, and two peers can participate in a direct room inside `builders`.
- Operator intent: Move detailed review work into a restricted direct room while keeping public-thread visibility clean.
- Expected business outcome: The operator can inspect the direct room, confirm only the two expected peers are members, and verify work lifecycle stays bound to the direct room.
- AGH surfaces used: CLI direct resolve/list/messages, API direct routes, native network tools, runtime audit/store evidence, provider-backed session when reachable.
- Real provider/LLM expectation: Reviewer agent produces a useful direct-room review artifact or trace message.
- Blocked live-provider boundary, if any: To be filled by `qa-execution`.

### Actors and Agent Roles

| Actor/Agent | Role | Expected Behavior | Evidence Source |
| --- | --- | --- | --- |
| Operator | Scenario driver | Resolves the direct room and verifies restricted state. | CLI/API output, browser screenshot. |
| Requester agent | Public initiator | Requests restricted follow-up from reviewer. | Public thread message. |
| Reviewer agent | Direct-room worker | Continues work in direct room using a new `work_id`. | Direct-room messages, work lookup, artifact. |

### Preconditions

- [ ] TC-SCEN-001 setup exists or equivalent public thread state is created.
- [ ] Two distinct peer IDs are available.
- [ ] Direct-room visibility is understood as restricted runtime visibility, not cryptographic privacy.
- [ ] Provider-backed agent session is reachable or exact blocked boundary is recorded.

### Journey Steps

1. **Operator resolves the direct room**
   - Surface: CLI
   - Input: `agh network directs resolve --session "$AGH_SESSION_ID" --channel builders --peer reviewer.sess-xyz -o json`
   - **Expected:** Response includes one `direct_id` matching `direct_[a-f0-9]{32}`, `peer_a`, `peer_b`, and `channel:"builders"`.

2. **Operator sends direct-room work**
   - Surface: CLI or native tool
   - Input: `agh network send --session "$AGH_SESSION_ID" --channel builders --surface direct --direct "$DIRECT_ID" --kind say --work work_review_launch --reply-to "$THREAD_MESSAGE_ID" --trace-id trace_launch_review --body '{"text":"Review the migration details privately.","intent":"handoff"}' -o json`
   - **Expected:** Response includes `surface:"direct"`, the resolved `direct_id`, `work_id:"work_review_launch"`, and no `thread_id`.

3. **Agent performs direct-room review**
   - Surface: provider-backed AGH session when reachable
   - Input: Ask the reviewer agent to inspect and respond in the direct room with an actionable review note.
   - **Expected:** The agent response remains in the same `direct_id`, advances or references the direct-room work, and does not appear in public thread messages.

4. **Operator checks isolation**
   - Surface: API/CLI
   - Input: Compare `agh network directs messages --channel builders --direct "$DIRECT_ID" -o jsonl` with `agh network threads messages --channel builders --thread thread_launch_review -o jsonl`.
   - **Expected:** Direct-room messages appear only in the direct-room output; public-thread output does not include restricted details.

5. **Disruption probe**
   - Probe: Submit a direct message using a mismatched or non-deterministic `direct_id`.
   - **Expected:** Runtime rejects the message with deterministic validation; no direct-room summary or message is created for the invalid room.

### Required Evidence

- Direct resolve CLI output.
- Direct-room API response.
- Work lookup for `work_review_launch`.
- Public-thread and direct-room message comparison.
- Live agent/LLM transcript or blocked provider boundary.

### Behavioral Evidence

- Operator journey: public-to-direct handoff with restricted direct-room inspection.
- Live agent/LLM behavior: reviewer agent review artifact or exact blocked provider boundary.
- Artifacts produced and used: direct-room messages, `work_id` lookup, CLI/API comparisons.
- Cross-surface assertions: direct-room state agrees across CLI/API/runtime and remains absent from public-thread messages.
- Disruption probes: invalid direct-room binding is rejected without creating a room.

### Pass Criteria

- Direct-room identity is deterministic and two-party.
- Direct-room messages are isolated from public thread queries.
- `work_id` is bound to the direct room and does not become a conversation ID.

### Failure Criteria

- Direct-room messages leak into public thread views.
- The same pair gets multiple active `direct_id` values.
- Direct rooms are described or displayed as encrypted/private beyond the implemented restriction.
