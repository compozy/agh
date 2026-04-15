# Stock Bridge Secret Resolution via Env Refs

## Summary

- Make persisted bridge secret bindings usable in the stock daemon by wiring a default env-backed `BridgeSecretResolver` into normal daemon construction.
- Keep bridge secret bindings reference-only. In the stock binary, `vault_ref` supports only `env:NAME`; this change does not add raw secret values to the bridge HTTP/UDS surface.
- Fail early on unsupported ref syntax when a binding is written, and fail clearly on missing env values during boot/restart with an actionable error instead of `daemon: bridge secret resolver is required`.

## Key Changes

- **Daemon composition**
  - Add a built-in env-backed resolver in `internal/daemon` that uses `Daemon.getenv`.
  - Install it by default in the stock path when `WithBridgeSecretResolver(...)` is not provided.
  - Preserve `WithBridgeSecretResolver(...)` as the override for tests and future backends.

- **Binding validation and contract**
  - Define the stock-daemon `vault_ref` format as `env:NAME`.
  - Add stock-path validation so `PUT /bridges/:id/secret-bindings/:binding_name` rejects empty env names and unsupported schemes such as generic `vault://...`.
  - Keep the existing `BridgeSecretResolver` interface stable; add an optional validator capability in the daemon path so the default resolver can validate refs before persistence without forcing an interface break for custom resolvers.
  - Update contract comments/OpenAPI descriptions and examples to state that the stock daemon currently supports `env:` refs only.

- **Runtime behavior**
  - Resolve `env:` bindings during bridge runtime launch and pass the resolved values into `InitializeBridgeBoundSecret` exactly as today.
  - On missing or empty env vars, return a precise config/auth error that names the binding and env var; keep the current lifecycle rollback behavior unchanged for this fix.
  - Do not change provider-scoped runtime handshake shape, extension ownership, or bridge lifecycle orchestration beyond replacing the missing-resolver failure with real stock resolution.

## Test Plan

- Unit: env ref parsing/validation accepts `env:TG_TOKEN` and rejects empty, malformed, or unsupported refs.
- Unit: stock daemon composition installs the default env resolver only when no custom resolver is injected.
- Unit: bridge secret binding writes reject unsupported refs in the stock path and still accept valid `env:` refs.
- Integration: persisted `env:` bindings resolve during boot/restart and appear in the provider initialize handshake bound-secret payload.
- Integration: missing env vars fail with the new actionable error message, not `bridge secret resolver is required`.

## Assumptions

- No new raw-secret write API or CLI is added in this fix.
- `vault_ref` remains the field name to avoid broader contract churn, even though stock support is limited to `env:` refs for now.
- A future AGH-managed secret store can be introduced later as another resolver backend and ref scheme without changing the bridge launch handshake again.
