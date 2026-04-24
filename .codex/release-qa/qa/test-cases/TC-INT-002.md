## TC-INT-002: Network Direct Reply Lifecycle

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify the core AGH Network direct-message lifecycle through public daemon/runtime surfaces.

### Preconditions

- Daemon/runtime e2e harness is available.
- Mock ACP agents or real ACP-compatible agents can join a network channel.

### Test Steps

1. Start an isolated AGH daemon with network enabled.
   **Expected:** Daemon reports network status `running`.

2. Create two agent sessions and join them to a shared channel.
   **Expected:** Both peers appear in network peer listings.

3. Send a `direct` message from one peer to the other.
   **Expected:** Target receives a network prompt with message ID, channel, kind, sender, and reply guidance.

4. Send `receipt` and `trace` lifecycle messages.
   **Expected:** Lifecycle messages are accepted, correlated, audited, and visible in network timeline/status.

### Edge Cases & Variations

| Variation            | Input                | Expected Result                                         |
| -------------------- | -------------------- | ------------------------------------------------------- |
| Missing target peer  | Unknown `--to`       | Send fails before publish with target-not-found error.  |
| Duplicate message ID | Same envelope replay | Duplicate is rejected or ignored with audit visibility. |
