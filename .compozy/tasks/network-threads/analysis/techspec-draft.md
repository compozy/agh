# TechSpec: AGH Network Conversation Containers and Work Threads

## Executive Summary

This TechSpec redesigns AGH Network v0 around explicit conversation containers instead of a flat channel timeline plus peer-room projection. The new model keeps `channel` as the audience and discovery scope, introduces `public_thread` as the public N-to-N conversation primitive, introduces `direct_room` as the restricted one-to-one conversation primitive, and renames `interaction_id` to `work_id` for lifecycle-bearing work inside either conversation type.

No PRD exists for this workstream; this TechSpec is authored directly from the user-approved product direction and codebase exploration. The primary trade-off is a larger hard cut now in exchange for a much cleaner protocol, sharper UX semantics, and less permanent ambiguity in every public surface.

## System Architecture

### Component Overview

| Component | Purpose | Boundary |
| --- | --- | --- |
| RFC / envelope model | Replace `kind:"direct"` with `surface:"thread"|"direct"` plus `thread_id` / `direct_id` and `work_id` | Protocol + shared runtime types |
| Network router | Route by channel and conversation container, preserve work lifecycle inside the container | `internal/network` only |
| Conversation registry | Materialize thread heads and direct rooms for query/UI use | runtime + store |
| Message store | Persist `surface`, `thread_id`, `direct_id`, `work_id`, and visibility-scoped timeline rows | globaldb |
| CLI / HTTP / UDS | Expose thread/direct listing, detail, message, and send surfaces symmetrically | control plane |
| Web workspace | Replace flat channel timeline + peer-first model with channel threads and direct rooms | `web/` |
| Bundled prompts / skills | Teach agents to respond in the current conversation container and use `work_id` only for lifecycle-bearing work | startup prompt + bundled skills |
| Docs / OpenAPI | Rewrite protocol/runtime docs and regenerate codegen in the same PR | `packages/site`, `openapi`, `web/src/generated` |

### Data Flow

1. A local session sends a conversation-bearing message through `agh network send`.
2. The daemon validates `surface`, conversation ID, `work_id`, and kind-specific rules.
3. The router resolves the target container:
   - `surface:"thread"` -> `(channel, thread_id)`
   - `surface:"direct"` -> `(channel, direct_id)`
4. The router persists the normalized message row and updates conversation metadata.
5. Delivery logic enqueues inbound prompts for local sessions and renders wrappers with conversation metadata.
6. HTTP/UDS/CLI/Web query the persisted conversation views instead of reconstructing them from flat channel data.

## Implementation Design

### Core Interfaces

```go
type Surface string

const (
	SurfaceThread Surface = "thread"
	SurfaceDirect Surface = "direct"
)

type ConversationRef struct {
	Channel  string
	Surface  Surface
	ThreadID *string
	DirectID *string
}
```

```go
type SendRequest struct {
	SessionID   string
	Conversation ConversationRef
	Kind        Kind
	To          *string
	WorkID      *string
	ReplyTo     *string
	TraceID     *string
	CausationID *string
	Body        json.RawMessage
}
```

```go
type ConversationStore interface {
	ListThreads(ctx context.Context, channel string, query ThreadQuery) ([]ThreadSummary, error)
	ListDirectRooms(ctx context.Context, channel string, query DirectRoomQuery) ([]DirectRoomSummary, error)
	ListMessages(ctx context.Context, ref ConversationRef, query MessageQuery) ([]ConversationMessage, error)
	WriteMessage(ctx context.Context, entry ConversationMessage) error
}
```

### Data Models

#### Envelope hard cut

- Add `surface:"thread"|"direct"` to conversation-bearing envelopes.
- Add `thread_id` for `surface:"thread"`.
- Add `direct_id` for `surface:"direct"`.
- Rename `interaction_id` -> `work_id`.
- Delete `kind:"direct"`.

#### Kind rules

- `greet`, `whois`:
  - MUST NOT carry `surface`, `thread_id`, `direct_id`, or `work_id`
  - remain channel/peer discovery only
- `say`, `capability`, `receipt`, `trace`:
  - MUST carry `surface`
  - MUST carry the corresponding conversation identifier
  - MAY carry `to`
  - MUST carry `work_id` when participating in lifecycle-bearing work

#### Conversation scoping

- `thread_id` is scoped to `(channel, thread_id)`.
- `direct_id` is scoped to `(channel, direct_id)`.
- `work_id` is scoped to exactly one conversation container.
- A `work_id` MUST NOT cross from a public thread into a direct room. Handoff across containers opens a new `work_id` and links by `reply_to`, `trace_id`, and `causation_id`.

#### AGH Runtime store shape

- `network_timeline_log`
  - add `surface TEXT NOT NULL`
  - add `thread_id TEXT NULL`
  - add `direct_id TEXT NULL`
  - rename `interaction_id` column to `work_id`
- add `network_threads`
  - `(channel, thread_id)` primary key
  - opener, root_message_id, opened_at, last_activity_at, message_count, participant_count
- add `network_direct_rooms`
  - `(channel, direct_id)` primary key
  - peer_a, peer_b, opened_at, last_activity_at, message_count, open_work_count
- `network_audit_log`
  - add `surface`, `thread_id`, `direct_id`, `work_id`

Delete targets:

- `interaction_id` across active runtime/API/web/docs surfaces
- `kind:"direct"` validation and examples
- peer-room-as-primary-navigation assumptions in `/network`

### API Endpoints

#### Existing paths retained but re-scoped

- `GET /api/network/status`
- `GET /api/network/channels`
- `GET /api/network/channels/{channel}`
- `POST /api/network/send`

#### New conversation listing/detail paths

- `GET /api/network/channels/{channel}/threads`
- `GET /api/network/channels/{channel}/threads/{thread_id}`
- `GET /api/network/channels/{channel}/threads/{thread_id}/messages`
- `GET /api/network/channels/{channel}/directs`
- `GET /api/network/channels/{channel}/directs/{direct_id}`
- `GET /api/network/channels/{channel}/directs/{direct_id}/messages`
- `POST /api/network/channels/{channel}/directs/resolve`
  - AGH runtime helper that resolves or creates the stable one-to-one `direct_id` for a local session peer pair

#### Send payload contract

`POST /api/network/send` request body:

- `session_id`
- `channel`
- `surface`
- `thread_id` or `direct_id`
- `kind`
- `to` optional but validated against surface rules
- `work_id`
- `reply_to`
- `trace_id`
- `causation_id`
- `body`

Validation:

- reject legacy `interaction_id`
- reject `kind:"direct"`
- reject `surface:"thread"` without `thread_id`
- reject `surface:"direct"` without `direct_id`
- reject `work_id` on `greet` / `whois`
- reject cross-container lifecycle continuation

## Integration Points

| Integration | Purpose | Approach |
| --- | --- | --- |
| Bundled network skill | Teach agents the new conversation model | rewrite `agh-network` skill with thread/direct-room semantics |
| Startup harness prompt | inject updated network section for channel-bound sessions | preserve one-time prompt section injection |
| OpenAPI + generated TS | keep web and docs in lockstep | regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` |
| Site docs | rewrite protocol/runtime/operator docs | ship in same PR as contract/runtime changes |

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
| --- | --- | --- | --- |
| `docs/rfcs/003_agh-network-v0.md` | modified | normative protocol hard cut | rewrite envelope, kinds, routing, lifecycle, examples |
| `docs/rfcs/004_agh-network-v1.md` | modified | trust RFC references core fields | update cross-references and signed-field set |
| `internal/network/*` | modified | core runtime semantics change | add surface/container validation, work lifecycle rename |
| `internal/store/*` | modified | persistence/query shape changes | numbered migration, new tables/indexes, query rewrite |
| `internal/api/contract/*` | modified | public contract break | delete old fields, add new fields, regenerate codegen |
| `internal/api/core/*` | modified | handler/query behavior break | add thread/direct endpoints and send validation |
| `internal/cli/*` | modified | operator + agent control plane changes | update send flags, add thread/direct list/detail flows |
| `web/src/hooks/routes/use-network-page.ts` | modified | route/search model changes | shift from channel/peer rooms to threads/direct rooms |
| `web/src/systems/network/*` | modified | primary UX changes | thread heads, direct-room list, conversation views |
| `packages/site/content/{runtime,protocol}` | modified | docs truth changes | rewrite protocol and operator docs in same PR |

## Testing Approach

### Unit Tests

- Envelope validation for `surface`, `thread_id`, `direct_id`, `work_id`, and `kind` hard cuts
- Legacy rejection tests for `interaction_id` and `kind:"direct"`
- Direct-room visibility tests and participant validation
- Work lifecycle tests proving container-scoped `work_id`
- Wrapper rendering tests proving prompt metadata includes conversation container and `work_id`

### Integration Tests

- SQLite migration tests for new tables/columns/indexes and restart behavior
- Router + store tests for:
  - public thread creation
  - direct-room resolution
  - work handoff from thread -> direct room opening a new `work_id`
- HTTP/UDS parity tests for thread/direct endpoints
- CLI integration tests for send/list/detail/inbox behavior
- Web integration tests for:
  - channel thread list
  - thread detail timeline
  - direct-room list/detail
  - composer sending into the active conversation container

### Required QA shape

- `make verify`
- real-scenario QA updates for:
  - public launch thread coordination
  - restricted reviewer handoff via direct room
  - summarize-back-to-thread workflow

## Development Sequencing

### Build Order

1. RFC and glossary hard cut — no dependencies
2. Shared runtime types and envelope validation — depends on step 1
3. SQLite migration, store queries, and conversation registries — depends on step 2
4. Router, delivery wrapper, and bundled prompt/skill updates — depends on step 3
5. HTTP/UDS/CLI contract and handler changes plus codegen — depends on step 4
6. Web `/network` redesign for threads/direct rooms — depends on step 5
7. Site docs, QA scenarios, and verification gates — depends on step 6

### Technical Dependencies

- `agh-schema-migration` for numbered SQLite changes
- `agh-contract-codegen-coship` for OpenAPI + generated TS
- `cy-web-docs-impact` expectations: web and docs must co-ship with backend contract change

## Monitoring and Observability

- Add structured fields to all network message logs:
  - `channel`
  - `surface`
  - `thread_id`
  - `direct_id`
  - `work_id`
  - `trace_id`
  - `causation_id`
- Preserve the claim-token redaction invariant
- Add metrics by surface:
  - messages sent/received/delivered
  - open public threads
  - open direct rooms
  - open work items
  - queue depth by conversation container

## Technical Considerations

### Key Decisions

- Public discussion and restricted bilateral discussion are separate conversation containers.
- `work_id` replaces `interaction_id` so lifecycle and conversation no longer share ambiguous language.
- `direct` becomes a conversation surface, not a message kind.
- `say` is the generic conversational event kind on either surface.
- Peer filters remain useful, but they are not the same as direct rooms.

### Known Risks

- The change is broad and touches protocol, runtime, docs, CLI, and web in one cut.
- Direct-room identity needs a stable AGH runtime helper to avoid fragmented bilateral history.
- The current site docs and tests are heavily anchored on `interaction_id` and peer-room wording; partial rewrite will create truth drift immediately.

## Architecture Decision Records

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) — Public N-to-N conversation and restricted one-to-one conversation become distinct channel-scoped containers.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) — Work lifecycle gets a precise name and no longer doubles as a conversation identifier.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) — `kind` describes events while `surface` describes where the event lives.
