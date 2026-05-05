# TechSpec: AGH Network Conversation Containers and Work Threads

## Executive Summary

This TechSpec redesigns AGH Network v0 around explicit conversation containers instead of a flat channel timeline plus peer-room projection. The accepted model keeps `channel` as the audience and discovery scope, introduces `public_thread` as the public N-to-N conversation primitive, introduces `direct_room` as the restricted one-to-one conversation primitive, and renames `interaction_id` to `work_id` for lifecycle-bearing work inside either conversation type.

No PRD exists for this workstream; this TechSpec is authored directly from the user-approved product direction, codebase research, the local agent-network knowledge base, and the round 1 Claude/Opus peer review. The primary trade-off is a larger hard cut now in exchange for a cleaner protocol, sharper UX semantics, deterministic storage invariants, and less permanent ambiguity in every public surface.

This is a greenfield alpha hard cut. The implementation must not ship compatibility aliases, dual fields, shadow migrations, or fallback readers for the old flat conversation model.

## MVP Boundary

MVP boundary for the upcoming task decomposition: tasks 01-08 must implement the complete hard cut for AGH Network conversation containers.

1. Task 01: Rewrite RFCs, glossary, and protocol examples for `surface`, `thread_id`, `direct_id`, and `work_id`.
2. Task 02: Update shared runtime envelope types, validators, direct-room ID derivation, work lifecycle enums, reason codes, and hard-cut Go symbols.
3. Task 03: Ship the numbered SQLite migration, final fresh-db schema, conversation side tables, `network_work`, same-transaction writes, store queries, and migration tests.
4. Task 04: Update router, delivery wrappers, lifecycle handling, hooks, observability, startup prompts, and bundled agent skills.
5. Task 05: Update HTTP, UDS, CLI, native tools, extension Host API, OpenAPI, and generated TypeScript in one change.
6. Task 06: Rebuild the `/network` web experience around channel threads and direct rooms.
7. Task 07: Rewrite site/runtime/operator docs, extension SDK docs, examples, and MCP/skill-facing guidance.
8. Task 08: Define the implementation verification scope: unit, integration, web, E2E, real-scenario QA coverage, and required verification gates.

`cy-create-tasks` must still append the required `$qa-report` and `$qa-execution` task pair after the implementation tasks. Because this feature has a web surface, the QA execution task must include Playwright/E2E coverage for the new thread and direct-room routes. Task 08 is the verification scope definition, not a replacement for the mandatory QA pair.

Post-MVP work is explicitly deferred: private group threads, more than two participants in restricted rooms, thread split/merge, cross-channel thread moves, unread-count sync, notification preferences, search ranking, retention policy controls, transcript export, federation policy, cryptographic private threads, and analytics dashboards beyond the metrics listed here.

Out of scope: compatibility aliases for `interaction_id`, accepting `kind:"direct"` after this change, migration bridges for old alpha conversation rows, public-thread embedded direct messages, work items that span multiple conversation containers, direct rooms with more than two peers, and any UI that renders non-implemented runtime controls.

Historical exception: archived `.compozy/tasks/_archived/*` artifacts may retain old `interaction_id` and `kind:"direct"` terminology because they are provenance records, not active product documentation. Active RFCs, docs, generated contracts, prompts, tests, and examples must hard cut.

## System Architecture

### Component Overview

| Component | Purpose | Boundary |
| --- | --- | --- |
| RFC / envelope model | Replace `kind:"direct"` with `surface:"thread"|"direct"` plus `thread_id` / `direct_id` and `work_id` | Protocol + shared runtime types |
| Network router | Route by channel and conversation container, preserve work lifecycle inside the container | `internal/network` only |
| Conversation registry | Materialize thread heads, direct rooms, participants, and active work for query/UI use | `internal/store/globaldb` |
| Message store | Persist `surface`, `thread_id`, `direct_id`, `work_id`, and visibility-scoped timeline rows | `internal/store` + `internal/store/globaldb` |
| Task domain | Remain the only durable task queue when network messages create task runs | `internal/task` only |
| CLI / HTTP / UDS | Expose thread/direct listing, detail, message, resolve, and send surfaces symmetrically | control plane |
| Native tools / MCP | Give agents a machine-readable path to inspect and speak in containers | `internal/tools`, `internal/daemon` |
| Extension Host API | Give extensions parity with HTTP/UDS for read/send/resolve operations | `internal/extension` |
| Web workspace | Replace flat channel timeline + peer-first model with channel threads and direct rooms | `web/` |
| Bundled prompts / skills | Teach agents to respond in the current conversation container and use `work_id` only for lifecycle-bearing work | startup prompt + bundled skills |
| Docs / OpenAPI | Rewrite protocol/runtime docs and regenerate codegen in the same PR | `packages/site`, `openapi`, `web/src/generated` |

### Data Flow

1. A local session sends a conversation-bearing message through CLI, HTTP, UDS, native tool, extension Host API, or bridge ingress.
2. The daemon validates `kind`, `surface`, conversation ID, `work_id`, `reply_to`, trust proof, and kind-specific body rules.
3. The router resolves the target container:
   - `surface:"thread"` -> `(channel, thread_id)`
   - `surface:"direct"` -> `(channel, direct_id)`
4. For direct rooms, control-plane helpers may resolve or create the room before send; the wire envelope still carries the explicit `direct_id`.
5. The store persists the normalized message row, binds or advances `work_id` when present, updates participants, and updates conversation summaries in the same SQLite transaction.
6. Delivery logic enqueues inbound prompts for local sessions and renders wrappers with conversation metadata.
7. HTTP/UDS/CLI/Web query persisted conversation views instead of reconstructing them from flat channel data.

## Architectural Boundaries

- `internal/network` owns protocol envelope validation, routing decisions, direct-room ID derivation, work lifecycle state transitions, and delivery wrapper metadata. It must not import `internal/api`, `internal/cli`, `web`, `packages/site`, or generated OpenAPI code.
- `internal/store` owns typed persistence DTOs, store-safe conversation references, and validation helpers for durable network records. It can be imported by `internal/network`.
- `internal/store/globaldb` owns SQLite schema, migrations, indexes, transaction helpers, and query implementations. It must not import `internal/network`; it implements interfaces consumed by `internal/network` using only `internal/store` and standard-library types.
- Do not create an `internal/conversation` package in this cut. `ConversationStore` is consumed in `internal/network` and implemented by `internal/store/globaldb`.
- `internal/task` remains the only package that owns durable task queue state and `task_runs` ownership transitions. Network code may observe or correlate task activity but must not duplicate `ClaimNextRun`, leases, or terminal state transitions.
- `internal/api/contract` owns DTO shape and OpenAPI generation inputs. Contract changes must co-ship with `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- `internal/api/core` owns HTTP handler validation and response assembly. It may call runtime/store services, but it must not bypass `internal/network` validation for send semantics.
- `internal/cli` and UDS clients are control-plane adapters. They must call the same contract/runtime primitives as HTTP and must not implement separate conversation resolution rules.
- `web/` owns operator UX state, routing, and rendering only. It must treat conversation summaries and messages as server truth and must not infer direct-room membership from flat channel rows.
- `packages/site` owns documentation prose and examples only. It must reflect the RFC and generated contract; it must not define independent protocol fields.
- `cmd/agh` and `internal/daemon` remain the composition root. New conversation services are constructed there or under the existing daemon composition boundary.

## Terminology

| Term | Definition |
| --- | --- |
| `channel` | Audience, discovery, and permission scope for public threads and direct rooms. |
| `public_thread` | Public N-to-N conversation container inside one channel. Wire value: `surface:"thread"`. |
| `direct_room` | Restricted one-to-one conversation container inside one channel. Wire value: `surface:"direct"`. |
| `message` | One AGH Network envelope/event persisted in a conversation or discovery stream. |
| `work_id` | Lifecycle-bearing work marker bound to exactly one conversation container. |
| `reply_to` | Fine-grained message-to-message edge. It never substitutes for `thread_id` or `direct_id`. |
| `trace_id` | Operational correlation across messages, handoffs, and task ingress. It never substitutes for `work_id`. |
| `causation_id` | Event/message that caused the current message. |
| `peer` | Participant identity. A peer is not a conversation container. |

## Implementation Design

### Wire Model

Conversation-bearing envelopes carry an explicit conversation surface:

```go
type Surface string

const (
	SurfaceThread Surface = "thread"
	SurfaceDirect Surface = "direct"
)
```

```go
type Envelope struct {
	Protocol    string          `json:"protocol"`
	ID          string          `json:"id"`
	Kind        Kind            `json:"kind"`
	Channel     string          `json:"channel"`
	Surface     *Surface        `json:"surface,omitempty"`
	ThreadID    *string         `json:"thread_id,omitempty"`
	DirectID    *string         `json:"direct_id,omitempty"`
	From        string          `json:"from"`
	To          *string         `json:"to,omitempty"`
	WorkID      *string         `json:"work_id,omitempty"`
	ReplyTo     *string         `json:"reply_to,omitempty"`
	TraceID     *string         `json:"trace_id,omitempty"`
	CausationID *string         `json:"causation_id,omitempty"`
	TS          int64           `json:"ts"`
	ExpiresAt   *int64          `json:"expires_at,omitempty"`
	Body        json.RawMessage `json:"body"`
	Proof       *Proof          `json:"proof,omitempty"`
	Ext         ExtensionMap    `json:"ext,omitempty"`
}
```

Normative kind set after the hard cut:

```go
const (
	KindGreet      Kind = "greet"
	KindWhois      Kind = "whois"
	KindSay        Kind = "say"
	KindCapability Kind = "capability"
	KindReceipt    Kind = "receipt"
	KindTrace      Kind = "trace"
)
```

`KindDirect` and `DirectBody` are deleted. A normal text message in a direct room is `kind:"say"` with `surface:"direct"`.

The marshal/unmarshal contract must preserve nullable field absence. For RFC 004/JCS verification, an absent `surface`, `thread_id`, `direct_id`, or `work_id` is not equivalent to a present zero value; receivers must verify the canonical bytes before injecting defaults.

### Kind Rules

| Kind | Conversation fields | Work field | Addressing |
| --- | --- | --- | --- |
| `greet` | MUST NOT carry `surface`, `thread_id`, or `direct_id` | MUST NOT carry `work_id` | channel discovery only |
| `whois` | MUST NOT carry `surface`, `thread_id`, or `direct_id` | MUST NOT carry `work_id` | channel/peer discovery only |
| `say` | MUST carry `surface` and matching container ID | MAY carry `work_id` when part of lifecycle-bearing work | `to` MAY target a visible peer without changing visibility |
| `capability` | MUST carry `surface` and matching container ID | MUST carry `work_id` when transferring work-linked capability | `to` MAY target a peer |
| `receipt` | MUST carry `surface` and matching container ID | MUST carry `work_id` | `to` SHOULD target the sender of the admitted message |
| `trace` | MUST carry `surface` and matching container ID | MUST carry `work_id` | `to` MAY target the work initiator |

### Core Go Types

`internal/network` owns validation and pure routing concepts:

```go
type ConversationRef struct {
	Channel  string
	Surface  Surface
	ThreadID string
	DirectID string
}

func (r ConversationRef) Validate() error
func (r ConversationRef) ContainerKey() string
func (r ConversationRef) IsThread() bool
func (r ConversationRef) IsDirect() bool
```

```go
type SendRequest struct {
	SessionID    string
	Conversation ConversationRef
	Kind        Kind
	To          *string
	WorkID      *string
	ReplyTo     *string
	TraceID     *string
	CausationID *string
	ExpiresAt   *int64
	ID          *string
	Body        json.RawMessage
	Ext         ExtensionMap
}

type SendResult struct {
	ID       string
	Subject  string
	Envelope Envelope
}
```

```go
type WorkState string

const (
	WorkStateSubmitted  WorkState = "submitted"
	WorkStateWorking    WorkState = "working"
	WorkStateNeedsInput WorkState = "needs_input"
	WorkStateCompleted  WorkState = "completed"
	WorkStateFailed     WorkState = "failed"
	WorkStateCanceled   WorkState = "canceled"
)

type Work struct {
	WorkID      string
	Ref         ConversationRef
	OpenedBy    string
	TargetPeer  string
	State       WorkState
	OpenedAt    time.Time
	UpdatedAt   time.Time
	TerminalAt  *time.Time
}
```

`internal/store` owns durable DTOs used by `globaldb` and consumed by `internal/network`:

```go
type NetworkConversationRef struct {
	Channel  string
	Surface  string
	ThreadID string
	DirectID string
}

type NetworkThreadSummary struct {
	Channel          string
	ThreadID         string
	RootMessageID    string
	Title            string
	OpenedByPeerID   string
	OpenedSessionID  string
	OpenedAt         time.Time
	LastActivityAt   time.Time
	MessageCount     int
	ParticipantCount int
	OpenWorkCount    int
	LastMessagePreview string
}

type NetworkDirectRoomSummary struct {
	Channel            string
	DirectID           string
	PeerA              string
	PeerB              string
	OpenedAt           time.Time
	LastActivityAt     time.Time
	MessageCount       int
	OpenWorkCount      int
	LastMessagePreview string
}

type NetworkDirectRoomEntry struct {
	Channel        string
	DirectID       string
	PeerA          string
	PeerB          string
	OpenedAt       time.Time
	LastActivityAt time.Time
}

type NetworkWorkEntry struct {
	WorkID           string
	Channel          string
	Surface          string
	ThreadID         string
	DirectID         string
	OpenedByPeerID   string
	OpenedSessionID  string
	TargetPeerID     string
	State            string
	OpenedAt         time.Time
	LastActivityAt   time.Time
	TerminalAt       *time.Time
}

type NetworkConversationMessage struct {
	MessageID   string
	SessionID   string
	Channel     string
	Surface     string
	ThreadID    string
	DirectID    string
	Direction   string
	PeerFrom    string
	PeerTo      string
	Kind        string
	WorkID      string
	ReplyTo     string
	TraceID     string
	CausationID string
	Intent      string
	Text        string
	PreviewText string
	Body        json.RawMessage
	Timestamp   time.Time
}

type NetworkConversationWriteResult struct {
	MessageID           string
	Duplicate           bool
	ConversationOpened  bool
	WorkOpened          bool
	WorkTransitioned    bool
	WorkState           string
	LastActivityAt      time.Time
}

type NetworkAuditEntry struct {
	ID        string
	SessionID string
	Direction string
	Kind      string
	Channel   string
	Surface   string
	ThreadID  string
	DirectID  string
	WorkID    string
	PeerFrom  string
	PeerTo    string
	MessageID string
	Reason    string
	Size      int
	Timestamp time.Time
}

type NetworkThreadQuery struct {
	Limit int
	After string
}

type NetworkDirectRoomQuery struct {
	PeerID string
	Limit  int
	After  string
}

type NetworkConversationMessageQuery struct {
	BeforeMessageID string
	AfterMessageID  string
	Kind            string
	WorkID          string
	Limit           int
}
```

`ConversationStore` is defined where consumed, in `internal/network`, and implemented by `internal/store/globaldb`:

```go
type ConversationStore interface {
	ResolveDirectRoom(ctx context.Context, entry store.NetworkDirectRoomEntry) (store.NetworkDirectRoomSummary, error)
	WriteConversationMessage(ctx context.Context, entry store.NetworkConversationMessage) (store.NetworkConversationWriteResult, error)
	ListThreads(ctx context.Context, channel string, query store.NetworkThreadQuery) ([]store.NetworkThreadSummary, error)
	GetThread(ctx context.Context, channel string, threadID string) (store.NetworkThreadSummary, error)
	ListDirectRooms(ctx context.Context, channel string, query store.NetworkDirectRoomQuery) ([]store.NetworkDirectRoomSummary, error)
	GetDirectRoom(ctx context.Context, channel string, directID string) (store.NetworkDirectRoomSummary, error)
	ListConversationMessages(ctx context.Context, ref store.NetworkConversationRef, query store.NetworkConversationMessageQuery) ([]store.NetworkConversationMessage, error)
	GetWork(ctx context.Context, workID string) (store.NetworkWorkEntry, error)
}
```

The interface intentionally uses `store.Network*` DTOs and `store.NetworkConversationRef` for all store-facing values. `internal/network.ConversationRef` remains the runtime validation type; before calling the store, network code converts it to `store.NetworkConversationRef`. This avoids an `internal/store/globaldb -> internal/network` dependency and preserves the boundary rule.

### Validators and Reason Codes

Add or rename these validators in `internal/network`:

```go
func ValidateSurface(surface Surface) error
func ValidateConversationID(id string, field string) error
func ValidateConversationRef(ref ConversationRef) error
func ValidateWorkID(id string) error
func ValidateWorkState(state WorkState) error
func ValidateWorkTransition(from WorkState, to WorkState) error
func ValidateEnvelopeConversation(envelope Envelope) error
func ValidateDirectRoomPeers(peerA string, peerB string) error
```

Validation rules must include symmetric container-field rejection:

- Any envelope with `thread_id` or `direct_id` set MUST also set `surface`.
- Any envelope with `surface:"thread"` MUST set `thread_id` and MUST NOT set `direct_id`.
- Any envelope with `surface:"direct"` MUST set `direct_id` and MUST NOT set `thread_id`.
- Any `greet` or `whois` envelope MUST omit `surface`, `thread_id`, `direct_id`, and `work_id`.
- Unknown or mismatched `surface` / container ID combinations fail with `ReasonCodeInvalidSurface` or `ReasonCodeConversationNotFound` before routing.

Reason code registry after the hard cut:

```go
const (
	ReasonCodeMalformed              ReasonCode = "malformed"
	ReasonCodeExpired                ReasonCode = "expired"
	ReasonCodeDuplicate              ReasonCode = "duplicate"
	ReasonCodeUnsupportedKind        ReasonCode = "unsupported_kind"
	ReasonCodeUnsupportedProfile     ReasonCode = "unsupported_profile"
	ReasonCodeVerificationFailed     ReasonCode = "verification_failed"
	ReasonCodeNotTarget              ReasonCode = "not_target"
	ReasonCodeNotFound               ReasonCode = "not_found"
	ReasonCodeBusy                   ReasonCode = "busy"
	ReasonCodeInternal               ReasonCode = "internal"
	ReasonCodeInvalidSurface         ReasonCode = "invalid_surface"
	ReasonCodeConversationNotFound   ReasonCode = "conversation_not_found"
	ReasonCodeWorkClosed             ReasonCode = "work_closed"
	ReasonCodeWorkContainerMismatch  ReasonCode = "work_container_mismatch"
	ReasonCodeLegacyFieldRejected    ReasonCode = "legacy_field_rejected"
)
```

Delete `ReasonCodeInteractionClosed`; use `ReasonCodeWorkClosed`.

### Old-to-New Symbol Rename List

| Old symbol / field | New symbol / field | Required action |
| --- | --- | --- |
| `Envelope.InteractionID` | `Envelope.WorkID` | hard rename JSON field to `work_id`; reject `interaction_id` on ingress |
| `NetworkSendRequest.InteractionID` | `NetworkSendRequest.WorkID` | contract + CLI + client hard rename |
| `NetworkSendPayload.InteractionID` | `NetworkSendPayload.WorkID` | contract + generated TS hard rename |
| `NetworkMessageEntry.InteractionID` | `NetworkConversationMessage.WorkID` | replace flat DTO with conversation DTO |
| `NetworkAuditEntry` without container fields | `NetworkAuditEntry.Surface/ThreadID/DirectID/WorkID` | add typed audit columns and filters |
| `KindDirect` | none | delete |
| `DirectBody` | `SayBody` on `surface:"direct"` | delete |
| `InteractionState` | `WorkState` | hard rename enum type |
| `StateSubmitted` | `WorkStateSubmitted` | hard rename constants |
| `StateWorking` | `WorkStateWorking` | hard rename constants |
| `StateNeedsInput` | `WorkStateNeedsInput` | hard rename constants |
| `StateCompleted` | `WorkStateCompleted` | hard rename constants |
| `StateFailed` | `WorkStateFailed` | hard rename constants |
| `StateCanceled` | `WorkStateCanceled` | hard rename constants |
| `Interaction` | `Work` | hard rename lifecycle struct |
| `Router.interactions` | `Router.works` or durable store-backed work lookup | remove in-memory-only interaction map as authority |
| `OpenInteraction` | `OpenWork` | hard rename |
| `ApplyInteractionEnvelope` | `ApplyWorkEnvelope` | hard rename |
| `ErrInteractionNotFound` | `ErrWorkNotFound` | hard rename |
| `ErrInteractionClosed` | `ErrWorkClosed` | hard rename |
| `LifecycleActionRejectDirect` | `LifecycleActionRejectWork` | hard rename or delete if obsolete |
| CLI `--interaction-id` | CLI `--work` | no alias |
| JSON `interaction_id` | JSON `work_id` | no alias |

### Direct Room Resolution Algorithm

Direct room resolution is deterministic and race-safe. It does not depend on message order.

Inputs:

```go
type DirectRoomResolveRequest struct {
	SessionID string `json:"session_id"`
	PeerID    string `json:"peer_id"`
}
```

Algorithm in `internal/network`:

```go
func DirectRoomIdentity(channel string, localPeer string, remotePeer string) (directID string, peerA string, peerB string, err error) {
	// 1. Trim and validate channel with the existing channel grammar.
	// 2. Trim and validate both peer IDs with ValidatePeerID.
	// 3. Reject identical peers.
	// 4. Sort peers lexicographically into peerA, peerB.
	// 5. Hash domain-separated identity bytes:
	//    "agh-network/direct-room/v1\x00" + channel + "\x00" + peerA + "\x00" + peerB
	// 6. Return "direct_" + first 32 lowercase hex chars of SHA-256.
}
```

Store semantics in `internal/store/globaldb`:

1. Open a network immediate transaction using the same pattern as `withTaskImmediateTransaction`.
2. `INSERT OR IGNORE` `(channel, direct_id, peer_a, peer_b, opened_at, last_activity_at, message_count, open_work_count)` into `network_direct_rooms`.
3. `SELECT` the row by `(channel, peer_a, peer_b)` inside the same transaction.
4. If the selected `direct_id` does not equal the deterministic ID, return `ErrDirectRoomCollision`.
5. If `INSERT OR IGNORE` was ignored because `(channel, direct_id)` already exists for a different peer pair and the `(channel, peer_a, peer_b)` select returns zero rows, return `ErrDirectRoomCollision`.
6. Do not update `last_activity_at` on resolve-only calls; only message writes move activity.

DDL enforces:

- `PRIMARY KEY (channel, direct_id)`
- `UNIQUE (channel, peer_a, peer_b)`
- `CHECK (peer_a < peer_b)`

Concurrent calls for the same `(channel, peer_a, peer_b)` return the same row. Concurrent calls for different pairs never share a row. Expanding `direct_room` beyond two peers is a future schema migration trigger because `peer_a` / `peer_b` and the unique pair constraint are intentionally two-party only.

Valid direct-room-opening writes are restricted to messages whose `direct_id` equals `DirectRoomIdentity(channel, peer_from, peer_to)` and whose peer pair is valid. A missing direct room with a non-deterministic `direct_id`, missing target peer, or same-peer pair is rejected instead of creating a room.

### Work Lifecycle and `task_runs`

`work_id` is a network-level lifecycle marker in v0. It is not a task queue, not a claim token, not a task-run ID, and not a replacement for `task_runs`.

Rules:

- `network_work` binds `work_id` to exactly one conversation container and tracks only network lifecycle state.
- `task_runs` remains the single durable work queue and the only owner of task claim, lease, heartbeat, complete, fail, release, and cancel transitions.
- A network message may cause task ingress through existing task-domain APIs. That task run is correlated by `trace_id`, `causation_id`, and canonical task metadata, not by reusing `work_id` as queue ownership.
- Network code must never call `ClaimNextRun`, mutate task-run claim tokens, or set task-run terminal state directly.
- If a work conversation needs an executable task, the implementation calls the existing task service ingress path and records the resulting task/run correlation in `task_runs.metadata_json.network_work_id` plus `network_message_id`, `network_channel`, `network_surface`, and either `network_thread_id` or `network_direct_id`.
- `task_runs.metadata_json.network_work_id` is an observability/correlation key only. It is never used for task claiming, lease ownership, task scheduling, or queue selection.
- Raw `claim_token` remains forbidden in network envelopes, message bodies, metadata, logs, prompts, HTTP/UDS responses, CLI output, and web UI.

### Same-Transaction Write Strategy

`WriteConversationMessage` replaces `WriteNetworkMessage` as the authoritative write path.

Required transaction shape:

```go
func (g *GlobalDB) withNetworkImmediateTransaction(
	ctx context.Context,
	action string,
	run func(exec networkSQLExecutor) error,
) (err error) {
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open connection for %s: %w", action, err)
	}
	defer func() { _ = conn.Close() }()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin immediate %s transaction: %w", action, err)
	}

	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, action))
		}
	}()

	if err := run(conn); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit %s transaction: %w", action, err)
	}
	finished = true
	return nil
}
```

Within one `BEGIN IMMEDIATE` transaction:

1. Validate the entry and normalize timestamps.
2. Insert `network_timeline_log` with `ON CONFLICT(message_id) DO NOTHING`.
3. If the insert was a duplicate, return `Duplicate=true` and do not update summaries.
4. Verify the target container exists or create it when this is a valid thread-opening or direct-room-opening write.
5. If `work_id` is present, select existing `network_work` row by `work_id`.
6. If no `network_work` row exists, insert one bound to the current container.
7. If a `network_work` row exists with a different `(channel, surface, thread_id, direct_id)`, reject with `ReasonCodeWorkContainerMismatch`.
8. If a `network_work` row is terminal, reject the non-duplicate write with `ReasonCodeWorkClosed`.
9. Update `network_work.state`, `last_activity_at`, and `terminal_at` when the message advances lifecycle state.
10. Insert or update public-thread participants for `surface:"thread"`.
11. Update `network_threads` or `network_direct_rooms` summary counters, `last_activity_at`, `last_message_preview`, and `open_work_count`.
12. Write the matching `network_audit_log` row with the same container fields.
13. Commit.

The implementation must not write timeline rows and summary rows in separate transactions. If summary update fails, the message insert rolls back.

Thread-opening rule: the first non-duplicate valid conversation-bearing message with a new `thread_id` opens the public thread, and that message becomes `root_message_id`. The opening message must pass all kind-specific validation before the thread row is created. Direct-room-opening writes must satisfy the deterministic direct-room rule above. Duplicate replay is idempotent only when `message_id` is identical and is caught by step 3 before any lifecycle transition; there is no broader post-terminal duplicate carve-out.

## Data Models

### Field Rationale

| Field | Shape | Purpose | Rationale |
| --- | --- | --- | --- |
| `surface` | nullable enum string: `thread` or `direct` | Selects the conversation container class for conversation-bearing messages | Nullable only for discovery kinds; separates where a message lives from what event happened |
| `thread_id` | nullable string | Identifies a public N-to-N conversation inside a channel | Nullable because direct-room messages use `direct_id`; indexed for thread timelines |
| `direct_id` | nullable string | Identifies a restricted one-to-one room inside a channel | Nullable because public-thread messages use `thread_id`; stable ID prevents fragmented bilateral history |
| `work_id` | nullable string | Identifies lifecycle-bearing work inside one conversation container | Replaces `interaction_id` and prevents lifecycle state from doubling as UX thread state |
| `reply_to` | nullable message ID | Points to the specific message being answered | Keeps fine-grained message causality separate from conversation membership |
| `trace_id` | nullable string | Correlates distributed runtime work and handoffs | Supports observability across related containers without reusing `work_id` |
| `causation_id` | nullable message ID or event ID | Records the event that caused this message | Supports auditability for summarize-back and handoff flows |
| `root_message_id` | non-null message ID on `network_threads` | Anchors the public thread head shown in the channel timeline | Avoids reconstructing thread heads from every message query |
| `participant_count` | integer on `network_threads` | Fast summary for web/channel thread lists | Matchable summary state belongs in a typed column, not JSON metadata |
| `peer_a`, `peer_b` | non-null peer IDs on `network_direct_rooms` | Defines direct-room membership | Enforces restricted visibility and stable one-to-one uniqueness |
| `open_work_count` | integer on thread/direct summary tables | Indicates active lifecycle work in a conversation | Supports operator triage without scanning all lifecycle events |
| `last_message_preview` | text on summary tables | Shows a safe preview in lists | Prevents web from querying full messages for list rendering |

Conversation ID grammar:

- `thread_id` MUST match `^thread_[a-z0-9][a-z0-9_-]{2,95}$`; it is scoped by `channel`, not globally unique.
- `direct_id` MUST match `^direct_[a-f0-9]{32}$`; it is scoped by `channel` and produced by `DirectRoomIdentity`.
- `work_id` SHOULD use the existing AGH ID helper with a `work_` prefix; validation rejects empty strings, whitespace, path separators, control characters, and IDs longer than 128 bytes.

### SQLite Migration

Current research found `globalSchemaMigrations` ending at version 15. Unless a new migration lands first, this change appends:

```go
{
	Version:  16,
	Name:     "rebuild_network_conversation_containers",
	Up:       migrateNetworkConversationContainers,
	Checksum: "2026-05-04-rebuild-network-conversation-containers",
}
```

Implementation tasks must inspect `globalSchemaMigrations` at task start. If version 16 is no longer free, use the next sequential version with no gaps and update both `Version` and `Checksum` to the actual migration identity. Tests must assert the chosen migration version recorded in the migration table, not a stale value copied from this planning snapshot.

Fresh DB schema in `globalSchemaStatements` and the selected migration version must converge to the same shape. `EnsureSchema` remains the fresh-create path only; it must not become an old-shape compatibility reconciler.

#### Fresh Schema DDL

AGH store connections already enable SQLite foreign keys through the shared DSN `_pragma=foreign_keys(ON)` and `configureSQLite` / `restoreForeignKeys` paths. This migration must preserve that invariant: if it temporarily disables foreign keys for a rebuild, it must restore `PRAGMA foreign_keys = ON` before returning, and tests must assert `PRAGMA foreign_keys = 1` plus one cascade/restrict behavior for the new network tables.

```sql
CREATE TABLE IF NOT EXISTS network_timeline_log (
	message_id   TEXT PRIMARY KEY,
	session_id   TEXT,
	channel      TEXT NOT NULL,
	surface      TEXT CHECK (surface IN ('thread', 'direct') OR surface IS NULL),
	thread_id    TEXT,
	direct_id    TEXT,
	direction    TEXT NOT NULL,
	peer_from    TEXT NOT NULL,
	peer_to      TEXT,
	kind         TEXT NOT NULL,
	work_id      TEXT,
	reply_to     TEXT,
	trace_id     TEXT,
	causation_id TEXT,
	intent       TEXT,
	text         TEXT,
	preview_text TEXT NOT NULL DEFAULT '',
	body_json    TEXT NOT NULL,
	timestamp    TEXT NOT NULL,
	CHECK (
		(surface IS NULL AND thread_id IS NULL AND direct_id IS NULL AND work_id IS NULL AND kind IN ('greet', 'whois'))
		OR (surface = 'thread' AND thread_id IS NOT NULL AND direct_id IS NULL)
		OR (surface = 'direct' AND direct_id IS NOT NULL AND thread_id IS NULL)
	),
	CHECK (kind IN ('greet', 'whois', 'say', 'capability', 'receipt', 'trace'))
);

CREATE INDEX IF NOT EXISTS idx_net_timeline_thread_ts
	ON network_timeline_log(channel, thread_id, timestamp, message_id)
	WHERE surface = 'thread';

CREATE INDEX IF NOT EXISTS idx_net_timeline_direct_ts
	ON network_timeline_log(channel, direct_id, timestamp, message_id)
	WHERE surface = 'direct';

CREATE INDEX IF NOT EXISTS idx_net_timeline_work_ts
	ON network_timeline_log(work_id, timestamp, message_id)
	WHERE work_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_net_timeline_presence_ts
	ON network_timeline_log(channel, timestamp, message_id)
	WHERE surface IS NULL;

CREATE INDEX IF NOT EXISTS idx_net_timeline_kind_ts
	ON network_timeline_log(kind, timestamp, message_id);
```

```sql
CREATE TABLE IF NOT EXISTS network_threads (
	channel              TEXT NOT NULL,
	thread_id            TEXT NOT NULL,
	root_message_id      TEXT NOT NULL,
	title                TEXT NOT NULL DEFAULT '',
	opened_by_peer_id    TEXT NOT NULL DEFAULT '',
	opened_session_id    TEXT NOT NULL DEFAULT '',
	opened_at            TEXT NOT NULL,
	last_activity_at     TEXT NOT NULL,
	message_count        INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
	participant_count    INTEGER NOT NULL DEFAULT 0 CHECK (participant_count >= 0),
	open_work_count      INTEGER NOT NULL DEFAULT 0 CHECK (open_work_count >= 0),
	last_message_preview TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (channel, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_network_threads_activity
	ON network_threads(channel, last_activity_at DESC, thread_id);
```

```sql
CREATE TABLE IF NOT EXISTS network_thread_participants (
	channel         TEXT NOT NULL,
	thread_id       TEXT NOT NULL,
	peer_id         TEXT NOT NULL,
	first_message_id TEXT NOT NULL,
	first_seen_at    TEXT NOT NULL,
	last_seen_at     TEXT NOT NULL,
	PRIMARY KEY (channel, thread_id, peer_id),
	FOREIGN KEY (channel, thread_id)
		REFERENCES network_threads(channel, thread_id)
		ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_network_thread_participants_peer
	ON network_thread_participants(peer_id, last_seen_at DESC);
```

```sql
CREATE TABLE IF NOT EXISTS network_direct_rooms (
	channel              TEXT NOT NULL,
	direct_id            TEXT NOT NULL,
	peer_a               TEXT NOT NULL,
	peer_b               TEXT NOT NULL,
	opened_at            TEXT NOT NULL,
	last_activity_at     TEXT NOT NULL,
	message_count        INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
	open_work_count      INTEGER NOT NULL DEFAULT 0 CHECK (open_work_count >= 0),
	last_message_preview TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (channel, direct_id),
	UNIQUE (channel, peer_a, peer_b),
	CHECK (peer_a < peer_b)
);

CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_activity
	ON network_direct_rooms(channel, last_activity_at DESC, direct_id);

CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_peer_a
	ON network_direct_rooms(channel, peer_a, last_activity_at DESC);

CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_peer_b
	ON network_direct_rooms(channel, peer_b, last_activity_at DESC);
```

```sql
CREATE TABLE IF NOT EXISTS network_work (
	work_id            TEXT PRIMARY KEY,
	channel            TEXT NOT NULL,
	surface            TEXT NOT NULL CHECK (surface IN ('thread', 'direct')),
	thread_id          TEXT,
	direct_id          TEXT,
	opened_by_peer_id  TEXT NOT NULL,
	opened_session_id  TEXT NOT NULL DEFAULT '',
	target_peer_id     TEXT NOT NULL DEFAULT '',
	state              TEXT NOT NULL CHECK (state IN ('submitted', 'working', 'needs_input', 'completed', 'failed', 'canceled')),
	opened_at          TEXT NOT NULL,
	last_activity_at   TEXT NOT NULL,
	terminal_at        TEXT,
	CHECK (
		(surface = 'thread' AND thread_id IS NOT NULL AND direct_id IS NULL)
		OR (surface = 'direct' AND direct_id IS NOT NULL AND thread_id IS NULL)
	),
	FOREIGN KEY (channel, thread_id)
		REFERENCES network_threads(channel, thread_id)
		ON DELETE RESTRICT,
	FOREIGN KEY (channel, direct_id)
		REFERENCES network_direct_rooms(channel, direct_id)
		ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_network_work_conversation
	ON network_work(channel, surface, thread_id, direct_id, last_activity_at DESC);

CREATE INDEX IF NOT EXISTS idx_network_work_state
	ON network_work(state, last_activity_at DESC);
```

```sql
CREATE TABLE IF NOT EXISTS network_audit_log (
	id         TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	direction  TEXT NOT NULL,
	kind       TEXT NOT NULL,
	channel    TEXT NOT NULL,
	surface    TEXT,
	thread_id  TEXT,
	direct_id  TEXT,
	work_id    TEXT,
	peer_from  TEXT NOT NULL,
	peer_to    TEXT,
	message_id TEXT NOT NULL,
	reason     TEXT,
	size       INTEGER NOT NULL,
	timestamp  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_net_audit_ts ON network_audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_net_audit_session ON network_audit_log(session_id);
CREATE INDEX IF NOT EXISTS idx_net_audit_conversation ON network_audit_log(channel, surface, thread_id, direct_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_net_audit_work ON network_audit_log(work_id, timestamp) WHERE work_id IS NOT NULL;
```

#### Migration Shape

The selected migration runs inside the existing `store.RunMigrations` transaction. It must:

1. Create `network_timeline_log_new` with the final schema.
2. Copy only `greet` and `whois` rows from the old `network_timeline_log`, with `surface`, `thread_id`, `direct_id`, and `work_id` set to `NULL`.
3. Drop old conversation rows (`say`, `direct`, `capability`, `receipt`, `trace`) because old alpha rows do not have a valid conversation container and must not be guessed into threads or direct rooms.
4. Drop the old `network_timeline_log`.
5. Rename `network_timeline_log_new` to `network_timeline_log`.
6. Create `network_threads`, `network_thread_participants`, `network_direct_rooms`, and `network_work`.
7. Rebuild `network_audit_log` to include `surface`, `thread_id`, `direct_id`, and `work_id`; old audit rows keep those fields `NULL`.
8. Recreate final indexes.
9. Restore and assert `PRAGMA foreign_keys = ON` if the migration disabled it during rebuild.
10. Update tests that assert table columns, indexes, foreign keys, and restrict/cascade behavior.

The migration must not:

- rename `interaction_id` to `work_id` for old rows that cannot prove container membership
- synthesize `thread_id` or `direct_id` from `reply_to`, `to`, peer rooms, or message kind
- preserve `kind:"direct"` in active rows
- leave old indexes that imply flat channel timelines as the primary query model

Delete targets in active code:

- old `network_timeline_log.interaction_id` column
- old flat `idx_net_timeline_channel_ts` query path for conversation timelines
- old peer-room message query as primary conversation path
- old `WriteNetworkMessage` implementation that writes without conversation summaries

### Side-Table vs JSON Decisions

- `network_threads` is a side table because thread heads, last activity, message counts, open work counts, and participant counts are matchable state used by API queries and web navigation.
- `network_thread_participants` is a side table because participant counts must be exact, incrementally maintainable, and queryable by peer without scanning all messages.
- `network_direct_rooms` is a side table because peer membership, stable room resolution, activity ordering, and open work counts require uniqueness constraints and indexes.
- `network_work` is a side table because `work_id` container binding and lifecycle state are safety invariants, not opaque annotations.
- `surface`, `thread_id`, `direct_id`, and `work_id` are typed columns on timeline/audit rows because routing, filtering, lifecycle validation, and observability depend on them.
- JSON metadata is allowed only for opaque provider/runtime annotations that are never used for routing, authorization, filtering, lifecycle continuation, summary counts, or UI primary navigation.
- The implementation must not add conversation ownership, room membership, lifecycle state, visibility flags, or work ownership to JSON blobs when a typed column or side table can express the invariant.

### RFC 004 Trust Integration

RFC 004 currently says v1 uses the v0 wire format and signs the full envelope excluding only `proof.sig`. This change updates RFC 004 so the signed content explicitly includes the new conversation fields.

For `agh-network/v1`, the JCS signed envelope includes:

- `protocol`
- `id`
- `kind`
- `channel`
- `surface` when present
- `thread_id` when present
- `direct_id` when present
- `from`
- `to`
- `work_id` when present
- `reply_to`
- `trace_id`
- `causation_id`
- `ts`
- `expires_at`
- canonical `body`
- canonical `ext`
- `proof.profile`, `proof.alg`, `proof.key_id`, and `proof.pubkey`

Only `proof.sig` is excluded from canonicalization. Field absence is signed by omission; a receiver must not insert default values before verifying. A verified-format `from` without a valid proof remains `rejected`, not `unverified`.

Tests must round-trip envelopes through JSON encoding and JCS canonicalization to prove absent nullable fields and present zero-valued nullable fields produce different signed canonical bytes. This specifically covers `surface`, `thread_id`, `direct_id`, and `work_id`.

Processing order changes from "route by kind + channel + to" to:

1. validate required envelope fields
2. reject legacy `interaction_id` and `kind:"direct"`
3. evaluate expiration
4. evaluate trust state
5. route by `channel`, `surface`, matching container ID, and `to`
6. apply work lifecycle if `work_id` is present
7. apply extension handling

NATS request/reply examples must replace `interaction_id` with `work_id`; NATS reply subjects still do not replace core envelope correlation.

## API, CLI, Tooling, and Extension Surfaces

### HTTP and UDS Route Map

Retained:

- `GET /api/network/status`
- `GET /api/network/peers`
- `GET /api/network/peers/{peer_id}`
- `GET /api/network/channels`
- `POST /api/network/channels`
- `GET /api/network/channels/{channel}`
- `POST /api/network/send`
- `GET /api/network/inbox`

Deleted/replaced:

- `GET /api/network/channels/{channel}/messages` is deleted as the primary timeline path.
- `GET /api/network/peers/{peer_id}/messages` is deleted as the direct-room path.

New:

- `GET /api/network/channels/{channel}/threads`
- `GET /api/network/channels/{channel}/threads/{thread_id}`
- `GET /api/network/channels/{channel}/threads/{thread_id}/messages`
- `GET /api/network/channels/{channel}/directs`
- `POST /api/network/channels/{channel}/directs/resolve`
- `GET /api/network/channels/{channel}/directs/{direct_id}`
- `GET /api/network/channels/{channel}/directs/{direct_id}/messages`
- `GET /api/network/work/{work_id}`

HTTP and UDS registrations must stay in parity. Handler tests must fail if one transport exposes a route that the other does not.

### Contract Payloads

```go
type NetworkSendRequest struct {
	SessionID   string                     `json:"session_id"`
	Channel     string                     `json:"channel"`
	Surface     string                     `json:"surface,omitempty"`
	ThreadID    string                     `json:"thread_id,omitempty"`
	DirectID    string                     `json:"direct_id,omitempty"`
	Kind        string                     `json:"kind"`
	To          string                     `json:"to,omitempty"`
	Body        json.RawMessage            `json:"body"`
	WorkID      string                     `json:"work_id,omitempty"`
	ReplyTo     string                     `json:"reply_to,omitempty"`
	TraceID     string                     `json:"trace_id,omitempty"`
	CausationID string                     `json:"causation_id,omitempty"`
	ExpiresAt   *int64                     `json:"expires_at,omitempty"`
	ID          string                     `json:"id,omitempty"`
	Ext         map[string]json.RawMessage `json:"ext,omitempty"`
}
```

Validation:

- reject JSON payloads containing `interaction_id`
- reject `kind:"direct"`
- reject `thread_id` or `direct_id` without `surface`
- reject `surface:"thread"` without `thread_id`
- reject `surface:"direct"` without `direct_id`
- reject `surface:"thread"` with `direct_id`
- reject `surface:"direct"` with `thread_id`
- reject `thread_id` and `direct_id` together
- reject `surface`, `thread_id`, `direct_id`, or `work_id` on `greet` / `whois`
- reject `receipt` / `trace` without `work_id`
- reject cross-container lifecycle continuation
- reject raw `claim_token` anywhere in body or ext

Response envelopes:

```go
type NetworkThreadSummaryPayload struct {
	Channel          string     `json:"channel"`
	ThreadID         string     `json:"thread_id"`
	RootMessageID    string     `json:"root_message_id"`
	Title            string     `json:"title,omitempty"`
	OpenedByPeerID   string     `json:"opened_by_peer_id,omitempty"`
	OpenedSessionID  string     `json:"opened_session_id,omitempty"`
	OpenedAt         *time.Time `json:"opened_at,omitempty"`
	LastActivityAt   *time.Time `json:"last_activity_at,omitempty"`
	MessageCount     int        `json:"message_count"`
	ParticipantCount int        `json:"participant_count"`
	OpenWorkCount    int        `json:"open_work_count"`
	LastMessagePreview string   `json:"last_message_preview,omitempty"`
}

type NetworkDirectRoomPayload struct {
	Channel            string     `json:"channel"`
	DirectID           string     `json:"direct_id"`
	PeerA              string     `json:"peer_a"`
	PeerB              string     `json:"peer_b"`
	OpenedAt           *time.Time `json:"opened_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
	MessageCount       int        `json:"message_count"`
	OpenWorkCount      int        `json:"open_work_count"`
	LastMessagePreview string     `json:"last_message_preview,omitempty"`
}

type NetworkConversationMessagePayload struct {
	MessageID   string          `json:"message_id"`
	Channel     string          `json:"channel"`
	Surface     string          `json:"surface,omitempty"`
	ThreadID    string          `json:"thread_id,omitempty"`
	DirectID    string          `json:"direct_id,omitempty"`
	Kind        string          `json:"kind"`
	Direction   string          `json:"direction"`
	PeerFrom    string          `json:"peer_from"`
	PeerTo      string          `json:"peer_to,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	WorkID      string          `json:"work_id,omitempty"`
	ReplyTo     string          `json:"reply_to,omitempty"`
	TraceID     string          `json:"trace_id,omitempty"`
	CausationID string          `json:"causation_id,omitempty"`
	Text        string          `json:"text,omitempty"`
	PreviewText string         `json:"preview_text,omitempty"`
	Body        json.RawMessage `json:"body"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
}
```

### CLI Commands and Output Shapes

All commands preserve `-o json`, `-o jsonl` where list streams are useful, and `-o toon` where the existing CLI supports it.

```bash
agh network status -o json
```

JSON shape: `{"network":{...,"open_threads":0,"open_direct_rooms":0,"open_work_items":0,"surface_metrics":[...]}}`.

```bash
agh network channels -o json
```

JSON shape: `{"channels":[{"channel":"builders","thread_count":3,"direct_room_count":2,...}]}`.

```bash
agh network threads list --channel builders --limit 25 -o json
agh network threads show --channel builders --thread thread_launch_db -o json
agh network threads messages --channel builders --thread thread_launch_db --limit 50 -o jsonl
```

JSON shapes:

- `{"threads":[NetworkThreadSummaryPayload...]}`
- `{"thread":NetworkThreadSummaryPayload}`
- JSONL messages: one `NetworkConversationMessagePayload` per line

```bash
agh network directs list --channel builders --peer reviewer.sess-xyz --limit 25 -o json
agh network directs resolve --session "${AGH_SESSION_ID}" --channel builders --peer reviewer.sess-xyz -o json
agh network directs show --channel builders --direct direct_0123abcd... -o json
agh network directs messages --channel builders --direct direct_0123abcd... --limit 50 -o jsonl
```

JSON shapes:

- `{"directs":[NetworkDirectRoomPayload...]}`
- `{"direct":NetworkDirectRoomPayload}`
- JSONL messages: one `NetworkConversationMessagePayload` per line

```bash
agh network send \
  --session "${AGH_SESSION_ID}" \
  --channel builders \
  --surface thread \
  --thread thread_launch_db \
  --kind say \
  --body '{"text":"I can review the migration.","intent":"availability"}' \
  -o json
```

```bash
agh network send \
  --session "${AGH_SESSION_ID}" \
  --channel builders \
  --surface direct \
  --direct direct_0123abcd... \
  --kind trace \
  --work work_review_42 \
  --reply-to msg_review_request \
  --trace-id trace_review_42 \
  --body '{"state":"working","message":"Inspecting the migration now."}' \
  -o json
```

Deleted flags:

- `--interaction-id`

New flags:

- `--surface thread|direct`
- `--thread <thread_id>`
- `--direct <direct_id>`
- `--work <work_id>`

### Native Agent Tools

Retain and update:

- `agh__network_status`
- `agh__network_channels`
- `agh__network_peers`
- `agh__network_inbox`
- `agh__network_send`

Add:

- `agh__network_threads`
- `agh__network_thread_messages`
- `agh__network_directs`
- `agh__network_direct_resolve`
- `agh__network_direct_messages`
- `agh__network_work`

`agh__network_send` schema must accept `surface`, `thread_id`, `direct_id`, and `work_id`, and must reject `interaction_id` with `additionalProperties:false`. Tool tests must continue to prove raw `claim_token` rejection.

### Extension Host API

Add Host API methods in `internal/extension/protocol`, mirror them in `internal/extension/contract`, implement them in `internal/extension/host_api_*`, and update SDK generation:

- `network/status`
- `network/channels`
- `network/peers`
- `network/threads`
- `network/thread/get`
- `network/thread/messages`
- `network/directs`
- `network/direct/resolve`
- `network/direct/messages`
- `network/work/get`
- `network/send`

Capability gates:

- read methods require `network.read`
- `network/direct/resolve` requires `network.write` because it may create durable state
- `network/send` requires `network.write`

Host API payloads must use the same contract DTOs as HTTP/UDS where practical. Extension Host API docs and generated TypeScript SDK roots must co-ship.

### Web Route Map

Current route: `web/src/routes/_app/network.tsx` -> `/_app/network`.

Replace the single flat route with TanStack file routes:

| Route file | Route ID | Purpose |
| --- | --- | --- |
| `web/src/routes/_app/network.tsx` | `/_app/network` | layout/shell, channel selector, redirects to first visible channel |
| `web/src/routes/_app/network.$channel.threads.tsx` | `/_app/network/$channel/threads` | public thread list for one channel |
| `web/src/routes/_app/network.$channel.threads.$threadId.tsx` | `/_app/network/$channel/threads/$threadId` | public thread timeline |
| `web/src/routes/_app/network.$channel.directs.tsx` | `/_app/network/$channel/directs` | direct-room list for one channel |
| `web/src/routes/_app/network.$channel.directs.$directId.tsx` | `/_app/network/$channel/directs/$directId` | direct-room timeline |

Implementation requirements:

- Generate route tree after adding files.
- Query keys must include `channel`, `surface`, and container ID.
- The composer must send into the active route container.
- The channel-level `/_app/network/$channel/threads` composer is a "New public thread" affordance, not a reply composer. Submitting it generates a valid `thread_id`, sends the root `kind:"say"` message with `surface:"thread"`, and redirects to `/_app/network/$channel/threads/$threadId`.
- If a generated `thread_id` collides, the web client retries with a new ID once and then surfaces the server validation error. It must not silently append to an existing thread.
- Public thread views must never query direct-room messages.
- Direct-room views must render `peer_a` / `peer_b` membership from server truth.
- Browser artifact capture should replace `network_selected_peer` with `network_selected_thread` and `network_selected_direct`.
- Existing Storybook/network fixtures must be rewritten to show channel thread heads, thread detail, direct list, and direct detail.

## Extensibility and Agent Manageability

This change is user-visible runtime capability. It is incomplete unless every agent-manageable and extension-facing path can operate on the new conversation containers.

### Extensions

- Add Host API methods listed above.
- Update extension contract generation and SDK exports.
- Add manifest capability gates `network.read` and `network.write` if they do not already exist.
- Add Host API tests for capability denial, malformed IDs, direct resolve race behavior, and send parity with HTTP validation.

### Hooks

Add a `network` hook family in `internal/hooks/events.go`:

```go
const HookEventFamilyNetwork HookEventFamily = "network"

const (
	HookNetworkThreadOpened      HookEvent = "network.thread.opened"
	HookNetworkDirectRoomOpened  HookEvent = "network.direct_room.opened"
	HookNetworkMessagePersisted  HookEvent = "network.message.persisted"
	HookNetworkWorkOpened        HookEvent = "network.work.opened"
	HookNetworkWorkTransitioned  HookEvent = "network.work.transitioned"
	HookNetworkWorkClosed        HookEvent = "network.work.closed"
)
```

Network hooks are async observation hooks in MVP. They cannot deny routing, mutate payloads, bypass trust checks, bypass raw-token redaction, or change task-run ownership. Dispatch happens at the state-transition call site after durable commit, not by tailing timeline/audit tables.

Delivery contract: network hooks are best-effort, fire-and-forget, post-commit notifications with no replay log in MVP. A crash after commit but before hook dispatch may lose the hook notification; a retry at a higher layer may emit another notification for the same persisted message. Hook consumers must deduplicate by `(event, message_id, work_id, trace_id)` where those fields exist, and hook failures must be logged without rolling back the network write.

Payloads include:

- `channel`
- `surface`
- `thread_id`
- `direct_id`
- `work_id`
- `message_id`
- `kind`
- `peer_from`
- `peer_to`
- `work_state`
- `trace_id`
- `causation_id`

Payloads must not include raw `claim_token` or unredacted secret material.

### Skills and Capabilities

- Rewrite bundled `agh-network` skill to teach channel -> thread/direct semantics.
- Keep capability artifact vocabulary unchanged; `capability` remains the canonical artifact name.
- Capability messages now use `kind:"capability"` on either `surface:"thread"` or `surface:"direct"`, with `work_id` when the transfer is lifecycle-bearing.
- No new capability bundle format is required in MVP.

### Tools and MCP Sidecars

- Native tools listed above become the primary agent path.
- Hosted MCP tool descriptors must expose the same JSON schemas as native tools.
- MCP sidecars must not receive raw claim tokens through network message bodies, ext, tool results, prompts, or logs.
- Tool descriptions must make direct-room visibility explicit: direct rooms are restricted to the two room peers plus runtime/audit access, not cryptographic privacy.

### Bundles and Registries

- No bundle activation schema change is required.
- Registries that index bundled skills must include the rewritten `agh-network` skill and updated snippets in `internal/skills/bundled/bundled_test.go`.
- Tool registry IDs should remain stable where possible; add new IDs rather than overloading old inbox/peer tools.

### Bridge SDK

- Bridge ingress may map external Slack-like thread/direct constructs into `surface`, `thread_id`, `direct_id`, and `work_id`.
- Bridge SDK docs must state that bridge adapters cannot fabricate direct-room membership outside the deterministic direct-room resolver.
- Bridge ingress should use `trace_id` / `causation_id` for cross-system correlation instead of abusing `work_id`.

### Config Lifecycle

No new `config.toml` keys are required for MVP. Existing `[network]` enablement, transport, and channel settings remain the runtime gate.

Required docs/tests:

- update config docs only to show the new send/query shapes
- assert default config output has no new network conversation keys
- ensure settings UI does not render controls for thread retention, unread sync, or notification preferences because those are post-MVP

## Agent Prompt and Skill Contract

Inbound wrappers must include conversation metadata:

```xml
<network-message
  id="msg_review_42"
  from="reviewer.sess-xyz"
  channel="builders"
  surface="direct"
  direct-id="direct_0123abcd..."
  kind="trace"
  work-id="work_review_42"
  reply-to="msg_review_request"
  trace-id="trace_review_42"
  causation-id="msg_review_request"
  trust="untrusted">
  <network-preview encoding="xml-escaped">Inspecting the migration now.</network-preview>
  <network-body encoding="base64-json">BASE64_CANONICAL_JSON</network-body>
</network-message>
```

Bundled `agh-network` skill rewrite outline:

- Operating model: channel is audience; public threads are N-to-N; direct rooms are restricted 1-to-1.
- Current context: read `AGH_SESSION_ID`, `AGH_SESSION_CHANNEL`, `AGH_PEER_ID`, and wrapper `surface`.
- Inspecting public work: `agh network threads list/show/messages`.
- Inspecting direct rooms: `agh network directs list/resolve/show/messages`.
- Sending in current public thread: `agh network send --surface thread --thread ...`.
- Sending in current direct room: `agh network send --surface direct --direct ...`.
- Work lifecycle: use `--work` only for lifecycle-bearing work; never use it as a conversation ID.
- Handoff: moving public work into a direct room opens a new `work_id`; link with `reply_to`, `trace_id`, and `causation_id`.
- Summarize back: direct-room conclusions are posted publicly as a new `say` in the public thread.
- Prompt injection defense: unchanged, but examples must include `surface`, container ID, and `work_id`.
- Raw token defense: raw `claim_token` remains forbidden.

Example public-thread envelope:

```json
{
  "protocol": "agh-network/v0",
  "id": "msg_thread_001",
  "kind": "say",
  "channel": "builders",
  "surface": "thread",
  "thread_id": "thread_launch_db",
  "from": "founder.sess-a",
  "to": null,
  "reply_to": null,
  "trace_id": "trace_launch_db",
  "causation_id": null,
  "ts": 1777903200,
  "body": {"text":"Let's review the migration plan.","intent":"discussion"},
  "ext": {}
}
```

Example direct-room work envelope:

```json
{
  "protocol": "agh-network/v0",
  "id": "msg_direct_trace_001",
  "kind": "trace",
  "channel": "builders",
  "surface": "direct",
  "direct_id": "direct_0123abcd0123abcd0123abcd0123abcd",
  "from": "reviewer.sess-b",
  "to": "founder.sess-a",
  "work_id": "work_review_42",
  "reply_to": "msg_direct_request_001",
  "trace_id": "trace_launch_db",
  "causation_id": "msg_thread_001",
  "ts": 1777903260,
  "body": {"state":"working","message":"Inspecting migration failure paths."},
  "ext": {}
}
```

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
| --- | --- | --- | --- |
| `docs/rfcs/003_agh-network-v0.md` | modified | normative protocol hard cut | rewrite envelope, kinds, routing, lifecycle, examples |
| `docs/rfcs/004_agh-network-v1.md` | modified | trust RFC references core fields and signed content | update processing model, signed-field set, NATS request/reply examples |
| `docs/_memory/glossary.md` | modified | current kind list includes `direct` | update canonical kind list and add conversation container terms |
| `internal/network/*` | modified | core runtime semantics change | add surface/container validation, deterministic direct ID, work lifecycle rename |
| `internal/store/types.go` | modified | persistence DTOs change | add thread/direct/work/message query DTOs and validation |
| `internal/store/globaldb/*` | modified | persistence/query shape changes | numbered migration, new tables/indexes, same-transaction write path |
| `internal/hooks/*` | modified | new hook family | add async network hook events and payloads |
| `internal/api/contract/*` | modified | public contract break | delete old fields, add new fields, regenerate codegen |
| `internal/api/core/*` | modified | handler/query behavior break | add thread/direct endpoints and send validation |
| `internal/api/udsapi/routes.go` | modified | UDS parity | add same routes as HTTP |
| `internal/cli/*` | modified | operator + agent control plane changes | update send flags, add thread/direct list/detail flows |
| `internal/tools/*` | modified | agent native tools change | update `network_send`, add thread/direct tools |
| `internal/daemon/native_tools.go` | modified | hosted tool dispatch | route new native tool methods through runtime/store |
| `internal/extension/*` | modified | extension manageability | add Host API methods, capability gates, SDK generation |
| `internal/skills/bundled/skills/agh-network/SKILL.md` | modified | agent prompt semantics | rewrite examples and rules |
| `web/src/routes/_app/network*` | modified | route/search model changes | shift from channel/peer rooms to threads/direct rooms |
| `web/src/hooks/routes/use-network-page.ts` | modified | flat view model obsolete | replace with route/query-driven container view models |
| `web/src/systems/network/*` | modified | primary UX changes | thread heads, direct-room list, conversation views |
| `web/e2e/fixtures/*` | modified | artifact schema change | record selected thread/direct instead of selected peer room |
| `packages/site/content/{runtime,protocol}` | modified | docs truth changes | rewrite protocol and operator docs in same PR |
| `openapi/agh.json` | generated | contract drift risk | regenerate with `make codegen` |
| `web/src/generated/agh-openapi.d.ts` | generated | web type drift risk | regenerate with `make codegen` |
| `.compozy/tasks/_archived/*` | no active change | historical artifacts may keep old terminology | do not edit unless a future archival cleanup explicitly requests it |

## Safety Invariants

1. Every conversation-bearing message belongs to exactly one container: `(channel, thread_id)` when `surface="thread"` or `(channel, direct_id)` when `surface="direct"`.
2. `greet` and `whois` never carry `surface`, `thread_id`, `direct_id`, or `work_id`.
3. Container fields are symmetric: `thread_id` or `direct_id` without `surface` is invalid; `surface` without the matching container ID is invalid; the opposite container ID must be absent.
4. The first non-duplicate valid conversation-bearing message with a new `thread_id` opens the public thread and becomes the root message.
5. Direct-room-opening writes are valid only when `direct_id` matches the deterministic channel + peer-pair identity.
6. `kind:"direct"` is rejected at every ingress path: network envelope validation, HTTP, UDS, CLI, native tools, extension Host API, tests, docs examples, and generated contract.
7. `interaction_id` is rejected at every public ingress path and removed from active runtime, store, web, docs, and generated artifacts.
8. A `work_id` is durably bound to one conversation container at creation and cannot be continued from another container.
9. `network_work` rows cannot dangle; SQLite foreign keys or constraint triggers enforce that the bound thread/direct container exists and cannot be deleted while work references it.
10. Moving work from a public thread to a direct room creates a new `work_id`; linkage uses `reply_to`, `trace_id`, and `causation_id`.
11. Direct-room visibility is restricted to the two room peers plus runtime/audit access; public thread queries never include direct-room messages.
12. Direct-room resolution is deterministic for `(channel, peer_a, peer_b)` regardless of peer order and never creates duplicate active rooms for the same pair.
13. Conversation summaries are derived from committed message rows in the same SQLite transaction that writes the message.
14. Post-terminal work rejects all new non-duplicate messages; exact `message_id` replay is idempotent before lifecycle handling.
15. Generated API and web types ship in the same change as contract source edits.
16. Agent prompt wrappers always include `channel`, `surface`, the matching conversation ID, `reply_to` when present, and `work_id` only when lifecycle-bearing work exists.
17. Claim-token redaction remains unchanged: raw claim tokens never cross network envelopes, HTTP/UDS responses, CLI output, native tool payloads, logs, prompts, docs examples, web UI, or persisted message bodies.
18. `network_work` never becomes a durable queue; `task_runs` remains the only durable task queue.
19. Network-created task runs carry `task_runs.metadata_json.network_work_id` when they originate from a work-bearing message, but that metadata never affects task claiming or queue ownership.
20. RFC 004 verified signatures include `surface`, `thread_id`, `direct_id`, and `work_id` when present.
21. Direct rooms are exactly two-party in MVP; adding group direct rooms requires a future schema migration and new ADR.
22. Network hooks are best-effort post-commit notifications in MVP; hook failure or loss never rolls back committed network writes.

## Test Strategy

### Unit Tests

- Envelope validation for `surface`, `thread_id`, `direct_id`, `work_id`, and `kind` hard cuts.
- Symmetric container-field rejection tests for IDs without `surface`, `surface` without matching ID, mismatched opposite IDs, and `greet` / `whois` with any conversation field.
- Legacy rejection tests for `interaction_id` and `kind:"direct"` across network, API contract decoding, CLI input, native tool schema, and Host API.
- Direct-room ID derivation tests proving peer-order independence, channel scoping, same-peer rejection, and stable hash output.
- Direct-room collision tests for the zero-row-after-`INSERT OR IGNORE` case when a deterministic ID collides with a different pair.
- Work lifecycle tests proving container-scoped `work_id`, terminal rejection, valid state transitions, and cross-container rejection.
- Work lifecycle idempotency tests proving exact duplicate `message_id` replay returns duplicate before lifecycle handling and all new post-terminal messages are rejected.
- Store DTO validation tests for new summaries, queries, messages, direct rooms, and work entries.
- Wrapper rendering tests proving prompt metadata includes conversation container and `work_id`.
- RFC 004 trust tests proving signed canonical content changes when `surface`, `thread_id`, `direct_id`, or `work_id` changes.
- RFC 004 JCS tests proving absent nullable fields and present zero-valued nullable fields produce different canonical bytes before verification.
- Hook event catalog tests update expected event count and validate network family matcher fields.
- Native tool schema tests proving `interaction_id` is rejected and raw `claim_token` is still rejected.

### Migration Tests

- Fresh DB contains final `network_timeline_log`, `network_threads`, `network_thread_participants`, `network_direct_rooms`, `network_work`, and updated `network_audit_log`.
- Reopen-after-restart preserves the final schema and records the actual next sequential migration version selected at implementation time.
- Old flat `network_timeline_log` rebuild copies only `greet` / `whois` rows and deletes old conversation rows.
- Old `interaction_id` column and old flat timeline indexes are absent after migration.
- Unique `(channel, peer_a, peer_b)` direct-room constraint is present.
- `network_work` rejects invalid surface/container combinations.
- `network_work` rejects missing referenced thread/direct containers and prevents deleting a referenced container.
- `network_thread_participants` cascade behavior is covered with `PRAGMA foreign_keys = 1` asserted.
- Migration tests run with `-race` where package patterns already use race-sensitive SQLite tests.

### Integration Tests

- Router + store public thread creation.
- Router + store direct-room resolve and concurrent resolve under goroutines.
- Work handoff from thread -> direct room opens a new `work_id` and links by `reply_to`, `trace_id`, and `causation_id`.
- Network-triggered task ingress writes `task_runs.metadata_json.network_work_id` and does not use it for queue ownership or claims.
- Same-transaction write rollback proves no summary update persists if timeline insert fails.
- HTTP/UDS parity tests for every thread/direct/work endpoint.
- CLI integration tests for list/show/messages/resolve/send output shapes.
- Extension Host API tests for read/write capabilities and validation parity.
- Hook dispatch tests proving network hooks fire after durable commit only, failures are logged, and hook failure does not roll back the committed message.
- Web integration tests for channel thread list, thread detail timeline, direct-room list/detail, channel-level new-thread composer behavior, and composer sending into the active route container.

### Coverage and Race Requirements

- Every touched Go package must keep at least 80% package coverage.
- Race-sensitive Go packages touched by this work, especially `internal/network`, `internal/store/globaldb`, `internal/api/core`, `internal/daemon`, and `internal/hooks`, require Linux-Race parity in CI or the closest existing race-enabled lane.
- Web route/system changes require Vitest coverage for route state, query keys, empty/error states, and composer payload shape.
- Contract changes require `make codegen-check` and generated TypeScript drift checks.

### Required QA Shape

- `make verify`
- real-scenario QA updates for:
  - public launch thread coordination
  - restricted reviewer handoff via direct room
  - summarize-back-to-thread workflow
  - direct-room resolve race with two agents attempting the same handoff

## Implementation Steps

1. RFC and glossary hard cut.
2. Shared runtime types, validators, reason codes, symbol renames, and direct-room ID helper.
3. SQLite migration, store DTOs, store queries, `network_work`, and same-transaction write helper.
4. Router, delivery wrappers, lifecycle handling, hooks, observability, and bundled prompt/skill updates.
5. HTTP/UDS/CLI/native tools/extension Host API contracts and codegen.
6. Web `/network` route tree and systems rewrite.
7. Site docs, extension SDK docs, examples, and MCP/tool docs.
8. Full verification, E2E, real-scenario QA, and task readiness review.

Technical dependencies:

- `agh-schema-migration` for numbered SQLite changes.
- `agh-contract-codegen-coship` for OpenAPI + generated TS.
- `cy-web-docs-impact` expectations: web and docs must co-ship with backend contract change.
- `agh-code-guidelines`, `golang-pro`, `nats`, and `agh-test-conventions` for implementation tasks touching Go/network/tests.

## Monitoring and Observability

Structured log fields on all network message logs:

- `channel`
- `surface`
- `thread_id`
- `direct_id`
- `work_id`
- `trace_id`
- `causation_id`
- `kind`
- `peer_from`
- `peer_to`
- `direction`
- `delivery_result`

Metrics names and cardinality:

| Metric | Unit | Type | Labels | Cardinality rule |
| --- | --- | --- | --- | --- |
| `network_messages_total` | messages | counter | `channel`, `surface`, `kind`, `direction`, `result` | no `thread_id`, `direct_id`, `work_id` labels |
| `network_conversation_messages_total` | messages | counter | `channel`, `surface` | no container ID labels |
| `network_threads_open_total` | threads | counter | `channel` | channel only |
| `network_direct_rooms_open_total` | rooms | counter | `channel` | channel only |
| `network_work_open_total` | work items | counter | `channel`, `surface` | no `work_id` label |
| `network_work_transitions_total` | transitions | counter | `channel`, `surface`, `state` | state enum only |
| `network_open_work_items` | work items | gauge | `channel`, `surface` | no `work_id` label |
| `network_delivery_queue_depth` | messages | gauge | `channel`, `surface` | no container ID label |
| `network_direct_resolve_total` | resolves | counter | `channel`, `result` | no peer labels |

High-cardinality IDs (`thread_id`, `direct_id`, `work_id`, message IDs) belong in structured logs and persisted audit rows, not metric labels.

Observability path:

- network runtime updates in-memory status counters for `/network/status`
- durable audit rows include container fields for task/observe aggregations
- network hook events provide extension-visible observation after commit
- no separate event bus is introduced

## Technical Considerations

### Key Decisions

- Public discussion and restricted bilateral discussion are separate conversation containers.
- `work_id` replaces `interaction_id` so lifecycle and conversation no longer share ambiguous language.
- `direct` becomes a conversation surface, not a message kind.
- `say` is the generic conversational event kind on either surface.
- Direct-room resolution is deterministic from channel + sorted peer pair.
- `network_work` is durable lifecycle metadata, not a queue.
- Peer filters remain useful, but they are not the same as direct rooms.

### Known Risks

- The change is broad and touches protocol, runtime, docs, CLI, extension APIs, tools, and web in one cut.
- Store transactions need careful idempotency so duplicate message IDs do not inflate summaries.
- RFC 004 canonicalization tests must be updated at the same time as envelope fields or verified-mode peers will drift.
- The current site docs and tests are heavily anchored on `interaction_id` and peer-room wording; partial rewrite will create truth drift immediately.
- Direct rooms are "restricted visibility", not cryptographic privacy; product copy must not overpromise.

## Architecture Decision Records

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - Public N-to-N conversation and restricted one-to-one conversation become distinct channel-scoped containers.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - Work lifecycle gets a precise name and no longer doubles as a conversation identifier.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - `kind` describes events while `surface` describes where the event lives.
