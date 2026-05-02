# GoClaw Protocol, Testing, and Multi-Agent Orchestration Patterns

Comprehensive analysis of `goclaw` codebase patterns for AGH (Artificial General Hivemind), focusing on protocol design, testing infrastructure, message bus architecture, RPC dispatch, and multi-agent orchestration.

## 1. Test Helper Design Patterns

### 1.1 Context Builder Pattern

**File:** `internal/testutil/context.go`

The testutil package provides lightweight context builders that inject tenant/user/agent identities without requiring database connection.

```go
// Minimal builder API — no DB required
func TenantCtx(tenantID uuid.UUID) context.Context
func UserCtx(tenantID uuid.UUID, userID string) context.Context
func AgentCtx(tenantID, agentID uuid.UUID) context.Context
func FullCtx(tenantID uuid.UUID, userID string, agentID uuid.UUID) context.Context
```

**Key patterns:**

- Builders compose by chaining `store.With*` setters (e.g., `store.WithTenantID(ctx, tenantID)`)
- No allocations; uses context value keys (defined in `store/context.go`)
- Safe for tests to panic on malformed UUIDs via `MustParseUUID()`
- Used as test-setup fixture in all context-dependent tests

### 1.2 Shared Database Pattern with Lazy Initialization

**Files:**

- `internal/testutil/doc.go`, `internal/testutil/db.go` (integration tag)
- `tests/integration/v3_test_helper.go`

**Pattern: sync.Once + skip on unavailable**

```go
var (
    sharedDB     *sql.DB
    sharedDBOnce sync.Once
    sharedDBErr  error
)

func TestDB(t *testing.T, migrationsDir string) *sql.DB {
    t.Helper()
    sharedDBOnce.Do(func() {
        // ... single connection + migrations, any error stored
    })
    if sharedDBErr != nil {
        t.Skipf("test PG not available: %v", sharedDBErr)  // graceful skip
    }
    return sharedDB
}
```

**Key design decisions:**

- Build tag `//go:build integration` — keeps default build dependency-free
- Single lazy initialization per test binary (no per-test DB setup)
- Skips gracefully if Postgres unavailable (not fail-hard)
- Migrations run once via `golang-migrate` with `migrate.Up()`
- `pg.InitSqlx(db)` called centrally to prevent nil deref in sqlx wrappers

### 1.3 Fixture Builder Pattern

**File:** `tests/integration/v3_fixture_builders.go`

Fixture builders create minimal valid entities for FK satisfaction without full ORM setup. Pattern emphasizes:

- Manual INSERT statements with minimal required columns
- Each builder returns IDs for use in downstream fixtures
- Cleanup via `t.Cleanup()` with FK-order deletion (children first)

**Example:**

```go
func seedTenantAgent(t *testing.T, db *sql.DB) (tenantID, agentID uuid.UUID) {
    tenantID = uuid.New()
    agentID = uuid.New()
    // INSERT tenant, agent (minimal fields)
    // t.Cleanup deletes in FK order: team_tasks → team_members → teams → agent
    return tenantID, agentID
}
```

**Fixture composition:**

- `seedTwoTenants()` — isolation testing (two independent tenants)
- `seedTeam()` — team + 2 members (lead + member)
- `seedSession()` — empty session
- `seedMCPServer()`, `seedSecureCLI()`, `seedAPIKey()` — resource-specific

**Key insight:** Fixtures use direct SQL INSERT (not store API) to avoid schema-versioning coupling.

### 1.4 Assertion and Verification Patterns

**Observation from test files:**

Tests use direct comparison + error checks rather than assertion libraries:

```go
if got != expected {
    t.Errorf("mismatch: got %q, want %q", got, expected)
}
```

**No custom assertion helpers found** — tests rely on table-driven subtests (`t.Run()`) for parameterized verification.

### 1.5 Mock Generation

**File:** `internal/testutil/generate.go`

Uses `mockgen` (go.uber.org/mock) with pre-generated mocks checked into repo:

```go
//go:generate mockgen -destination=mock_session_store.go -package=testutil github.com/nextlevelbuilder/goclaw/internal/store SessionStore
```

**Pattern:**

- One `go:generate` per store interface
- Mocks checked in (no runtime generation)
- Used for unit tests that need store interface without hitting DB

---

## 2. Protocol and Wire Format Design

### 2.1 Frame Type System

**File:** `pkg/protocol/frames.go`

Three frame types:

```go
const (
    FrameTypeRequest  = "req"    // client → server
    FrameTypeResponse = "res"    // server → client
    FrameTypeEvent    = "event"  // server push
)
```

**Request frame:**

```go
type RequestFrame struct {
    Type   string          `json:"type"`    // always "req"
    ID     string          `json:"id"`      // client-generated, matches response
    Method string          `json:"method"`  // RPC method name
    Params json.RawMessage `json:"params,omitempty"` // deferred unmarshaling
}
```

**Response frame:**

```go
type ResponseFrame struct {
    Type    string      `json:"type"`     // always "res"
    ID      string      `json:"id"`       // matches request ID
    OK      bool        `json:"ok"`       // success flag
    Payload any         `json:"payload,omitempty"` // typed when ok=true
    Error   *ErrorShape `json:"error,omitempty"`   // when ok=false
}
```

**Event frame (server push):**

```go
type EventFrame struct {
    Type         string        `json:"type"`      // always "event"
    Event        string        `json:"event"`     // event name (e.g., "agent", "chat")
    Payload      any           `json:"payload,omitempty"`
    Seq          int64         `json:"seq,omitempty"`        // ordering number
    StateVersion *StateVersion `json:"stateVersion,omitempty"` // version counters
}
```

**Key design decisions:**

- Type field used for demultiplexing (first read determines path)
- Params left as `json.RawMessage` — deferred unmarshaling by handler (cheap parser rejection)
- OK boolean + Error shape enables structured error responses (code, message, details, retryable flag)
- StateVersion for optimistic state sync (presence, health version counters)

### 2.2 RPC Method Constants

**File:** `pkg/protocol/methods.go`

~100+ method constants organized by priority:

- Phase 1 CRITICAL: `agent`, `chat.send`, `config.get`, `sessions.list`
- Phase 2 NEEDED: `skills.*`, `cron.*`, `channels.*`, `teams.*`
- Phase 3 NICE TO HAVE: `logs.tail`, `browser.act`, `zalo.*`

**Naming convention:** `package.resource.action` (e.g., `teams.tasks.create`)

### 2.3 Event Types and Payloads

**Files:**

- `pkg/protocol/events.go` — event name constants + subtypes
- `pkg/protocol/team_events.go` — typed payloads for delegation/team task events

**Event lifecycle constants:**

```go
const (
    EventAgent = "agent"           // agent phase + result
    EventChat = "chat"             // chat completion
    EventCron = "cron"             // cron execution
    EventTeamTaskCreated = "team.task.created"
    EventDelegationStarted = "delegation.started"
    EventDelegationCompleted = "delegation.completed"
)
```

**Agent event subtypes** (in payload.type):

```go
AgentEventRunStarted = "run.started"
AgentEventToolCall = "tool.call"
AgentEventActivity = "activity"  // phase: thinking, tool_exec, compacting
```

**Typed delegation payload:**

```go
type DelegationEventPayload struct {
    DelegationID      string
    SourceAgentID     string  // UUID string
    TargetAgentID     string  // UUID string
    Mode              string  // "async" | "sync"
    Status            string  // lifecycle: pending, completed, failed
    ElapsedMS         int
    Error             string
    CreatedAt         string
}
```

**Key insight:** Event payloads use string IDs (parsed as UUID by consumers), never agent_key. See `docs/agent-identity-conventions.md`.

### 2.4 Error Shape and Codes

**File:** `pkg/protocol/errors.go`

```go
type ErrorShape struct {
    Code         string `json:"code"`      // error category (e.g., "UNAUTHORIZED")
    Message      string `json:"message"`   // localized message
    Details      any    `json:"details,omitempty"`
    Retryable    bool   `json:"retryable,omitempty"`    // client hint
    RetryAfterMs int    `json:"retryAfterMs,omitempty"` // rate limit backoff
}
```

**Error codes:**

```go
ErrInvalidRequest = "INVALID_REQUEST"
ErrUnauthorized = "UNAUTHORIZED"
ErrNotFound = "NOT_FOUND"
ErrAlreadyExists = "ALREADY_EXISTS"
ErrResourceExhausted = "RESOURCE_EXHAUSTED"
ErrTenantAccessRevoked = "TENANT_ACCESS_REVOKED"
```

---

## 3. Message Bus Architecture

### 3.1 Two-Bus Model

**GoClaw uses two distinct buses:**

#### Bus 1: Internal MessageBus (Channels)

**File:** `internal/bus/bus.go`

Routes messages between channels (Telegram, Discord, etc.) and agent runtime.

```go
type MessageBus struct {
    inbound  chan InboundMessage      // from channels → agent
    outbound chan OutboundMessage     // from agent → channels
    handlers map[string]MessageHandler // channel name → handler
    subscribers map[string]EventHandler // for broadcast
}
```

**Inbound message model:**

```go
type InboundMessage struct {
    Channel      string
    SenderID     string
    ChatID       string
    Content      string
    Media        []MediaFile
    SessionKey   string
    TenantID     uuid.UUID    // tenant scope
    AgentID      string       // target agent
    UserID       string       // per-user memory/bootstrap
    HistoryLimit int          // context window from channel config
    ToolAllow    []string     // per-group tool allowlist
}
```

**Outbound message model:**

```go
type OutboundMessage struct {
    Channel  string
    ChatID   string
    Content  string
    Media    []MediaAttachment  // with MIME type + caption
}
```

**Operations:**

```go
PublishInbound(msg)              // blocking enqueue
TryPublishInbound(msg) bool      // non-blocking (drops if buffer full)
ConsumeInbound(ctx) (msg, ok)    // blocking dequeue
PublishOutbound(msg)             // to channels
SubscribeOutbound(ctx) (msg, ok) // subscribe to outgoing

RegisterHandler(channel, handler)
GetHandler(channel) (handler, ok)

Subscribe(id, handler)
Unsubscribe(id)
Broadcast(event)                 // non-blocking per subscriber, panic-safe
```

**Broadcast safety:**

- Panicking handlers do NOT crash bus
- Caught and logged with subscriber ID + panic value
- Other handlers still deliver (recover inside lambda)

#### Bus 2: DomainEventBus (Consolidation)

**File:** `internal/eventbus/domain_event_bus.go`

Typed event bus for v3 consolidation pipeline with worker pool, dedup, retry.

```go
type DomainEventBus interface {
    Publish(event DomainEvent)              // non-blocking
    Subscribe(eventType, handler) unsubscribe
    Start(ctx context.Context)
    Drain(timeout) error
}
```

**Event model:**

```go
type DomainEvent struct {
    ID       string      // UUID v7 for ordering
    Type     EventType   // e.g., EventSessionCompleted
    SourceID string      // dedup key (session key, run ID)
    TenantID string      // MUST be UUID string
    AgentID  string      // MUST be UUID string (or empty)
    UserID   string
    Timestamp time.Time
    Payload  any         // typed per EventType
}
```

**Worker pool design:**

```go
Config {
    QueueSize     int
    WorkerCount   int
    RetryAttempts int
    RetryDelay    time.Duration
    DedupTTL      time.Duration
}
```

**Dedup mechanism (dedup.go):**

- TTL-based expiry map
- Background cleanup goroutine (sweeps at TTL/2 intervals)
- `Add(sourceID) bool` — returns true if new, false if duplicate
- Empty sourceID skips dedup

**Retry with exponential backoff:**

```go
for attempt := range cfg.RetryAttempts {
    err := safeCall(handler, event)
    if err == nil { return }
    if attempt < cfg.RetryAttempts-1 {
        time.Sleep(delay)
        delay *= 2
    }
}
```

### 3.2 Publish-Time Validation

**File:** `internal/eventbus/validate_agent_id.go`

Observer that logs warnings on non-UUID AgentID (drift detection):

```go
func validateAgentID(event DomainEvent) {
    if event.AgentID == "" { return } // OK — team/system event
    if _, err := uuid.Parse(event.AgentID); err != nil {
        slog.Warn("eventbus.non_uuid_agent_id",
            "event_type", event.Type,
            "non_uuid_agent_id", event.AgentID,
            "source_id", event.SourceID,
        )
    }
}
```

**Key insight:** Non-blocking observability — warning only, does not reject.

### 3.3 Cache Invalidation Events

**File:** `internal/bus/types.go`

Cache invalidation uses MessageBus broadcast (not persisted):

```go
const (
    TopicCacheAgent = "cache:agent"
    TopicCacheSkills = "cache:skills"
    TopicCacheCron = "cache:cron"
    // ... ~10 cache kinds
)

type CacheInvalidatePayload struct {
    Kind     string    // CacheKindAgent, CacheKindSkills, etc.
    Key      string    // agent_key, agent_id, etc. Empty = invalidate all
    TenantID uuid.UUID // uuid.Nil = global (master admin), scopes to tenant otherwise
}
```

**Broadcast helper:**

```go
func BroadcastForTenant(pub EventPublisher, name string, tenantID uuid.UUID, payload any) {
    pub.Broadcast(Event{Name: name, TenantID: tenantID, Payload: payload})
}
```

### 3.4 Deduplication Pattern (Channels)

**File:** `internal/bus/dedupe.go`

TTL-based message dedup for channels (matching TypeScript `createDedupeCache()`):

```go
type DedupeCache struct {
    mu      sync.Mutex
    entries map[string]int64 // key → unix millis expiry
    ttl     time.Duration    // TTL
    maxSize int              // max entries before eviction
}

// Defaults: ttl=20min, maxSize=5000
func (d *DedupeCache) IsDuplicate(key string) bool {
    now := time.Now().UnixMilli()
    cutoff := now - d.ttl.Milliseconds()

    // Check if exists and in window
    if ts, ok := d.entries[key]; ok && ts >= cutoff { return true }

    // Lazy prune expired
    // Record this key with expiry = now + ttl
    d.entries[key] = now + d.ttl.Milliseconds()
    return false
}
```

**Cleanup strategy:**

- Prunes expired entries lazily on each check
- Evicts oldest entries if over maxSize (random order sufficient)

### 3.5 Inbound Message Debouncer

**File:** `internal/bus/inbound_debounce.go`

Buffers rapid consecutive messages from same sender (channel:chatID:senderID), merges on silence.

```go
type InboundDebouncer struct {
    debounceMs time.Duration
    buffers    map[string]*debounceBuffer
    flushFn    func(InboundMessage)
}

func (d *InboundDebouncer) Push(msg InboundMessage) {
    if d.debounceMs <= 0 { d.flushFn(msg); return } // disabled
    if len(msg.Media) > 0 { d.flushKey(key); d.flushFn(msg); return } // media bypasses

    // Buffer text, restart timer
    buf := d.buffers[debounceKey(msg)]
    buf.messages = append(buf.messages, msg)
    timer = time.AfterFunc(d.debounceMs, func() { d.flushKey(key) })
}

func mergeInboundMessages(msgs []InboundMessage) InboundMessage {
    // Join content with newlines, concat media, use last message for metadata
}
```

**Key design:**

- Media messages bypass debounce (flush buffered text first)
- Merging uses last message fields (simplifies timestamp, metadata handling)

---

## 4. RPC Method Handler Pattern

### 4.1 MethodHandler Type and Router

**File:** `internal/gateway/router.go`

```go
type MethodHandler func(ctx context.Context, client *Client, req *protocol.RequestFrame)

type MethodRouter struct {
    handlers   map[string]MethodHandler
    server     *Server
    tenantStore store.TenantStore   // optional
    permCache  *cache.PermissionCache // optional
}

func (r *MethodRouter) Handle(ctx context.Context, client *Client, req *protocol.RequestFrame) {
    // 1. Lookup handler
    handler, ok := r.handlers[req.Method]
    if !ok {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, "unknown method"))
        return
    }

    // 2. Permission check (skip for connect, health, browser pairing)
    if req.Method != protocol.MethodConnect && req.Method != protocol.MethodHealth {
        if !pe.CanAccess(client.role, req.Method) {
            client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrUnauthorized, "permission denied"))
            return
        }
    }

    // 3. Inject context: locale, tenantID, tenantSlug, role
    ctx = store.WithLocale(ctx, locale)
    ctx = store.WithTenantID(ctx, client.TenantID())
    ctx = store.WithRole(ctx, client.Role())

    // 4. Call handler
    handler(ctx, client, req)
}
```

**Registration:**

```go
router.Register(protocol.MethodTeamsList, m.handleList)
router.Register(protocol.MethodTeamsCreate, m.handleCreate)
```

### 4.2 Method Handler Implementation Pattern

**Example: Teams Create**

```go
type TeamsMethods struct {
    teamStore     store.TeamStore
    agentStore    store.AgentStore
    cfg           *config.Config
    eventBus      bus.EventPublisher
}

// Register all methods in one call
func (m *TeamsMethods) Register(router *gateway.MethodRouter) {
    router.Register(protocol.MethodTeamsList, m.handleList)
    router.Register(protocol.MethodTeamsCreate, m.handleCreate)
    router.Register(protocol.MethodTeamsDelete, m.handleDelete)
}

type teamsCreateParams struct {
    TeamName    string   `json:"teamName"`
    LeadAgentID string   `json:"leadAgentId"`
    MemberIDs   []string `json:"memberIds"`
}

func (m *TeamsMethods) handleCreate(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
    locale := store.LocaleFromContext(ctx)

    // 1. Nil check (soft dependency on store)
    if m.teamStore == nil {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInternal, i18n.T(locale, i18n.MsgTeamsNotConfigured)))
        return
    }

    // 2. Parse params
    var params teamsCreateParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, i18n.T(locale, i18n.MsgInvalidJSON)))
        return
    }

    // 3. Validate required fields
    if params.TeamName == "" {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, i18n.T(locale, i18n.MsgRequired, "teamName")))
        return
    }

    // 4. Parse UUIDs
    leadID, err := uuid.Parse(params.LeadAgentID)
    if err != nil {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, i18n.T(locale, i18n.MsgInvalidID, "leadAgentId")))
        return
    }

    // 5. Business logic + DB transaction
    team, err := m.teamStore.CreateTeam(ctx, &store.Team{Name: params.TeamName, LeadAgentID: leadID})
    if err != nil {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInternal, err.Error()))
        return
    }

    // 6. Broadcast event
    m.eventBus.Broadcast(bus.Event{
        Name: protocol.EventTeamCreated,
        TenantID: store.TenantIDFromContext(ctx),
        Payload: protocol.TeamCreatedPayload{TeamID: team.ID.String()},
    })

    // 7. Return result
    client.SendResponse(protocol.NewOKResponse(req.ID, team))
}
```

**Handler structure (consistent across all methods):**

1. Nil-check optional stores (soft dependencies)
2. Extract locale from context for i18n
3. Parse params via `json.Unmarshal(req.Params, &params)`
4. Validate required fields (return early with i18n error)
5. Parse UUIDs with error handling
6. Business logic (store call, calculations)
7. Broadcast relevant events (e.g., TeamCreated)
8. Return typed response (NewOKResponse or NewErrorResponse)

### 4.3 Permission and Ownership Checks

**Pattern 1: Role-based access**

```go
if !permissions.HasMinRole(client.Role(), permissions.RoleAdmin) {
    client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrUnauthorized, "admin required"))
    return
}
```

**Pattern 2: Session ownership**

```go
if !requireSessionOwner(ctx, m.sessions, m.cfg, client, req.ID, params.SessionKey) {
    return  // error already sent by helper
}
```

**Pattern 3: Team membership**

```go
if !permissions.HasMinRole(client.Role(), permissions.RoleAdmin) {
    callerID := store.UserIDFromContext(ctx)
    if ok, err := m.teamStore.HasTeamAccess(ctx, teamID, callerID); err != nil || !ok {
        client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotFound, "not a member"))
        return
    }
}
```

### 4.4 Error Response Patterns

**Localization:**

```go
locale := store.LocaleFromContext(ctx)  // from connect params or Accept-Language header
i18n.T(locale, i18n.MsgRequired, "fieldName")  // fmt.Sprintf-style templates
```

**Error codes:**

- `ErrInvalidRequest` — malformed JSON, missing required field
- `ErrUnauthorized` — permission denied
- `ErrNotFound` — resource not found
- `ErrAlreadyExists` — duplicate key
- `ErrResourceExhausted` — quota limit hit
- `ErrInternal` — server error (catch-all for unhandled)

---

## 5. Cross-Cutting Concerns

### 5.1 Feature Gating via Edition System

**File:** `internal/edition/edition.go`

Simple preset-based feature gating (no complex rules engine):

```go
type Edition struct {
    Name                  string
    MaxAgents             int
    MaxTeams              int
    MaxTeamMembers        int
    MaxChannels           map[string]int
    MaxSubagentConcurrent int
    MaxSubagentDepth      int
    KGEnabled             bool
    RBACEnabled           bool
    TeamFullMode          bool
    VectorSearch          bool
}

var (
    Standard = Edition{
        Name: "standard",
        KGEnabled: true,
        RBACEnabled: true,
        TeamFullMode: true,
        VectorSearch: true,
    }
    Lite = Edition{
        Name: "lite",
        MaxAgents: 5,
        MaxTeams: 1,
        MaxTeamMembers: 5,
        MaxChannels: map[string]int{"telegram": 1, "discord": 1},
        MaxSubagentConcurrent: 2,
        KGEnabled: false,
        RBACEnabled: false,
    }
)

// Global atomic state
var current atomic.Pointer[Edition]

func Current() Edition { return *current.Load() }
func SetCurrent(e Edition) { current.Store(&e) }
```

**Usage:**

```go
if edition.Current().KGEnabled { /* knowledge graph available */ }
if edition.Current().MaxAgents > 0 && agentCount >= edition.Current().MaxAgents {
    return ErrResourceExhausted
}
```

**Design decisions:**

- Preset only, no runtime customization
- Atomic pointer for lock-free concurrent reads
- Used at startup via `SetCurrent()` (often triggered by DB backend detection)
- No per-tenant editions (global instance setting)

### 5.2 Internationalization (i18n)

**File:** `internal/i18n/i18n.go`

Simple message catalog with locale fallback:

```go
type Catalog = map[string]string  // key → template

var catalogs = map[string]map[string]string{} // locale → catalog

func register(locale string, msgs map[string]string) {
    catalogs[locale] = msgs
}

func T(locale, key string, args ...any) string {
    msg := lookup(locale, key)
    if len(args) > 0 {
        return fmt.Sprintf(msg, args...)
    }
    return msg
}

func lookup(locale, key string) string {
    // Try requested locale
    if cat, ok := catalogs[locale]; ok {
        if msg, ok := cat[key]; ok { return msg }
    }
    // Fallback to English
    if locale != LocaleEN {
        if cat, ok := catalogs[LocaleEN]; ok {
            if msg, ok := cat[key]; ok { return msg }
        }
    }
    // Return key as-is if not found anywhere
    return key
}

func Normalize(locale string) string {
    if IsSupported(locale) { return locale }
    if len(locale) >= 2 {
        prefix := locale[:2]  // "en-US" → "en"
        if IsSupported(prefix) { return prefix }
    }
    return DefaultLocale
}
```

**Catalog registration (catalog_en.go, etc.):**

```go
func init() {
    register(LocaleEN, map[string]string{
        MsgRequired: "Field '%s' is required",
        MsgNotFound: "%s '%s' not found",
        MsgUnknownMethod: "Unknown method: %s",
    })
}
```

**Supported locales:**

- `en` (English, default)
- `vi` (Vietnamese)
- `zh` (Chinese)

**Key insight:** No dependency on i18next or external libs — pure Go maps + fallback.

### 5.3 Encryption (AES-256-GCM)

**File:** `internal/crypto/aes.go`

Symmetric encryption for API keys and sensitive tokens:

```go
const prefix = "aes-gcm:"

func Encrypt(plaintext, key string) (string, error) {
    if key == "" || plaintext == "" { return plaintext, nil }

    keyBytes, err := DeriveKey(key)  // 32-byte AES key
    block, _ := aes.NewCipher(keyBytes)
    gcm, _ := cipher.NewGCM(block)

    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return prefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext, key string) (string, error) {
    if !IsEncrypted(ciphertext) {
        slog.Warn("crypto.unencrypted_value_read")  // backward compat: plaintext allowed
        return ciphertext, nil
    }

    // Decode, extract nonce, decrypt, verify tag
}

func IsEncrypted(value string) bool { return strings.HasPrefix(value, prefix) }

func DeriveKey(input string) ([]byte, error) {
    // Accept: hex (64 chars), base64 (44 chars), or raw 32 bytes
    // Single function handles all formats
}
```

**Usage:**

- API keys stored as `aes-gcm:...` in DB
- Empty key = plaintext passthrough (dev/test scenarios)
- Backward compatible (unencrypted values readable, logged as warning)

---

## 6. Multi-Agent Orchestration Patterns

### 6.1 Delegation System

**Event types:**

```go
EventDelegationStarted = "delegation.started"
EventDelegationCompleted = "delegation.completed"
EventDelegationFailed = "delegation.failed"
EventDelegationCancelled = "delegation.cancelled"
EventDelegationProgress = "delegation.progress"
EventDelegationAccumulated = "delegation.accumulated"
EventDelegationAnnounce = "delegation.announce"
```

**Typed payloads:**

```go
type DelegationEventPayload struct {
    DelegationID       string
    SourceAgentID      string  // UUID
    SourceAgentKey     string  // agent_key
    TargetAgentID      string  // UUID
    TargetAgentKey     string  // agent_key
    Mode               string  // "sync" or "async"
    Task               string  // task description
    Status             string  // pending, completed, failed
    ElapsedMS          int
    Error              string
}

type DelegationProgressPayload struct {
    SourceAgentID string
    Active        []DelegationProgressItem  // per-delegation progress
}

type DelegationProgressItem struct {
    DelegationID string
    TargetAgentKey string
    ElapsedMS int
    Activity string  // "thinking", "tool_exec", "compacting"
    Tool string      // current tool name
}
```

**Accumulated delegation (async with siblings still running):**

```go
type DelegationAccumulatedPayload struct {
    DelegationID       string
    SourceAgentID      string
    TargetAgentKey     string
    SiblingsRemaining  int     // count of still-running siblings
    ElapsedMS          int
}
```

**Announce (all siblings complete):**

```go
type DelegationAnnouncePayload struct {
    SourceAgentID string
    Results       []DelegationAnnounceResultSummary // per-delegatee summary
    CompletedTaskIDs []string  // team task IDs resolved by delegation
    TotalElapsedMS int
    HasMedia bool
}
```

### 6.2 Team Task Lifecycle Events

**Event types:**

```go
EventTeamTaskCreated = "team.task.created"
EventTeamTaskClaimed = "team.task.claimed"
EventTeamTaskProgress = "team.task.progress"
EventTeamTaskCompleted = "team.task.completed"
EventTeamTaskFailed = "team.task.failed"
EventTeamTaskApproved = "team.task.approved"
EventTeamTaskRejected = "team.task.rejected"
EventTeamTaskCommented = "team.task.commented"
EventTeamTaskAssigned = "team.task.assigned"
EventTeamTaskAttachmentAdded = "team.task.attachment_added"
```

**Typed payload:**

```go
type TeamTaskEventPayload struct {
    TeamID           string
    TaskID           string
    TaskNumber       int
    Subject          string
    Status           string  // lifecycle state
    OwnerAgentKey    string
    Reason           string  // for rejections
    ProgressPercent  int
    ProgressStep     string
    ActorType        string  // "agent", "human", "system"
    ActorID          string  // agent key, user ID, or system identifier
    CommentText      string  // for commented events
}
```

### 6.3 Generic Batch Queue for Orchestration

**File:** `internal/orchestration/batch_queue.go`

Lock-free producer-consumer queue with deduplication by key:

```go
type BatchQueue[T any] struct {
    queues sync.Map // key → *batchQueueState[T]
}

// Pattern: First enqueue returns isProcessor=true (that goroutine processes)
func (bq *BatchQueue[T]) Enqueue(key string, entry T) bool {
    v, _ := bq.queues.LoadOrStore(key, &batchQueueState[T]{})
    q := v.(*batchQueueState[T])
    q.mu.Lock()
    defer q.mu.Unlock()
    q.entries = append(q.entries, entry)
    if q.running {
        return false  // processor already running
    }
    q.running = true
    return true      // caller is processor
}

// Processor drains all buffered entries for this key
func (bq *BatchQueue[T]) Drain(key string) []T {
    // ... atomically swap nil, return all buffered
}

// Processor checks if more work arrived while processing
func (bq *BatchQueue[T]) TryFinish(key string) bool {
    // ... check if entries > 0: false (more work)
    //     if entries == 0: mark idle, delete, return true (done)
}

// Processor loop pattern
if isProcessor := bq.Enqueue(key, entry); isProcessor {
    for {
        batch := bq.Drain(key)
        processBatch(batch)
        if bq.TryFinish(key) { break }  // TOCTOU-safe
    }
}
```

**Design:** TOCTOU-safe (prevents race between check and finish via lock held).

### 6.4 Child Result Aggregation

**File:** `internal/orchestration/child_result.go`

Unified result struct capturing agent run outcome (v2 or v3 pipeline):

```go
type ChildResult struct {
    Content       string
    Media         []bus.MediaFile  // path + MIME type + filename
    InputTokens   int64
    OutputTokens  int64
    Runtime       time.Duration
    Iterations    int
    Status        string  // "completed", "failed", "cancelled"
}

// Convert v2 RunResult → ChildResult
func CaptureFromRunResult(r *agent.RunResult, runtime time.Duration) ChildResult {
    return ChildResult{
        Content: r.Content,
        Media: MediaResultToBusFiles(r.Media),
        InputTokens: int64(r.Usage.PromptTokens),
        OutputTokens: int64(r.Usage.CompletionTokens),
        Runtime: runtime,
        Iterations: r.Iterations,
        Status: "completed",
    }
}

// Convert v3 PipelineResult → ChildResult
func CaptureFromPipelineResult(r *pipeline.RunResult, runtime time.Duration) ChildResult {
    // ... similar conversion
}
```

---

## 7. Testing Patterns and Best Practices

### 7.1 Integration Test Structure

**Typical integration test:**

```go
func TestTeamStore_CreateTeam(t *testing.T) {
    db := testDB(t)  // skip if PG unavailable
    ctx := context.Background()

    // Seed minimal fixtures
    tenantID, agentID := seedTenantAgent(t, db)

    // Create store instance
    store := pg.NewTeamStore(db)

    // Execute business logic
    team, err := store.CreateTeam(ctx, &store.Team{
        TenantID: tenantID,
        Name: "Test Team",
        LeadAgentID: agentID,
    })

    // Assert results
    if err != nil { t.Fatalf("CreateTeam: %v", err) }
    if team.ID == uuid.Nil { t.Error("team ID not set") }
    if team.Name != "Test Team" { t.Errorf("name mismatch") }

    // Cleanup via t.Cleanup() in seedTenantAgent
}
```

### 7.2 Table-Driven Subtests

**Pattern:**

```go
tests := []struct {
    name        string
    edition     Edition
    maxAgents   int
    wantLimited bool
}{
    {"Standard has no limits", Standard, 0, false},
    {"Lite has limits", Lite, 5, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        if got := tt.edition.IsLimited(); got != tt.wantLimited {
            t.Errorf("IsLimited() = %v; want %v", got, tt.wantLimited)
        }
    })
}
```

### 7.3 Panic Recovery Testing

**Testing panic safety:**

```go
func TestBroadcast_PanickingHandler_DoesNotCrashBus(t *testing.T) {
    mb := New()
    defer mb.Close()

    mb.Subscribe("panicker", func(e Event) { panic("subscriber exploded") })
    mb.Subscribe("normal", func(e Event) { /* ... */ })

    // Must not panic — bus catches and logs
    mb.Broadcast(Event{Name: "test"})

    // Bus still operational
    mb.Broadcast(Event{Name: "test2"})
}
```

---

## 8. Summary of Key Design Principles

### Protocol Design

1. **Frame type demultiplexing** — Type field determines JSON schema
2. **Deferred param unmarshaling** — Params left as `json.RawMessage` for handler-specific parsing
3. **Structured errors** — Code + message + retryable flag + optional details
4. **Typed event payloads** — One struct per event type, UUID strings for identity

### Testing

1. **Shared DB with lazy init + graceful skip** — not per-test setup
2. **Fixture builders with t.Cleanup()** — FK-order deletion, no ORM
3. **Context builders (no DB)** — fast unit test setup
4. **Pre-generated mocks** — checked in, no runtime generation

### Message Bus

1. **Two-bus model** — MessageBus for channels, DomainEventBus for consolidation
2. **Dedup with TTL + lazy cleanup** — not background GC
3. **Panic-safe broadcast** — handler panics logged, don't crash others
4. **Non-blocking publish** — drops on buffer full, warns in logs

### RPC Dispatch

1. **MethodRouter with registry** — map[method]handler registered via Register()
2. **Permission check in dispatcher** — before context injection
3. **Context injection (locale, tenant, role)** — uniform across all handlers
4. **Consistent error response path** — early return pattern

### Feature Gating

1. **Preset editions only** — Standard, Lite, or custom (no rules engine)
2. **Atomic global state** — lock-free reads via atomic.Pointer
3. **Soft dependencies** — handlers nil-check optional stores

### Orchestration

1. **Generic BatchQueue[T]** — TOCTOU-safe processor election
2. **Unified ChildResult** — aggregates v2/v3 agent outcomes
3. **Event-driven lifecycle** — delegation progress + team task status via events
4. **Agent identity invariant** — UUID strings in events, never agent_key

---

## 9. References and File Locations

**Protocol:**

- `/pkg/protocol/frames.go` — Request/Response/Event types
- `/pkg/protocol/methods.go` — RPC method constants
- `/pkg/protocol/events.go` — Event names and subtypes
- `/pkg/protocol/team_events.go` — Delegation and team task payloads
- `/pkg/protocol/errors.go` — Error codes

**Testing:**

- `/internal/testutil/` — Context builders, DB helper
- `/tests/integration/v3_test_helper.go` — Shared DB, fixture builders
- `/internal/gateway/client_testing.go` — TestClient constructor

**Message Bus:**

- `/internal/bus/bus.go` — MessageBus (channels)
- `/internal/bus/types.go` — InboundMessage, OutboundMessage, Event
- `/internal/bus/dedupe.go` — Dedup cache (TTL-based)
- `/internal/bus/inbound_debounce.go` — Message buffering + merging
- `/internal/eventbus/` — DomainEventBus (consolidation)

**Gateway:**

- `/internal/gateway/server.go` — Server + WebSocket setup
- `/internal/gateway/router.go` — MethodRouter dispatcher
- `/internal/gateway/methods/` — Handler implementations (50+ files)

**Cross-Cutting:**

- `/internal/edition/edition.go` — Feature gating presets
- `/internal/i18n/i18n.go` — Message catalog + locale fallback
- `/internal/crypto/aes.go` — AES-256-GCM encryption

**Orchestration:**

- `/internal/orchestration/batch_queue.go` — Generic producer-consumer
- `/internal/orchestration/child_result.go` — Result aggregation (v2/v3)
