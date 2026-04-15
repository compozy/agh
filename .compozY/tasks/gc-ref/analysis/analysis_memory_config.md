# GoClaw Memory, Consolidation & Config Patterns — Analysis for AGH

## 1. Memory Persistence Model (3-Tier Architecture)

**Source:** `internal/memory/`, `internal/consolidation/`

### Architecture

- **Working memory** (session) → **Episodic** (summaries) → **Semantic** (knowledge graph)
- Storage: PostgreSQL with pgvector for semantic search
- Query: Full-text search (FTS) + vector similarity hybrid (vector 0.3 + text 0.7, tunable)

### EpisodicStore

- Session summaries with L0 (auto-inject) abstracts
- TTL-based expiration (default 90 days, pruned every 6h)
- Content hashing: SHA256 short digest for dedup
- Scoping: `(AgentID, UserID, TenantID)`
- Idempotency via `SourceID` format: `{sessionKey}:{compactionCount}`

### Chunking Strategy

- Paragraph-aware with overlap (default 1000 chars, 200 char overlap)
- Embedding model: `text-embedding-3-small` (configurable)

---

## 2. Context Compaction/Summarization

**Source:** `internal/consolidation/`, `internal/agent/loop_compact.go`

### Mid-Loop Compaction Strategy

- Summarizes first ~70% of messages, keeps last ~30% intact
- Configurable `keepLastMessages` (default 4)
- Summary prompt preserves task IDs, decisions, statuses, URLs, identifiers **verbatim**
- Max 8000 char excerpt from session before summarization

### CompactionConfig

```go
type CompactionConfig struct {
    ReserveTokensFloor int     // minimum reserve tokens (default 20000)
    MaxHistoryShare    float64 // max % of context for history (default 0.85)
    KeepLastMessages   int     // messages to preserve (default 4)
    MemoryFlush        MemoryFlushConfig
}
```

### Event-Driven Pipeline

1. `SessionCompleted` → episodic worker
2. Episodic worker summarizes + publishes `EpisodicCreated`
3. Semantic worker extracts KG from summary + publishes `EntityUpserted`
4. Dedup worker merges duplicate entities

### L0 Abstract Generation

- Short 1-sentence topic + summary for system prompt injection
- Used for auto-inject relevance ranking

---

## 3. Extractive Memory (Regex Fallback)

**Source:** `internal/agent/extractive_memory.go`

### Mechanism

Regex-based fallback when LLM memory flush returns `NO_REPLY` or fails:

**Categories extracted:**

- **Decisions:** "decided to", "agreed on", "we'll use" patterns
- **Preferences:** "I prefer", "don't do", "always", "never" patterns
- **Technical facts:** "API is", "endpoint is", "version is" + URLs + file paths + dates

**Integration:**

- Runs in `extractiveMemoryFallback()` after LLM flush timeout/error
- Limited to last 20 messages
- Set-based dedup preserving insertion order
- Output: `memory/{YYYY-MM-DD}-auto-extract.md` (appends, never overwrites)

### Adaptation for AGH

```go
// Fallback extraction when LLM-based memory flush fails
func extractFromMessages(msgs []Message) []MemoryEntry {
    var entries []MemoryEntry
    patterns := map[string]*regexp.Regexp{
        "decision":   regexp.MustCompile(`(?i)(decided to|agreed on|we'll use|let's go with)\s+(.+)`),
        "preference": regexp.MustCompile(`(?i)(I prefer|don't do|always|never)\s+(.+)`),
        "fact":       regexp.MustCompile(`(?i)(API is|endpoint is|version is)\s+(.+)`),
    }
    // ... extract + dedup
    return entries
}
```

---

## 4. Knowledge Graph Integration

**Source:** `internal/knowledgegraph/extractor.go`

### Extraction Flow

- LLM-based entity/relation extraction (configurable provider, default Claude)
- Confidence filtering: default 0.75 minimum threshold
- Chunking for texts > 12000 chars (paragraph-aware, merge results)
- Dedup: entities by `external_id` (keep higher confidence), relations by `(source, type, target)` tuple

### Output Normalization

- Entity names + IDs lowercased + trimmed
- Relation types lowercased
- JSON sanitization: fixes malformed decimals ("0. 85" → "0.85"), trailing commas

### Retry Logic

- Truncates to 8000 chars on first length-exceed, retries
- Skips failed chunks (non-fatal) instead of failing entire extraction

---

## 5. Config Loading Chain

**Source:** `internal/config/`

### Priority: File → Env → Defaults → Merge → Validate

1. **Load file:** JSON5 from `GOCLAW_CONFIG` (or default `~/.goclaw/config.json`)
2. **Apply env overrides:** Secrets only from env (e.g., `GOCLAW_ANTHROPIC_API_KEY`), auto-enable channels if credentials provided
3. **Merge with defaults:** `config.Default()` provides hardcoded baseline
4. **Per-agent resolution:** `ResolveAgent(agentID)` merges defaults + per-agent spec
5. **Path expansion:** `ExpandHome("~/.goclaw/...")` resolves to user home

### Per-Agent Override Pattern (Pointer Fields)

```go
type AgentConfig struct {
    Model      *string          // nil = use default
    Provider   *string          // nil = use default
    Sandbox    *SandboxConfig   // nil = use default
    Memory     *MemoryConfig    // nil = use default
    Compaction *CompactionConfig // nil = use default
}
```

Pointer fields allow partial overrides — only non-nil values replace defaults.

### Memory Config

```go
type MemoryConfig struct {
    Enabled           bool    // default true
    EmbeddingProvider string  // auto-select
    EmbeddingModel    string  // text-embedding-3-small
    MaxResults        int     // 6
    MinScore          float64 // 0.35
    VectorWeight      float64 // hybrid search tuning
    TextWeight        float64 // hybrid search tuning
    Dreaming          *DreamingConfig
}
```

---

## 6. Generic Cache with TTL

**Source:** `internal/cache/`

### Interface

```go
type Cache[V any] interface {
    Get(ctx context.Context, key string) (V, bool)
    Set(ctx context.Context, key string, value V, ttl time.Duration)
    Delete(ctx context.Context, key string)
    DeleteByPrefix(ctx context.Context, prefix string)
    Clear(ctx context.Context)
}
```

### InMemoryCache Implementation

- **Thread-safe:** `sync.Map` backed
- **TTL support:** Zero TTL = no expiry
- **Lazy eviction:** On Get, check `expiresAt` and delete if expired
- **Periodic sweep:** Optional background goroutine at configurable interval
- **Size capping:** Optional `maxSize` with oldest-first eviction (evict 20% when exceeded)

### Options Pattern

```go
func NewInMemoryCache[V any](opts ...CacheOption[V]) *InMemoryCache[V]

func WithMaxSize[V any](n int) CacheOption[V]
func WithSweepInterval[V any](d time.Duration) CacheOption[V]
```

---

## 7. Workspace Resolution (6-Scenario Model)

**Source:** `internal/workspace/`

### Scenarios

| Scope         | Path Pattern                       | Memory Scope    |
| ------------- | ---------------------------------- | --------------- |
| Delegate      | delegator's shared workspace       | user (isolated) |
| Team shared   | `teams/{teamID}/`                  | shared          |
| Team isolated | `teams/{teamID}/{userID}/`         | user            |
| Personal open | `{tenantPath}/{agentID}/{userID}/` | user            |
| Predefined    | `{tenantPath}/{agentID}/`          | shared          |

### Key Patterns

- **Single resolution:** `WorkspaceContext` resolved ONCE at run start, immutable for entire run
- **Context propagation:** `WorkspaceContext.FromContext(ctx)` / `WithContext(ctx, wc)`
- **Path traversal defense:** `sanitizeSegment()` — alphanumeric + `-_` only
- **Permission isolation:** 0755 (personal) vs 0750 (team)
- **Tenant scoping:** Master tenant uses base dir directly, non-master uses `base/tenants/{slug}/`

---

## 8. Memory Flush (Pre-Compaction)

**Source:** `internal/agent/memoryflush.go`

### Trigger Conditions

- Session approaching context limit
- Memory flush enabled (default true)
- Not already flushed in this compaction cycle (dedup guard)

### Execution Flow

1. Build system prompt + history summary + flush prompt
2. Call LLM with file-writing tools only (max 5 iterations)
3. If `NO_REPLY` or timeout → fallback to regex extraction
4. Mark flush complete + save session

### Tool Access

Limited to file tools only — no arbitrary execution during memory flush.

---

## Recommended Adaptations for AGH

### Immediate (Low Effort)

1. **Generic `Cache[V]` interface** with TTL + lazy eviction — reusable across session, config, and agent caches
2. **Per-agent config override via pointer fields** — AGH already has TOML config, add pointer-based partial merge
3. **Regex extractive memory fallback** — cheap insurance when LLM memory flush fails
4. **`sanitizeSegment()`** path helper — path traversal defense for workspace paths

### Medium Term

5. **Compaction with verbatim preservation** — keep task IDs, decisions, URLs in summaries
6. **Event-driven consolidation pipeline** (SessionCompleted → episodic → semantic → dedup)
7. **Workspace context resolved once** — immutable per-session, propagated via `context.Context`

### Future

8. **L0 abstract auto-inject** — short summaries from past sessions injected into system prompt
9. **Knowledge graph extraction** with chunking + confidence filtering
10. **Hybrid search** (vector + FTS) for memory retrieval
