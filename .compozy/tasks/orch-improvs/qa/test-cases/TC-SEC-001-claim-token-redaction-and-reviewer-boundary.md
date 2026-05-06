# TC-SEC-001: Claim Token Redaction And Reviewer Boundary

**Priority:** P0

**Objective:** Prove token-fenced task ownership and reviewer authority boundaries hold across
runtime state, prompts, native tools, API responses, SSE, web UI, and docs-visible examples.

**Requirements Covered:** tasks 05, 09-10, 15, 17, 22-24, 26-27, 29; ADR-001, ADR-007, ADR-008,
ADR-009.

## Preconditions

- Isolated QA lab with a claimed active run and at least one review request.
- Access to worker session, reviewer session, operator CLI, HTTP, UDS, and web UI.
- Ability to inspect task context bundle, task stream payloads, and native tool visibility.

## Test Steps

1. Inspect worker `/agent/context` and session prompt overlay.
   **Expected:** Context includes bounded task/run data, but no raw `claim_token` field or
   `agh_claim_*` value.

2. Inspect reviewer `/agent/context` for a bound review session.
   **Expected:** Reviewer sees the reviewed run bundle and continuation context only through
   review binding; reviewer does not receive a worker lease or raw claim token.

3. Inspect operator-facing HTTP, UDS, CLI JSON, web UI, and SSE event payloads.
   **Expected:** No raw claim token is exposed in public surfaces or generated examples.

4. Attempt worker terminal mutation with a missing or wrong claim token.
   **Expected:** Mutation is rejected and `tasks.current_run_id` remains consistent.

5. Attempt review verdict submission from operator context through native tool access.
   **Expected:** Native `submit_run_review` is unavailable without a persisted reviewer binding.

6. Attempt review verdict submission through web UI.
   **Expected:** No verdict submission control exists; web remains read-only for verdict authority.

7. Attempt to mutate review state by channel message or bridge notification.
   **Expected:** No task review state changes; only task-service or explicit transport authority can
   persist verdicts.

## Behavioral Evidence

- Redaction search output for API responses, prompts, task stream frames, and browser content.
- Negative native-tool availability transcript.
- Rejected mutation responses with status/error details.
- Store or API evidence that state remained unchanged after unauthorized attempts.

## Disruption Probes

- Include token-like text inside task summaries and review text to prove text redaction catches
  nested string values.
- Reconnect SSE after a redacted event and verify replay does not leak the token.

