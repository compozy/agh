# TC-UI-001: Web Orchestration Tab Operator Truth

**Priority:** P1

**Objective:** Prove the web task detail and run detail surfaces display runtime truth for
execution profiles, review state, bridge notification diagnostics, and SSE resume state without
exposing controls the runtime does not support.

**Requirements Covered:** tasks 24-27, 28-29; ADR-003, ADR-007, ADR-010.

## Preconditions

- Daemon-served web app points at the isolated QA daemon through `AGH_WEB_API_PROXY_TARGET`.
- A task exists with an execution profile, at least one review request, and a bridge subscription.
- Playwright or `browser-use:browser` can drive a desktop viewport and a mobile viewport.

## Test Steps

1. Open the task detail page and select the Orchestration tab.
   **Expected:** The tab renders execution profile, task reviews, bridge notifications, and stream
   resume cards without layout overlap.

2. Inspect the execution profile card.
   **Expected:** The profile summary matches CLI/HTTP/UDS data; edit/delete controls are disabled
   while an active run exists and enabled when `current_run_id` clears.

3. Open the profile JSON editor, submit malformed JSON, then submit a valid full replacement.
   **Expected:** Malformed input shows validation feedback without mutation; valid input persists
   through task-service profile authority and refreshes all visible data.

4. Inspect task-level and run-level review cards.
   **Expected:** Review request, status, outcome, reason, missing work, and continuation guidance
   match runtime state; the UI does not expose review verdict submission.

5. Inspect bridge notification diagnostics.
   **Expected:** Zero-state, failure-state, and delivered-state diagnostics match CLI/API output.

6. Inspect stream resume state before and after an event.
   **Expected:** The UI seeds from `latest_event_seq`, receives named task/review/notification SSE
   events, updates state from `idle` to `connected` or event-driven statuses, and reports parse or
   connection errors truthfully.

7. Repeat the core inspection at mobile width.
   **Expected:** Text and controls remain readable, no card nests inside another card, and no
   controls overlap.

## Behavioral Evidence

- Browser trace or screenshots for desktop and mobile viewports.
- Task id, latest event sequence, subscription id, and review id displayed in the UI.
- Cross-surface comparison output from CLI or API.
- Playwright report or browser-use transcript.

## Disruption Probes

- Open the Orchestration tab after page load to catch disabled-to-enabled stream-state regressions.
- Simulate named SSE events and fallback `onmessage` frames.
- Force profile mutation rejection while an active run is present.

