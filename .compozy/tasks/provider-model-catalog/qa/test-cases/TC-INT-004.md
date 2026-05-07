# TC-INT-004: `/api/openai/v1/models` HTTP-Only Registration + Filter

**Priority:** P0
**Type:** Integration
**Systems:** `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, `internal/api/core/model_catalog.go`.
**Requirement:** TechSpec OpenAI-Compatible Projection, Task 07.
**Status:** Not Run

## Objective

Verify `/api/openai/v1/models` is registered only on HTTP, returns the OpenAI-shaped projection with `agh` metadata, accepts `provider_id` filter, and is absent from UDS routes.

## Preconditions

- [ ] Daemon running with seeded catalog and bearer auth.

## Test Steps

1. **HTTP `GET /api/openai/v1/models`.**
   - **Expected:** 200; body `{"object":"list","data":[...]}`; each item has `id`, `object="model"`, `created=0`, `owned_by=<provider_id>`, `agh.{provider_id, display_name, supports_tools, supports_reasoning, availability_state, reasoning_efforts, context_window, max_output_tokens, sources}`.
2. **`provider_id` filter.**
   - Command: `GET /api/openai/v1/models?provider_id=codex`.
   - **Expected:** Subset filtered; deterministic order.
3. **Unknown `provider_id`.**
   - Command: `GET /api/openai/v1/models?provider_id=unknown-xyz`.
   - **Expected:** 200 with empty `data` array; no error.
4. **UDS does not expose the route.**
   - Command: hit UDS path `/api/openai/v1/models`.
   - **Expected:** 404 (route not registered); UDS routes table only includes the native catalog dispatcher.
5. **Refresh route absent for OpenAI projection.**
   - Command: HTTP `POST /api/openai/v1/models`.
   - **Expected:** 404 / method not allowed; refresh remains exclusive to native catalog routes.
6. **Source identity exposed in `agh.sources`.**
   - **Expected:** Array of `source_id` strings ordered consistently with native projection.

## Audit Coverage

- C5, C8.
- SI-9 (no secret in OpenAI payload).

## Pass Criteria

- All steps match documented behavior.

## Failure Criteria

- UDS exposes the OpenAI route.
- Filter ignored.
- `agh` metadata missing.
