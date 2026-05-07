# TC-FUNC-005: `models.dev` Source - TTL, Disable, Legacy Aliases

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog/modelsdev.go`
**Requirement:** TechSpec `Models.dev Source` + Config Lifecycle.
**Status:** Not Run

## Objective

Verify the `models.dev` source honors the configurable TTL, endpoint, timeout, and disable switch; tolerates current and legacy schema aliases; and never proves account-level availability.

## Preconditions

- [ ] `httptest`-based stub server mirroring `models.dev/api.json`.
- [ ] Config writes `[model_catalog.sources.models_dev]` with stub endpoint.

## Test Steps

1. **Current-schema parse.**
   - Stub returns canonical fields (`reasoning`, `tool_call`, `limit.context`, `cost.input`, `cost.output`).
   - **Expected:** Rows include `supports_reasoning`, `supports_tools`, `context_window`, `cost_*` populated.
2. **Legacy-schema parse.**
   - Stub returns `supportsReasoning`, `supports_reasoning`, `supportsTools`, `supports_tools`, `contextWindow`, `maxInputTokens`, `maxOutputTokens`, `pricing.input`, `pricing.output`.
   - **Expected:** All fields parse identically; tolerant aliases tested.
3. **TTL respected.**
   - Trigger refresh; immediately call list with `Refresh=false`.
   - **Expected:** Cached rows returned without HTTP call within TTL.
4. **Disable switch.**
   - Set `[model_catalog.sources.models_dev] enabled = false`.
   - **Expected:** Source status `refresh_state="idle"`, no outbound HTTP, rows absent for the source.
5. **Override endpoint and timeout.**
   - Set `endpoint = "http://127.0.0.1:0/api.json"`, `timeout = "1ms"`.
   - **Expected:** Source status records timeout error; redacted `last_error`; prior stale rows preserved.
6. **No account availability.**
   - Stub returns models for `codex` provider with `available=true` field.
   - **Expected:** Projection ignores `available` from `models.dev` (kind keeps `available=null`); availability remains `unknown` unless live/extension says otherwise.
7. **Provider-scoped status row.**
   - Stub spans 3 AGH providers; refresh once.
   - **Expected:** `model_catalog_sources` has 3 rows (one per provider) for `models_dev`; no blank-provider sentinel row.

## Audit Coverage

- C6 task tree (Task 03 + Task 05 wiring).
- SI-5, SI-13.

## Pass Criteria

- All schema variants parse.
- TTL/disable/override honored.
- Provider-scoped status rows preserved.

## Failure Criteria

- Any legacy alias fails parse.
- Disabled source still calls HTTP.
- Account availability inferred from `models.dev`.
