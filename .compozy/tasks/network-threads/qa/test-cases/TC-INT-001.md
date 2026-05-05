## TC-INT-001: Direct-Room Resolve Race - Two Agents Converge On One Room

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05
**Execution Class:** Integration / E2E harness

---

### Objective

Verify that concurrent direct-room resolution for the same `(channel, peer_a, peer_b)` returns one deterministic `direct_id`, creates one room row, and keeps subsequent messages in the same room.

### Preconditions

- [ ] Fresh QA lab or runtime harness is available.
- [ ] Two distinct peer IDs are available.
- [ ] Store and API surfaces can be queried after concurrent operations.
- [ ] `make test-e2e-runtime` is available as supporting harness evidence.

### Test Steps

1. **Start concurrent direct resolve operations**
   - Input: Resolve the same direct room from two peers or two goroutines through the supported API/CLI/harness path.
   - **Expected:** Both operations return the same non-empty `direct_id`.

2. **Inspect persisted direct rooms**
   - Input: `agh network directs list --channel builders --peer reviewer.sess-xyz -o json` or equivalent API/harness query.
   - **Expected:** Exactly one direct room exists for the peer pair in `builders`.

3. **Send messages after the race**
   - Input: Send one `surface:"direct"` message from each participant into the returned room.
   - **Expected:** Both messages persist under the same `direct_id`.

4. **Verify public thread isolation**
   - Input: Query the related public thread messages.
   - **Expected:** Direct-room race messages do not appear in public-thread output.

5. **Disruption probe**
   - Probe: Force an invalid same-peer or mismatched-room resolve/send attempt.
   - **Expected:** Runtime rejects the attempt with deterministic validation and creates no new room.

### Behavioral Evidence

- Operator journey: prevent fragmented direct-room history during simultaneous handoff.
- Live agent/LLM behavior: optional; this is primarily integration evidence.
- Artifacts produced and used: runtime harness logs, CLI/API output, direct-room messages.
- Cross-surface assertions: direct ID and message counts agree across CLI/API/store or harness snapshots.

### Related Test Cases

- TC-SCEN-002
- TC-REG-001

