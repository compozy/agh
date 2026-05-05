# Inter-Agent Communication Patterns — Slice Analysis

> Slice: how AGH peers actually TALK to each other (the conversational protocol on top of NATS transport). Not "who runs", not "where it persists" — what verbs/fields they have to coordinate.

---

## 1. TL;DR

- Today AGH peers can do **broadcast (`say`)**, **directed request (`direct`)**, **lightweight ack (`receipt`)**, **lifecycle progress (`trace`)**, **presence (`greet`)**, **identity/capability lookup (`whois`)** and **transferable how-to (`capability`)**. That's seven kinds, all with strong correlation primitives (`interaction_id`, `reply_to`, `causation_id`, `trace_id`).
- The envelope is rich on **identity, threading, and lifecycle**, but **conversationally thin**: there is no first-class verb for *handoff*, *delegation acceptance*, *bid/offer*, *vote*, *cancel*, *pause*, or *escalate-to-human*. `intent` exists on `say`/`direct` bodies (`internal/network/envelope.go:251,260`) but it is a free-form string with no registry, no semantics, and no router-side meaning.
- **No mention/addressing scheme inside body text.** `to` selects exactly one peer; broadcast goes to everybody; there is no "@bob, @carol" subset, no role tag (`@role:reviewer`), no thread-anchored mention. Multica-style `mention://` parsing (`/.resources/multica/server/internal/util/mention.go:13`) is a useful precedent we lack.
- **Capability discovery is one-shot, not negotiated.** `whois` returns a static brief or rich catalog; there is no `offer`/`accept`/`decline`, no quorum/voting kind, and no contract-net-style auction even though all the hooks (`interaction_id`, `expires_at`) are in place.
- **Hand-off across sessions is invisible to the protocol.** A peer cannot transfer ownership of an `interaction_id` to another peer; the lifecycle state machine in `internal/network/lifecycle.go:217-235` actively rejects any third-party from speaking on an open interaction.
- **Status broadcasts ride on `trace`, but `trace` is locked to a directed interaction.** Peers can't publish "I'm idle / I'm busy / I claim this" to the channel — the only public status signal is the periodic `greet` heartbeat, which carries no workload state.
- For real autonomy we need a small set of new verbs (`handoff`, `claim`, `bid`, `vote`, `status`, `cancel`/`pause`, `escalate`) plus a **registered `intent` taxonomy** layered on top of today's `say`/`direct`, plus mention parsing in body text, plus an addressing extension that supports `to: ["bob", "carol"]` or `to: "@role:reviewer"`.

---

## 2. Current envelope and verb set (supported today)

### 2.1 Envelope fields — `internal/network/envelope.go:168-185`

| Field            | Type                | Purpose today                                               | Coordination affordance                                       | Gap for autonomy                                                                        |
| ---------------- | ------------------- | ----------------------------------------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `protocol`       | string              | `agh-network/v0` literal                                    | Version pin                                                   | None.                                                                                   |
| `id`             | string (`msg_*`)    | Unique message id                                           | Anti-replay key, target of `reply_to`/`causation_id`          | None.                                                                                   |
| `kind`           | enum (7 values)     | Wire-level routing decision                                 | Determines lifecycle treatment                                | Closed registry — no extension namespace, no `x-` prefix policy.                        |
| `channel`        | string              | NATS subject prefix                                         | Logical room                                                  | Single channel per envelope: cannot fan-out to multiple rooms in one shot.              |
| `from`           | peer-id             | Sender                                                      | Identity/auth                                                 | None.                                                                                   |
| `to`             | `*string` (single)  | One peer or null=broadcast                                  | Direct addressing                                             | **No multi-target.** No role/group address. No mention list parallel to `to`.           |
| `interaction_id` | `*string`           | Lifecycle key per `internal/network/lifecycle.go:23-32`     | Locks a 1:1 conversation between Initiator/Target             | **Two-party only**, ownership cannot move (`validateInteractionDirection:217-235`).     |
| `reply_to`       | `*string`           | Message-level threading                                     | Lets agents recover the parent message                        | No support for "list of replied-to" (parallel responses).                               |
| `trace_id`       | `*string`           | Correlated trace across many messages                       | Distributed tracing                                           | Unused by lifecycle FSM; purely informational.                                          |
| `causation_id`   | `*string`           | "This message is caused by …"                               | DAG of causality                                              | None (good).                                                                            |
| `ts`             | int64               | Unix epoch                                                  | Replay window via `replayDeadline` (`router.go:915`)          | None.                                                                                   |
| `expires_at`     | `*int64`            | TTL                                                         | Soft deadline; bounds replay window                           | Not enforced as response timeout; agents must DIY.                                      |
| `body`           | RawMessage          | Kind-specific payload                                       | Carries `intent`, text, artifacts                             | `intent` is unregistered free-form string.                                              |
| `proof`          | `*Proof` map        | Reserved for future signature                               | Forward-compat                                                | Not used today.                                                                         |
| `ext`            | `ExtensionMap`      | `agh.capabilities_brief`, `agh.include`, `agh.workflow…`    | Out-of-band metadata                                          | No registered keys for "mentions", "priority", "human-required", "deadline", "budget".  |

### 2.2 Verb registry — `internal/network/envelope.go:16-34`

| Kind         | Body fields (file:line)                                                   | Routing                                          | Lifecycle effect (`internal/network/lifecycle.go:261-289`)  |
| ------------ | ------------------------------------------------------------------------- | ------------------------------------------------ | ----------------------------------------------------------- |
| `greet`      | `GreetBody{PeerCard, Summary}` — `envelope.go:218`                        | Always broadcast on `agora.v1.<channel>.bcast`   | Refreshes presence; ignored by FSM.                         |
| `whois`      | `WhoisBody{Type,Query,PeerCard}` — `envelope.go:238`                      | Broadcast (request) or directed (response)       | Side-effect: capability discovery via `agh.capability_*` ext. |
| `say`        | `SayBody{Text,Intent,Artifacts}` — `envelope.go:248`                      | Broadcast only                                   | Never opens or advances an interaction.                     |
| `direct`     | `DirectBody{Text,Intent,Artifacts}` — `envelope.go:258`                   | Always directed; requires `interaction_id`       | Opens interaction, transitions `submitted`→`working`.       |
| `capability` | `CapabilityBody{CapabilityEnvelopePayload}` — `envelope.go:268-288`       | Directed or broadcast                            | When directed: same lifecycle as `direct`.                  |
| `receipt`    | `ReceiptBody{ForID,Status,ReasonCode,Detail}` — `envelope.go:291`         | Always directed                                  | `accepted`/`duplicate` = unchanged; `rejected`→`failed`; `canceled`→`canceled`. |
| `trace`      | `TraceBody{State,Message,Result,ArtifactRefs}` — `envelope.go:302-307`    | Always directed                                  | Drives state machine `submitted/working/needs_input/completed/failed/canceled`. |

### 2.3 What this verb set is good for

- **Pub/sub broadcast**: `say` to all (`router.go:413`).
- **Strict 1:1 RPC**: `direct` + `interaction_id` + `receipt` + `trace` is a complete request/response with progress.
- **Heartbeat presence**: `greet` re-published on `GreetInterval` via `Router.StartHeartbeat:201`.
- **Identity probe**: `whois` request → response (also doubles as capability-catalog fetch via `ext.agh.include=["capability_catalog"]`, `capability_catalog.go:43`).
- **Catalog publish**: `capability` broadcasts a sharable how-to artifact.

### 2.4 What it is missing

- **Multi-recipient direct.** `To` is `*string` not `[]string`. To send a question to three peers you must publish three envelopes, each with a fresh `interaction_id` — and you cannot then collect their answers under one umbrella.
- **In-text mentions.** No parser for `@peer-id` or `@role:tag` inside `SayBody.Text`. Multica solved this with `mention://member/<uuid>` markdown links (`mention.go:13`); we have no equivalent so the only way to single someone out in a broadcast is by typing their name and praying they read it.
- **Verbs of intent.** No `handoff`, `claim`, `bid`, `accept`, `decline`, `withdraw`, `vote`, `endorse`, `escalate`, `pause`/`resume`. Today every "verb" must be smuggled into `SayBody.Intent` (a free-form string with zero registry — `envelope.go:251`).
- **Public status broadcasts.** `trace` is locked to directed interactions (`shouldTrackSentLifecycle:867-878`). A peer cannot publish "status: idle" or "status: busy with task X" to the channel.
- **Hand-off semantics.** `validateInteractionDirection` (`lifecycle.go:217-235`) refuses any envelope where `from`/`to` does not match the original Initiator/Target pair, so transferring ownership requires closing the interaction and opening a new one — losing causal threading.
- **Capability *negotiation*.** `whois` answers "do you have X?" but the protocol has no `bid` ("I'll do X for cost C") or `claim` ("I'm taking X off the queue"). Contract-net coordination has to be hand-rolled in `SayBody.Intent`.
- **Voting / consensus.** No `vote` kind, no quorum primitive, no `endorse`/`veto` (Architect's `intent`-based consensus, `docs/ideas/anp/conversa.jsonl:13`, was anticipated but not implemented).
- **Rich `expires_at` handling.** `expires_at` is honored only as a replay-window upper bound (`router.go:917`); the protocol does not generate a "request-timed-out" `trace.failed` automatically.

---

## 3. Communication patterns inventory

| Pattern                              | Supported today?                                                                                                | Needed for autonomy?                                                  | Gap                                                                                                |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **Broadcast `say`**                  | Yes (`KindSay`, `router.go:412-414`)                                                                            | Yes — open call for help                                              | No mention-list to highlight specific recipients.                                                  |
| **Direct request/reply with corr-id** | Yes — `direct`+`interaction_id`+`receipt`+`trace` (lifecycle FSM in `lifecycle.go:142-158`)                   | Yes — the load-bearing primitive                                      | Pair-locked; cannot fan-out.                                                                       |
| **Mention (`@peer`)**                | **No** — neither in `to` nor parsed from text                                                                   | Yes — first thing humans expect from chat semantics                   | Add `Mentions []string` extension and/or markdown parser (Multica precedent).                      |
| **Multicast / subset broadcast**     | **No** — `to` is single                                                                                         | Yes — "all reviewers", "@team:codex"                                  | Either `to: []string`, or new `audience` field, or role-as-channel pattern.                        |
| **Hand-off (transfer ownership)**    | **No** — third-party rejected by `validateInteractionDirection`                                                 | Yes — escalation, domain switch, human-in-loop                        | New `KindHandoff` that mutates `Interaction.Target`; or an extension on `direct` with `succession`. |
| **Capability query (`whois`)**       | Yes — request/response with capability catalog projection (`capability_catalog.go:42-81`)                       | Yes — necessary for skill discovery                                   | One-shot, no follow-up "I'd like to use this capability now" handshake.                            |
| **Capability publish/teach**         | Yes — `KindCapability` carrying `CapabilityEnvelopePayload`                                                     | Yes                                                                   | Receiver has no normative way to *accept* / *acknowledge adoption* of a published recipe.          |
| **Bid / offer (contract net)**       | **No** — only `intent` string in body                                                                           | Yes — "who wants to grab task T?"                                     | Add `KindOffer` (or `KindBid`) tied to a `task_ref`; reuse `interaction_id` for the auction round. |
| **Claim / accept**                   | **No** explicit verb (`receipt(accepted)` is a protocol ack, not work acceptance — `lifecycle.go:307-311`)      | Yes — separate "I will do this" from "I received your bytes"          | New `KindAccept` (or `intent: "accept"` on `direct`) bound to an `offer.id`.                       |
| **Vote / consensus / approve**       | **No**                                                                                                          | Yes — multi-agent governance (review boards, plan approval)           | New `KindVote` with `motion_id`, `valence`, `weight`; quorum logic on initiator side.              |
| **Status broadcast (idle/busy)**     | Partial — only `greet` heartbeat carries presence; no workload field                                            | Yes — load-aware coordination, dispatch                               | Either new `KindStatus` or extend `GreetBody` with `availability` enum.                            |
| **Cancel / withdraw**                | Partial — `receipt(canceled)` flips terminal state but can only be sent within the existing 1:1 interaction     | Yes — initiator-side cancel, third-party `cancel-all`                 | New `KindCancel` (or `intent: "cancel"`) addressable by `interaction_id` *or* `trace_id`.          |
| **Pause / resume**                   | **No**                                                                                                          | Useful for human-in-loop                                              | New `KindPause` / `KindResume`, or use `trace.state="needs_input"` (already exists).               |
| **Escalate to human**                | **No** explicit verb                                                                                            | Yes — autonomy needs an exit door                                     | `intent: "escalate"` on `say`, with `ext.agh.escalation` carrying urgency/owner/channel hints.     |
| **Blackboard / shared state update** | **No** — there is no replaceable record kind (Architect call-out, agora-spec-v0.2.md §16)                       | Helpful for shared todo lists, plan boards                            | Replaceable kind keyed by `(channel, peer, d-tag)`; deferred to v0.3 in design docs.               |
| **Reaction / endorse / signal**      | **No**                                                                                                          | Lightweight social signal between agents                              | Tiny new kind `KindReact` or `say` with `intent: "react"` and `ext.agh.target_id`.                 |
| **Threading**                        | Yes — `reply_to`+`interaction_id`+`trace_id` triangle. But no "thread root" or breakout sub-thread.             | Yes                                                                   | Add `thread_id` (root anchor) distinct from `interaction_id` (1:1 contract).                       |

---

## 4. Reference comparisons

### 4.1 Hermes — `delegate_task` tool (`/.resources/hermes/tools/delegate_tool.py`)

- **Verbs as events on a delegation bus** (`DelegateEvent`, `delegate_tool.py:497-515`):

  | Event                       | Semantics                                  |
  | --------------------------- | ------------------------------------------ |
  | `delegate.task_spawned`     | Subagent created                           |
  | `delegate.task_progress`    | Streaming progress text                    |
  | `delegate.task_completed`   | Terminal success                           |
  | `delegate.task_failed`      | Terminal failure                           |
  | `delegate.task_thinking`    | Inner reasoning surface                    |
  | `delegate.tool_started`     | Worker invoked a tool                      |
  | `delegate.tool_completed`   | Worker finished a tool                     |

- **Roles** (`delegate_tool.py:308-323`): `leaf` vs `orchestrator`, controlling whether further delegation is allowed (depth limit `_get_max_spawn_depth`).
- **Pause/kill switch** (`set_spawn_paused`, `delegate_tool.py:154-161`) and `interrupt_subagent` give first-class operator verbs we have no equivalent for.

**Lessons for AGH**: model worker lifecycle as discrete kinds (`spawned`, `progress`, `completed`, `failed`, `thinking`, `tool_started`, `tool_completed`) so the channel transcript fully captures sub-agent execution; today AGH lifecycle has just six abstract states (`InteractionState`, `envelope.go:136-143`).

### 4.2 Multica — `Message` envelope and mention parser

- **Message envelope** (`/.resources/multica/server/pkg/protocol/messages.go:6-9`): radically simple `{ Type string, Payload json.RawMessage }`. Specific payload types per intent: `TaskDispatchPayload`, `TaskProgressPayload`, `TaskCompletedPayload`, `TaskMessagePayload`, `ChatMessagePayload`, `ChatDonePayload`, `ChatSessionReadPayload`. **One verb per intent** — no overloaded `intent` string.
- **In-text mentions** (`/.resources/multica/server/internal/util/mention.go:7-44`):
  - `Mention{Type: "member|agent|issue|all", ID}` parsed from markdown links of the form `[@Label](mention://type/id)`.
  - `IsMentionAll()` short-circuits broadcast routing.
  - Deduplicated.
- **Polymorphic addressing**: agents and humans share the same mention namespace.

**Lessons for AGH**: (a) in-band mention parser is essential for "say to channel but ping these peers"; (b) splitting payloads per concern is cleaner than overloading `intent`.

### 4.3 Openclaw — outbound message channels (`/.resources/openclaw/src/infra/outbound/message.ts`)

- Treats outbound messages as a **delivery contract**: `MessageSendParams` carries `replyToId` (line 73), `deliveryMode: "direct" | "gateway"` (line 239), and an explicit `OutboundMirror` for fan-out.
- **`replyToId` is first-class** at the API boundary, not just on the wire — every send specifies whether it is a fresh thread or a reply.
- Polling counterpart `MessagePollParams` (line 96) lets a sender wait for replies — synchronous request/reply is exposed at the SDK level.

**Lessons for AGH**: today our caller has to thread `ReplyTo`/`InteractionID`/`CausationID` themselves (see the verbose `replyGuidanceContext` in `delivery.go:842-989`); we should expose a higher-level `Reply()` helper that fills these.

### 4.4 Agora-spec-v0.2 (our own design draft)

- 5 core kinds: `greet`, `say`, `direct`, `recipe`, `whois` + 4 optional: `receipt`, `echo`, `revoke`, `trace` (`agora-spec-v0.2.md:329-549`).
- **`echo`** (`agora-spec-v0.2.md:469-493`) — reputational attestation about another agent. AGH lacks any reputation/feedback verb.
- **`tribute`** field on envelope (`agora-spec-v0.2.md:738-754`) — payment/value envelope. AGH has nothing equivalent.
- **`thread`** field separate from `reply_to` (`agora-spec-v0.2.md:213,387`) — distinct thread root vs message reply. AGH conflates these into `interaction_id`.

### 4.5 Draft 3 — "Kiosko" seven acts (`/docs/ideas/network/draft_3.md:121-156`)

The design vocabulary AGH never landed:

| Verb         | Semantic                                                                  |
| ------------ | ------------------------------------------------------------------------- |
| `enter`      | Join a square (≈ our greet)                                               |
| `hail`       | Broadcast to square (≈ our `say`)                                         |
| `interject`  | Reply to ongoing thread (≈ our `direct` with `reply_to`)                  |
| `whisper`    | Private 1-1 (≈ `direct`)                                                  |
| `cry`        | **Advertise a service with price/sample** — we have no analog             |
| `ask`        | **Request help with budget/deadline** — we have no analog                 |
| `strike`+`receipt` | **Two-sided handshake to commit** — we have only the `receipt` half |

### 4.6 ANP draft 5 — 3 protocolar verbs + kinds (`/docs/ideas/network/draft_5.md:140-156`)

- Reduces verbs to `publish | request | ack` and uses `kind` integers as schema tags (NIP-style). Includes:
  - `kind=4` response (with `req_id` in tags)
  - `kind=5` **NACK with enumerated reason** (`unknown_kind | rate_limited | schema_mismatch | sig_invalid | unauthorized | canonicalization_error | expired`).
- AGH's `ReasonCode` enum (`envelope.go:97-109`) is the closest cousin but is admission-time only; there is no "I refuse to serve this work" NACK.

### 4.7 Multi-agent-patterns-analysis (our own doc)

- §3.1 (`docs/ideas/orchestration/multi-agent-patterns-analysis.md:36-61`): explicitly recognizes that AGH lacks an inter-agent **handoff state-snapshot** primitive even though `ext` is the obvious place for `agh.handoff_version`/`agh.handoff_digest`/`agh.handoff_source`.

---

## 5. Concrete proposals

### 5.1 Envelope additions (`internal/network/envelope.go`)

```go
// New optional fields on Envelope
type Envelope struct {
    // … existing …
    ThreadID    *string  `json:"thread_id,omitempty"`    // root of a multi-message thread; distinct from interaction_id
    Audience    []string `json:"audience,omitempty"`     // multi-target directed send (peer-ids OR @role:foo OR @group:bar)
    Mentions    []string `json:"mentions,omitempty"`     // peer-ids/@roles called out inside body text
    Priority    *string  `json:"priority,omitempty"`     // "low" | "normal" | "high" | "urgent"
    DeadlineAt  *int64   `json:"deadline_at,omitempty"`  // soft response deadline (distinct from wire-level expires_at)
}
```

**Reasoning**:

- `ThreadID` lets sub-conversations branch from a parent (today `interaction_id` doubles as both, blocking the third-party-handoff use case).
- `Audience` finally allows `to=["bob","carol"]` or `to=["@role:reviewer"]`. Validation rule: if both `to` and `audience` are present, `audience` is the source of truth; `to` becomes a hint for the dominant addressee.
- `Mentions` is parser-populated (see §5.4); having it on the envelope avoids re-parsing in every consumer.
- `Priority`/`DeadlineAt` are widely-used coordination signals; they unblock backlog routing.

### 5.2 New `Kind` values (`internal/network/envelope.go:16-34`)

| New kind        | Body sketch                                                                                         | Lifecycle effect                                                                                                                  |
| --------------- | --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `KindStatus`    | `StatusBody{ State: "idle"/"busy"/"away", Detail string, Workload []string }`                       | Channel-broadcast presence enrichment; refreshed on `GreetInterval` or on transition.                                             |
| `KindOffer`     | `OfferBody{ TaskRef, CapabilityRef, Cost, Deadline, Sample }`                                       | Opens a *light* interaction: acceptable replies are `KindAccept` or `KindDecline`.                                                |
| `KindAccept`    | `AcceptBody{ ForOfferID, Terms }`                                                                   | Advances `submitted`→`working` like a `direct`, but also marks the offer's auction round as decided.                              |
| `KindDecline`   | `DeclineBody{ ForOfferID, Reason }`                                                                 | No state change; counts toward declination tally.                                                                                 |
| `KindHandoff`   | `HandoffBody{ Target, Reason, ContextDigest, ResumePrompt }`                                        | Mutates `Interaction.Target` atomically (see §5.6); allows third-party continuation under same `interaction_id`.                  |
| `KindCancel`    | `CancelBody{ ForInteractionID or ForTraceID, Reason }`                                              | Same as `receipt(canceled)` but addressable by trace umbrella, callable by initiator outside the rigid two-party check.           |
| `KindVote`      | `VoteBody{ MotionID, Valence: "yea"/"nay"/"abstain", Weight, Note }`                                | No FSM; initiator aggregates; quorum policy lives in caller.                                                                      |
| `KindReact`     | `ReactBody{ TargetID, Reaction string }` (lightweight social signal)                                | None.                                                                                                                             |
| `KindEscalate`  | `EscalateBody{ Reason, Urgency, ReassignTo (optional human channel) }`                              | Causes a `trace.state="needs_input"` to be emitted on parent interaction.                                                         |

**Verb economy**: this is +9 kinds. We can collapse `Accept`/`Decline` and `React` into a single `KindSignal` with a discriminator if churn is a concern. The `agora-spec-v0.2.md` precedent of having `echo` as a small dedicated verb argues for separate kinds.

### 5.3 Registered `intent` taxonomy (zero new wire fields)

If we want to avoid introducing kinds, register a closed set of `intent` values on `say`/`direct` bodies with router-side switch:

```go
const (
    IntentAsk         = "ask"          // open question to channel/peer
    IntentReply       = "reply"        // narrative answer (already common today)
    IntentClarify     = "clarify"      // request missing input
    IntentReport      = "report"       // status/progress narrative
    IntentDelegate    = "delegate"     // hand task to specific peer
    IntentHandoff     = "handoff"      // transfer ownership of an interaction
    IntentClaim       = "claim"        // "I'm taking this"
    IntentBid         = "bid"          // "I can do it for X"
    IntentVote        = "vote"         // structured vote in body
    IntentReact       = "react"        // emoji-style ack
    IntentEscalate    = "escalate"     // human attention required
    IntentCancel      = "cancel"       // initiator-side withdraw
)
```

The router's `previewForBody` (`delivery.go:1018-1049`) already varies preview by body type; it should also pick the rendered prefix by `intent` so the agent sees `<network-message kind="say" intent="ask">…` rather than the bare kind.

**Recommendation**: do BOTH — high-traffic verbs (`status`, `handoff`, `vote`, `cancel`) become first-class kinds; lower-frequency conversational verbs (`ask`, `report`, `clarify`, `react`) stay on `intent` with a registry.

### 5.4 In-text mention parser (`internal/network/mentions.go`, new)

Borrow the Multica regex (`/.resources/multica/server/internal/util/mention.go:13`):

```go
// Matches @peer-id, @role:reviewer, @group:codex, @all in body text
var MentionRe = regexp.MustCompile(`@([a-z0-9][a-z0-9._:-]{0,127}|all)`)

func ExtractMentions(text string) []string { /* dedup, normalize */ }
```

Wire it into `SayBody`/`DirectBody` decoding so `Envelope.Mentions` is populated automatically. Update `formatNetworkMessage` (`delivery.go:742-804`) to emit `<network-mentions>` block so the receiving agent's prompt clearly sees who was tagged.

### 5.5 Helper functions on `Router` (`internal/network/router.go`)

Today every reply path reconstructs threading by hand (see `Router.buildWhoisResponseEnvelope:642-697`). Introduce typed builders:

```go
func (r *Router) Reply(ctx, sessionID string, parent Envelope, body Body, opts ...ReplyOption) (SendResult, error)
func (r *Router) ReplyAll(ctx, sessionID string, parent Envelope, body Body, opts ...ReplyOption) (SendResult, error)
func (r *Router) Handoff(ctx, sessionID string, parent Envelope, target string, body HandoffBody) (SendResult, error)
func (r *Router) Status(ctx, sessionID string, body StatusBody) (SendResult, error)
func (r *Router) Vote(ctx, sessionID string, motionID, valence string) (SendResult, error)
```

Each helper:

1. Inherits `channel`, `interaction_id`, `thread_id`, `trace_id` from `parent`.
2. Sets `reply_to=parent.id`, `causation_id=parent.id`.
3. Validates that the new kind is allowed in the current `Interaction.State`.

This eliminates the verbose human-readable guidance (`delivery.go:806-989`) — the agent calls `agh network reply` and the daemon does the right thing.

### 5.6 Handoff lifecycle (`internal/network/lifecycle.go`)

Add `LifecycleActionHandoff` and amend `validateInteractionDirection` (`lifecycle.go:217-235`) so that when `env.Kind == KindHandoff`, the new `Target` becomes the interaction owner. Persist the handoff event (audit `RecordHandoff` in `audit.go`) and emit a synthetic `trace(state="working", message="handoff to <peer>")` so the audit trail and SSE notifier reflect the change.

### 5.7 Status as enriched greet (alternative to new kind)

Rather than a brand-new `KindStatus`, extend `GreetBody`:

```go
type GreetBody struct {
    PeerCard     PeerCard
    Summary      string
    Availability string   `json:"availability,omitempty"` // idle|busy|paused|offline
    Workload     []string `json:"workload,omitempty"`     // active interaction_ids
    Capacity     *int     `json:"capacity,omitempty"`     // max concurrent tasks
}
```

This costs no new kind and immediately turns the heartbeat into a coordination beacon.

### 5.8 Audit additions (`internal/network/audit.go`)

`AuditDirectionSent/Received/Rejected/Delivered` (`audit.go:18-26`) needs `AuditDirectionHandoff`, `AuditDirectionVoteCast`, `AuditDirectionOfferAccepted`, `AuditDirectionOfferDeclined` so that the cross-session activity timeline can render coordination history without re-deriving it from `intent` strings. The `NetworkMessageEntry` writer (`audit.go:256-308`) already projects `Intent` into the row — extend it to project `Mentions`, `ThreadID`, `Audience` for query-time filters.

---

## 6. Prompt-side counterpart

(Cross-link: identity/prompt slice should formalize this section.)

For agents to actually USE the new patterns, their system prompt needs:

1. **Verb cheat-sheet at the top.** Today the prompt only learns about kinds via the long protocol-guidance block in `delivery.go:27-39`. A compact table — "use `direct` for 1:1 work, `say` for open questions, `handoff` to pass ownership, `status` to advertise availability, `vote` to weigh in on a motion" — should be injected once per session, not appended to every inbound message.
2. **Mention awareness.** The system prompt must instruct the agent: "When you see `@your-peer-id` in `<network-mentions>`, treat it as a direct ping even if the envelope was a broadcast." This needs the mention parser from §5.4 to run before delivery.
3. **Status rhythm.** The prompt should commit the agent to publishing a `status` (or enriched `greet`) when it transitions idle⇄busy; today it never does.
4. **Hand-off etiquette.** Prompt should describe when a `KindHandoff` is appropriate (out-of-domain question, wrong workspace, escalation) and require the outgoing agent to include a `ContextDigest` so the new owner can resume.
5. **Auction etiquette.** When responding to a `KindOffer`, the agent must reply with `KindAccept` or `KindDecline` within `deadline_at`; on accept, transition the interaction; on decline, leave it open for others.
6. **Vote semantics.** Prompt should explain the motion-id correlation pattern: "to start a vote, send `say` with `intent: 'vote'` and `ext.agh.motion={id, options, quorum}`; to cast, send `KindVote` with `motion_id`."
7. **Cancellation.** Prompt should grant the agent permission to send `KindCancel` against its own outstanding interactions when the user revokes the original request, and to *honor* a `KindCancel` from the original initiator.
8. **Identity binding.** The agent must always know its own `peer_id` (already injected via `$AGH_PEER_ID`?). The identity slice should standardize the env vars.

Without these prompt-side changes, the new wire verbs are dead letters.

---

## 7. Open questions

1. **Addressing scheme for groups/roles.** Two options:
   - (a) **Reserve `@role:` and `@group:` namespaces** in `peer_id` validation and let the registry advertise a peer's roles in `PeerCard.Capabilities`. Pros: minimal changes; cons: peers cannot join multiple roles cleanly.
   - (b) **Introduce a separate `Audience` field** (§5.1) and let the router fan out at receive time. Pros: clean; cons: requires registry-side role index and changes admission audit (today `audit.PeerTo` is one string).
2. **Threading vs interaction.** Should `thread_id` (root) and `interaction_id` (1:1 contract) coexist or merge? The agora-spec-v0.2 draft kept them separate (`agora-spec-v0.2.md:213`); our current code merges them (`router.go:929`) and that merger blocks hand-off. Recommendation: keep `interaction_id` as the lifecycle key, add `thread_id` as a separate root anchor populated automatically when the first envelope of a chain is sent.
3. **Idempotency for new verbs.** `markSeen` (`router.go:764-778`) deduplicates by `envelope.id` only. For `KindOffer`/`KindAccept`/`KindVote`, the meaningful idempotency key is `(motion_id, voter)` or `(offer_id, bidder)` — should we (a) push that to application code, (b) add a per-kind idempotency key field, or (c) keep using `envelope.id` and let app code carry the semantic key in `ext`?
4. **Hand-off and outstanding receipts.** When ownership transfers mid-interaction, what happens to the in-flight `direct` whose `expires_at` has not fired? Two policies: (i) auto-emit a synthetic `receipt(canceled)` to the previous target, (ii) leave open and let the new target resume responsibility. The lifecycle FSM has no notion of "third-party take-over" today; we must pick.
5. **Status frequency vs noise.** If `status` is its own kind, when must it fire? On every transition? On heartbeat tick? Sub-second oscillations would flood the channel. Recommendation: `status` only on transitions OR on `GreetInterval` boundaries, never both within one interval.
6. **Per-channel verb capability.** Should a channel be allowed to declare *which* kinds it accepts? (e.g., a "stand-up" channel only accepts `say`, `status`, `react`; a "tasks" channel accepts everything.) Today every channel accepts every kind; constraining this would require a channel-policy primitive that the registry doesn't yet have.
7. **`ext` namespace governance.** We already use `agh.capability_*`, `agh.workflow_*`, `agh.handoff_*`. Need an explicit registry doc and a `ext.X-*` reservation for caller-defined keys to avoid collisions when third parties ship kinds through `intent` strings.
8. **Backward compatibility (greenfield rule).** Per `CLAUDE.md`, "Greenfield Alpha — Zero Legacy Tolerance" — we can renumber/replace freely. Recommendation: introduce all new kinds in one wire-format bump (`agh-network/v1`) rather than dribbling them in alongside `v0`.

---

## File references (load-bearing)

- `/Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go` — wire format, kinds, bodies
- `/Users/pedronauck/Dev/compozy/agh/internal/network/router.go` — send/receive, dedup, lifecycle dispatch
- `/Users/pedronauck/Dev/compozy/agh/internal/network/peer.go` — registry, capability catalog storage, whois matching
- `/Users/pedronauck/Dev/compozy/agh/internal/network/delivery.go` — agent-facing prompt rendering of network messages, reply guidance
- `/Users/pedronauck/Dev/compozy/agh/internal/network/lifecycle.go` — interaction state machine and two-party direction lock
- `/Users/pedronauck/Dev/compozy/agh/internal/network/capability_brief.go` and `capability_catalog.go` — capability projection / discovery
- `/Users/pedronauck/Dev/compozy/agh/internal/network/audit.go` — audit + timeline projection
- `/Users/pedronauck/Dev/compozy/agh/internal/session/manager_hooks.go` — `hookInputClassNetworkMessage` injection point
- `/Users/pedronauck/Dev/compozy/agh/internal/session/network_peer.go` — capability projection from session config to network
- `/Users/pedronauck/Dev/compozy/agh/internal/transcript/transcript.go` — replay schema (informs how new kinds must persist)
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/network/agora-spec-v0.2.md` — design draft (echo, tribute, thread, recipe)
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/network/draft_3.md` — Kiosko seven-acts vocabulary (cry/ask/strike)
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/network/draft_5.md` — ANP NACK reasons + 3-verb model
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/orchestration/multi-agent-patterns-analysis.md` — choreography vs orchestration mapping
- `/Users/pedronauck/Dev/compozy/agh/.resources/hermes/tools/delegate_tool.py` — `DelegateEvent` enum (lines 497-515) — worker lifecycle as first-class events
- `/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/pkg/protocol/messages.go` — payload-per-intent envelope
- `/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/internal/util/mention.go` — mention parser precedent
- `/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/infra/outbound/message.ts` — first-class `replyToId` + delivery mode
