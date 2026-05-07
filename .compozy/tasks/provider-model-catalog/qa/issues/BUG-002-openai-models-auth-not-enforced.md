# BUG-002: `/api/openai/v1/models` returns catalog data without bearer auth

## Status

Open

## Severity

Critical

## Affected Test Cases

- TC-SEC-002: `/api/openai/v1/models` Auth + OpenAI-Shaped Errors
- TC-INT-004: `/api/openai/v1/models` HTTP-Only Registration + Filter
- TC-SCEN-002: Agent-Manageable Catalog Parity Journey

## Environment

- QA manifest: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`
- AGH home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`
- Base URL: `http://127.0.0.1:62444`
- Branch: `fix-migrations`
- Runtime version: `2debf0cf-dirty`

## Reproduction

1. Start the isolated daemon from the QA manifest.
2. Seed the `codex` catalog via Settings API and restart the daemon so the config source is registered.
3. Refresh the config source:

   ```bash
   ./bin/agh provider models refresh codex --source config --force -o json
   ```

4. Call the OpenAI-compatible model projection without an `Authorization` header:

   ```bash
   curl -i "http://127.0.0.1:62444/api/openai/v1/models?provider_id=codex"
   ```

5. Call it with an invalid bearer token:

   ```bash
   curl -i -H "Authorization: Bearer bad-token" \
     "http://127.0.0.1:62444/api/openai/v1/models?provider_id=codex"
   ```

## Expected Result

Per `_techspec.md`, Task 07, docs, and TC-SEC-002:

- Missing auth returns 401 or 403.
- Bad bearer auth returns 401 or 403.
- Both failures use the OpenAI-shaped envelope:

  ```json
  {"error":{"message":"...","type":"invalid_request_error","code":"unauthorized"}}
  ```

- No catalog rows are returned to an unauthenticated or unauthorized HTTP client.

## Actual Result

- Missing auth returned HTTP 200 with catalog rows.
- Bad bearer auth returned HTTP 200 with catalog rows.
- The seeded `manual-gpt` model was visible in both responses.

Evidence:

- `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/http-openai-models-codex-no-auth.txt`
- `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/http-openai-models-codex-bad-token.txt`

## Root-Cause Notes

- `internal/api/httpapi/routes.go` registers `/api/openai/v1/models` inside the regular `/api` group.
- `internal/api/httpapi/middleware.go` only applies loopback binding and CORS rejection paths to this route.
- `internal/config.HTTPConfig` currently contains only `host` and `port`; no generic bearer-token authority exists for HTTP `/api/*`.
- `internal/api/httpapi/model_catalog_test.go` covers loopback/CORS-shaped OpenAI errors but does not cover missing or invalid bearer tokens.

## Impact

The OpenAI-compatible projection leaks provider model catalog data to any client that can reach the loopback-bound HTTP daemon. This violates the accepted QA contract and the generated API/docs contract for unauthorized requests.

## Fix Requirements

- Define the daemon-owned HTTP API bearer-auth authority or revise the accepted contract through an explicit TechSpec/ADR update.
- If bearer auth remains required, add production enforcement for `/api/openai/v1/models` without breaking first-party local web access.
- Add regression tests proving missing and invalid bearer credentials do not return catalog data and use the OpenAI error envelope.
- Update docs and generated contracts if the accepted HTTP auth model changes.
