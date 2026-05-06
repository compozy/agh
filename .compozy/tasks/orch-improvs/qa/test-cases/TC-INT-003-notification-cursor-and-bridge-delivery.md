# TC-INT-003: Notification Cursor And Bridge Delivery Semantics

**Priority:** P0

**Objective:** Prove bridge task subscriptions use durable notification cursors for accepted-final
terminal delivery, diagnostics, replay, delete lifecycle, and failure handling without mutating
review state or advancing on failed delivery.

**Requirements Covered:** tasks 06-07, 21, 24-25, 29; ADR-003, ADR-005, ADR-007.

## Preconditions

- Isolated QA lab with bridge delivery configured.
- Task with review policy enabled.
- Ability to force one bridge delivery failure and then a successful delivery.

## Test Steps

1. Create a bridge task subscription through HTTP.
   **Expected:** UDS and CLI list/show expose the same subscription id, cursor id, target, and
   zero-state diagnostics.

2. Generate a terminal run that requires review.
   **Expected:** Terminal event is durable, but bridge final delivery is deferred while review or
   continuation is active.

3. Force a bridge delivery failure for the next delivery attempt.
   **Expected:** Diagnostic error is recorded and cursor sequence does not advance.

4. Approve the final reviewed run.
   **Expected:** Accepted-final terminal event becomes eligible for delivery.

5. Allow bridge delivery to succeed.
   **Expected:** Cursor advances to the delivered event sequence and diagnostics record delivery id
   and timestamp.

6. Restart daemon and replay from stored cursor state.
   **Expected:** No duplicate accepted-final notification is delivered, and later terminal events
   resume from the stored sequence.

7. Delete the bridge task subscription.
   **Expected:** HTTP, UDS, CLI, web data hooks, and diagnostics stop showing the subscription; no
   orphan delivery path remains active.

## Behavioral Evidence

- Subscription id, cursor id, cursor sequence before and after failure, delivery id, and timestamps.
- CLI/HTTP/UDS output for create/list/show/delete.
- Bridge delivery logs or events for failed and successful attempts.
- Task event sequence for deferred and delivered terminal events.

## Disruption Probes

- Replay an old task event whose current task/review state no longer matches accepted-final.
- Create two subscriptions for the same task and prove cursor identity remains distinct.
- Restart daemon after failed delivery and before success.

