# Hard-Cut Rename: Bridges + Network Channels

## Summary

- Execute a rename in two ordered passes: first free the term `channel` by renaming the existing external messaging subsystem from `channel` to `bridge`; only then rename AGH Network `space` to `channel`.
- Treat this as a hard cut across code, storage, APIs, CLI, extension protocol, specs, and RFCs. Do not ship aliases, dual JSON fields, alternate CLI flags, deprecated paths, schema fallbacks, or old/new namespaces side by side.
- Keep `internal/network` as the package name. Rename the namespace concept inside it to `channel`. Rename the external adapter subsystem package from `internal/channels` to `internal/bridges`.

## Implementation Changes

- Phase 1: rename the external adapter domain from `channel` to `bridge` in `internal/channels -> internal/bridges`, daemon wiring, observability, API handlers, CLI client/commands, OpenAPI, codegen, tests, docs, and `.compozy/tasks/channel-adapters` artifacts. Use `Bridge`, `BridgeStatus`, `BridgeRoute`, `BridgeSecretBinding`, `BridgeDeliveryMetrics`, `BridgeService`, `bridgeRuntime`, and `bridgepkg`.
- Phase 1 public namespace changes: CLI `agh channel ...` becomes `agh bridge ...`; HTTP/UDS `/api/channels...` becomes `/api/bridges...`; OpenAPI tag `channels` becomes `bridges`; capability `channel.adapter` becomes `bridge.adapter`; host methods `channels/messages/ingest`, `channels/instances/get`, `channels/instances/report_state`, and extension service `channels/deliver` become `bridges/messages/ingest`, `bridges/instances/get`, `bridges/instances/report_state`, and `bridges/deliver`; vault/test fixture prefixes `vault://channels/...` become `vault://bridges/...`.
- Phase 1 storage changes: rename `channel_instances`, `channel_secret_bindings`, `channel_routes`, and `channel_ingest_dedup` to `bridge_instances`, `bridge_secret_bindings`, `bridge_routes`, and `bridge_ingest_dedup`. Rewrite schema builders/assertions to the final names only; do not add migration code for the old table names.
- Phase 2: rename AGH Network `space` to `channel` across `internal/network`, session opt-in, config, store, API contract, CLI, prompt wrappers, bundled skill docs, and RFC examples. Inside `internal/network`, use `Channel`, `ChannelInfo`, `ValidateChannel`, `JoinChannel`, `LeaveChannel`, `ListChannels`, and NATS subjects `agh.network.v0.<channel>.broadcast` / `agh.network.v0.<channel>.peer.<route_token>`.
- Phase 2 cross-package naming rule: inside `internal/network` use the bare noun `Channel`; outside `internal/network` use `NetworkChannel` for session/config/store structs where a bare `Channel` would be ambiguous. User-facing JSON, TOML, and CLI still use `channel`.
- Phase 2 public namespace changes: session create payload/flags `space` and `--space` become `channel` and `--channel`; session env `AGH_SESSION_SPACE` becomes `AGH_SESSION_CHANNEL`; network API `/api/network/spaces` becomes `/api/network/channels`; peer filter query `space` becomes `channel`; network send payload/response/envelope field `space` becomes `channel`; contract payloads `NetworkSpacePayload` / `NetworkSpacesResponse` become `NetworkChannelPayload` / `NetworkChannelsResponse`; config `network.default_space` becomes `network.default_channel`.
- Phase 2 storage changes: rename `sessions.space` to `sessions.channel`, `network_audit_log.space` to `network_audit_log.channel`, `store.NetworkAuditEntry.Space` to `Channel`, session metadata JSON `space` to `channel`, and related query/filter helpers/assertions. Rewrite current schema helpers and migration helpers to the final column names only; do not preserve `space` fallback paths.
- Phase 3: update specs and docs in place so the repo has one vocabulary everywhere. Rewrite `docs/rfcs/003_agh-network-v0.md` and `docs/rfcs/004_agh-network-v1.md` to use `channel`, update subject mappings, examples, and diagrams, keep the v0/v1 relationship text consistent after the rename, rename channel-adapter techspec and ADR docs to bridge-adapter naming, regenerate API spec/codegen outputs, and update help text and prompt wrappers.

## Public APIs / Interfaces / Types

- CLI: `agh bridge {list|get|create|update|enable|disable|restart|routes|test-delivery}` replaces `agh channel ...`; `agh network channels` replaces `agh network spaces`; `agh network send --channel`; `agh session new --channel`.
- HTTP/UDS API: `/api/bridges...` replaces `/api/channels...`; `/api/network/channels` replaces `/api/network/spaces`; request/response fields `space` become `channel` in session creation, network send, network peers, network inbox, and surfaced envelopes.
- Extension protocol: provide capability `bridge.adapter`; host methods under `bridges/*`; extension service `bridges/deliver`; runtime init payload exposes bridge instance data and bound secrets under bridge naming.
- Config, env, and schema: `network.default_channel` replaces `network.default_space`; `AGH_SESSION_CHANNEL` replaces `AGH_SESSION_SPACE`; session meta and DB schema use `channel`; bridge DB schema uses `bridge_*` names only.

## Test Plan

- Rename and update all unit and integration tests that assert old nouns in CLI paths, API routes, JSON field names, NATS subjects, env vars, DB table or column names, OpenAPI tags, and extension protocol method names.
- Add explicit regression checks that the old surfaces are gone: no `/api/channels`, no `/api/network/spaces`, no `--space`, no `AGH_SESSION_SPACE`, no `channel.adapter`, no `channels/*`, no `sessions.space`, no `network_audit_log.space`, and no `channel_*` bridge tables.
- Keep the behavioral coverage intact for bridge lifecycle, route resolution, delivery target resolution, bridge runtime reload and secret binding, network join/leave/list/send/inbox, prompt wrappers, session startup env injection, audit logging, and OpenAPI/codegen snapshots.
- Final verification gate for the implementation turn: `make verify` must pass.

## Assumptions

- Historical memory ledgers may remain as historical artifacts; current product, docs, and spec surfaces must use the new vocabulary only.
- Because this is unreleased alpha, the rename is applied in place under the current API and protocol roots instead of adding compatibility shims or dual-version support.
- The web app does not currently consume the bridge or network surfaces at runtime beyond generated contract artifacts, so regenerating API spec and codegen is sufficient unless a new consumer appears during implementation.
