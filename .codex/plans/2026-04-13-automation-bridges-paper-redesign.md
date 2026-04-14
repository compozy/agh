# Bridges Paper Implementation Plan

## Summary

- Add a new `Bridges` page in the web app at `/bridges`, covering the four Paper states: empty, bridge selected, bridge with no routes, and create bridge modal.
- Implement the feature as a dedicated `web/src/systems/bridges` system, with the route owning orchestration and presentational components staying pure.
- Close the two real backend gaps before the UI depends on them: an HTTP bridge-provider catalog and `last_success_at` telemetry. Do not add fake delivery-tuning controls.

## Key Changes

- Add `GET /api/bridges/providers` to HTTP and UDS, backed by the bridge runtime rather than the generic UDS-only extensions API.
- Extend bridge-capable extension manifests with a `bridge` section. For `bridge.adapter` extensions, require `bridge.platform` and `bridge.display_name`.
- Reuse the manifest description as provider card copy. Do not invent extra UI metadata.
- Extend bridge delivery telemetry with `last_success_at` in broker metrics, observer health, shared contracts, OpenAPI, and generated web types.
- Do not implement per-bridge `retry` or `timeout` settings in this round. The runtime does not support them yet, and the UI must not imply otherwise.
- Add a new `Bridges` item to the main sidebar and a new route file `web/src/routes/_app/bridges.tsx`.
- Create `web/src/systems/bridges` with typed API adapters, query keys/options, hooks, and Paper-aligned components.
- Keep filtering and selection local to the route: scope pills, search query, selected bridge, create-dialog state, and test-delivery dialog state.
- Render the page with existing AGH design primitives and tokens. Only generalize shared components when there is an obvious cross-route fit.
- Use installed providers dynamically in the create modal. Do not hardcode Slack/Discord/Teams cards.
- Implement `Test Delivery` as a small modal that edits `mode`, `peer_id`, `thread_id`, and `group_id`, then calls the existing dry-run endpoint.
- In create flow, submit `enabled = true` and `status = "starting"` by default.

## Public Interfaces / Type Changes

- New endpoint: `GET /api/bridges/providers`
- New shared payloads: `BridgeProviderPayload` and `BridgeProvidersResponse`
- Extended bridge service surface: `ListProviders(ctx context.Context)`
- Extended bridge extension manifest shape with a `bridge` section
- Extended bridge health payloads with `last_success_at`

## Tests

- Backend unit tests for manifest validation, provider catalog projection, broker success telemetry, and payload conversions.
- Backend transport/integration tests for `GET /api/bridges/providers` and updated health/spec output.
- Frontend adapter, route, and component tests for empty state, selected bridge state, no-routes state, create modal, and test-delivery flow.
- Final verification:
  - `make codegen`
  - `make web-lint`
  - `make web-typecheck`
  - `make web-test`
  - `make verify`

## Assumptions

- Provider catalog is sourced from installed `bridge.adapter` extensions, not static cards.
- `Test Delivery` is modal-based even though the Paper artboards only show the footer CTA.
- Only supported delivery defaults in this round are `mode`, `peer_id`, `thread_id`, and `group_id`.
- Backlog microcopy must reflect real telemetry only. No invented “pending retry” metrics.
- `network` remains a separate subsystem. This work is limited to `bridges`.
