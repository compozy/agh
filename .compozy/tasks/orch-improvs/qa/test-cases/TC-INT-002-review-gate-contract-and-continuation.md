# TC-INT-002: Review Gate Contract, Binding, And Continuation Authority

**Priority:** P0

**Objective:** Prove review requests, routing, reviewer binding, native verdict submission,
idempotency, continuation runs, and failure policies are owned by task service state and not by
channels, scheduler, web UI, bridge notifications, or prompt text.

**Requirements Covered:** tasks 09-10, 14-17, 19, 22-23, 29; ADR-006, ADR-007, ADR-008, ADR-009.

## Preconditions

- Isolated QA lab with worker and reviewer identities.
- Review profile configured with original-worker exclusion.
- Access to CLI, HTTP, UDS, and native tool invocation from a reviewer session.

## Test Steps

1. Request a run review through CLI and inspect it through HTTP and UDS.
   **Expected:** Request id, run id, status, policy, attempt, and reviewer selectors match across
   surfaces.

2. Route the review request.
   **Expected:** ReviewRouter selects or creates an eligible reviewer session and persists the
   binding.

3. Attempt to call `submit_run_review` outside the reviewer-bound session.
   **Expected:** Tool is unavailable or rejected with a deterministic authorization error.

4. Attempt to submit a verdict from the original worker when original-worker review is disabled.
   **Expected:** Submission is rejected and no review state changes.

5. Submit a rejected verdict from the bound reviewer session.
   **Expected:** Verdict, reason, missing work, and next-round guidance are persisted; exactly one
   continuation run is created.

6. Replay the same rejected verdict with the same delivery id.
   **Expected:** Existing verdict and continuation run are returned; no duplicate continuation run
   appears.

7. Replay the same delivery id with conflicting verdict content.
   **Expected:** Conflict is rejected and persisted state remains unchanged.

8. Submit timeout/error/invalid-output attempts until failure policy applies.
   **Expected:** Attempt rows increase monotonically, retries stop at configured bounds, and final
   failure state matches policy.

9. Submit an approved verdict for the corrected run.
   **Expected:** Review is recorded as terminal approved outcome, with no rewrite of prior run
   status or channel-owned verdict.

## Behavioral Evidence

- Review ids, run ids, delivery ids, continuation run ids, attempt numbers, and review round.
- Native tool availability proof for bound and unbound sessions.
- HTTP/UDS/CLI parity outputs.
- Store or API evidence proving exactly one continuation run per rejected review.

## Disruption Probes

- Crash or stop daemon immediately after rejected verdict persistence and before continuation claim.
- Force a no-route review request and verify deterministic task-service diagnostic state.
- Submit a channel message that looks like a verdict and verify it has no authority.

