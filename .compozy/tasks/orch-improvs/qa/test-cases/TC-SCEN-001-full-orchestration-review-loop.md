# TC-SCEN-001: Full Orchestration Review Continuation Loop

**Priority:** P0

**Objective:** Prove the user-visible orchestration program works as one durable system: a task
profile selects runtime behavior, a worker produces incomplete work, a reviewer rejects it, a
continuation run receives missing-work guidance, the reviewer approves the correction, and the
bridge receives exactly the accepted final notification.

**Requirements Covered:** tasks 08-13, 15-19, 21-27, 29; ADR-005, ADR-007, ADR-008, ADR-009,
ADR-010.

## Preconditions

- Fresh isolated QA lab from `agh-qa-bootstrap`.
- Live or native provider path available for the release-grade run.
- Worker and reviewer agents configured with distinct identities.
- Review policy enabled for the target task.
- Bridge task subscription created for terminal accepted-final delivery.

## Test Steps

1. Start the isolated daemon and web proxy from the bootstrap manifest.
   **Expected:** Daemon health, UDS access, and web proxy target resolve to the isolated lab.

2. Create a task execution profile that selects the worker identity, provider/model override,
   sandbox mode, and participant policy.
   **Expected:** CLI, HTTP, and UDS reads return the same normalized profile and generated JSON
   fields.

3. Create a task with review policy enabled and start the worker run.
   **Expected:** The run is claimed only by the eligible worker, and `tasks.current_run_id` is set.

4. Have the worker complete intentionally incomplete work.
   **Expected:** The terminal run is persisted, `current_run_id` clears, and a review request is
   created without rewriting the run terminal status.

5. Route the review request to the configured reviewer.
   **Expected:** The reviewer session is bound to the persisted review request, and the original
   worker is not selected when original-worker review is disallowed.

6. Submit a rejected verdict through `submit_run_review` from the bound reviewer session with
   missing-work items and next-round guidance.
   **Expected:** The verdict is persisted through task-service authority, a continuation run is
   created once, and `parent_review_id` lineage is recorded.

7. Inspect `/agent/context` and the worker session prompt overlay for the continuation run.
   **Expected:** The task context bundle contains the same review continuation guidance and no raw
   claim token.

8. Complete the continuation run with corrected work.
   **Expected:** A follow-up review request is created for the next round and review round
   counters increase monotonically.

9. Submit an approved verdict from the bound reviewer session.
   **Expected:** The task is accepted without rewriting prior run statuses, and review state shows
   the final approved outcome.

10. Confirm bridge terminal notification delivery and cursor state.
    **Expected:** Exactly one accepted-final terminal notification is delivered, cursor sequence
    advances only after confirmed delivery, and diagnostics expose last delivery metadata.

11. Inspect the same task in CLI, HTTP, UDS, native-tool output, and web orchestration tab.
    **Expected:** Profile, review state, continuation lineage, `latest_event_seq`, and bridge
    diagnostics match across every surface.

## Behavioral Evidence

- Operator journey transcript for task/profile/review/notification commands.
- Native reviewer tool call transcript or model/tool trace.
- Task, run, review, subscription, cursor, and event sequence identifiers.
- Browser screenshot or trace of the web orchestration tab after approval.
- Bridge delivery evidence with delivery id and cursor sequence.

## Disruption Probes

- Restart daemon after step 6 and before claiming the continuation run.
- Replay the rejected verdict with the same delivery id.
- Attempt to replay the rejected verdict with conflicting payload.
- Temporarily fail bridge delivery before the successful final delivery.

