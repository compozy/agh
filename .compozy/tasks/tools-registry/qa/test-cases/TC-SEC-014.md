# TC-SEC-014 — Hook payloads, events, and result envelopes redact sensitive fields

- **Priority:** P1
- **Type:** Security / redaction
- **Trace:** Task 04, Task 06, Safety Invariants 11, 12

## Objective

Prove that fields marked sensitive in the descriptor `input_schema` (or by hook configuration) are redacted from result envelopes, hook payloads, telemetry events, and SSE streams. Confirm the `Redactions` list in `ToolResult` accurately points to the redacted positions.

## Preconditions

- A `native_go` tool whose `input_schema` marks one property as sensitive (test harness flag).
- A pre-call hook that logs payloads.
- Sentinel input value: `tools.sensitive.field:LEAK_v1`.

## Test Steps

1. Invoke the tool with the sensitive field set to the sentinel.
   - **Expected:** Successful response; `result.redactions` lists the path.
2. Inspect daemon logs, hook payload logs, event journal.
   - **Expected:** Sentinel does not appear; redacted placeholder used.
3. Inspect SSE event payload for `tool.call_completed`.
   - **Expected:** Sensitive field absent from envelope; only path/redaction-kind exposed.
4. Run sentinel scan across all `qa/logs/` and `qa/traces/` for `tools.sensitive.field:LEAK_v1`.
   - **Expected:** Zero matches.

## Edge Cases

- Hooks cannot widen visibility: a hook that sets `decision.visible_to_session = true` MUST NOT bypass redaction.
- Result truncation interacts with redaction: if redacted region is truncated, both metadata flags are present.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing partial
- **Command/Spec:** `go test ./internal/tools -run TestRedactionEnvelope`
