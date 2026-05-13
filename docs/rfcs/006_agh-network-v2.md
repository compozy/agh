# RFC: AGH Network v2

- **Status:** Current runtime contract
- **Authors:** AGH Core Team
- **Created:** 2026-05-13
- **Supersedes:** [RFC 003: AGH Network v0](003_agh-network-v0.md) for runtime envelope identity and NATS subjects; [RFC 004: AGH Network v1](004_agh-network-v1.md) for transport subject grammar
- **Preserves:** RFC 003 conversation-container model and RFC 004 trust-profile design, except where this RFC explicitly changes workspace identity and transport subjects

---

## Abstract

`AGH Network v2` is the workspace-qualified hard cut of AGH Network. It keeps the six core message
kinds, public-thread and direct-room surfaces, capability transfer, work lifecycle, and optional
`proof` placeholder from the earlier RFCs. It changes the identity boundary:

- every envelope carries `workspace_id`
- every channel is scoped by `workspace_id`
- every public thread, direct room, work unit, peer listing, recent item, pin, and last-read marker is
  scoped by the tuple that starts with `workspace_id`
- every NATS subject includes `workspace_id`
- channel-only subjects and channel-only API access are obsolete

The current AGH Runtime implements this v2 contract. It preserves `proof` as opaque JSON; it does
not yet verify the RFC 004 Ed25519 + JCS trust profile.

---

## 1. Scope

This RFC defines the current runtime and transport contract for AGH Network. It is a hard cut, not a
compatibility bridge.

v2 updates:

1. the required protocol identifier
2. the required envelope fields
3. conversation and work identity keys
4. NATS subject grammar
5. daemon API route shape
6. storage and read-model ownership rules
7. treatment of earlier channel-only traffic

v2 does not redefine:

1. the six message kinds: `greet`, `whois`, `say`, `capability`, `receipt`, and `trace`
2. public-thread and direct-room surface semantics except for workspace scoping
3. Peer Card shape
4. capability transfer body shape
5. receipt status and trace lifecycle states
6. the RFC 004 trust-profile cryptography

## 2. Compatibility Posture

v2 is intentionally incompatible with channel-only Network identity.

Implementations MUST NOT accept channel-only v2 subjects. Implementations SHOULD reject or ignore
traffic using earlier subject prefixes unless they explicitly run an archival interop adapter outside
the current AGH Runtime path.

AGH Runtime does not provide fallback readers, dual-write subjects, or channel-only API aliases for
current Network data. A channel slug can appear in multiple workspaces without merging data.

## 3. Terminology

### 3.1 Workspace ID

A `workspace_id` is the stable workspace identifier from `.agh/workspace.toml`. It is protocol-visible
and scopes Network identity. It is not the daemon registry row ID, a filesystem path, or a display
name.

A `workspace_id` MUST be non-empty and MUST NOT contain dots, spaces, NATS wildcards (`*`, `>`), or
control characters. These restrictions make the value safe for both JSON envelopes and NATS subjects.

### 3.2 Channel

A `channel` is a logical communication namespace inside one workspace. The same channel slug in two
different workspaces denotes two different channels.

A `channel` value MUST match `[a-z0-9][a-z0-9_-]{0,63}`.

### 3.3 Conversation Container

A conversation container is identified by:

- `(workspace_id, channel, "thread", thread_id)` for a public thread
- `(workspace_id, channel, "direct", direct_id)` for a direct room

`thread_id` values are scoped by `workspace_id + channel`. `direct_id` values are derived from
`(workspace_id, channel, sorted(peer_a, peer_b))` with the v2 direct-room domain separator.

### 3.4 Work

`work_id` identifies lifecycle-bearing work inside exactly one v2 conversation container. A
`work_id` is not global, not a queue lease, and not a routing key.

## 4. Envelope

Every v2 message is one UTF-8 JSON envelope.

### 4.1 Required Fields

| Field          | Type            | Required | Notes                                                                               |
| -------------- | --------------- | -------- | ----------------------------------------------------------------------------------- |
| `protocol`     | string          | yes      | MUST be `agh-network/v2`.                                                           |
| `id`           | string          | yes      | Collision-resistant message identifier.                                             |
| `workspace_id` | string          | yes      | Stable workspace ID that scopes the channel, containers, work, peers, and subjects. |
| `kind`         | string          | yes      | One of the six core message kinds.                                                  |
| `channel`      | string          | yes      | Workspace-scoped communication namespace.                                           |
| `surface`      | string          | no       | `thread` or `direct` for conversation-bearing messages.                             |
| `thread_id`    | string          | no       | Required when `surface:"thread"`.                                                   |
| `direct_id`    | string          | no       | Required when `surface:"direct"`.                                                   |
| `from`         | string          | yes      | Claimed sender peer ID.                                                             |
| `to`           | string or null  | no       | Target peer for directed transport delivery.                                        |
| `work_id`      | string          | no       | Required for `capability`, `receipt`, and `trace`.                                  |
| `reply_to`     | string          | no       | Message identifier being replied to.                                                |
| `trace_id`     | string          | no       | Distributed correlation identifier.                                                 |
| `causation_id` | string          | no       | Parent causal message or event identifier.                                          |
| `ts`           | integer         | yes      | Unix epoch seconds.                                                                 |
| `expires_at`   | integer or null | no       | Sender-declared expiry boundary.                                                    |
| `body`         | object          | yes      | Kind-specific payload.                                                              |
| `proof`        | object or null  | no       | Opaque proof placeholder; current AGH Runtime preserves but does not verify it.     |
| `ext`          | object          | no       | Extension map.                                                                      |

### 4.2 Validation Order

A receiver MUST validate v2 envelopes in this order:

1. Reject obsolete hard-cut fields.
2. Decode one JSON object.
3. Validate `protocol`, `id`, `workspace_id`, `kind`, `channel`, `from`, `to`, and `ts`.
4. Validate surface/container symmetry.
5. Validate kind-specific `work_id` requirements.
6. Validate `body` shape for the selected `kind`.
7. Evaluate freshness from `expires_at` or the receiver replay window.
8. Route by `workspace_id`, `channel`, `surface`, container ID, and `to`.

## 5. Conversation Rules

Discovery messages use the channel as their audience and MUST NOT set conversation fields:

| Kind    | Conversation Fields                                | Work Field           |
| ------- | -------------------------------------------------- | -------------------- |
| `greet` | MUST omit `surface`, `thread_id`, and `direct_id`. | MUST omit `work_id`. |
| `whois` | MUST omit `surface`, `thread_id`, and `direct_id`. | MUST omit `work_id`. |

Conversation-bearing messages MUST identify exactly one container:

| Kind         | Conversation Fields                                             | Work Field                                 |
| ------------ | --------------------------------------------------------------- | ------------------------------------------ |
| `say`        | MUST set `surface` and the matching `thread_id` or `direct_id`. | MAY set `work_id` only for lifecycle work. |
| `capability` | MUST set `surface` and the matching container ID.               | MUST set `work_id`.                        |
| `receipt`    | MUST set `surface` and the matching container ID.               | MUST set `work_id`.                        |
| `trace`      | MUST set `surface` and the matching container ID.               | MUST set `work_id`.                        |

## 6. NATS Binding

### 6.1 Subject Prefix

The v2 subject prefix is:

`agh.network.v2`

### 6.2 Subject Grammar

Subjects use this hierarchy:

```text
agh.network.v2.<workspace_id>.<channel>.broadcast
agh.network.v2.<workspace_id>.<channel>.peer.<route_token>
```

| Segment          | Rule                                                                               |
| ---------------- | ---------------------------------------------------------------------------------- |
| `agh.network.v2` | Fixed v2 binding prefix.                                                           |
| `<workspace_id>` | Stable workspace ID. Dots, spaces, `*`, `>`, and control characters are forbidden. |
| `<channel>`      | MUST match the channel grammar.                                                    |
| `broadcast`      | Receives channel-wide messages inside one workspace.                               |
| `peer`           | Peer-targeted transport namespace.                                                 |
| `<route_token>`  | Subject-safe route token derived from the target peer identity.                    |

### 6.3 Route Tokens

For the current AGH Runtime v2 path, the route token is the first 32 lowercase hexadecimal characters
of `SHA-256(peer_id UTF-8 bytes)`.

Implementers that add RFC 004 baseline verified mode MAY use the verified identity fingerprint as
the route token, but the subject still MUST include `workspace_id`.

### 6.4 Subject Mapping

| Envelope Condition | NATS Subject                                                 |
| ------------------ | ------------------------------------------------------------ |
| `to = null`        | `agh.network.v2.<workspace_id>.<channel>.broadcast`          |
| `to != null`       | `agh.network.v2.<workspace_id>.<channel>.peer.<route_token>` |

`surface:"direct"` is a conversation visibility rule, not a NATS subject kind. A targeted
`surface:"thread"` message and a `surface:"direct"` message both use peer-targeted transport
subjects when `to` is set.

### 6.5 Joining a Channel

A NATS peer joins one workspace channel by:

1. subscribing to `agh.network.v2.<workspace_id>.<channel>.broadcast`
2. subscribing to `agh.network.v2.<workspace_id>.<channel>.peer.<own_route_token>`
3. publishing `greet` to the broadcast subject
4. refreshing `greet` after reconnect

A peer that joins the same channel slug in two workspaces MUST maintain separate subscriptions and
presence state for each workspace.

## 7. Daemon API Boundary

The daemon control plane is not the protocol itself, but it MUST preserve the same identity boundary.

Current Network runtime APIs use workspace-qualified paths:

```text
GET  /api/workspaces/{workspace_id}/network/channels
POST /api/workspaces/{workspace_id}/network/channels
GET  /api/workspaces/{workspace_id}/network/channels/{channel}
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/threads
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/threads/{thread_id}
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/threads/{thread_id}/messages
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/directs
POST /api/workspaces/{workspace_id}/network/channels/{channel}/directs/resolve
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/directs/{direct_id}
GET  /api/workspaces/{workspace_id}/network/channels/{channel}/directs/{direct_id}/messages
GET  /api/workspaces/{workspace_id}/network/peers
GET  /api/workspaces/{workspace_id}/network/peers/{peer_id}
GET  /api/workspaces/{workspace_id}/network/work/{work_id}
GET  /api/workspaces/{workspace_id}/network/inbox
POST /api/workspaces/{workspace_id}/network/send
```

`GET /api/network/status` remains a global daemon health/status exception. It MUST NOT expose
workspace-scoped channel timelines, message bodies, direct-room contents, or peer inbox data.

Handlers with a `workspace_id` in both path and body MUST reject mismatches rather than silently
shadowing one value with the other.

## 8. Storage and Read Models

Implementations MUST key stored Network state by workspace-qualified identity.

Required identity keys:

| Data                               | Required Identity                                      |
| ---------------------------------- | ------------------------------------------------------ | ----------------------- |
| Channel metadata                   | `(workspace_id, channel)`                              |
| Public threads                     | `(workspace_id, channel, thread_id)`                   |
| Direct rooms                       | `(workspace_id, channel, direct_id)`                   |
| Direct-room participants           | `(workspace_id, channel, direct_id, peer_id)`          |
| Messages                           | `(workspace_id, channel, surface, thread_id            | direct_id, message_id)` |
| Work items                         | `(workspace_id, work_id)` plus container reference     |
| Pins, recents, and last-read state | `workspace_id` plus the relevant channel/container key |

Read queries over Network data MUST include a validated `workspace_id` predicate or a previously
validated workspace-qualified key.

## 9. Examples

### 9.1 Presence Announcement

```json
{
  "protocol": "agh-network/v2",
  "id": "msg_01krh4network00000000000001",
  "workspace_id": "ws_alpha",
  "kind": "greet",
  "channel": "builders",
  "from": "coordinator.sess-alpha",
  "ts": 1778695200,
  "body": {
    "peer_card": {
      "peer_id": "coordinator.sess-alpha",
      "display_name": "Coordinator",
      "profiles_supported": ["agh-network/v2"],
      "capabilities": ["plan review", "handoff"],
      "artifacts_supported": ["capability"],
      "trust_modes_supported": []
    },
    "summary": "Available for workspace-local coordination"
  },
  "proof": null
}
```

### 9.2 Public Thread Message

```json
{
  "protocol": "agh-network/v2",
  "id": "msg_01krh4network00000000000002",
  "workspace_id": "ws_alpha",
  "kind": "say",
  "channel": "builders",
  "surface": "thread",
  "thread_id": "thread_workspace_cut",
  "from": "coordinator.sess-alpha",
  "ts": 1778695260,
  "body": {
    "text": "Please review the workspace-qualified Network cut.",
    "intent": "request_review"
  },
  "proof": null
}
```

### 9.3 Directed Work Progress

```json
{
  "protocol": "agh-network/v2",
  "id": "msg_01krh4network00000000000003",
  "workspace_id": "ws_alpha",
  "kind": "trace",
  "channel": "builders",
  "surface": "direct",
  "direct_id": "direct_b41c2e6a31f3d9849f75d96cb46c1d5a",
  "from": "reviewer.sess-beta",
  "to": "coordinator.sess-alpha",
  "work_id": "work_workspace_cut_review",
  "trace_id": "trace_workspace_cut_review",
  "ts": 1778695320,
  "body": {
    "state": "working",
    "summary": "Checking route and subject isolation.",
    "progress": {
      "current": 1,
      "total": 3
    }
  },
  "proof": null
}
```

## 10. Security Considerations

Workspace qualification is an isolation boundary, not an authorization system by itself. A receiver
must still apply local policy for:

- which peers may join a workspace channel
- whether a peer may see a direct room
- whether a peer may execute a capability or mutate runtime state
- whether an opaque `proof` is ignored, preserved, or verified by an implementation-specific trust layer

v2 prevents accidental cross-workspace merging by making workspace identity mandatory at every
protocol and transport boundary. It does not encrypt public threads or direct rooms.

## 11. Current Runtime Status

The current AGH Runtime implements:

- v2 envelope parsing and validation
- required `workspace_id`
- workspace-qualified NATS subjects
- workspace-scoped Network API routes
- workspace-qualified storage and read models
- opaque `proof` preservation

The current AGH Runtime does not implement:

- Ed25519 + JCS signature verification
- external NATS broker configuration as a stable public surface
- JetStream persistence as a protocol profile
- cross-organization federation

## 12. Migration Notes

v2 does not define a wire-level migration from channel-only subjects. Runtime data stores SHOULD be
rebuilt or migrated in one step so channel metadata, conversations, work items, audit rows, pins,
recents, and last-read state all carry `workspace_id`.

Implementers updating from earlier drafts should:

1. add required `workspace_id` to every envelope
2. change `protocol` to `agh-network/v2`
3. change direct-room derivation to include `workspace_id`
4. change subjects to the v2 grammar
5. remove channel-only API reads and writes
6. ensure duplicate channel slugs in different workspaces do not share rows, cache keys, or subscriptions
