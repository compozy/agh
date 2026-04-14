## TC-SEC-005: Network Peer Write Rejected on Channel Mismatch

**Priority:** P0
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that a network peer is denied task operations when the requested or bound network channel does not match the peer's authenticated ingress channel. The `ErrTaskChannelMismatch` sentinel must be returned and no data mutation occurs.

---

### Preconditions
- [ ] AGH daemon running with task subsystem and network layer initialized
- [ ] Network peer authenticated on ingress channel `"builders"`
- [ ] Task exists with `network_channel: "builders"` (for channel-bound operations)
- [ ] Task exists with `network_channel: "ops"` (for mismatch testing)
- [ ] Network channel validator configured on the TaskManager

---

### Test Steps
1. **Network peer enqueues run on matching channel (control)**
   - Input: Peer on channel `"builders"` calls `EnqueueRunFromPeer` for a task bound to channel `"builders"`
   - **Expected:** Run enqueued successfully. No error.

2. **Network peer enqueues run with mismatched requested channel**
   - Input: Peer on ingress channel `"builders"` calls `EnqueueRunFromPeer` requesting channel `"ops"`
   - **Expected:** `ErrTaskChannelMismatch` returned. Error message includes both channel names. No run persisted.

3. **Network peer enqueues run for task bound to different channel**
   - Input: Peer on ingress channel `"builders"` targets a task with `network_channel: "ops"`
   - **Expected:** `ErrTaskChannelMismatch` returned. Error message identifies the task ID and both channel values.

4. **Network peer creates task on mismatched channel**
   - Input: Peer on ingress channel `"builders"` attempts to create a task with `network_channel: "ops"` via network task bridge
   - **Expected:** Channel validation fails. Task not created.

5. **Verify HTTP error mapping for channel mismatch**
   - Input: Trigger channel mismatch via HTTP-facing network endpoint
   - **Expected:** Response maps to appropriate HTTP status (403 or 409). Error body includes `"channel_mismatch"` classification.

---

### Attack Vectors
- [ ] Network peer attempts cross-channel task manipulation to interfere with work on another channel
- [ ] Channel name casing mismatch bypass (e.g., `"Builders"` vs `"builders"`) -- validated via TrimSpace and normalization
- [ ] Channel name with leading/trailing whitespace to bypass exact match
- [ ] Empty channel string to bypass channel validation

---

### Related Test Cases
- TC-SEC-002: Server-derived origin identity
- TC-SEC-008: Unauthorized scope read for network peers
