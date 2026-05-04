# TC-FUNC-027 — Dispatch ordering: schema → policy/availability recheck → hooks → handle → limiter → telemetry

- **Priority:** P0
- **Type:** Functional / dispatch
- **Trace:** Task 04, Safety Invariants 1, 2, 11

## Objective

Prove `Registry.Call` runs in the canonical order:

1. Validate input against `input_schema`.
2. Re-evaluate availability.
3. Re-evaluate policy (`EffectiveToolDecision`).
4. Run `tool.pre_call` hooks.
5. Call provider `Handle.Call`.
6. Apply `ResultLimiter` (truncate / redact).
7. Run `tool.post_call` or `tool.post_error` hooks.
8. Emit telemetry events.

## Test Steps

1. Invoke a tool with invalid input.
   - **Expected:** Error before policy/availability checks; no provider invocation; no hooks.
2. Invoke with valid input but session lacks lineage atom.
   - **Expected:** Recheck denies; no provider invocation; `tool.post_error` does NOT fire because policy denial precedes hooks (per dispatch ordering).
3. Pre-call hook denies.
   - **Expected:** Handle never called; `tool.post_error` may fire with hook denial.
4. Provider returns error.
   - **Expected:** `tool.post_error` fires; result limiter not applied to error path; telemetry recorded.
5. Provider returns oversized result.
   - **Expected:** Result limiter truncates with `truncated = true`; `tool.post_call` runs on truncated envelope; telemetry includes `result_bytes`, `truncated`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestDispatchOrdering`
