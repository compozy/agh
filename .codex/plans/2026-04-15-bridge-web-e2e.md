# Bridge Web E2E Closure

## Summary

- Close `/bridges` as an operator-ready surface: edit existing bridges, bind secrets, control lifecycle, and reflect live bridge health/status.
- Keep the detail panel as the operational cockpit; do not add a separate settings page.
- Land missing shared-contract work first so the web can consume the feature through generated types instead of ad hoc request code.

## Key Changes

### Shared API / Backend

- Publish missing secret-binding endpoints in the shared OpenAPI spec and regenerate generated clients/types:
  - `GET /api/bridges/{id}/secret-bindings`
  - `PUT /api/bridges/{id}/secret-bindings/{binding_name}`
  - `DELETE /api/bridges/{id}/secret-bindings/{binding_name}`
- Add bridge-health SSE endpoints in HTTP and UDS:
  - `GET /api/bridges/health/stream`
- Implement `StreamBridgeHealth` in `internal/api/core`, reusing the existing SSE helper pattern and emitting:
  - one initial `snapshot`
  - subsequent `snapshot` events only when bridge health changes
  - one `error` event before exit if polling fails
- Keep the SSE contract out of OpenAPI, matching the project's existing stream convention.

### Web Data Layer

- Extend bridge system types with update payloads, secret-binding payloads, and bridge health stream snapshot types.
- Add adapters for:
  - `updateBridge`
  - `listBridgeSecretBindings`
  - `putBridgeSecretBinding`
  - `deleteBridgeSecretBinding`
  - `enableBridge`
  - `disableBridge`
  - `restartBridge`
- Add query keys/options/hooks for secret bindings and mutation hooks for update, lifecycle, and secret CRUD.
- Add a dedicated bridge health stream hook that consumes `/api/bridges/health/stream` and patches React Query cache snapshots without replacing the normal queries as the source of truth.

### Web UI

- Reuse the existing bridge form model so create and edit share the same mutable bridge fields.
- Add an edit dialog for mutable bridge fields:
  - `display_name`
  - `dm_policy`
  - `routing_policy`
  - `provider_config`
  - `delivery_defaults`
- Upgrade the detail panel into the full operational surface with:
  - `Edit`
  - `Enable` / `Disable`
  - `Restart`
  - inline secret-slot binding rows
- Use an env-first secret UX for the stock daemon:
  - operator types `AGH_BRIDGE_*`
  - UI submits `vault_ref: env:NAME`
  - UI fixes `kind` to `slot.name`
- Show effective live status using `health.status` when present, falling back to persisted `bridge.status`.
- After config or secret changes, mark the bridge as restart-required until a successful restart or enable clears the hint.

## Test Plan

- Add backend tests for the new spec entries and bridge health stream behavior.
- Add web adapter and hook tests for update, lifecycle, secret CRUD, and SSE-driven cache updates.
- Add component and route integration tests for:
  - editing a bridge
  - binding/removing secrets
  - lifecycle controls
  - health/status updates from the SSE stream
- Run:
  - `make web-lint`
  - `make web-typecheck`
  - `make web-test`
  - `make verify`

## Assumptions

- The stock daemon path is the target UX, so secrets are optimized around `env:NAME`.
- Realtime means live bridge health/status updates, not live route streaming.
- Secret-binding CRUD must be added to the shared spec/codegen because the endpoints exist in the server but are currently absent from generated web types.
