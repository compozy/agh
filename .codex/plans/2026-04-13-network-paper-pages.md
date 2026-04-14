# Network Paper Pages Plan

## Summary

- Add the new page `/_app/network` in `web/` with two internal tabs, `Channels` and `Peers`, covering the six Paper states: channels selected, channels empty, channel without messages, create-channel modal, peers selected, and peers empty.
- Follow the RFC and current session model: joining a `channel` means creating a **new session** for each selected agent with `channel` set. Do not invent a standalone persistent channel entity outside that flow.
- Keep `Peers` observability-only in v1. Do not fake `Disconnect` or `Remove` actions before the backend models them honestly.

## Key Changes

### Backend

- Add `POST /api/network/channels` with body `{ channel, workspace_id, agent_names[] }`. The handler validates the channel name and creates **one new session per agent** using the existing `createSession` flow with `channel` populated.
- Add `GET /api/network/channels/{channel}` for the right-panel channel snapshot and `GET /api/network/channels/{channel}/messages` for the read-only timeline.
- Create a dedicated persisted network message log for the timeline, separate from the existing audit log: store accepted `say` envelopes once per `message_id`, with `channel`, `from`, `session_id` when present, `intent`, `text/body`, and `timestamp`. This avoids duplicate rows between local send/receive paths.
- Extend network audit/telemetry to include `delivered`, because the `Peers` view needs truthful `sent / received / rejected / delivered` counters.
- Add `GET /api/network/peers/{peer_id}` for the selected peer detail: identity, local `session_id` when available, current channel, last heartbeat, and message aggregates.
- Enrich local peers with a human `display_name` derived from session/agent metadata when possible. Remote peers continue to use their card `display_name` or `peer_id` fallback.

### Web

- Create `web/src/systems/network` following the app-renderer-systems pattern: adapters, types, query keys, query options, hooks, and pure presentational components.
- Add the route `/_app/network` with route-level orchestration for tab state, search state, selected row, and create-channel modal state.
- Reuse existing project tokens/components such as `WorkspacePageShell`, `Pill`, `PillButton`, `Input`, `Button`, and `Dialog`. Keep any new visual patterns local to `systems/network/components` unless a second real consumer appears.
- Add `Network` to the sidebar.

### Intentional Adaptations From Paper

- The `Create Channel` modal selects **local agents**, not remote peers and not already-running sessions.
- The channel timeline is read-only and shows `say` messages only. `direct`, `receipt`, and low-level trace traffic do not belong in this UI.
- The `Peers` detail panel stays truthful to the current backend/network model and exposes read-only identity, channels, and counters without fake control actions.

## Public Interfaces / Types

- New endpoints:
  - `POST /api/network/channels`
  - `GET /api/network/channels/{channel}`
  - `GET /api/network/channels/{channel}/messages`
  - `GET /api/network/peers/{peer_id}`
- New payloads:
  - `CreateNetworkChannelRequest`
  - `CreateNetworkChannelResponse`
  - `NetworkChannelDetailPayload`
  - `NetworkChannelMessagePayload`
  - `NetworkPeerDetailPayload`
- Domain/store changes:
  - new append-only storage for timeline messages keyed by `message_id`
  - `NetworkAuditEntry.Direction` extended with `delivered`
  - `NetworkPeerPayload` enriched with a display label suitable for the UI

## Test Plan

### Backend

- Creating a channel creates new sessions with the requested `channel` and `workspace_id`, without moving existing sessions.
- Timeline persistence stores accepted `say` envelopes without duplicating a `message_id`.
- Peer metrics include `delivered`.
- Peer detail and channel detail return complete, ordered payloads.

### Frontend

- The `network` route renders correctly for `Channels` and `Peers`.
- Empty states match the Paper layouts.
- The create-channel modal lists local agents and submits the correct mutation.
- Selected-channel and no-messages states use the real API.
- Peer detail remains read-only and renders counters, channel membership, and local session information when available.

### Final Verification

- `make codegen`
- `make web-lint`
- `make web-typecheck`
- `make web-test`
- `make verify`

## Assumptions

- A channel is materialized by the set of sessions created with that channel name; there is no separate `network_channels` persistence layer in this delivery.
- Each selected agent in the modal produces a new session in the new channel.
- `Peers` has no control actions in v1.
- When the Paper conflicts with the RFC/current daemon model, the implementation follows the truthful daemon model while staying as close as possible to the approved layout.
