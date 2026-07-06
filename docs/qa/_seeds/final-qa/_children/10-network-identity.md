---
name: 10-network-identity
title: AGH Network + Identity + Peers + Channels — Real-LLM QA Plan
description: Behavior-first QA scenarios for the AGH Network v0/v1 protocol surface (`internal/network`) plus the daemon-validated caller identity (`internal/agentidentity`). Covers two-instance peer-to-peer roundtrips, proof-stripping defense, claim_token redaction at network ingress, channel join/leave/discovery, capability artifact transfer, control-message dispatch, cross-version negotiation, embedded NATS isolation, identity rotation, partition recovery, DoS resistance, and real Claude Code agent-to-agent calls. Real-LLM provider lanes against Claude Code; never mocks at the network layer.
type: final-qa-child
module: network-identity
parent: ../_parent.md
provider_lanes: [claude-code]
authoritative_runtime_truth: internal/CLAUDE.md
references:
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/final-qa/_references/openclaw-qa-patterns.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/final-qa/_references/hermes-qa-patterns.md
  - /Users/pedronauck/Dev/compozy/agh/CLAUDE.md
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md
  - qmd://agh-rfcs-local/003-agh-network-v0.md
  - qmd://agh-rfcs-local/004-agh-network-v1.md
---

# 10 — AGH Network + Identity + Peers + Channels

## 1. Module scope

The AGH Network module is the only daemon surface that owns inter-AGH-instance
communication. It implements the AGH Network v0 envelope contract (RFC 003)
plus the v1 Baseline Trust Profile (RFC 004) on top of an embedded NATS
server, exposes channel join/leave + peer discovery + outbound `Send` over
HTTP/UDS/CLI, and is **the sole package permitted to import NATS** per the
backend architecture rule on inter-package coordination
(`internal/CLAUDE.md` "Direct function calls through interfaces — no event
bus, no reflection-based routing, no NATS as inter-package coordination.
NATS is permitted **only** inside `internal/network`…").

The agent identity package validates daemon-bound caller identity for any
agent-facing CLI/UDS operation that crosses the daemon boundary. It is in
scope here because identity proof-stripping is the v1 trust invariant and
because `AGH_SESSION_ID`/`AGH_AGENT` is what authorizes a network-peer
session to enqueue task runs from another peer
(`Manager.EnqueueRunFromPeer`, `internal/network/tasks.go:222`).

Packages in scope (file:line citations are repo-absolute):

| Surface                         | Path                                                                          | Authoritative API                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ------------------------------- | ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Manager (composition root)      | `/Users/pedronauck/Dev/compozy/agh/internal/network/manager.go`               | `NewManager` (`:127-147`), `JoinChannel` (`:339-364`), `LeaveChannel` (`:512-558`), `Send` (`:569-583`), `Inbox`/`WaitInbox` (`:724-749`), `Status` (`:673-721`), `Shutdown` (`:753-826`), `handleInboundMessage` (`:828-887`), `handleDisconnect`/`handleReconnect` (`:1065-1103`), audit-only-on-durable-write recorders (`:1111-1204`), `transportListener` (`:1206-1231`).                                                                                                                                                                                                                                                                                                              |
| Wire envelope + validation      | `/Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go`              | `Envelope` (`:169-185`), `ProtocolV0` (`:11`), seven-kind enumeration (`KindGreet`/`KindWhois`/`KindSay`/`KindDirect`/`KindCapability`/`KindReceipt`/`KindTrace` `:16-24`), `ReceiptStatus` (`:45-71`), `ReasonCode` (`:94-131`), `InteractionState` (`:134-160`), `Proof` map (`:162-163`), `ExtensionMap` (`:165-166`), `PeerCard` (`:227-235`).                                                                                                                                                                                                                                                                                                                                          |
| Validate / freshness            | `/Users/pedronauck/Dev/compozy/agh/internal/network/validate.go`              | `ParseEnvelope`/`NormalizeEnvelope` (`:55-89`), `ValidateChannel` (`:91-97`) gated by `internal/network/rules/channel.go:5` `^[a-z0-9][a-z0-9_-]{0,63}$`, `ValidatePeerID` (`:99-104`) `^[a-z0-9][a-z0-9._-]{0,127}$`, `RouteToken` deterministic SHA-256 (`:107-116`), kind-specific body decoders (`:226-282`), `validateKindEnvelopeRules` "greet peer_card.peer_id must match from" (`:284-325`), capability-digest verification (`:419-471`), `validateEnvelopeFreshness` `expires_at`/`max_replay_age` (`:327-343`), `DefaultMaxReplayAge` 5 minutes (`:47`).                                                                                                                          |
| Router (lifecycle + presence)   | `/Users/pedronauck/Dev/compozy/agh/internal/network/router.go`                | `NewRouter` (`:131-163`), `Send` presence preflight (`:251-290`), `Receive` (`:293-315`), `prepareReceiveState` (`:349-397`) — `not_target`, `duplicate` admission, `dispatchReceivedEnvelope` (`:399-425`), `handleWhois` (`:559-635`), `handleReceivedCapability` (`:438-454`), `applyReceiveLifecycle` (`:469-517`) — `interaction_closed`, `markSeen` deduplication window (`:782-796`), `replayDeadline` (`:933-945`).                                                                                                                                                                                                                                                                |
| Peer registry                   | `/Users/pedronauck/Dev/compozy/agh/internal/network/peer.go`                  | `PeerRegistry` (`:55-62`), `RegisterLocalWithCapabilityCatalog` (`:136-194`), `LeaveLocal` (`:197-216`), `RefreshRemoteWithCapabilityCatalog` (`:299-356`) with `2 * greetInterval` expiry (`:351`), `expireRemotesLocked` (`:519-530`), `LookupPresence` (`:382-405`), `ListPeers` (`:414-449`), `ListChannels` (`:452-482`), `MatchLocalPeers` (`:276-289`), `DefaultPeerCard` (`:108-121`).                                                                                                                                                                                                                                                                                              |
| Embedded transport              | `/Users/pedronauck/Dev/compozy/agh/internal/network/transport.go`             | `NewTransport` (`:94-161`) starts an embedded `nats-server/v2` with token auth + `nats.InProcessServer`, `Publish` (`:234-261`) honors `defaultTransportPublishTimeout=5s` if no deadline, `Subscribe` (`:264-281`), `Drain`/`Shutdown` (`:284-353`), `BroadcastSubject` `agh.network.v0.<channel>.broadcast` (`:356-362`), `DirectSubject` `agh.network.v0.<channel>.peer.<route-token>` (`:364-375`), `subjectPrefix = "agh.network.v0"` (`:23`).                                                                                                                                                                                                                                         |
| Audit                           | `/Users/pedronauck/Dev/compozy/agh/internal/network/audit.go`                 | `FileAuditWriter` (`:66-138`), `RecordSent`/`RecordReceived`/`RecordRejected`/`RecordDelivered` (`:140-163`), `TaskIngressAuditWriter` (`:60-62, :167-203`), `NormalizeAuditEntry` (`:288-326`), greet-presence dedupe window `2 * GreetIntervalDuration` (`manager.go:281-288`).                                                                                                                                                                                                                                                                                                                                                                                                          |
| Delivery coordinator            | `/Users/pedronauck/Dev/compozy/agh/internal/network/delivery.go`              | `MaxQueueDepth` overflow → `queue_overflow` rejection (audited via `recordDropped` `manager.go:1202-1204`); `accept`/`drop`/`waitInbox`/`onTurnEnd` per-session worker semantics. Tests: `internal/network/manager_test.go:631-636` proves overflow audited as rejected with reason `queue_overflow`.                                                                                                                                                                                                                                                                                                                                                                                     |
| Task ingress (network → tasks)  | `/Users/pedronauck/Dev/compozy/agh/internal/network/tasks.go`                 | `TaskIngressContext` (`:60-70`), `EnqueueRunFromPeer` (`:222`), `CreateTaskFromPeer` (`:103`), `UpdateTaskFromPeer` (`:137`), `CancelTaskFromPeer` (`:186`), `ErrTaskIngressUnavailable`/`ErrTaskIngressPeerNotFound`/`ErrTaskIngressCapabilityDenied` (`:24-32`).                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Capability brief & catalog      | `/Users/pedronauck/Dev/compozy/agh/internal/network/capability_brief.go`, `capability_catalog.go` | Whois capability discovery payload + canonical `agh-network/v0` capability digest matching (verify drift against `aghconfig.CanonicalCapabilityDigest` invoked in `validate.go:446-468`).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Identity validation             | `/Users/pedronauck/Dev/compozy/agh/internal/agentidentity/identity.go`        | `Resolve` (`:139-162`), env vars `AGH_SESSION_ID`/`AGH_AGENT` (`:18-23`), UDS headers `X-AGH-Session-ID`/`X-AGH-Agent`/`X-AGH-Workspace-ID` (`:24-30`), error codes `identity_required`/`identity_stale`/`identity_mismatch`/`identity_unauthorized`/`identity_lookup_unavailable` (`:48-58`), exit codes `64`/`65`/`69`/`77` (`:34-45`), `ErrorPayload`/`MarshalErrorJSON`/`MarshalErrorJSONL` (`:131-347`).                                                                                                                                                                                                                                                                              |
| HTTP/UDS API                    | `/Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/network_test.go`, `internal/api/spec/spec.go:1178-1224` | Routes `GET /api/network/status`, `GET /api/network/peers`, `GET /api/network/peers/:peer_id`, `GET /api/network/peers/:peer_id/messages`, `GET /api/network/channels`, `GET /api/network/channels/:channel`, `GET /api/network/channels/:channel/messages`, `POST /api/network/channels`, `POST /api/network/send`, `GET /api/network/inbox`. Registered in both HTTP and UDS handler registries (`internal/api/udsapi/handlers_test.go:150-255`).                                                                                                                                                                                                                                       |
| CLI                             | `/Users/pedronauck/Dev/compozy/agh/internal/cli/client.go:912-973`            | `agh network status`, `agh network peers`, `agh network channels`, `agh network send`, `agh network inbox`. CLI shape verified by `internal/cli/network_client_test.go:23-52`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Network config                  | `/Users/pedronauck/Dev/compozy/agh/internal/config/config.go:204-213`         | `NetworkConfig{Enabled, DefaultChannel, Port, MaxPayload, GreetInterval, MaxReplayAge, MaxQueueDepth}` plus toml validation in `config_test.go:2283-2313` enforcing positive `max_payload`/`max_queue_depth`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |

Out of scope (other children): autonomy kernel claim/lease semantics
(module 04), session manager state machine (module 03 — only the
`AGH_SESSION_ID` injection seam is in scope here), automation cron/webhook
trigger production (module 09), web UI rendering of network surfaces
(module 08).

## 2. Authoritative invariants under test

These come straight from `internal/CLAUDE.md`, RFC 003, RFC 004, and the
implementation. Every scenario maps back to one or more of these IDs.
Coverage IDs follow the openclaw lowercase dotted/dashed convention.

| Coverage ID                                | Invariant                                                                                                                                                                                                                                                                | Source                                                                                                                  |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------- |
| `network.protocol-implementable`           | The AGH Network protocol must remain implementable outside AGH. No internal AGH type may leak into the wire envelope (envelope schema, kinds, lifecycle states are RFC-defined; nothing else enters the body).                                                            | Repo-wide moat statement in root `CLAUDE.md` Vocabulary & Product Strategy section + RFC 003 Section 4.4 product boundary. |
| `network.nats-isolation`                   | Only `internal/network` may import `github.com/nats-io/*`. No other package publishes/subscribes to NATS subjects.                                                                                                                                                       | `internal/CLAUDE.md` "NATS is permitted only inside `internal/network`…".                                                |
| `network.subject-prefix.v0`                | Embedded transport publishes only on `agh.network.v0.<channel>.broadcast` and `agh.network.v0.<channel>.peer.<route-token>`.                                                                                                                                              | `internal/network/transport.go:23,356-375`.                                                                              |
| `network.channel-grammar`                  | Channel matches `^[a-z0-9][a-z0-9_-]{0,63}$`. Dots, whitespace, NATS wildcards (`>`, `*`) are forbidden.                                                                                                                                                                  | `internal/network/rules/channel.go:5`; RFC 003 Section 3.2.                                                              |
| `network.peer-id-grammar`                  | Peer ID matches `^[a-z0-9][a-z0-9._-]{0,127}$`. Route token is `sha256(peer_id)[:32 hex]`.                                                                                                                                                                                  | `internal/network/validate.go:42, 99-116`; RFC 003 Section 6.1.                                                          |
| `network.envelope.required-fields`         | `protocol`, `id`, `kind`, `channel`, `from`, `ts`, `body` are required. `to` required for `direct`/targeted `whois`/targeted `receipt`/targeted `trace`. `interaction_id` required for `direct`/`receipt`/`trace`.                                                          | RFC 003 Section 5.1.2; `internal/network/validate.go:153-205, 284-325`.                                                  |
| `network.envelope.protocol-pin`            | `protocol` MUST be exactly `agh-network/v0`; any other value is `ErrInvalidField`.                                                                                                                                                                                       | `internal/network/validate.go:154-159`.                                                                                  |
| `network.body.greet-from-binding`          | `greet.peer_card.peer_id` MUST equal envelope `from`. Mismatch is `ErrInvalidBody`.                                                                                                                                                                                      | `internal/network/validate.go:291-294`.                                                                                  |
| `network.body.capability-digest`           | `capability.digest` MUST equal `aghconfig.CanonicalCapabilityDigest(def)`. Drift is `ErrVerificationFailed`.                                                                                                                                                              | `internal/network/validate.go:446-468`.                                                                                  |
| `network.replay-window`                    | An envelope with no `expires_at` is rejected if `now - ts > MaxReplayAge` (default 5 min). With `expires_at`, the deadline takes priority. Duplicate `id` within the window is dropped (`markSeen`).                                                                       | `internal/network/validate.go:47, 327-343`; `internal/network/router.go:782-796, 933-945`.                               |
| `identity.proof-stripping-defense`         | An identity in verified format (`nickname@fingerprint`) WITHOUT valid `proof` MUST classify as `rejected`, NOT `unverified`.                                                                                                                                              | `internal/CLAUDE.md` Security Invariants; RFC 004 Section 3.3.                                                           |
| `identity.proof-invalid-rejected`          | A baseline-profile proof present but invalid (signature mismatch, key-id mismatch, fingerprint-from mismatch) MUST classify as `rejected` with a stable typed error.                                                                                                       | RFC 004 Section 4.7-4.8.                                                                                                 |
| `identity.unverified-classification`       | An identity NOT in verified format and with no `proof` (or with an unsupported profile) is `unverified` — accepted but not trusted.                                                                                                                                       | RFC 004 Section 3.2.                                                                                                     |
| `network.no-claim-token-in-metadata`       | The network layer MUST reject envelopes whose `ext` carries a raw `agh_claim_*` token. Only `claim_token_hash` is permitted on the wire.                                                                                                                                  | `internal/CLAUDE.md` Security Invariants ("Network layer rejects raw `claim_token` in metadata.").                       |
| `peer-card.distinct-from-agent-card`       | A Peer Card describes a network identity (peer_id + profiles + capabilities + artifacts + trust modes). It is NOT an Agent Card (which describes a specific agent's affordances). The two MUST not be conflated in storage or wire payloads.                              | RFC 003 Section 6.2; `internal/network/peer.go:34-46`.                                                                   |
| `channel.join-discovery`                   | After `JoinChannel`, the local peer is visible in `ListPeers(channel)`; remote peers refreshed via greet are visible until `2 * greet_interval` since `LastSeen`.                                                                                                          | `internal/network/manager.go:339-364, 645-655`; `internal/network/peer.go:299-356`.                                       |
| `channel.private-not-listed`               | Private channels (channels a daemon has not joined and is not invited into) MUST NOT appear in `ListChannels` output to non-members.                                                                                                                                     | RFC 003 Sections 6.3, 8 + `internal/network/peer.go:452-482` (only counts channels the registry knows about).            |
| `delivery.ordering-preserved`              | For two messages M1 then M2 published from the same peer to the same direct subject, the receiver observes M1 before M2.                                                                                                                                                  | NATS subject ordering guarantee + `internal/network/router.go:898-915` flush-after-publish.                              |
| `delivery.queue-overflow-audited`          | When `MaxQueueDepth` is reached, the network audits a `rejected` entry with reason `queue_overflow`.                                                                                                                                                                     | `internal/network/manager.go:1202-1204`; `internal/network/manager_test.go:631-636`.                                     |
| `transport.publish-timeout`                | Outbound publishes use a context-attached timeout (`defaultTransportPublishTimeout = 5s` if the caller has no deadline). `http.DefaultClient` is forbidden in production paths anywhere in `internal/`.                                                                    | `internal/network/transport.go:21-23, 234-261`; `internal/CLAUDE.md` "External-call timeouts".                           |
| `transport.embedded-isolation`             | Each daemon's embedded NATS server binds `127.0.0.1:<port>` with a per-process token; restart releases the port; two daemons on the same host with distinct ports do not collide.                                                                                         | `internal/network/transport.go:94-160, 186-213`; `internal/network/manager.go:311-318`.                                  |
| `transport.partition-recovery`             | A disconnect emits `network.disconnected`; reconnect emits `network.reconnected` and re-publishes greets for every active session via `handleReconnect`.                                                                                                                  | `internal/network/manager.go:1065-1103`.                                                                                  |
| `identity.rotation`                        | When a session re-joins with a new `peer_id` (key rotation in v1), the previous `peer_id` is removed from the local index (`localsByChannel`); messages already in flight on the old direct subject deliver until heartbeat expiry; new messages route to the new fingerprint subject. | `internal/network/peer.go:179-194` (peer-id collision check + replace), `:497-506` (`removeLocalIndexesLocked`).        |
| `dos.bogus-peer-card-ratelimit`            | A flood of malformed `whois` requests does not exhaust daemon resources: rejected envelopes are audited but no work is enqueued, and the daemon stays responsive to peers + HTTP/UDS during the flood.                                                                     | `internal/network/router.go:317-347` (parse-error path), `internal/network/audit.go:150-163`, `internal/network/manager.go:828-848` (warn-and-drop).      |
| `cross-version.negotiation`                | Two peers with different protocol identifiers (`agh-network/v0` vs `agh-network/v1`) cannot silently corrupt: a v0 receiver of a v1 envelope rejects it as `ErrInvalidField` because `validate.go:157-159` pins `protocol == ProtocolV0` exactly.                          | `internal/network/validate.go:154-159`; RFC 003 Section 1.3 + RFC 004 Section 6.2 (v1 uses `agh.network.v1` subject prefix). |
| `event.lineage-correlation.network`        | Every audit row carries `kind`, `channel`, `peer_from`, `peer_to`, `message_id`, plus `interaction_id`/`reply_to`/`trace_id`/`causation_id` when present.                                                                                                                  | `internal/network/audit.go:288-326`; `internal/network/manager.go:1233-1265` (log fields include `agh.workflow_id`/`handoff_*`). |
| `agentidentity.daemon-authoritative`       | `Resolve` requires `AGH_SESSION_ID` + `AGH_AGENT` and confirms via daemon-authoritative `SessionLookup`. Spoofed env values whose session is not Active are `ErrIdentityStale`. Mismatched agent name is `ErrIdentityMismatch`. Workspace mismatch is `ErrIdentityUnauthorized`. | `internal/agentidentity/identity.go:139-262`.                                                                            |
| `agentidentity.exit-code-determinism`      | Identity errors map to deterministic process exit codes 64/65/69/77; no error path emits exit 0.                                                                                                                                                                         | `internal/agentidentity/identity.go:34-45, 349-365`.                                                                     |

## 3. Operating model

QA mode is **real-scenario** (per the standing directive on real-scenario
QA). Every scenario:

- Runs against **two** isolated AGH_HOME directories (Instance A + Instance B)
  whenever a peer-to-peer roundtrip is required — each with its own daemon,
  unique daemon HTTP/UDS port pair, and unique embedded NATS port. Both
  AGH_HOMEs use the worktree-isolation helper (`agh-worktree-isolation`
  skill) so SQLite, ports, and tmux-bridge sockets never collide.
- Resolves provider auth from the bootstrap manifest according to each
  provider contract: bound-secret, brokered, and explicitly isolated-home
  lanes use `PROVIDER_HOME` / `PROVIDER_CODEX_HOME`, while `native_cli`
  lanes with `home_policy=operator` preserve the operator `HOME` unless the
  scenario explicitly validates isolated provider-home behavior. Live runs
  that exercise real Claude Code peers are gated `live: conditional` — they
  require the credential broker / pooled Claude Code login described in
  `openclaw-qa-patterns.md` §4.
- Uses real Claude Code (`claude-opus-4-7[1m]` for "agent A" and
  `claude-sonnet-4-6` for "agent B" by default) as subprocess agents. The
  network protocol itself is exercised with daemon-driven CLI commands; the
  agent driver is plugged in only when the scenario needs the agent to send
  / receive on its own.
- Emits four artifacts under `.artifacts/qa/<run-id>/net-XX/`:
  - `net-XX-report.md` (Worked / Failed / Blocked / Follow-up)
  - `net-XX-summary.json` (machine-readable; per-instance and aggregate)
  - `net-XX-events.json` (network audit rows + EventStore rows scoped to the
    scenario window, both instances)
  - `net-XX-output.log` (combined daemon stderr + tcpdump-of-loopback when
    flagged)
- Asserts against:
  - `network.audit` rows (`store.NetworkAuditEntry` produced by
    `audit.go:288-326`),
  - `network.message` timeline rows (`store.NetworkMessageEntry`),
  - HTTP/UDS responses from `/api/network/*`,
  - direct loopback NATS subject inspection (Instance A's audit log proves
    Instance B's send round-tripped),
  - structured log events (`network.peer.joined`, `network.peer.left`,
    `network.message.sent`, `network.message.received`,
    `network.message.rejected`, `network.disconnected`,
    `network.reconnected`).

Scenarios are numbered `NET-01..NET-NN`; each is a fenced `qa-scenario`
block plus a flow narrative. Reproduce by running them sequentially; many
can be parallelized under unique worktree isolation.

## 4. Provider matrix

| Mode               | When                                                                                                | Driver                                                                                                                                          |
| ------------------ | --------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `real-claude-code` | Default. Required for NET-12 (real agent-to-agent roundtrip) and any scenario that drives an agent. | `claude-opus-4-7[1m]` for the originating agent; `claude-sonnet-4-6` for the responder.                                                         |
| `daemon-driver`    | Default for scenarios that exercise only the network protocol surface (no agent prompt needed).     | `agh network send` / `agh network inbox` / `agh network channels` / `agh network peers` driven from the QA harness.                              |
| `mock-acp` (gate)  | Determinism gate for race-sensitive scenarios where real-LLM nondeterminism would obscure the test. | `internal/e2elane` mock ACP server. Used only as a deterministic agent placeholder; the network code paths remain real.                          |

We do NOT include an `aimock` lane; per the openclaw provider-mode
honest framing, AIMock is additive and not a replacement for the
deterministic mock. We do NOT exercise OpenClaw or Hermes at the network
layer here — driver-agnosticism is covered in module 03 (ACP).

## 5. Preconditions (apply to every scenario)

- Fresh QA bootstrap via the `agh-qa-bootstrap` skill, **twice** (one per
  instance). Manifest paths saved to
  `bootstrap-manifest-A.json` and `bootstrap-manifest-B.json`;
  `bootstrap-A.env` / `bootstrap-B.env` exported into the harness shell
  before any `agh` command.
- Two unique `AGH_HOME` directories per worktree (per the worktree-isolation
  directive).
- Bound-secret, brokered, and explicitly isolated-home auth staged into
  per-instance `PROVIDER_HOME` / `PROVIDER_CODEX_HOME`; `native_cli`
  providers with `home_policy=operator` intentionally use the operator
  `HOME` / native login state unless the scenario explicitly validates
  isolated provider-home behavior.
- Both daemons started in background. HTTP / UDS listeners reachable on
  their unique ports.
- Each daemon's `network.enabled = true`, `network.port = <unique>`,
  `network.greet_interval = 2` (so QA does not wait full 30s default), and
  `network.max_replay_age = 60` for tighter replay-window assertions.
- `make verify` is green on the SUT branch before QA runs (per the
  Critical Rules).

Per-instance config example (Instance A; Instance B mirrors with B-prefixed
paths and a distinct port):

```text
AGH_HOME=$HOME/.qa/net-10/$SCENARIO/agh-home-A
AGH_DAEMON_HTTP=127.0.0.1:31230
AGH_DAEMON_UDS=$AGH_HOME/sock/uds.sock
PROVIDER_HOME=$AGH_HOME/provider-home
PROVIDER_CODEX_HOME=$AGH_HOME/provider-codex-home
NETWORK_PORT=14222
GREET_INTERVAL=2
DEFAULT_CHANNEL=qa-builders
```

## 6. Cleanup (applies to every scenario)

- `agh daemon stop` for both instances (or kill PID from manifest).
- Inspect `task_runs` for any stuck `claimed`/`running` rows arising from
  network-driven enqueues; if found, attach to the scenario report and DO
  NOT clean — it's evidence.
- Archive `events.db`, `agh.db`, and the network audit JSONL
  (`$AGH_HOME/network/audit.jsonl`) for both instances before tearing down.
- Tear down both AGH_HOMEs only after evidence artifacts are written.

## 7. Mandatory scenarios

### NET-01 — Two-instance peer-to-peer direct message roundtrip

```yaml qa-scenario
id: net-01-two-instance-roundtrip
title: Two AGH instances on the same host with isolated AGH_HOME and distinct NATS ports send a direct peer-to-peer message; round-trip succeeds; both sides classify the identity as unverified (v0 default)
theme: network.peer-to-peer
coverage:
  primary:
    - channel.join-discovery
    - network.subject-prefix.v0
    - network.envelope.required-fields
    - identity.unverified-classification
  secondary:
    - transport.embedded-isolation
    - delivery.ordering-preserved
    - event.lineage-correlation.network
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Instance A daemon running on HTTP :31230, NATS :14222.
  - Instance B daemon running on HTTP :31231, NATS :14223.
  - Same channel `qa-builders` declared in `network.default_channel`.
  - Both daemons configured with the SAME embedded NATS cluster route OR
    each instance subscribed to the other's NATS port via the configured
    transport routing (per RFC 003 Section 10 NATS profile). Document the
    chosen topology in the scenario report.
docs_refs:
  - qmd://agh-rfcs-local/003-agh-network-v0.md (Sections 6, 10)
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:339-364
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:251-290
  - /Users/pedronauck/Dev/compozy/agh/internal/network/transport.go:23,356-375
steps:
  - On Instance A, `agh network channels` → A sees `qa-builders` after a
    session joins; record `peer_id_A`.
  - On Instance B, same — record `peer_id_B`.
  - From A: `agh network peers --channel qa-builders` lists both A and B
    after one greet interval (~2s for this scenario).
  - From A: `agh network send --channel qa-builders --to <peer_id_B>
    --kind direct --interaction-id qa-net01-int-1 --body
    '{"text":"hello B from A","intent":"qa.greet"}' -o json`. Capture the
    returned `message_id`.
  - On B: `agh network inbox --session-id <B-session> -o json`.
expected_behavior:
  - A's audit log writes one `direction=sent kind=direct` row with
    `channel=qa-builders`, `peer_from=<peer_id_A>`,
    `peer_to=<peer_id_B>`, `interaction_id=qa-net01-int-1`,
    `message_id=<id>`.
  - B's audit log writes one `direction=received kind=direct` row with the
    same `interaction_id` and `message_id`.
  - B's inbox returns one envelope with `protocol="agh-network/v0"`,
    `to=peer_id_B`, body `{"text":"hello B from A","intent":"qa.greet"}`.
  - The envelope traveled on subject `agh.network.v0.qa-builders.peer.<route_token(peer_id_B)>`
    (record `route_token_B = sha256(peer_id_B)[:32]` and grep audit).
  - Trust state classification on B's side: `unverified` (no `proof`,
    `from` not in `nickname@fingerprint` format).
evidence:
  - `net-01-events-A.json` (audit rows scoped to scenario window).
  - `net-01-events-B.json`.
  - `net-01-output.log` containing one
    `network.message.sent` log on A and one `network.message.received` on
    B with matching `message_id`.
failure_signatures:
  - B's inbox empty: subject-routing broken or replay window dropped the
    envelope.
  - Audit row missing on either side: durable-write-then-broadcast invariant
    violated.
  - Envelope `protocol` field not `agh-network/v0`: `network.envelope.protocol-pin`
    violated.
cleanup:
  - `agh network leave --channel qa-builders` on both sides; verify peer
    counts drop to zero in `agh network status -o json`.
```

### NET-02 — Proof-stripping defense (verified-format identity without proof = REJECTED)

```yaml qa-scenario
id: net-02-proof-stripping-rejected
title: A peer whose `from` matches the verified format `nickname@fingerprint` but carries no `proof` (or null `proof`) is REJECTED, never classified as unverified
theme: network.identity-trust
coverage:
  primary:
    - identity.proof-stripping-defense
  secondary:
    - network.envelope.required-fields
    - event.lineage-correlation.network
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Instance A and Instance B running. Channel `qa-builders` joined on both.
  - Test harness can craft a raw envelope (using a fixture envelope JSON)
    and inject it onto Instance A's NATS subject — bypassing AGH's CLI to
    emulate a malicious external sender.
docs_refs:
  - qmd://agh-rfcs-local/004-agh-network-v1.md (Section 3.3)
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md (Security Invariants — Identity proof-stripping defense)
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:317-347
  - /Users/pedronauck/Dev/compozy/agh/internal/network/validate.go:153-185
steps:
  - Construct envelope JSON with `from = "alice@39f713d0a644253f04529421b9f51b9b"`,
    `proof = null`, `kind = "say"`, body `{"text":"proof-strip test"}`,
    valid `protocol`/`channel`/`ts`/`id`.
  - Publish directly to subject
    `agh.network.v0.qa-builders.broadcast` of Instance A's embedded NATS
    via the daemon's NATS Drain seam (or via the harness with the issued
    token).
  - Capture audit rows on A.
expected_behavior:
  - A's audit log writes one `direction=rejected` row with
    `peer_from=alice@39f713d0a644253f04529421b9f51b9b`, reason field
    indicating identity rejection (`reason_code=verification_failed` per
    `internal/network/router.go:1069-1080` mapping or
    `reason_code=malformed` if v0 currently returns malformed for this
    edge — record exact value).
  - No delivery row on B; no inbox entry; no `task.run` enqueue cascade.
  - Trust state recorded as `rejected`, NEVER `unverified`. If the daemon
    classifies as `unverified`, this scenario fails — that is the
    proof-stripping vulnerability the invariant exists to prevent.
evidence:
  - A's audit JSONL fragment showing the rejection.
  - Statement of recorded `trust_state` (must be `rejected`).
failure_signatures:
  - Trust state recorded as `unverified`: critical security violation;
    `identity.proof-stripping-defense` violated. Block the release.
  - Envelope delivered to B: complete failure of the trust invariant.
cleanup:
  - Tear down both instances.
```

### NET-03 — Invalid proof signature rejected with stable typed error

```yaml qa-scenario
id: net-03-invalid-proof-signature
title: A peer with a baseline-profile proof that has a tampered signature is REJECTED with `reason_code=verification_failed`
theme: network.identity-trust
coverage:
  primary:
    - identity.proof-invalid-rejected
  secondary:
    - event.lineage-correlation.network
risk: critical
live: conditional
provider: daemon-driver
preconditions:
  - QA harness key-pair generator that produces a valid Ed25519 keypair
    for `nickname@fingerprint` matching the v1 baseline trust profile.
  - When v1 trust profile is implemented in code, this scenario runs
    `live: true`. Until then, this scenario is `live: conditional` — runs
    only when the v1 verification path is exercised by the daemon code.
docs_refs:
  - qmd://agh-rfcs-local/004-agh-network-v1.md (Section 4.7)
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:1069-1080
  - /Users/pedronauck/Dev/compozy/agh/internal/network/validate.go (proof parsing — confirm in-scope when v1 lands)
steps:
  - Generate keypair K. Compute fingerprint = `sha256(pubkey)[0:32]`.
  - Build `from = "qa-bot@<fingerprint>"`. Sign canonical envelope (JCS,
    omitting `proof.sig`). Tamper the resulting signature by flipping the
    last byte.
  - Publish onto Instance A's broadcast subject.
expected_behavior:
  - A's audit row `direction=rejected` with `reason_code=verification_failed`.
  - The error returned to any HTTP/UDS surface that surfaces the rejection
    is a stable typed error wrapping `ErrVerificationFailed` (per
    `internal/network/validate.go:36-39`).
  - Trust state on the rejection record is `rejected`.
evidence:
  - Audit row + the typed error surface.
failure_signatures:
  - Audit row absent: rejection path silently swallowed the error.
  - Trust state `unverified`: invariant violated.
  - Error type not stable / not derivable from `ErrVerificationFailed`:
    upstream callers cannot branch.
cleanup:
  - Tear down.
```

### NET-04 — Valid proof but unknown public key (classification rule)

```yaml qa-scenario
id: net-04-unknown-public-key
title: A peer with a valid Ed25519 proof under an unknown public key is `unverified` (no local trust anchor) — confirm exact rule against the implementation, not the assumption
theme: network.identity-trust
coverage:
  primary:
    - identity.unverified-classification
  secondary:
    - identity.proof-stripping-defense
risk: high
live: conditional
provider: daemon-driver
preconditions:
  - Same v1 keypair generator as NET-03 but the resulting public key has
    NOT been advertised in any local Peer Card on Instance A or B.
docs_refs:
  - qmd://agh-rfcs-local/004-agh-network-v1.md (Section 3.2 trust states + Section 4.8 status interpretation)
notes:
  - The exact rule must be verified against the implementation. RFC 004
    Section 4.8 says "verified means all verification steps succeeded" —
    if all steps succeed but the daemon has no local roots / allowlist,
    the daemon's policy decides verified vs unverified vs rejected. This
    scenario asserts the implemented behavior matches the documented one
    and reports the exact rule observed.
steps:
  - Generate K_unknown. Sign envelope correctly. Inject onto A.
  - Capture A's audit `trust_state` field (or whichever field surfaces
    classification — note the implementation detail in the report).
expected_behavior:
  - The classification is exactly one of `verified` (all steps pass and
    local policy admits) OR `unverified` (steps fail at "Valid and
    allowed" gate per RFC 004 Section 3.1 mermaid). It is NEVER
    `rejected` solely for "unknown key" UNLESS local policy explicitly
    forbids unknown keys.
  - Whichever rule the implementation uses MUST be (a) deterministic, and
    (b) documented in the scenario report so future readers can audit it.
evidence:
  - Audit row showing the classification.
  - Linked source file:line that implements the rule.
failure_signatures:
  - Classification differs across two identical runs: nondeterminism.
  - Classification contradicts RFC 004 Section 3 without an ADR
    documenting the deviation: spec drift.
cleanup:
  - Tear down.
```

### NET-05 — Raw `claim_token` in `ext` rejected at network ingress

```yaml qa-scenario
id: net-05-claim-token-rejected
title: An envelope whose `ext` carries a raw `agh_claim_*` token is dropped at network ingress with an audit event; the sender is notified with a stable typed error
theme: network.security
coverage:
  primary:
    - network.no-claim-token-in-metadata
  secondary:
    - event.lineage-correlation.network
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Instance A and B running. Channel joined.
  - Harness can craft an envelope with `ext = {"agh.claim_token":"agh_claim_FAKE_QA_<rand>"}`
    or any other `ext` key carrying a raw `agh_claim_*` literal.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md (Security Invariants — "Network layer rejects raw `claim_token` in metadata.")
  - /Users/pedronauck/Dev/compozy/agh/internal/task/lease.go (RedactClaimTokens, the redaction regex)
steps:
  - Build envelope X with the polluting `ext`. Send via `agh network send`
    OR direct NATS injection for the malicious-sender variant.
  - Capture A's outbound audit + B's inbound audit.
  - Run the audit grep:
    `rg -n 'agh_claim_[A-Za-z0-9_-]{12,}' net-05-events-A.json net-05-events-B.json net-05-output.log`
expected_behavior:
  - The envelope is REJECTED at A's outbound boundary (preferred — caller
    sees a stable typed error mentioning forbidden ext key) OR at B's
    inbound boundary (acceptable fallback — audit row direction=rejected,
    reason mentions raw `claim_token`).
  - The grep returns zero matches across audit + log — the raw token
    NEVER appears in any sink even if the envelope was rejected.
  - The sender's error is wrapped through a stable typed error so callers
    can branch on `errors.Is`.
evidence:
  - Audit row showing the rejection with reason.
  - Empty grep output.
failure_signatures:
  - Envelope delivered to B with raw token intact: complete failure of
    the security invariant.
  - Raw token appears anywhere in audit/log: redaction violated.
  - Caller error is a generic 500 with no typed wrapping: caller can't
    branch.
cleanup:
  - None.
```

### NET-06 — Channel create + join, broadcast publish, ordering preserved

```yaml qa-scenario
id: net-06-channel-broadcast-ordering
title: Two peers create channel A; one publishes two messages M1 then M2; the other receives in order M1, M2; envelope-id deduplication holds
theme: network.channel-broadcast
coverage:
  primary:
    - channel.join-discovery
    - delivery.ordering-preserved
    - network.replay-window
  secondary:
    - network.envelope.required-fields
risk: high
live: false
provider: daemon-driver
preconditions:
  - Two daemons; channel `qa-net06` not yet joined by anyone.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:339-364, 941-978
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:413, 782-796
  - /Users/pedronauck/Dev/compozy/agh/internal/network/transport.go:356-375
steps:
  - On A: `agh network channels join qa-net06` (creates session-bound peer
    and subscribes to broadcast subject).
  - On B: same.
  - On A: `agh network send --channel qa-net06 --kind say --body
    '{"text":"M1"}' -o json` → `id_M1`.
  - Immediately: `agh network send --channel qa-net06 --kind say --body
    '{"text":"M2"}' -o json` → `id_M2`.
  - On B: `agh network inbox --channel qa-net06 -o jsonl` over a 5-second
    window.
  - Re-publish M1 via direct NATS injection to test duplicate dedup; B
    must NOT see M1 twice.
expected_behavior:
  - B receives M1 then M2 in that order (timestamp + sequence verified
    against B's audit log).
  - Re-injected M1 is dropped silently (no duplicate audit `received`,
    or audit shows reason `duplicate`).
  - `network.subject-prefix.v0` holds — both messages published on
    `agh.network.v0.qa-net06.broadcast`.
evidence:
  - B's inbox jsonl (ordered).
  - A's audit (two sent rows).
  - B's audit (one received row per id, duplicate rejected on second
    M1).
failure_signatures:
  - Out-of-order delivery: NATS ordering broken or audit timestamps
    nondeterministic.
  - Duplicate M1 delivered: dedupe `markSeen` window broken.
cleanup:
  - Both leave the channel.
```

### NET-07 — Channel image roundtrip via `kind=capability` artifact (binary blob with integrity hash)

```yaml qa-scenario
id: net-07-channel-binary-roundtrip
title: One peer publishes a binary blob (PNG, base64-encoded inside a capability artifact body); the receiver reads it; SHA-256 integrity hash matches and the canonical capability digest verifies
theme: network.channel-artifact
coverage:
  primary:
    - network.body.capability-digest
    - network.envelope.required-fields
  secondary:
    - delivery.ordering-preserved
    - network.replay-window
risk: high
live: false
provider: daemon-driver
preconditions:
  - 64KB-class fixture PNG saved at `$LAB/fixtures/qa-net07.png`.
  - `network.max_payload >= 96 KB` to give headroom over base64 expansion.
  - Capability def authored with `id=qa-net07`, `summary=...`, `outcome=...`,
    so `aghconfig.CanonicalCapabilityDigest` produces a stable digest.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/validate.go:419-471 (capability digest match)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:415, 438-454 (capability dispatch)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go:267-288 (CapabilityEnvelopePayload)
  - /Users/pedronauck/Dev/compozy/agh/internal/config (CanonicalCapabilityDigest helper)
steps:
  - Compute `sha256_local = sha256(file)`.
  - Build `kind=capability` envelope with `body.capability.id=qa-net07`,
    `body.capability.digest=<canonical>`, plus the PNG as a
    base64-encoded artifact in the capability's `examples` or attached
    via the agreed extension key (record the chosen path; if AGH does
    not yet support binary artifacts inside capability bodies, scope
    the scenario to a `direct` envelope whose body carries
    `{"text":"see ext.image","artifacts":[<base64>]}` — note in the
    report which path was used).
  - Send from A to B; B retrieves via inbox; B re-encodes and computes
    `sha256_received`.
expected_behavior:
  - `sha256_local == sha256_received` byte-for-byte.
  - `network.body.capability-digest` validation passes — neither side
    rejects the envelope.
  - Audit + message timeline rows reflect the round trip.
failure_signatures:
  - Hash mismatch: any envelope-mutation in the path (e.g. JSON
    re-encoding losing precision).
  - Capability digest validation rejects on either side: digest drift.
cleanup:
  - Delete fixture.
```

### NET-08 — Control-message dispatch (typed receipt → daemon handler emits result)

```yaml qa-scenario
id: net-08-control-message-dispatch
title: A typed control envelope (`kind=receipt` with `status=accepted`) reaches the daemon and the typed handler dispatches a follow-up `trace` event; the receiver observes both the original receipt and the trace continuation
theme: network.control-dispatch
coverage:
  primary:
    - network.envelope.required-fields
    - event.lineage-correlation.network
  secondary:
    - delivery.ordering-preserved
risk: high
live: false
provider: daemon-driver
preconditions:
  - Two daemons; same channel; an `interaction_id=qa-net08-int-1` already
    initiated by a prior `direct` send (A→B).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:399-425, 469-517 (lifecycle dispatch)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/lifecycle.go (LifecycleAction taxonomy)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go:290-310 (ReceiptBody, TraceBody)
steps:
  - Pre-send: A→B `direct` with `interaction_id=qa-net08-int-1`.
  - B sends `receipt` `{for_id=<original.id>, status=accepted, interaction_id=qa-net08-int-1}` back to A.
  - B follows up with `trace` `{state=working, message="picked up"}`.
  - On A: capture audit + inbox.
expected_behavior:
  - A's inbox shows one `receipt` envelope (typed body has
    `status=accepted` and no `reason_code`, per
    `validate.go:493-497`).
  - A's inbox shows one `trace` envelope with `state=working`.
  - Lifecycle action recorded by router transitions correctly through
    the LifecycleAction taxonomy (`internal/network/lifecycle.go`).
  - Both envelopes carry the same `interaction_id`.
evidence:
  - Audit + inbox rows.
failure_signatures:
  - Receipt with `status=accepted` carrying a `reason_code`: validator
    bypassed.
  - Trace not delivered after receipt: lifecycle path broken.
  - `interaction_id` lost across hops: correlation lost.
cleanup:
  - Send terminal `trace state=completed` to close.
```

### NET-09 — Cross-version peer (v0 ↔ v1 protocol mismatch handled cleanly)

```yaml qa-scenario
id: net-09-cross-version-negotiation
title: A v1 peer sends a `protocol=agh-network/v1` envelope to a v0-only daemon; the v0 daemon rejects with a stable typed error AND never silently corrupts the inbound message
theme: network.cross-version
coverage:
  primary:
    - cross-version.negotiation
    - network.envelope.protocol-pin
  secondary:
    - event.lineage-correlation.network
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Instance A is v0-only (current code, `validate.go:154-159` pins
    `protocol == ProtocolV0`).
  - Instance B emits a v1-shaped envelope (harness — handcraft an
    envelope with `protocol="agh-network/v1"`).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/validate.go:154-159
  - qmd://agh-rfcs-local/003-agh-network-v0.md (Section 1.3 upgrade path)
  - qmd://agh-rfcs-local/004-agh-network-v1.md (Section 6.2 v1 subject prefix `agh.network.v1`)
steps:
  - Inject the v1 envelope onto A's NATS via either:
    (a) the v0 broadcast subject (forces v0 daemon to parse it), or
    (b) the v1 subject `agh.network.v1.qa-builders.broadcast` if A is
    not subscribed to that prefix (the envelope MUST NOT reach A).
  - Capture A's audit + log.
expected_behavior:
  - When (a): A's audit logs `direction=rejected` with the typed
    `ErrInvalidField` error (protocol mismatch) — no silent corruption,
    no partial body decode side-effect.
  - When (b): A receives nothing because A subscribes only to
    `agh.network.v0.*`.
  - In neither case is a v1 envelope silently treated as v0.
evidence:
  - Audit row(s) + structured log line `network.message.rejected` with
    reason-code `malformed` (per `reasonCodeForReceiveError`
    `internal/network/router.go:1069-1080`) or `unsupported_profile`.
failure_signatures:
  - A processes the v1 envelope as v0 and routes its body downstream:
    silent corruption; critical safety failure.
  - A subscribes to `agh.network.v1.*` while declaring only v0 support:
    capability misadvertisement.
cleanup:
  - None.
```

### NET-10 — Embedded NATS profile boots, port collision resolved, no leak across restart

```yaml qa-scenario
id: net-10-embedded-nats-isolation
title: Daemon boots with embedded NATS on chosen port; port collision returns a clean typed error; restart releases the previous bind without orphaned socket
theme: network.transport
coverage:
  primary:
    - transport.embedded-isolation
  secondary:
    - network.subject-prefix.v0
risk: high
live: false
provider: daemon-driver
preconditions:
  - Test starts with port `:14222` already bound by a sentinel listener.
  - Then the sentinel is released and the daemon retries on the same
    port.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/transport.go:94-161, 186-213
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:235-248, 311-318
steps:
  - With sentinel bound: start daemon configured for port 14222. Daemon
    boot MUST surface a clean typed error citing the port; no daemon
    half-up.
  - Release sentinel.
  - Restart daemon; transport boots; `agh network status -o json`
    reports `enabled=true`, `status=running`, `listener_port=14222`.
  - `agh daemon stop`; verify `lsof -i :14222` is empty after shutdown
    (both NATS and any client subscriber drained per
    `transport.go:284-353`).
  - Restart again on same port; second boot must succeed with no
    "address already in use" lag.
expected_behavior:
  - Bind error path: typed; cleanup of partial init via `rollbackInit`
    (`manager.go:307-310, 321-336`).
  - After restart: zero socket leak.
evidence:
  - `lsof` output snapshots before/after each phase.
  - Daemon log fragments showing `network.started` and the eventual
    `network.stopped`.
failure_signatures:
  - Daemon comes up half-bound: rollback failed.
  - Port still bound after `agh daemon stop`: socket leak.
cleanup:
  - Stop daemon. Verify no orphan listeners.
```

### NET-11 — NATS subject discipline (only `internal/network` publishes/subscribes)

```yaml qa-scenario
id: net-11-nats-subject-discipline
title: Static + runtime audit confirms only `internal/network` imports `nats-io/*` and only it publishes/subscribes to `agh.network.v0.*`
theme: network.architecture
coverage:
  primary:
    - network.nats-isolation
    - network.subject-prefix.v0
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Repo at the SUT commit, fresh checkout.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md
  - /Users/pedronauck/Dev/compozy/agh/internal/network/transport.go (only NATS importer)
steps:
  - Static heuristics:
    - `rg -n '"github.com/nats-io' --glob '!internal/network/**' --glob '!**/*_test.go'`
      returns zero hits.
    - `rg -n '"github.com/nats-io' internal/network/` confirms only the
      transport file imports nats-io.
    - `rg -n 'agh\.network\.v0' --glob '!internal/network/**' --glob '!**/*_test.go'`
      returns zero hits.
    - `rg -n '\.Publish\(.*agh\.network' --glob '!internal/network/**'`
      returns zero hits.
  - Runtime: with both daemons running and a real send roundtrip, tail
    NATS subject inspection: every `Publish` call originates from
    `internal/network/transport.go:Publish` or
    `internal/network/router.go:publishEnvelope`.
expected_behavior:
  - All static greps zero-hit.
  - Runtime: every NATS subject observed matches `agh.network.v0.<channel>.*`.
evidence:
  - `net-11-static-grep.txt` capturing all rg outputs.
  - Runtime trace of `Publish` callers (via stack snapshot or pprof
    label).
failure_signatures:
  - Any non-network package imports nats-io: architecture violated.
  - Any subject outside `agh.network.v0.*` published in production:
    leak of NATS as inter-package coordination.
cleanup:
  - None (read-only audit).
```

### NET-12 — Real Claude Code agent (Instance A) asks Instance B's exposed agent a question via channel; transcripts correlate via lineage keys

```yaml qa-scenario
id: net-12-real-agent-to-agent
title: A real Claude Code session in Instance A speaks (via `agh network send`) to Instance B's real Claude Code session; the responder generates a real reply; both sides' transcripts and network audits correlate via `interaction_id`/`trace_id`
theme: network.real-agent
coverage:
  primary:
    - channel.join-discovery
    - identity.unverified-classification
    - event.lineage-correlation.network
  secondary:
    - delivery.ordering-preserved
    - network.envelope.required-fields
risk: high
live: conditional
provider: real-claude-code
preconditions:
  - Both AGH instances have valid `claude` CLI auth in the effective Claude
    home for the lane: operator `HOME` by default, or per-instance
    `PROVIDER_HOME` only when the scenario explicitly validates isolated
    native auth.
  - Both instances have an agent session active in `qa-builders` channel
    with capability `code` advertised in the local Peer Card.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:569-583, 645-655
  - /Users/pedronauck/Dev/compozy/agh/internal/session/manager_start.go (AGH_SESSION_ID injection seam)
  - /Users/pedronauck/Dev/compozy/agh/internal/agentidentity/identity.go:139-162
steps:
  - On A: `agh session new --agent claude --workspace $LAB-A` → `S_A`.
  - Prompt `S_A`: "Use `agh network send` to ask peer B (peer_id=<peer_id_B>)
    via channel `qa-builders`, kind=direct, interaction_id=qa-net12-int-1,
    'Tell me one short fact about JSON-RPC.' Then call `agh network inbox
    --interaction-id qa-net12-int-1` and return the reply text."
  - On B: `agh session new --agent claude --workspace $LAB-B` → `S_B`.
  - Pre-instruct `S_B` (skill or workspace AGENT.md): when an inbox
    arrives via `qa-builders`, reply via `agh network send` with the same
    `interaction_id`.
  - Drive the loop until `S_A` returns the answer.
  - Capture both transcripts, both network audit JSONLs, both events.db.
expected_behavior:
  - `S_A`'s transcript shows the outbound + the reply.
  - `S_B`'s transcript shows the inbound + the reply it sent.
  - Network audit on A: one `sent` (direct, qa-net12-int-1) + one
    `received` (direct, same interaction_id, opposite direction).
  - Network audit on B: mirror image.
  - Trust state on both sides: `unverified` (v0). No `proof` validated.
  - Lineage keys correlate: both transcripts can be joined to the
    network audit rows via `interaction_id`. Optional: scenario sets a
    `trace_id` and asserts the trace_id flows through both transcripts'
    `agh.session.event.v1` payloads.
evidence:
  - `S_A`-transcript.json, `S_B`-transcript.json.
  - `net-12-events-A.json`, `net-12-events-B.json`.
  - Forbidden-needle scan: zero `agh_claim_*` matches across all four
    artifacts.
failure_signatures:
  - `S_A` fails to call `agh network send` (skill/auth issue) — escalate
    to module 06.
  - `S_B` does not reply within 60 seconds: control loop broken or
    inbox notification not wired to the agent driver — fail.
  - Audit and transcript cannot be joined: lineage gap.
cleanup:
  - Stop both sessions, leave channel, stop daemons.
```

### NET-13 — Channel discovery (private channels not listed to non-members)

```yaml qa-scenario
id: net-13-channel-discovery-privacy
title: An agent in channel `qa-public` cannot enumerate channel `qa-private` (which only Instance A's session-2 has joined); private channels stay invisible to non-members in `agh network channels` output
theme: network.discovery
coverage:
  primary:
    - channel.private-not-listed
    - channel.join-discovery
  secondary:
    - peer-card.distinct-from-agent-card
risk: high
live: false
provider: daemon-driver
preconditions:
  - Instance A has session 1 in `qa-public` and session 2 in `qa-private`.
  - Instance B has session 3 in `qa-public` only.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/peer.go:452-482, 680-700
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:659-670
steps:
  - On B: `agh network channels -o json`. Capture full list.
  - On B: `agh network peers --channel qa-private -o json`. Expected:
    empty list OR error ("not a member" / "not found").
  - On B: `agh network peers --channel qa-public -o json`. Expected:
    A's session-1 + B's session-3 visible.
expected_behavior:
  - `qa-private` does not appear in B's `network channels` output.
  - `qa-private` peer enumeration on B returns zero peers (B has no
    presence in that channel).
  - `qa-public` enumeration on B includes A's session-1.
  - Peer Card vs Agent Card distinction: `Peer Card` is what is
    advertised on the network; the per-agent Agent Card (skills,
    capabilities, AGENT.md prompt overlays) stays inside the originating
    daemon's session record and never leaks onto the wire except through
    `whois`-discovered `capabilities` (peer-card field), which carries
    capability *identifiers*, not full Agent Card definitions.
evidence:
  - JSON outputs from both calls.
  - Static check that the peer registry's `ListChannels` only counts
    channels the local registry knows about (`peer.go:452-482`).
failure_signatures:
  - `qa-private` visible to B: privacy invariant violated.
  - Agent Card details (full skill prompts, etc.) leaking into Peer Card
    advertisements: privacy + protocol-implementability invariant
    violated.
cleanup:
  - Both leave their channels.
```

### NET-14 — Outbound HTTP timeouts (no `http.DefaultClient` in any production network path)

```yaml qa-scenario
id: net-14-outbound-http-timeouts
title: No external HTTP call from `internal/network` (or anywhere reached from a network ingress path) uses `http.DefaultClient`; every external client has an explicit timeout
theme: network.security
coverage:
  primary:
    - transport.publish-timeout
  secondary:
    - network.nats-isolation
risk: high
live: false
provider: daemon-driver
preconditions:
  - Repo at SUT commit.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md (Security Invariants — "External-call timeouts.")
  - /Users/pedronauck/Dev/compozy/agh/internal/network/transport.go:21-23, 234-261
steps:
  - Static heuristics:
    - `rg -n 'http\.DefaultClient' internal/network/ internal/agentidentity/`
      returns zero hits in production code.
    - `rg -n 'http\.DefaultClient' internal/ --glob '!**/*_test.go'`
      returns zero hits in any package reachable from a network ingress
      path (the network ingress code path is the call graph rooted at
      `Manager.handleInboundMessage`).
    - For every `http.Client{}` constructed in `internal/network/` or
      `internal/agentidentity/`, verify a non-zero `Timeout` is set OR
      verify the call is wrapped in `context.WithTimeout`.
  - Runtime: under a synthetic flood (NET-16 setup), watch for goroutines
    blocked in HTTP calls; none should remain after the flood subsides.
expected_behavior:
  - All static greps zero-hit.
  - Runtime: no goroutine blocked in `(*http.Client).Do` longer than the
    test timeout (`pprof goroutine` snapshot).
evidence:
  - `net-14-static-grep.txt`.
  - pprof snapshot (`agh debug pprof goroutine`).
failure_signatures:
  - Any production hit on `http.DefaultClient`: invariant violated.
  - Goroutine blocked indefinitely in HTTP: missing timeout.
cleanup:
  - None.
```

### NET-15 — Identity rotation (peer rotates public key; old key gracefully retired)

```yaml qa-scenario
id: net-15-identity-rotation
title: A session leaves the channel and rejoins under a new `peer_id` (representing key rotation in v1); old route-token's direct subject naturally drains; in-flight messages already addressed to the OLD `peer_id` deliver through the lifecycle of the existing subscription, NEW messages route to the new fingerprint
theme: network.identity-rotation
coverage:
  primary:
    - identity.rotation
    - channel.join-discovery
  secondary:
    - delivery.ordering-preserved
    - transport.partition-recovery
risk: high
live: false
provider: daemon-driver
preconditions:
  - Two daemons; a session on Instance A joined as `peer_id_old` in
    channel `qa-rot`.
  - Instance B has recent presence cache for `peer_id_old` (greet within
    `2 * greet_interval`).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/peer.go:179-194 (peer-id collision check), :497-506 (`removeLocalIndexesLocked`), :519-530 (`expireRemotesLocked`)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:415-443 (prepareJoinLocalPeer rejoin path)
steps:
  - On B: `agh network send --channel qa-rot --to peer_id_old
    --kind direct --interaction-id qa-rot-int-1 --body
    '{"text":"to old"}'` (in flight before A rotates).
  - On A: `agh session network leave qa-rot` then
    `agh session network join qa-rot --peer-id peer_id_new`.
  - On B: send another `direct` to `peer_id_old`. Expected:
    `ErrTargetPeerNotFound` after old presence expired (or after
    presence cache TTL elapses).
  - On B: send another `direct` to `peer_id_new`. Expected: delivered.
  - Wait `2 * greet_interval` and re-list peers on B; old peer must have
    expired from remote cache.
expected_behavior:
  - The first message (`to old`) was delivered to A's old subscription
    if it arrived before the leave; otherwise rejected as
    `ErrTargetPeerNotFound`. Either outcome is acceptable; document the
    chosen ordering.
  - The send to `peer_id_old` AFTER rotation + cache expiry returns
    `ErrTargetPeerNotFound` (a typed error).
  - The send to `peer_id_new` succeeds.
  - No raw-token leak across the rotation window.
  - No silent cross-routing — old subject never receives a message
    intended for the new peer.
evidence:
  - Audit on A and B; `agh network peers -o json` snapshots before, at,
    and after the rotation window.
failure_signatures:
  - Old subject still receives messages addressed to `peer_id_new`:
    cross-routing bug.
  - Presence cache leak: `peer_id_old` visible after `2*greet_interval`.
cleanup:
  - Leave the channel.
```

### NET-16 — Network partition: A loses connectivity to B; reconnect within window delivers buffered messages; outside window cleanly fails

```yaml qa-scenario
id: net-16-partition-recovery
title: Simulated NATS-cluster partition between A's and B's embedded servers; on reconnect within the configured window, queued messages flow; beyond the window, the daemon emits `network.disconnected`+`network.reconnected` and re-greets, but messages dropped to overflow are audited as `queue_overflow`
theme: network.partition
coverage:
  primary:
    - transport.partition-recovery
    - delivery.queue-overflow-audited
  secondary:
    - channel.join-discovery
    - event.lineage-correlation.network
risk: high
live: false
provider: daemon-driver
preconditions:
  - Two daemons running. Channel joined on both. `network.max_queue_depth=8`
    so overflow is fast to provoke.
  - Harness can interrupt connectivity between A's and B's NATS server
    (e.g. firewall rule on loopback, or daemon-side `transport.Drain`
    for one connection, or kill the inter-server route).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:1065-1103 (handleDisconnect/handleReconnect)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager_test.go:631-636 (queue_overflow audit assertion)
steps:
  - Send 5 direct messages A→B. Verify B receives all 5.
  - Disconnect (firewall rule).
  - Send 12 direct messages A→B with the partition active.
  - Reconnect.
  - Wait for `network.reconnected` log + post-reconnect re-greet
    (`handleReconnect` re-publishes greets per `manager.go:1082-1102`).
  - Capture B's inbox.
expected_behavior:
  - On the disconnect, A's audit logs `network.disconnected`.
  - On reconnect, A logs `network.reconnected` with `sessions=<n>`.
  - B receives messages 1-8 (within `max_queue_depth`); messages 9-12
    are audited `direction=rejected reason=queue_overflow` on whichever
    side enforced the cap.
  - Messages dropped due to overflow have audit rows that include their
    `message_id` so operators can investigate.
  - No silent corruption: no message arrives with mutated body.
evidence:
  - A's audit + log fragments showing disconnect/reconnect.
  - B's inbox + audit rows showing exactly the expected delivered count
    and the overflow rejections.
failure_signatures:
  - More than `max_queue_depth` messages delivered: cap not enforced.
  - Re-greet missing after reconnect: `transport.partition-recovery`
    invariant violated.
  - Queue-overflow not audited: silent message drop.
cleanup:
  - Tear down firewall rule. Stop both daemons.
```

### NET-17 — DoS resistance: 10k bogus peer-card requests/min are rate-limited; daemon stays responsive

```yaml qa-scenario
id: net-17-dos-resistance
title: Under a flood of 10k bogus `whois` request envelopes per minute, the daemon stays responsive (HTTP/UDS responsive within 200ms p95) and bogus envelopes are audited as rejected; no goroutine leak; no memory blow-up
theme: network.dos-resistance
coverage:
  primary:
    - dos.bogus-peer-card-ratelimit
    - delivery.queue-overflow-audited
  secondary:
    - transport.publish-timeout
risk: high
live: false
provider: daemon-driver
preconditions:
  - Instance A running. NATS port reachable from harness with the
    embedded server's auth token (test-only seam).
  - Baseline pprof snapshot recorded before flood.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:317-347 (parse-error rejection)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:828-848 (warn-and-drop on receive failure)
steps:
  - Pre-flood: hit `GET /api/network/status` 10x; record p95 latency.
  - Start a goroutine pump emitting 10000 envelopes/min, ~167/sec, of
    malformed `whois` requests (e.g. invalid `to`, missing
    `interaction_id`, unknown profile in `proof`). Run for 60s.
  - During flood: hit `GET /api/network/status` continuously; record
    p95.
  - Concurrently: a clean `agh network send` from A→B is still
    delivered.
  - Stop flood; capture pprof snapshot; capture audit row count for the
    rejection window.
expected_behavior:
  - HTTP/UDS p95 latency during flood is within 2x baseline (and below
    200ms absolute).
  - Audit log records each rejection — and yet the audit writer does
    not block the inbound path (file writer is debounced/buffered or
    non-blocking; if not, `dos.bogus-peer-card-ratelimit` ought to
    require buffering, document the trade-off).
  - Goroutine count returns to baseline after flood ends (within 5
    seconds).
  - Resident memory growth bounded (< 200 MB additional during flood).
  - Clean A→B `direct` send during the flood is delivered.
evidence:
  - p95 latency before/during/after flood.
  - pprof goroutine + heap snapshots before/after.
  - Audit row count for the flood window.
failure_signatures:
  - HTTP/UDS unresponsive during flood: rate-limit gap.
  - Goroutine leak post-flood: rejection path leaking goroutines.
  - Memory continues growing: unbounded buffering of bogus envelopes.
  - Clean send during flood does not deliver: HoL blocking on inbound
    queue.
cleanup:
  - Stop pump. Verify daemon stable.
```

### NET-18 — Agent identity proof: env spoof + UDS header mismatch produces stable typed errors and exit codes

```yaml qa-scenario
id: net-18-agent-identity-proof
title: A caller spoofing `AGH_SESSION_ID`/`AGH_AGENT` against a session that is not active, or with a mismatched agent name, or against the wrong workspace, gets stable typed errors and the documented deterministic exit codes; identity errors never become exit 0
theme: network.identity-validation
coverage:
  primary:
    - agentidentity.daemon-authoritative
    - agentidentity.exit-code-determinism
  secondary:
    - identity.proof-stripping-defense
risk: critical
live: false
provider: daemon-driver
preconditions:
  - Instance A daemon running. One active session `S_active`. One stopped
    session `S_stopped`. One never-existed `S_ghost`.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/agentidentity/identity.go:139-262, 349-365
  - /Users/pedronauck/Dev/compozy/agh/internal/agentidentity/identity_test.go (test patterns to replicate end-to-end)
steps:
  - Run an agent CLI subcommand with `AGH_SESSION_ID=` (empty) →
    expected `identity_required`, exit 64.
  - Same with `AGH_AGENT=` empty → `identity_required`, exit 64.
  - Same with `AGH_SESSION_ID=S_ghost` → `identity_stale`, exit 65.
  - Same with `AGH_SESSION_ID=S_stopped` → `identity_stale`, exit 65.
  - Same with `AGH_SESSION_ID=S_active` but `AGH_AGENT=wrong-name` →
    `identity_mismatch`, exit 65.
  - Same with `AGH_SESSION_ID=S_active`, `AGH_AGENT=ok`, but UDS header
    `X-AGH-Workspace-ID=foreign-workspace` → `identity_unauthorized`,
    exit 77.
  - Daemon down → `identity_lookup_unavailable`, exit 69.
expected_behavior:
  - Each call returns the stable JSON error payload from
    `MarshalErrorJSON` / `MarshalErrorJSONL` with `code`, `message`,
    `action`, `exit_code` (`identity.go:131-347`).
  - Exit codes match the constants 64/65/69/77 exactly.
evidence:
  - JSONL captures of each error payload.
  - Process exit code captures.
failure_signatures:
  - Any code path returns exit 0 with an error payload: critical bug.
  - Error payload missing `action`: contract drift.
  - Stale session ID classified as `identity_lookup_unavailable`:
    classification mistake (RFC 003 forensic: stale ACP id distinction).
cleanup:
  - Stop daemon.
```

### NET-19 — Audit row + structured log lineage coverage matrix

```yaml qa-scenario
id: net-19-audit-lineage-coverage
title: Every send/receive/reject/deliver path emits an audit row that carries the documented correlation keys, including any `interaction_id`/`reply_to`/`trace_id`/`causation_id` plus the `agh.workflow_id`/`agh.handoff_*` extension fields when present
theme: network.observability
coverage:
  primary:
    - event.lineage-correlation.network
  secondary:
    - delivery.queue-overflow-audited
    - delivery.ordering-preserved
risk: high
live: false
provider: daemon-driver
preconditions:
  - Two daemons; channel joined; one full lifecycle session prepared
    (greet → say → direct → receipt → trace).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/audit.go:288-326
  - /Users/pedronauck/Dev/compozy/agh/internal/network/manager.go:1233-1265 (networkLogFields)
steps:
  - Drive one full lifecycle: A→B `say`, A→B `direct`+`interaction_id`,
    B→A `receipt accepted`, B→A `trace state=working`, B→A
    `trace state=completed`. Set
    `ext={"agh.workflow_id":"wf-net19","agh.handoff_version":"1"}`.
  - For each emitted audit row + log line, parse and assert:
    - `kind`, `channel`, `peer_from`, `peer_to`, `message_id` are present
      and non-empty (where applicable).
    - `interaction_id`, `reply_to`, `trace_id`, `causation_id` are
      preserved when set on the envelope.
    - `agh.workflow_id` flows into structured log fields per
      `networkLogFields` (`manager.go:1252-1264`).
    - `agh_claim_*` raw token NEVER appears.
expected_behavior:
  - Coverage matrix passes for every audit row across the lifecycle.
  - Structured logs carry the workflow + handoff fields.
evidence:
  - Coverage matrix in `net-19-summary.json` keyed by
    `(direction, kind)`.
failure_signatures:
  - Any required correlation key missing on any direction-kind pair:
    observability gap.
  - `agh.workflow_id` stripped on the wire: extension model violated.
  - Raw `agh_claim_*` in any payload: redaction violated.
cleanup:
  - Stop both daemons.
```

## 8. Optional / nice-to-have scenarios (run if time)

These extend coverage without being strictly required for ship.

### NET-20 — Whois capability discovery returns brief vs catalog correctly

```yaml qa-scenario
id: net-20-whois-capability-discovery
title: A `whois request` carrying a capability-catalog discovery extension returns the responder's capability catalog only when the responder advertised it; absent advertisement returns brief only
theme: network.discovery
coverage:
  primary:
    - peer-card.distinct-from-agent-card
  secondary:
    - channel.join-discovery
risk: medium
live: false
provider: daemon-driver
preconditions:
  - Two peers A, B in `qa-net20`. B advertises `CapabilityCatalog`
    (`internal/network/manager.go:432-441`).
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/router.go:606-714 (whois request handling)
  - /Users/pedronauck/Dev/compozy/agh/internal/network/peer.go:618-651 (capability catalog retention)
steps:
  - From A, send a `whois request` for B with the discovery extension
    set; capture the `whois response` envelope.
  - Re-run with B not advertising a catalog; capture the response.
expected_behavior:
  - First run: response includes the catalog projection of B's
    advertised capabilities.
  - Second run: response includes only the Peer Card brief.
  - Agent Card details never present on the wire.
failure_signatures:
  - Catalog returned when not advertised: leak.
  - Brief mode missing fields: spec drift.
cleanup:
  - Leave channel.
```

### NET-21 — `expires_at` honored over `max_replay_age`

```yaml qa-scenario
id: net-21-expires-at-priority
title: An envelope with `expires_at` in the past is rejected as expired even when `now - ts <= max_replay_age`
theme: network.replay
coverage:
  primary:
    - network.replay-window
  secondary:
    - network.envelope.required-fields
risk: medium
live: false
provider: daemon-driver
preconditions:
  - Two daemons; max_replay_age=600s; envelope with `ts=now-5s,
    expires_at=now-1s`.
code_refs:
  - /Users/pedronauck/Dev/compozy/agh/internal/network/validate.go:327-343
steps:
  - Inject the past-expiry envelope onto A.
expected_behavior:
  - Audit row direction=rejected, reason_code=expired.
  - Envelope NOT delivered.
failure_signatures:
  - Delivered: priority of `expires_at` over replay-age violated.
cleanup:
  - None.
```

## 9. Coverage matrix (this child)

| Coverage ID                            | Scenarios                                  |
| -------------------------------------- | ------------------------------------------ |
| `network.protocol-implementable`       | NET-09, NET-13                             |
| `network.nats-isolation`               | NET-11, NET-14                             |
| `network.subject-prefix.v0`            | NET-01, NET-06, NET-10, NET-11             |
| `network.channel-grammar`              | NET-06 (implicit in `qa-net06` validation) |
| `network.peer-id-grammar`              | NET-15 (rotation exercises grammar)        |
| `network.envelope.required-fields`     | NET-01, NET-06, NET-08, NET-12             |
| `network.envelope.protocol-pin`        | NET-09                                     |
| `network.body.greet-from-binding`      | NET-13 (Peer Card discipline)              |
| `network.body.capability-digest`       | NET-07                                     |
| `network.replay-window`                | NET-06, NET-21                             |
| `identity.proof-stripping-defense`     | NET-02, NET-18                             |
| `identity.proof-invalid-rejected`      | NET-03                                     |
| `identity.unverified-classification`   | NET-01, NET-04, NET-12                     |
| `network.no-claim-token-in-metadata`   | NET-05                                     |
| `peer-card.distinct-from-agent-card`   | NET-13, NET-20                             |
| `channel.join-discovery`               | NET-01, NET-06, NET-12, NET-13, NET-15     |
| `channel.private-not-listed`           | NET-13                                     |
| `delivery.ordering-preserved`          | NET-01, NET-06, NET-08, NET-12, NET-15     |
| `delivery.queue-overflow-audited`      | NET-16, NET-17, NET-19                     |
| `transport.publish-timeout`            | NET-14, NET-17                             |
| `transport.embedded-isolation`         | NET-01, NET-10                             |
| `transport.partition-recovery`         | NET-15, NET-16                             |
| `identity.rotation`                    | NET-15                                     |
| `dos.bogus-peer-card-ratelimit`        | NET-17                                     |
| `cross-version.negotiation`            | NET-09                                     |
| `event.lineage-correlation.network`    | NET-01, NET-02, NET-05, NET-08, NET-09, NET-12, NET-19 |
| `agentidentity.daemon-authoritative`   | NET-18                                     |
| `agentidentity.exit-code-determinism`  | NET-18                                     |

Total: 19 mandatory + 2 optional = 21 scenarios. Every coverage ID is
exercised by at least one scenario; every critical-risk ID is exercised
by at least two.

## 10. Forbidden-needle list (transcript, audit, log payloads)

Per the openclaw `forbiddenNeedles` pattern. None of the following may
appear in any audit row, message timeline row, structured log line, SSE
event, or HTTP/UDS response across any NET scenario:

- Any literal raw `agh_claim_<>=12 random char>` (regex
  `agh_claim_[A-Za-z0-9_-]{12,}`). Hash form (`claim_token_hash`) is
  permitted (`internal/CLAUDE.md` Security Invariants).
- Any provider API key shape: `sk-`, `xoxb-`, `AKIA`, `ya29.`,
  `claude_oauth_*`.
- Any private key encoding: `-----BEGIN PRIVATE KEY-----`, raw 32-byte
  Ed25519 secret keys (verify via base64-decoded length `== 64`).
- Any reference to deleted legacy vocabulary in network artifacts:
  `recipe`, `workflow`, `procedure`, `playbook` for AGH artifacts (per
  `docs/_memory/glossary.md` — canonical term is `capability`).
- Any internal-only AGH type names that would imply protocol coupling
  (e.g. `taskpkg.ActorContext`, `session.Info`) — these MUST NOT appear
  inside any wire envelope `body` or `ext`. They are allowed in audit
  metadata (which never crosses daemon boundaries).
- Any v0 envelope advertising `protocol="agh-network/v1"` (or vice-versa)
  — would indicate cross-version smuggling.

A single scenario test failure on this list is shippability-critical and
must be triaged immediately.

## 11. Reporting contract

Each scenario writes the four-artifact set required by the openclaw
operator-flow pattern (markdown report + JSON summary + observed events
+ combined log) for **both** instances when peer-to-peer behavior is
exercised. The aggregate `net-summary.json` for this child carries the
coverage matrix from §9 alongside per-scenario `outcome ∈ {worked,
failed, blocked, follow-up}` and machine-readable timing.

The scenario operator runs in-character (per the `real-scenario-qa`
skill); every run ends with a Worked / Failed / Blocked / Follow-up
section covering all 19 mandatory scenarios. A child run is shippable
only when:

- Every mandatory scenario is `worked` or has an explicit accepted
  follow-up.
- NET-02, NET-05, NET-09, NET-11, NET-14, NET-18 are ALL `worked` —
  these are the non-negotiable safety + architecture audits.
- No forbidden-needle hit anywhere.
- `make verify` passed on the SUT branch before this child ran (cite
  commit SHA in `net-summary.json`).
- Every `live: conditional` scenario either ran with pooled credentials
  or has an explicit "skipped — credentials unavailable" entry that
  references the pooled-credential broker plan from
  `openclaw-qa-patterns.md` §4.
