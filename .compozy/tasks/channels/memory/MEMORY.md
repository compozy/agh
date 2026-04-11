# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 02 established the daemon-owned registry and policy-driven routing layer in `internal/channels/`; later channel tasks can depend on canonical instance lifecycle and route ownership there instead of extending `globaldb` directly.
- Task 03 added the channel-side outbound target seam on `channels.Service.ResolveDeliveryTarget` and `channels.BuildDeliveryTarget`; later channel, API, CLI, and automation tasks should reuse that contract instead of decoding `delivery_defaults` themselves.
- Task 06 established the daemon-owned outbound delivery runtime in `internal/channels.Broker`; follow-on channel tasks should reuse its per-route ordering, bounded/coalescing queue semantics, ack tracking, and delivery snapshots instead of streaming raw session events to adapters.
- Task 07 composed the daemon-owned channel runtime in `internal/daemon`: boot now wires the registry, delivery broker, session notifier projection, bound-secret launch resolver, and default extension-manager Host API/dependency injection together as one runtime exposed through `Daemon.channels`.
- Task 08 exposed daemon-owned channel management over shared HTTP and UDS `/api/channels` endpoints plus generated OpenAPI; follow-on CLI and observability work should reuse the shared `internal/api/contract` channel DTOs and `core.ChannelService` transport seam instead of inventing channel transport models per client.
- Task 10 added daemon-owned per-instance channel observability: `internal/observe` now aggregates status, route counts, auth/runtime failures, and broker backlog/drop telemetry per channel instance; `/api/observe/health` exposes an additive `channels` summary, `/api/channels` returns `channel_health`, and channel detail responses include nested `health`. Follow-on UI and adapter tasks should consume these health surfaces instead of inferring channel state from extension-process health alone.

## Shared Decisions

- `ResolveOrCreateRoute` is the session-continuity path: it reuses the existing session bound to a canonical routing key and refreshes route activity metadata.
- `UpsertRoute` is the explicit ownership-rebind path: it replaces the stored session/agent for an existing canonical routing key without creating duplicate route rows.
- Channel instance lifecycle now treats `enabled=false` and `status=disabled` as the same disabled state, and re-entry from disabled must go through `starting`.
- Delivery-target resolution merges instance `delivery_defaults` with explicit overrides, keeps explicit values when both are present, and defaults the resolved target mode to `direct-send` when neither side sets one.
- `channel.adapter` is the extension capability surface for channel-capable adapters, and it must negotiate `channels/deliver` during `initialize`; capability-service validation now derives required extension service methods from declared `provides`.
- Channel-scoped launch material is delivered only through `initialize.runtime.channel`, which carries the selected `ChannelInstance` plus bound secrets for that instance; no generic vault or secret lookup Host API was added for extensions.
- The reserved channel Host API surface for follow-on tasks is `channels/messages/ingest`, `channels/instances/get`, and `channels/instances/report_state`, gated by `channel.write` and `channel.read`.
- Channel Host API instance lookup, state reporting, and inbound ingest now authorize against the bound `initialize.runtime.channel` instance; mismatched ownership or `channel_instance_id` values are treated as not-found instead of exposing foreign instance metadata.
- Inbound `channels/messages/ingest` serializes work per canonical routing-key hash, uses `channel_ingest_dedup` for duplicate suppression, and only persists the dedup record after prompt initiation succeeds; expired dedup rows are cleaned opportunistically on ingest using configurable TTL and cleanup cadence.
- ACP/session projection into the delivery broker stays in `internal/extension` via prompt registration and `ChannelDeliveryNotifier`; `internal/channels` remains ACP-agnostic to avoid a `channels -> acp` import cycle.
- Negotiated outbound delivery now flows through `internal/extension.Manager.DeliverChannel` and the `channels/deliver` request/ack/snapshot contract; adapter work should build on that runtime caller instead of inventing a second transport seam.
- Daemon composition for channels stays in `internal/daemon`: follow-on API, CLI, and observability work should consume the composed `Daemon.channels` runtime and the existing extension-manager injection path instead of rebuilding registry, broker, or bound-secret resolution elsewhere.

## Shared Learnings
- Very fast prompted sessions can persist `agent_message` or `done` events before channel delivery registration completes; `HostAPIHandler.submitPrompt` now performs a short persisted-event seed fetch so prompt registration can backfill those early events.
- Channel adapters can receive RPC code `-32003` / message `Not initialized` if they call Host API methods immediately after `initialize` starts; future adapters should not assume Host API readiness until the initialize response returns and should use bounded retry or deferred startup work instead of fixed sleeps.

## Open Risks
- Delivery broker state is still in-memory only; task 06 covers extension reconnect/restart within the daemon lifetime, but daemon-process restart persistence is still future work if later tasks require it.

- Global-scope channel instances still need a defined workspace and default-agent selection strategy before inbound Host API ingest can create new sessions for them.

- Channel launch selection is currently keyed by extension name and rejects multiple enabled channel instances for the same extension; future multi-instance adapter work will need an explicit daemon-side selection strategy.

## Handoffs
