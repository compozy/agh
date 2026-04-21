## TC-INT-003: Capability transfer lifecycle preservation

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that `kind:"capability"` preserves the useful delivery and lifecycle behavior that `recipe` previously had: directed transfer can open an interaction, broadcast transfer reaches channel peers, receipts/traces advance the same interaction, and post-terminal capability activity is rejected.

---

### Preconditions

- [ ] Two or more peers are available in a local test channel.
- [ ] The executor can run router/delivery/lifecycle integration checks.
- [ ] A valid transferable capability payload exists for the sending peer.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Channel | `builders` or equivalent local test channel | Shared by both peers |
| Directed interaction ID | `int_capability_lifecycle` | Used for the directed flow |
| Broadcast payload | Valid `kind:"capability"` with `to = null` | Used to confirm fan-out behavior |
| Terminal follow-up | Capability send after completion/cancellation | Used to confirm lifecycle enforcement |

---

### Test Steps

1. Send a directed `kind:"capability"` envelope with `interaction_id` from peer A to peer B.
   - **Expected:** Delivery succeeds, the interaction opens, and both sender-side and receiver-side bookkeeping reflect the new interaction.

2. Have peer B respond with a `receipt` and then a `trace` on the same `interaction_id`.
   - **Expected:** The interaction progresses through the expected lifecycle states with capability terminology preserved in summaries/audit output.

3. Send a broadcast `kind:"capability"` envelope in the same channel.
   - **Expected:** The capability reaches the expected channel peers without being forced through directed `direct` semantics.

4. Close or complete the directed interaction, then send another `kind:"capability"` message against the terminal interaction.
   - **Expected:** The runtime rejects the post-terminal capability message with the same lifecycle rule used for closed interactions.

5. If the protocol/runtime supports mixed flows, send a `direct` follow-up in the same interaction after the initial capability handoff.
   - **Expected:** `direct` and `capability` remain distinct kinds while still participating in the same interaction lifecycle correctly.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Broadcast transfer | `to = null` | Delivers to channel peers without directed lifecycle assumptions |
| Directed transfer | Explicit `to` with `interaction_id` | Opens an interaction and supports receipt/trace progression |
| Post-terminal send | Capability after `completed` or `canceled` | Rejected |
| Sender bookkeeping | Outbound send with `interaction_id` | Sender-side interaction state remains coherent for later replies |

---

### Traceability

- Tasks: `task_03`
- TechSpec: `Testing Approach`, `Build Order` item 5
- ADRs: `ADR-003`
- Primary surfaces: `internal/network/router.go`, `internal/network/delivery.go`, `internal/network/lifecycle.go`

---

### Evidence to Capture

- Delivery results for directed and broadcast capability sends
- Receipt/trace output tied to the same interaction
- Rejection output for post-terminal capability activity
- Any audit or summary text proving recipe-era wording is gone

---

### Notes

- This case is the operational continuity gate for the unification. If it fails, the model rename worked only at the type layer and not at the runtime behavior layer.
