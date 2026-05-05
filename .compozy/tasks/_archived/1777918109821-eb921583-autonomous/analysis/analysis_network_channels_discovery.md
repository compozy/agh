# Network Channels — Discovery, Membership, and Capability-Based Routing

> Slice owner: network channel autonomy
> Sources reviewed: `internal/network/*`, `internal/api/{contract,core,udsapi,httpapi}/*`, `internal/cli/network.go`, `internal/session/*`, `web/src/systems/network/*`, `docs/ideas/network/*`, `docs/ideas/orchestration/multi-agent-patterns-analysis.md`, `.resources/{hermes,multica,openclaw,paperclip}`.

---

## 1. TL;DR

Today an AGH session is **born already bound to exactly one channel string**, and that channel is opaque metadata — not an addressable, declarative thing the agent can reason about. The runtime can list channels, but only as a side-effect of which sessions happen to be alive (`PeerRegistry.ListChannels`, `internal/network/peer.go:452`). There is no channel manifest, no per-channel purpose/topic surfaced over the wire, no `whois`/`greet` for channels, no SSE stream that announces channel creation, and no API for an agent to enumerate, filter, join, or spawn a channel from inside a turn. Capability data exists per peer (`capability_brief.go`, `capability_catalog.go`) but is never indexed by topic/role and never matched against channels. The end-state — "agent boots, asks 'what channels exist? which are relevant to my role/skills?', joins them, listens, reacts" — is structurally impossible on the current code path: every join goes through `Sessions.Create(... Channel: …)` from a human-driven `POST /api/network/channels` (`internal/api/core/network_details.go:44`, `:584`).

The five concrete gaps are: (a) channel is a free string with no schema or topic vocabulary; (b) a session has exactly one channel and cannot multi-home; (c) `ListChannels` returns counts only — no purpose, capabilities, declared topics, or owner; (d) there is no agent-callable "create ad-hoc channel" path that doesn't require spawning a fresh session per agent; (e) there is no channel discovery event stream — agents cannot learn about channels created after their session started.

---

## 2. Current channel model

### 2.1 Lifecycle (ASCII)

```
                        ┌────────────────────────────────────────────┐
 HUMAN (UI/CLI)         │  POST /api/network/channels                 │
   │                    │  body { channel, purpose, workspace_id,     │
   │                    │         agent_names[] }                     │
   ▼                    └────────────────────────────────────────────┘
 BaseHandlers.CreateNetworkChannel
   internal/api/core/network_details.go:44
        │
        ├── normalizeNetworkChannel  → regex `^[a-z0-9][a-z0-9_-]{0,63}$`
        │     internal/network/rules/channel.go:5
        │
        ├── for each agent_name:
        │     Sessions.Create(CreateOpts{Channel, Workspace, AgentName, …})
        │       internal/api/core/network_details.go:584
        │
        └── networkStore.WriteNetworkChannel(NetworkChannelEntry{
                  Channel, WorkspaceID, Purpose, CreatedBy})
              ↳ persisted in agh.db `network_channels` table
                (metadata only — purpose is human prose, not machine-routable)

  Per-session activation
  internal/session/manager_helpers.go:108
        │
        └── joinNetworkPeer(session, capabilities)
              │
              └── lifecycle.JoinChannel(NetworkPeerJoin{
                      SessionID, PeerID:=agent@sessionId, Channel, Capabilities})
                    internal/session/manager_helpers.go:148

  network.Manager.JoinChannel
  internal/network/manager.go:335
        │
        ├── peers.RegisterLocalWithCapabilityCatalog(...)
        │     internal/network/peer.go:136
        │
        ├── acquireBroadcastSubscription(channel)
        │     internal/network/manager.go:914
        │     ↳ SUBSCRIBE  agh.network.v0.<channel>.broadcast
        │
        ├── subscribeDirect(channel, peerID)
        │     internal/network/manager.go:979
        │     ↳ SUBSCRIBE  agh.network.v0.<channel>.peer.<route_token>
        │
        └── startAuditedHeartbeat → publishGreet (router.PublishGreet)
              internal/network/router.go:174
              ↳ GreetBody{ PeerCard{ PeerID, DisplayName, Capabilities[] (IDs only),
                           ProfilesSupported, ArtifactsSupported, TrustModesSupported,
                           Ext{ "agh.capabilities_brief": [{id,summary}] }}}
              ↳ PUBLISH    agh.network.v0.<channel>.broadcast

  PeerRegistry (in-memory, per-channel)
  internal/network/peer.go:55
        localsByID:        sessionID -> LocalPeer
        localsByChannel:   channel   -> {peerID -> sessionID}
        remotesByChannel:  channel   -> {peerID -> RemotePeerEntry (TTL=2*greet)}

  Outbound discovery (capability-catalog whois)
  internal/network/router.go:600  + capability_catalog.go:42
        Whois request body: { type:"request", query:"<peerID|displayName|cap-id|profile|trust>" }
        Optional ext:       { "agh.include":["capability_catalog"], "agh.capability_ids":[…] }
        Match path:         PeerRegistry.MatchLocalPeers(channel, query) (peer.go:276)
                            → containsString over peer-card slices (peer.go:660-668)
        ↳ NO channel-level whois. Whois only finds peers WITHIN your channel.
```

### 2.2 What an agent inside a session can actually see

| Question | Mechanism | File |
| --- | --- | --- |
| "What channel am I on?" | injected at session create only | `session/session.go:54,78` |
| "Who else is on my channel?" | implicit via `say`/`greet` envelopes appearing in inbox | `delivery.go` (no enumeration tool) |
| "What channels exist?" | NOT REACHABLE from the agent — only via human REST/CLI | `udsapi/routes.go:228` (`GET /api/network/channels`) |
| "Can I join another channel?" | NO API — `JoinChannel` only fires from session activation | `manager_helpers.go:108` |
| "Can I create a channel?" | NO — `POST /api/network/channels` requires `agent_names[]` and spawns NEW sessions | `network_details.go:44` |

---

## 3. What works for autonomy already

Being honest about the foundations that are in place:

- **Capability projection is real.** Peers advertise `capabilities[]` (string IDs) plus a compact brief in ext (`capability_brief.go:48`). On request, the rich catalog (outcome, requirements, examples, etc.) is delivered via `whois` with `agh.include:["capability_catalog"]` (`capability_catalog.go:42-81`, `router.go:642`). Agents can already discover *peer* capabilities precisely.
- **Greet heartbeats give live presence per channel.** `Heartbeat` republishes the peer card on `GreetIntervalDuration()` (`manager.go:576-620`), and remote entries auto-expire at `2 × greet` (`peer.go:351`). The "who is here" view is correct in real time.
- **The transport supports NATS wildcards by construction.** Subjects are `agh.network.v0.<channel>.broadcast` and `…<channel>.peer.<route>` (`transport.go:355-374`). Wildcard subscriptions like `agh.network.v0.*.broadcast` would be one line in `subscribeDirect`. The substrate is multi-channel-ready; the policy layer above is single-channel-only.
- **Per-channel persisted metadata exists.** `store.NetworkChannelEntry { Channel, WorkspaceID, Purpose, CreatedBy, CreatedAt, UpdatedAt }` is written on creation (`network_details.go:611`) and queryable via `networkStore.ListNetworkChannels` (`network_details.go:454`). The schema slot for "purpose" is already there — it just isn't broadcast or matched.
- **Bundles can declare channels.** `DeclaredNetworkChannelPayload { Name, Description, Primary }` (`internal/api/contract/bundles.go:83`) and `BundleNetworkSettingsPayload.ConfiguredDefaultChannel` (`bundles.go:94`) exist. A bundle activation already knows "this extension wants channel X with description Y." Today that's a UI hint; it could be the seed for an autodiscover manifest.
- **The protocol envelope reserves room.** The `agora-spec-v0.2` design (`docs/ideas/network/agora-spec-v0.2.md:339-349`) explicitly puts `interests: ["translation:*"]` inside `greet.body` for client-side filtering — the wire shape is approved; just not implemented in `GreetBody` (`envelope.go:217-225`).
- **Channel grammar is centralized.** `rules/channel.go:5` and `network.ValidateChannel` (`validate.go:91`) — adding "ad-hoc channel" naming policies (e.g., a `wf-` prefix for workflow channels) is one regex change.

---

## 4. What is missing

Concrete gaps phrased as questions an autonomous agent would actually ask.

### 4.1 "What channels exist? Show me the manifest."

`Manager.ListChannels` (`manager.go:650`) returns `[]ChannelInfo { Channel, PeerCount }` — a string and a number. The richer payload at `GET /api/network/channels` (`network_details.go:471`) adds `WorkspaceID, Purpose, CreatedBy, LastActivityAt, MessageCount, KindCounts` but is HTTP/UDS only and never reaches the agent's prompt. There is **no `network.ChannelManifest`** with topic, declared roles needed, owner peer, parent/child relation, expiry, or capability hints. Hermes solved exactly this with a JSON channel directory, refreshed every 5 min, persisted to disk, queryable by name (`/.resources/hermes/gateway/channel_directory.py:60-99`). AGH has no analog.

### 4.2 "Filter live channels by topic / capability / role."

There is no filter at all. `ListChannels` takes no query (`manager.go:650`). `MatchLocalPeers` matches *peers* on a single string against PeerCard slices (`peer.go:276`); there is no `MatchChannels(query)` indexing channel purpose, declared capabilities, or required roles. The `agora` spec calls this out explicitly: `greet.body.interests` for client-side filtering (`docs/ideas/network/agora-spec-v0.2.md:339`); not implemented in `GreetBody` (`envelope.go:217`).

### 4.3 "Auto-join based on my declared skills."

Two blockers:

1. **A session is mono-channel.** `Session.Channel` is a single immutable string (`session/session.go:54`); `joinNetworkPeer` joins exactly one (`manager_helpers.go:148`). The peer registry enforces this — `localsByID: sessionID → LocalPeer` is a 1:1 map (`peer.go:60`). To "join channel B" the agent has to either leave A or spawn a new session. There is no `Manager.JoinAdditionalChannel(sessionID, channel)`.
2. **An agent definition does not declare interests.** `AgentDef` (`config/agent.go:17-28`) carries `Name, Provider, Command, Model, Tools, Permissions, MCPServers, Hooks, Capabilities` — there is no `Channels`, `Interests`, `SubscribesTo`, `DefaultChannels`, or `ChannelMatchers` field. Even bundle-declared channels (`contract/bundles.go:83`) only feed UI defaults, never automated subscription.

### 4.4 "Create an ad-hoc channel for the sub-team I'm spinning up."

The only creation path (`POST /api/network/channels`, `network_details.go:44`) is a coarse setup ritual: it requires `workspace_id`, `purpose`, and `agent_names[]`, and **spawns a brand-new session per named agent** (`network_details.go:584`). For a coordinator that wants to say "I'm starting a refactor; spin up channel `wf-refactor-x` and add my existing reviewer + tester sessions" — there is no API. There is no `Manager.CreateChannel(spec)` separate from session creation, no `POST /api/network/channels/:channel/members`, and no `/network/sessions/:id/channels` (multi-channel attach).

### 4.5 "Notify me when a channel I care about appears."

There is no channel-creation event stream. The audit pipeline (`audit.go`) records *envelopes*, not channel lifecycle. SSE for network status streams nothing channel-shaped (search confirmed: no `EventNetworkPeerJoined` / `EventChannel*` types anywhere in `internal/`). The web UI re-polls `useNetworkChannels` (`web/src/systems/network/hooks/use-network.ts`); the agent has no equivalent. The `agora-spec-v0.2` model expected this to ride on `greet` broadcasts so any peer subscribed to the broadcast subject would learn about new peers and (transitively) new channels — but AGH never broadcasts a `channel.announce` kind.

### 4.6 Other autonomy holes

- **No "channel-scope whois".** Today `whois` matches peer cards (`peer.go:276`); there's no kind to ask "describe channel X". A peer joining channel `wf-refactor-x` mid-flight has no way to ask "what is this channel for, what roles are needed, what's the parent task?"
- **`task ↔ channel` binding is one-directional and validation-only.** `tasks.go:366-416` enforces `task.NetworkChannel == ingressChannel`; it never creates a channel for a task. Yet the *idea* of "task spawns coordinating channel" is exactly what the task slice already wants (every task carries `NetworkChannel`, `contract/tasks.go:29`).
- **`bundles.declared_channels` is dead inventory.** Bundles list channel intent (`bundles.go:83`) but activation never registers them in `network_channels` and never broadcasts them. The plumbing reaches `BundleNetworkSettings` (`core/bundles.go:123`) and stops at the UI.
- **Capabilities never index channels.** `CapabilityCatalog` (`config/capabilities.go:43`) is per-agent. There is no inverted index `capability_id → channels offering it` and no protocol kind to advertise "channel X needs capability Y".

---

## 5. Reference comparisons

How peer projects handle the same concept space.

| Concern | AGH (today) | Hermes | Multica | OpenClaw | Paperclip |
| --- | --- | --- | --- | --- | --- |
| **Channel directory / manifest** | None for agents — only HTTP `GET /channels` returns aggregates (`network_details.go:471`) | **Yes**: `~/.hermes/channel_directory.json`, rebuilt every 5 min from live platform adapters + session history (`gateway/channel_directory.py:19-99`) | N/A — uses `useChatStore` singleton (`packages/core/chat/index.ts:1`); chats per workspace, no peer-discovered channels | "Channels" = chat platform accounts (Discord, Slack, WhatsApp), not agent-to-agent rooms (`docs/concepts/multi-agent.md:104`) | `bridge/stream/:channel` is a per-plugin SSE topic (`server/src/routes/plugins.ts:346`); not agent-discovery |
| **Filter by topic / capability** | Only `whois` over peer-card slices (`peer.go:660`); no channel-level filter | Resolves human-friendly channel names → IDs (`channel_directory.py:resolve_channel_name`); platform/type filtering | Subscriber lists by issue (`packages/views/issues/hooks/use-issue-subscribers.ts`) | Bindings by channel + account + agent ID (config-time, not runtime queryable) | (pluginId, channel, companyId) composite subscription key (`plugin-stream-bus.ts:5`) |
| **Auto-join based on agent declaration** | None — agent picks no channels; session is mono-channel | Agent reads home channels from config; one home per platform (`gateway/channel_directory.py: home_channel`) | N/A | `bindings` config maps channel account → agent at startup; not runtime auto-join (`docs/concepts/multi-agent.md:140`) | Per-plugin manifest declares channels at install time |
| **Ad-hoc channel creation by an agent** | None — only via human REST + must spawn agent sessions | `send_message` can target a new chat ID directly; channel creation is platform-side (Discord/Slack API) | Issue-thread is the unit; auto-created on first reply | `/agents` slash, `/focus` for **thread bindings** — sub-agent stays bound to a thread per session (`docs/tools/subagents.md:142`); thread is the ad-hoc unit | `ctx.streams.emit(channel, ...)` from a worker creates a stream channel on demand (`plugin-worker-manager.ts:576`) |
| **Discovery event stream** | None | Polls every 5 min + writes JSON (`channel_directory.py:60`) | TanStack Query realtime hooks (`packages/core/realtime/use-realtime-sync.ts`) | None — config-driven | Synthetic open/close events per stream (`plugin-worker-manager.ts:695`) |
| **Multi-channel session** | NO — `localsByID` 1:1 map (`peer.go:60`) | One agent serves N channels via Gateway routing | One workspace = many issues = many threads | One agent has many bindings (`docs/concepts/multi-agent.md:120`) | One worker holds N stream channels (`plugin-worker-manager.ts:393`) |
| **Channel = bound to task / workflow** | One-way validation only (`network/tasks.go:366`) | Sessions reference origin platform/chat | Issues are first-class threads | Thread bindings expire on idle (`docs/tools/subagents.md:148`) | Channels keyed by `(pluginId, channel, companyId)` |

**Takeaway**: Hermes is the closest reference. It has a JSON manifest, name → id resolution, periodic refresh, and per-platform "home channel" defaults — exactly the surface AGH is missing. OpenClaw's *thread bindings* are the closest analog of "ad-hoc channel for sub-agents" (a thread is created on first message, sub-agents bind to it, idle TTL evicts). Paperclip's `(pluginId, channel, companyId)` triple is the right multi-tenant key model for "scoped" channels.

---

## 6. Concrete proposals

Listed in dependency order. Every change is local to the `network` package, the `session` join surface, the API contract, and the CLI.

### 6.1 First-class `Channel` entity (in `internal/network/`)

Promote channels from "string keys in a map" to a typed entity:

```go
// internal/network/channel.go (NEW)
type ChannelManifest struct {
    Channel       string                  // existing grammar
    Purpose       string                  // promoted from store.NetworkChannelEntry
    Topics        []string                // free tags ("translation", "code-review")
    RolesNeeded   []string                // capability IDs the channel wants
    Owner         string                  // peer_id of creator (or "system")
    ParentChannel *string                 // for sub-team channels
    TaskID        *string                 // when spawned by a task
    Visibility    Visibility              // "public" | "workspace" | "private"
    ExpiresAt     *int64                  // TTL for ad-hoc channels
    CreatedAt     int64
    Ext           ExtensionMap
}
```

Persist alongside the existing `store.NetworkChannelEntry` (already at `internal/store/globaldb/`). Wire into `peers.ListChannels` so the runtime returns manifests, not just `{Channel, PeerCount}`.

### 6.2 New protocol kinds (additive, no break)

Add to `internal/network/envelope.go:14-24`:

- `KindChannelAnnounce` — broadcast by manifest owner on creation/update; body = `ChannelManifest`. Carries on subject `agh.network.v0.<channel>.broadcast` (already subscribed) AND on a NEW well-known meta subject `agh.network.v0.__meta__.channels` so peers in *other* channels learn about new ones.
- `KindChannelWhois` — request/response for `ChannelManifest` lookup by name or topic glob. Mirrors the existing peer `whois` (`router.go:600`).
- Optional: extend `WhoisBody` to accept `query_type: "channel"` instead of a new kind, to keep the kind set small.

This realizes the "discovery event stream" (#4.5) and the "channel whois" (#4.6) at the protocol layer.

### 6.3 Multi-channel sessions

`PeerRegistry.localsByID: map[sessionID]LocalPeer` (`peer.go:60`) becomes `map[sessionID][]LocalPeer` keyed by `(sessionID, channel)`. `JoinChannel`/`LeaveChannel` accept multiple channels per session. `Session` keeps a default "home" channel for backwards compat, but gains:

```go
// internal/session/interfaces.go
type NetworkPeerLifecycle interface {
    JoinChannel(ctx, NetworkPeerJoin) error
    JoinAdditionalChannel(ctx, sessionID, channel string, capabilities []NetworkPeerCapability) error  // NEW
    LeaveChannel(ctx, sessionID, channel string) error  // CHANGED: accept channel
    ListSessionChannels(ctx, sessionID) ([]string, error)  // NEW
}
```

This is the structural unblock for #4.3 (auto-join). Without it nothing else helps.

### 6.4 Agent-declared interests

Extend `AgentDef` (`internal/config/agent.go:17`) and the on-disk AGENT.md frontmatter:

```yaml
network:
  interests:
    - "translation:*"             # subject globs (matches Topics)
  required_capabilities:          # auto-join channels needing these
    - "code.review"
  default_channels: ["dev"]
  multi_home: true
```

On session activation (`manager_helpers.go:108`), after the legacy single-channel join, the new `autoJoinByInterests` step queries `Manager.ListChannelManifests(ctx, query)` and joins matching ones via `JoinAdditionalChannel`. Re-runs on every `KindChannelAnnounce` received (`manager.go:handleInboundMessage`). Fixes #4.3 without removing the static `Channel` field.

### 6.5 Agent-callable channel APIs

Both UDS and HTTP need new endpoints, mirrored in `internal/cli/network.go`:

```
GET    /api/network/channels                 (existing, extend response with manifests)
GET    /api/network/channels?topic=*&cap=*   (NEW filter query)
POST   /api/network/channels                 (existing — mark as "human bootstrap")
POST   /api/network/channels/:channel/announce       (NEW: broadcast/update manifest)
POST   /api/network/sessions/:session/channels       (NEW: join an existing channel)
DELETE /api/network/sessions/:session/channels/:channel (NEW: leave one channel)
GET    /api/network/sessions/:session/channels       (NEW: enumerate)
GET    /api/network/channels/stream                  (NEW: SSE — channel.announced/closed)
```

CLI counterparts:

```
agh network channels list [--topic foo] [--needs cap.id]
agh network channels create --channel wf-x --purpose "..." --topic refactor --task <id>
agh network channels join    --session <id> --channel wf-x
agh network channels leave   --session <id> --channel wf-x
agh network channels watch                  # streams new manifests
```

These give a session-running agent a full self-service surface (#4.1, #4.4, #4.5).

### 6.6 Task → channel coupling (close the loop with `tasks.go`)

When `task.network_channel` is set on `CreateTaskFromPeer` (`network/tasks.go:105`) and the channel does **not** yet exist, auto-create the manifest:

```
ChannelManifest{
  Channel:    spec.NetworkChannel,
  Purpose:    "Coordination for task " + spec.ID,
  TaskID:     &spec.ID,
  Owner:      peerCtx.peer.PeerID,
  Visibility: "workspace",
  ExpiresAt:  spec.Deadline + 24h,
}
```

Broadcast `KindChannelAnnounce`. This is the "task spawns sub-agent channel" pattern from the brief, executable today on top of the existing task ingress wiring.

### 6.7 Bundle declared channels become live announcements

When a bundle activation completes (`internal/api/core/bundles.go:138`), iterate `DeclaredChannels[]` (`contract/bundles.go:98`) and: (a) `WriteNetworkChannel` if absent, (b) emit `KindChannelAnnounce` on the meta subject. Closes the dead-inventory gap in #4.6. Zero new RFC needed.

### 6.8 Capability-indexed channel matching

Inside `peers` add an index `topicsByChannel` and `rolesNeededByChannel`. Add `MatchChannels(query ChannelQuery) []ChannelManifest` with `query.{Topic, RequiredCapability, Owner, ParentChannel}`. Use it from the new `autoJoinByInterests` (6.4) and from the filter REST endpoint (6.5).

### 6.9 Web/UI alignment

`web/src/systems/network/types.ts` already imports `OperationResponse<"listNetworkChannels">` — when the contract grows manifests, the UI is a pure typegen update. The "create ad-hoc channel" dialog (`network-create-channel-dialog.tsx`) becomes optional; manifest broadcasts make the panel real-time.

---

## 7. Open questions

1. **Channel namespace policy.** Should ad-hoc channels live under a reserved prefix (`wf-`, `task-`, `tmp-`) so the grammar in `rules/channel.go:5` can encode lifecycle? Or is the `Visibility` field on the manifest enough?
2. **Discovery scope — broadcast vs gossip.** Announcing every channel to every peer scales O(channels × peers). Should we cap at workspace boundary (current `network_channels.workspace_id` filter) or use NATS hierarchical wildcards (`agh.network.v0.__meta__.<workspace>.channels`)?
3. **Multi-home identity.** When session S joins channels A and B, does it use the same `peer_id` or a `peer_id@channel` derivation? `peer_id` collisions across channels are tolerated by `localsByChannel` today (`peer.go:179-184`), but a deterministic per-channel suffix would simplify auditing (`audit.go:236`).
4. **Auto-join authority.** Should *any* declared interest auto-join, or should an `Owner`/`Visibility` ACL gate it? Today there is no ACL anywhere in `network/` — this would be the first.
5. **Manifest mutability.** Can a non-owner peer correct/extend a manifest (e.g., add `RolesNeeded`)? If yes, conflict resolution rules (last-writer-wins by `ts`? owner-only writes? CRDT?). The agora council picked "owner-only with replaceable kind 30000+" (`docs/ideas/network/draft_5.md:154`); AGH has no equivalent.
6. **Migration of `session.Channel`.** Do we keep the single `Channel` field for backwards compat or replace with `Channels []string`? Per repo CLAUDE.md (greenfield, no legacy tolerance) — replace, but the change ripples through `meta.json`, `Info`, the prompt overlay (`prompt_overlay.go:18`), and every test in `internal/session/`.
7. **Channel-discovery cost on the prompt budget.** Hermes's directory is offline (file refresh). If AGH injects manifests into the agent's prompt context, we need a budgeting story — likely: only `RolesNeeded` and `Topics` in the steady-state context, full manifest only on `whois`.
8. **Where does `autoJoinByInterests` live — session or network?** Session knows the agent; network knows the manifests. A new tiny `internal/network/match.go` consumed by `session/manager_helpers.go` keeps the dependency direction clean (`session → network`, never the reverse — matches the import rule in CLAUDE.md).
