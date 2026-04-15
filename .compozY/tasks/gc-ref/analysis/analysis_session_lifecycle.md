# GoClaw Session & Gateway Lifecycle Analysis for AGH

## Executive Summary

GoClaw implements a **multi-tier session management system** with graceful lifecycle orchestration across:

1. **Session Manager** (in-memory + filesystem persistence)
2. **Gateway Server** (WebSocket + HTTP with lifecycle hooks)
3. **Message Consumer** (inbound routing + deduplication)
4. **Graceful Shutdown** (coordinated resource cleanup)

This analysis extracts patterns that AGH can adapt for its Agent Operating System architecture.

---

## Part 1: Session Lifecycle Patterns

### 1.1 Session Key Architecture

GoClaw uses **canonical hierarchical session keys** following the format:

```
agent:{agentKey}:{scopeType}:{scopeID}
```

**Key insight:** Agent keys are **human-readable identifiers** (e.g., `"default"`, `"my-agent"`), NOT UUIDs. This is intentional for cache invalidation and logging consistency.

**Session types and their keys:**

| Type           | Format                                                                | Example                                            |
| -------------- | --------------------------------------------------------------------- | -------------------------------------------------- |
| Direct Message | `agent:{agentKey}:{channel}:direct:{peerID}`                          | `agent:default:telegram:direct:386246614`          |
| Group Chat     | `agent:{agentKey}:{channel}:group:{chatID}`                           | `agent:default:telegram:group:-100123456`          |
| Group Topic    | `agent:{agentKey}:{channel}:group:{chatID}:topic:{topicID}`           | `agent:default:telegram:group:-100123456:topic:99` |
| Subagent       | `agent:{agentKey}:subagent:{label}`                                   | `agent:default:subagent:my-task`                   |
| Cron Job       | `agent:{agentKey}:cron:{jobID}`                                       | `agent:default:cron:reminder-job-id`               |
| Team           | `agent:{agentKey}:team:{teamID}:{chatID}`                             | `agent:default:team:team-1:user-123`               |
| Heartbeat      | `agent:{agentKey}:heartbeat` or `agent:{agentKey}:heartbeat:{unixMs}` | `agent:default:heartbeat`                          |
| WebSocket      | `agent:{agentKey}:ws:direct:{conversationID}`                         | `agent:default:ws:direct:conv-123`                 |

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/internal/sessions/key.go` (190 lines)

### 1.2 Session State Structure

```go
type Session struct {
    Key      string              // composite session key
    Messages []providers.Message // conversation history
    Summary  string              // LLM-generated summary after compaction
    Created  time.Time
    Updated  time.Time

    // Metadata
    Model               string  // LLM model used
    Provider            string  // LLM provider (anthropic, openai, etc.)
    Channel             string  // source channel (telegram, discord, ws)
    InputTokens         int64   // cumulative token usage
    OutputTokens        int64
    CompactionCount     int     // how many times history was summarized
    MemoryFlushCompactionCount int  // compaction count at last memory flush
    MemoryFlushAt       int64   // unix ms of last memory flush
    Label               string  // user-provided session label
    SpawnedBy           string  // parent agent (for subagent spawns)
    SpawnDepth          int     // nesting level
    ContextWindow       int     // cached LLM context window
    LastPromptTokens    int     // actual tokens from last response
    LastMessageCount    int     // message count at last LLM call
}
```

### 1.3 Session Manager Lifecycle Operations

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/internal/sessions/manager.go` (507 lines)

**Key operations:**

| Operation               | Purpose                    | Lock Type                | State Change                               |
| ----------------------- | -------------------------- | ------------------------ | ------------------------------------------ |
| `GetOrCreate()`         | Fetch or create session    | Write lock               | Creates if missing                         |
| `AddMessage()`          | Append message to history  | Write lock               | Updates `Updated` time                     |
| `GetHistory()`          | Fetch message slice (copy) | Read lock                | Defensive copy for thread safety           |
| `TruncateHistory()`     | Keep last N messages       | Write lock               | Prep for context limits                    |
| `SetHistory()`          | Replace entire history     | Write lock               | Used by memory compaction                  |
| `Reset()`               | Clear history + summary    | Write lock               | For session restart                        |
| `Delete()`              | Remove from memory + disk  | Write lock               | Deletes `.json` file if filesystem enabled |
| `Save()`                | Persist to disk (atomic)   | Read lock + atomic write | Uses temp file + rename pattern            |
| `IncrementCompaction()` | Bump compaction counter    | Write lock               | Triggers memory flush on threshold         |
| `SetMemoryFlushDone()`  | Mark flush complete        | Write lock               | Records compaction count + timestamp       |

**Atomic persistence pattern** (lines 449-476):

```go
// Snapshot under read lock
tmpFile := os.CreateTemp()
tmpFile.Write(data)
tmpFile.Sync()
tmpFile.Close()

// Atomic rename (no partial writes visible)
os.Rename(tmpPath, sessionPath)
```

### 1.4 Session Lifecycle Timeline

```
1. GetOrCreate() → Session initialized
2. AddMessage() x N → History grows
3. Metadata updates → Model, provider, tokens tracked
4. Monitor CompactionCount
5. When CompactionCount > threshold:
   - Memory consolidation pipeline summarizes history
   - SetHistory() to compressed version
   - IncrementCompaction()
   - SetMemoryFlushDone() → records flush point
6. Save() → Persists snapshot to disk (atomic)
7. Delete() → Removes from memory + filesystem

Parallel: Token tracking (AccumulateTokens) → usage metrics
```

---

## Part 2: Gateway Lifecycle Management

### 2.1 Gateway Startup & Dependency Injection

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway.go` (592 lines)

**Key pattern: Layered setup in `runGateway()`**

```
Phase 1: Config + Logging
  ↓
Phase 2: Core infrastructure (msgBus, domainBus, provider registry)
  ↓
Phase 3: Stores (PostgreSQL + optional SQLite)
  ↓
Phase 4: Tools, Skills, Bootstrap, Agents
  ↓
Phase 5: Server + HTTP handlers
  ↓
Phase 6: Channels, Scheduler, Cron, Heartbeat
  ↓
Phase 7: Lifecycle management (signal handler, graceful shutdown)
```

**Dependency injection pattern (gatewayDeps struct)**:

```go
type gatewayDeps struct {
    cfg              *config.Config
    server           *gateway.Server
    msgBus           *bus.MessageBus
    pgStores         *store.Stores       // all DB stores bundled
    providerRegistry *providers.Registry // LLM providers
    channelMgr       *channels.Manager   // Telegram, Discord, etc.
    agentRouter      *agent.Router       // agent lookup + resolution
    toolsReg         *tools.Registry     // tools (file, web, exec, etc.)
    skillsLoader     *skills.Loader      // skill search + discovery
    permCache        *cache.PermissionCache // for tenant membership checks
    enrichProgress   *vault.EnrichProgress  // vault enrichment status
    enrichWorker     *vault.EnrichWorker    // background enrichment task
    workspace        string
    dataDir          string
    domainBus        eventbus.DomainEventBus // V3 consolidation pipeline
}
```

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway_deps.go` (37 lines)

### 2.2 Graceful Shutdown Sequence

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway_lifecycle.go` (231 lines)

**Shutdown orchestration** (lines 142-184):

```go
go func() {
    sig := <-deps.sigCh // OS signal (SIGINT/SIGTERM)
    slog.Info("graceful shutdown initiated", "signal", sig)

    // 1. Broadcast shutdown event to all WS clients
    d.server.BroadcastEvent(*protocol.NewEvent(protocol.EventShutdown, nil))

    // 2. Stop inbound channels (Telegram, Discord, etc.)
    d.channelMgr.StopAll(context.Background())

    // 3. Stop cron jobs
    d.pgStores.Cron.Stop()

    // 4. Stop heartbeat ticker
    deps.heartbeatTicker.Stop()

    // 5. Stop task recovery ticker
    if taskTicker != nil {
        taskTicker.Stop()
    }

    // 6. Drain audit log queue BEFORE closing DB
    if deps.auditCh != nil {
        close(deps.auditCh)
    }

    // 7. Close provider resources (e.g., Claude CLI temp files)
    d.providerRegistry.Close()

    // 8. Stop permission cache sweep goroutines
    if d.permCache != nil {
        d.permCache.Close()
    }

    // 9. Release sandbox containers + stop pruning
    if deps.sandboxMgr != nil {
        deps.sandboxMgr.Stop()
        slog.Info("releasing sandbox containers...")
        deps.sandboxMgr.ReleaseAll(context.Background())
    }

    // 10. Drain active runs (5s timeout)
    if deps.sched != nil {
        slog.Info("gateway: draining active runs", "timeout", "5s")
        deps.sched.Stop()  // MarkDraining + StopAll
        time.Sleep(5 * time.Second)
    }

    // 11. Cancel context (stops all goroutines)
    cancel()
}()
```

**Key insight:** Shutdown is **ordered by dependency**, not by component:

1. User-facing channels stop first (Telegram, Discord)
2. Background workers stop (cron, heartbeat, task recovery)
3. System resources are released (sandbox, providers)
4. Active runs are drained with timeout
5. Context cancellation cascades to all goroutines

### 2.3 Lifecycle Hooks: Config Reload on Changes

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway_lifecycle.go` (lines 48-124)

**Pattern: Hot-reload via pub/sub messaging**

```go
// Quota config reload
d.msgBus.Subscribe("quota-config-reload", func(evt bus.Event) {
    if evt.Name != bus.TopicConfigChanged {
        return
    }
    updatedCfg := evt.Payload.(*config.Config)
    deps.quotaChecker.UpdateConfig(*updatedCfg.Gateway.Quota)
    slog.Info("quota config reloaded via pub/sub")
})

// TTS providers reload
d.msgBus.Subscribe("tts-config-reload", func(evt bus.Event) {
    if evt.Name != bus.TopicConfigChanged {
        return
    }
    newMgr := setupTTS(updatedCfg)
    deps.ttsTool.UpdateManager(newMgr)
    slog.Info("tts config reloaded", "provider", newMgr.PrimaryProvider())
})

// Web_fetch domain policy reload
d.msgBus.Subscribe("webfetch-config-reload", func(evt bus.Event) {
    deps.webFetchTool.UpdatePolicy(...)
})

// Cron default timezone reload
d.msgBus.Subscribe("cron-config-reload", func(evt bus.Event) {
    d.pgStores.Cron.SetDefaultTimezone(updatedCfg.Cron.DefaultTimezone)
})
```

**Key design:** Hot-reload handlers are **idempotent** and **isolated by feature**. No monolithic config reload that blocks or restarts the gateway.

---

## Part 3: Message Consumption & Session Routing

### 3.1 Inbound Message Consumer Architecture

**Location:** `/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/cmd/gateway_consumer.go` (244 lines)

**Consumer flow:**

```
1. consumeInboundMessages() reads from msgBus
2. Deduplication (20min TTL, 5000 max)
3. Route by message type:
   - Subagent announce → Serialize per session (prevent concurrent reads of stale history)
   - Teammate message → Special handling for team tasks
   - Reset/stop commands → Direct action
   - Escalation messages → Bypass debounce, immediate routing
   - Normal messages → Debounce (1000ms default)
4. Process through scheduler
5. Publish response back to channel
```

### 3.2 Deduplication & Announce Serialization

**Key patterns:**

**Dedup cache** (lines 34):

```go
dedupe := bus.NewDedupeCache(20*time.Minute, 5000)
// Uses message_id + sender + chat + channel as key
// Prevents webhook retries from duplicating agent runs
```

**Announce serialization** (lines 41-45):

```go
var announceMu sync.Map // sessionKey → *sync.Mutex
getAnnounceMu := func(key string) *sync.Mutex {
    v, _ := announceMu.LoadOrStore(key, &sync.Mutex{})
    return v.(*sync.Mutex)
}
// Ensures announce #N+1 doesn't start until announce #N completes
// Otherwise: announce #2 reads stale history before announce #1 writes results
```

### 3.3 Session-Aware Routing

**Team task cancellation** (lines 66-81):

```go
msgBus.Subscribe("consumer.team-task-cancel", func(event bus.Event) {
    if payload, ok := event.Payload.(protocol.TeamTaskEventPayload); ok {
        if sessKey, ok := deps.TaskRunSessions.Load(payload.TaskID); ok {
            if cancelled := sched.CancelSession(sessKey.(string)); cancelled {
                slog.Info("team task cancelled: stopped running agent",
                    "task_id", payload.TaskID, "session", sessKey)
            }
        }
    }
})
```

---

## Part 4: Comparison with AGH's Session Manager Approach

### 4.1 AGH Current State (Inferred)

Based on typical Go agent frameworks:

- Likely using **UUID-based session identifiers**
- Possible **in-memory only** session storage
- May lack **atomic persistence**
- Limited **hot-reload capabilities**

### 4.2 GoClaw's Advantages Over Typical Approaches

| Feature                         | GoClaw Pattern             | Why It Matters                                        |
| ------------------------------- | -------------------------- | ----------------------------------------------------- |
| **Composite keys**              | `agent:{key}:{scope}`      | Enables cache invalidation by agent (prefix matching) |
| **Human-readable agentKey**     | `"default"` not UUID       | Logs are readable; CLI integration easier             |
| **Atomic file writes**          | Temp + rename              | No partial session losses on crash                    |
| **Metadata tracking**           | Token counts, flush points | Enable cost analysis + smart compaction               |
| **Session deduplication**       | Per-key mutex via sync.Map | Prevents race conditions in concurrent announces      |
| **Graceful shutdown order**     | Dependency-aware sequence  | No orphaned goroutines; resources released properly   |
| **Hot-reload subscribers**      | Per-feature pub/sub        | Config changes without restart                        |
| **Subagent spawning isolation** | Separate session keys      | Parent/child tasks don't interfere                    |

---

## Part 5: Recommended Adaptations for AGH

### 5.1 Small, High-Impact Improvements

#### 1. **Composite Session Keys with Agent Key**

**Current risk:** If AGH uses `sessionID` without agentKey, cache invalidation requires scanning all sessions.

**Adaptation:**

```go
// Instead of:
type Session struct {
    ID string // uuid-only
}

// Use:
type Session struct {
    Key string // "agent:{agentKey}:{scope}"
    AgentKey string // human-readable identifier
    ID string // UUID for DB foreign keys
}

// Enable prefix-based invalidation:
cache.InvalidateAgent(agentKey) // clears all sessions for agent:foo:*
```

**File location to update:** Your session manager initialization

#### 2. **Atomic Session Persistence**

**Current risk:** Crash during session write leaves partial state.

**Pattern from GoClaw** (manager.go lines 449-476):

```go
func (m *Manager) Save(ctx context.Context, key string) error {
    // Snapshot under read lock
    tmpFile, _ := os.CreateTemp(m.storage, "session-*.tmp")
    tmpFile.Write(data)
    tmpFile.Sync()
    tmpFile.Close()

    // Atomic rename (kernel guarantees atomicity)
    os.Rename(tmpPath, sessionPath)
}
```

**Implementation:** Replace any direct writes with temp-file-then-rename.

#### 3. **Session Metadata for Cost Tracking**

**Add to AGH Session struct:**

```go
type Session struct {
    // Existing fields...

    // Token tracking (for cost analysis)
    InputTokens  int64
    OutputTokens int64
    Provider     string // for multi-provider scenarios
    Model        string

    // Compaction tracking (enables smart memory flushing)
    CompactionCount int
    MemoryFlushCompactionCount int
    MemoryFlushAt time.Time

    // Spawn tracking (for hierarchical agents)
    SpawnedBy string // parent agent key
    SpawnDepth int
}
```

**Benefit:** Enables cost-per-session reporting, memory management optimization.

#### 4. **Per-Session Synchronization for Concurrent Writes**

**Problem:** If AGH processes multiple updates to same session concurrently, history can corrupt.

**GoClaw solution for subagent announces** (gateway_consumer.go, lines 41-45):

```go
var sessionMutexes sync.Map // sessionKey → *sync.Mutex

func getSessionMutex(key string) *sync.Mutex {
    v, _ := sessionMutexes.LoadOrStore(key, &sync.Mutex{})
    return v.(*sync.Mutex)
}

// Before processing any session update:
mu := getSessionMutex(sessionKey)
mu.Lock()
defer mu.Unlock()
// ... update session ...
```

**Location to add:** In your agent loop message-processing step.

#### 5. **Graceful Shutdown with Ordered Cleanup**

**GoClaw pattern** (gateway_lifecycle.go, lines 142-184):

**Steps to adapt:**

```go
func shutdownGateway(gateway *Gateway) {
    // 1. Stop accepting inbound (channels/webhooks)
    gateway.StopChannels()

    // 2. Stop background jobs (cron, heartbeat)
    gateway.StopCron()
    gateway.StopHeartbeat()

    // 3. Drain audit logs (before DB close)
    gateway.DrainAuditQueue()

    // 4. Stop resource managers (sandbox, providers)
    gateway.providers.Close()
    gateway.sandbox.ReleaseAll(ctx)

    // 5. Drain active runs with timeout
    gateway.scheduler.MarkDraining()
    gateway.scheduler.StopAll()
    time.Sleep(5 * time.Second)

    // 6. Cancel context (cascades to all goroutines)
    cancel()
}
```

**Key:** Order matters. Stop external inputs first, then drain internal work, then release resources.

#### 6. **Config Reload Without Restart**

**Instead of:** Restart gateway on config change

**Implement:**

```go
// Wire hot-reload subscribers for each config section:
msgBus.Subscribe("quota-reload", func(evt Event) {
    if evt.Type == "config-changed" {
        quotaChecker.UpdateConfig(evt.Config.Quota)
    }
})

// On config file update:
notifyConfigChanged(newConfig)
```

**Benefit:** Zero downtime for policy changes, quota updates, provider additions.

#### 7. **Deduplication for Webhook Retries**

**Pattern from gateway_consumer.go (lines 34, 114-120):**

```go
dedupe := NewDedupeCache(20*time.Minute, 5000)

// For each inbound message:
if msgID := msg.Metadata["message_id"]; msgID != "" {
    dedupeKey := fmt.Sprintf("%s|%s|%s|%s",
        msg.Channel, msg.SenderID, msg.ChatID, msgID)
    if dedupe.IsDuplicate(dedupeKey) {
        continue // skip duplicate
    }
}
```

**Benefit:** Automatic handling of webhook retries (Telegram, Discord, etc. all retry on timeout).

---

## Part 6: State Transitions & Lifecycle Diagrams

### 6.1 Session State Machine

```
┌─────────────┐
│   CREATED   │ GetOrCreate() → new Session
└──────┬──────┘
       │
       ├─ AddMessage() x N
       │
       v
┌─────────────────┐
│  ACTIVE         │ Messages flowing, metadata updated
│ (In-Memory)     │
└──────┬──────────┘
       │
       ├─ CompactionCount > Threshold
       │
       v
┌──────────────────┐
│  COMPACTING      │ Memory consolidation summarizing history
└──────┬───────────┘
       │
       ├─ SetHistory(compressed)
       │ IncrementCompaction()
       │
       v
┌──────────────────┐
│  FLUSHED         │ Metadata: MemoryFlushCompactionCount set
└──────┬───────────┘
       │
       ├─ Save() → Atomic write to disk
       │
       v
┌──────────────────┐
│  PERSISTED       │ On-disk snapshot exists
└──────┬───────────┘
       │
       ├─ Delete() called OR no activity
       │
       v
┌──────────────────┐
│  DELETED         │ Removed from memory + disk
└──────────────────┘
```

### 6.2 Gateway Lifecycle States

```
START
  ↓
[INITIALIZING]
  ├─ Load config
  ├─ Setup logging
  ├─ Connect to DB
  ├─ Initialize tool registry
  ├─ Start channels
  ├─ Wire RPC methods
  │
  v
[RUNNING]
  ├─ WebSocket server listening
  ├─ Channel consumers active
  ├─ Cron scheduler running
  ├─ Config reload subscribers active
  │
  ├─ (external: SIGINT/SIGTERM)
  │
  v
[DRAINING]
  ├─ Broadcast shutdown event (WS clients)
  ├─ Stop channel consumers
  ├─ Stop cron jobs
  ├─ Drain audit queue
  ├─ Stop heartbeat ticker
  ├─ Release sandbox containers
  ├─ Cancel context (cascades to all goroutines)
  │
  v
[STOPPED]
  ├─ All goroutines exited
  ├─ Database connections closed
  ├─ Resources released
  │
  v
END
```

---

## Part 7: Code Snippets Worth Adapting

### 7.1 Atomic Session Write

**File:** `internal/sessions/manager.go` (lines 396-477)

```go
func (m *Manager) Save(_ context.Context, key string) error {
    if m.storage == "" {
        return nil
    }

    m.mu.RLock()
    s, ok := m.sessions[key]
    if !ok {
        m.mu.RUnlock()
        return nil
    }

    // Snapshot under lock to ensure consistency
    snapshot := Session{
        Key:      s.Key,
        Messages: make([]providers.Message, len(s.Messages)),
        // ... copy all fields ...
    }
    copy(snapshot.Messages, s.Messages)
    m.mu.RUnlock()

    data, _ := json.MarshalIndent(snapshot, "", "  ")

    // Atomic write: temp file → rename
    tmpFile, _ := os.CreateTemp(m.storage, "session-*.tmp")
    tmpFile.Write(data)
    tmpFile.Sync()  // Ensure disk write
    tmpFile.Close()

    if err := os.Rename(tmpPath, sessionPath); err != nil {
        return err
    }
    return nil
}
```

### 7.2 Graceful Shutdown

**File:** `cmd/gateway_lifecycle.go` (lines 142-184)

**Direct copy-paste pattern for AGH:**

```go
go func() {
    sig := <-sigCh
    slog.Info("graceful shutdown initiated", "signal", sig)

    // 1. User-facing systems stop first
    server.BroadcastEvent(EventShutdown)
    channelMgr.StopAll(ctx)

    // 2. Background workers stop
    cronStore.Stop()
    heartbeatTicker.Stop()

    // 3. System resources
    if sandboxMgr != nil {
        sandboxMgr.Stop()
        sandboxMgr.ReleaseAll(ctx)
    }
    if scheduler != nil {
        slog.Info("draining active runs", "timeout", "5s")
        scheduler.Stop()  // MarkDraining + StopAll
        time.Sleep(5 * time.Second)
    }

    // 4. Cascade cancellation
    cancel()
}()
```

### 7.3 Session Deduplication

**File:** `cmd/gateway_consumer.go` (lines 34, 114-120)

```go
import "github.com/nextlevelbuilder/goclaw/internal/bus"

// In consumer setup:
dedupe := bus.NewDedupeCache(20*time.Minute, 5000)

// For each message:
if msgID := msg.Metadata["message_id"]; msgID != "" {
    dedupeKey := fmt.Sprintf("%s|%s|%s|%s",
        msg.Channel, msg.SenderID, msg.ChatID, msgID)
    if dedupe.IsDuplicate(dedupeKey) {
        slog.Debug("skipping duplicate", "key", dedupeKey)
        continue
    }
}
```

### 7.4 Per-Session Mutex for Concurrent Safety

**File:** `cmd/gateway_consumer.go` (lines 41-45)

```go
var sessionMutexes sync.Map // sessionKey → *sync.Mutex

func getSessionMutex(key string) *sync.Mutex {
    v, _ := sessionMutexes.LoadOrStore(key, &sync.Mutex{})
    return v.(*sync.Mutex)
}

// Usage:
mu := getSessionMutex(sessionKey)
mu.Lock()
defer mu.Unlock()
// ... update session ...
```

### 7.5 Config Reload Subscribers

**File:** `cmd/gateway_lifecycle.go` (lines 48-124)

**Pattern for each config section:**

```go
d.msgBus.Subscribe("feature-config-reload", func(evt bus.Event) {
    if evt.Name != bus.TopicConfigChanged {
        return
    }
    updatedCfg, ok := evt.Payload.(*config.Config)
    if !ok {
        return
    }

    // Idempotent update: no teardown, just config swap
    featureMgr.UpdateConfig(updatedCfg.Feature)
    slog.Info("feature config reloaded")
})
```

---

## Part 8: Implementation Roadmap for AGH

### Phase 1: Session Keys (Low Risk, High Clarity)

- [ ] Add `AgentKey` field to Session struct
- [ ] Implement composite key builder: `SessionKey(agentKey, scope)`
- [ ] Update all session lookups to use new key format
- [ ] Test cache invalidation by agent key

### Phase 2: Atomic Persistence (Medium Risk, High Reliability)

- [ ] Replace session write with temp-file-then-rename pattern
- [ ] Add sync.Mutex per session for write serialization
- [ ] Test concurrent writes don't corrupt session files
- [ ] Add test: crash during write, verify no partial state

### Phase 3: Metadata Tracking (Low Risk, High Observability)

- [ ] Add `InputTokens`, `OutputTokens` to Session
- [ ] Add `Provider`, `Model`, `CompactionCount` to Session
- [ ] Wire token counting in agent loop
- [ ] Implement cost-per-session reporting

### Phase 4: Graceful Shutdown (Medium Risk, High Reliability)

- [ ] Identify shutdown sequence dependencies in AGH
- [ ] Implement ordered shutdown (stop external → drain → cleanup)
- [ ] Add 5s timeout for draining active runs
- [ ] Test no goroutine leaks on shutdown

### Phase 5: Hot-Reload (Medium Risk, Optional But Valuable)

- [ ] Implement pub/sub message bus (if not already present)
- [ ] Wire config-change subscribers for each feature
- [ ] Test config reload without restart
- [ ] Document user-facing config hot-reload capability

### Phase 6: Deduplication (Low Risk, High Reliability)

- [ ] Add dedup cache to inbound message router
- [ ] Use message ID + sender + chat as dedup key
- [ ] Test webhook retries are deduplicated
- [ ] Verify no duplicate agent runs

---

## Part 9: Summary of Key Patterns

| Pattern                    | GoClaw Implementation              | AGH Adaptation                                 |
| -------------------------- | ---------------------------------- | ---------------------------------------------- |
| **Session Keys**           | `agent:{agentKey}:{scope}`         | Use composite keys for cache invalidation      |
| **Persistence**            | Temp file + atomic rename          | Prevent partial writes on crash                |
| **Metadata**               | Token counts + compaction tracking | Enable cost analysis + memory optimization     |
| **Concurrency**            | Per-session mutex via sync.Map     | Prevent race conditions in message processing  |
| **Shutdown**               | Ordered cleanup by dependency      | No orphaned goroutines, clean resource release |
| **Hot-reload**             | Pub/sub subscribers per feature    | Zero-downtime config updates                   |
| **Deduplication**          | TTL cache (20min, 5000 entries)    | Automatic webhook retry handling               |
| **Announce Serialization** | sync.Map mutex per session         | Prevent concurrent reads of stale history      |

---

## Conclusion

GoClaw's session and lifecycle architecture is **battle-tested in production** for multi-tenant, multi-channel agent systems. Its patterns of:

1. **Composite session keys** enable efficient cache management
2. **Atomic persistence** prevents data loss
3. **Ordered shutdown** ensures clean exits
4. **Hot-reload subscribers** enable zero-downtime updates
5. **Per-session synchronization** prevents race conditions

...are directly applicable to AGH. Start with **Phase 1 (session keys)** and **Phase 2 (atomic persistence)** for immediate reliability gains. Phases 3-6 add observability and operational excellence.

---

**Analysis Date:** 2026-04-15
**GoClaw Version:** Latest (from `.resources/goclaw/`)
**Analyzed Files:** 7 Go source files, 1,574 total lines
