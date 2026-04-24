## TC-INT-001: Network Backpressure Is Audited

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

### Objective

Verify that inbound network queue overflow is operationally visible instead of silently dropping messages.

### Preconditions

- Network manager can run with an audit writer.
- A test session can join a channel and be held busy to force queueing.

### Test Steps

1. Start a network manager with `max_queue_depth=1` and an audit sink.
   **Expected:** Manager starts and reports network status.

2. Mark the target session as prompting/busy and accept two inbound messages.
   **Expected:** The first queued message is evicted when the second arrives.

3. Query status and audit output.
   **Expected:** `messages_rejected` increments and audit contains a rejected entry for the evicted message with reason `queue_overflow`.

4. Release the prompt and drain the remaining message.
   **Expected:** The remaining message is delivered and the delivered counter increments.

### Edge Cases & Variations

| Variation                | Input                       | Expected Result                                      |
| ------------------------ | --------------------------- | ---------------------------------------------------- |
| Multiple overflow events | Three messages with depth 1 | Each evicted message is audited once.                |
| Audit sink failure       | Store write fails           | Drop is logged and delivery continues without panic. |
