# GoClaw Reference Analysis — Consolidated Findings for AGH

> 12 sub-análises cobrindo ~300KB de documentação extraída de 1485 arquivos Go do goclaw.
> Foco: padrões práticos e adaptáveis, não features inteiras.

---

## Índice dos Relatórios Detalhados

| Arquivo                                                              | Foco                                                                                         | Tamanho |
| -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | ------- |
| [`analysis_agent_loop.md`](./analysis_agent_loop.md)                 | Core agent loop, context injection, history, pruning, tool loop, sanitization, orchestration | 45KB    |
| [`analysis_pipeline_hooks.md`](./analysis_pipeline_hooks.md)         | 8-stage pipeline, hooks system, permission model, sandbox, callback wiring                   | 36KB    |
| [`analysis_providers_gateway.md`](./analysis_providers_gateway.md)   | Provider interface, ACP protocol, resolution chain, DI, message processing, consumer         | 43KB    |
| [`analysis_mcp_tools_skills.md`](./analysis_mcp_tools_skills.md)     | MCP lifecycle, tool registry, lazy loading, skill catalog, connection pool                   | 32KB    |
| [`analysis_protocol_testing.md`](./analysis_protocol_testing.md)     | Wire protocol, message bus, RPC dispatch, test helpers, orchestration, feature gating        | 35KB    |
| [`analysis_memory_config.md`](./analysis_memory_config.md)           | 3-tier memory, context compaction, extractive memory, KG, config chain, cache, workspace     | 9KB     |
| [`analysis_safego_concurrency.md`](./analysis_safego_concurrency.md) | Panic recovery, lane scheduler, event bus drain, component lifecycle                         | 16KB    |
| [`analysis_session_lifecycle.md`](./analysis_session_lifecycle.md)   | Session keys, atomic persistence, shutdown ordering, token tracking, dedup                   | 26KB    |
| [`analysis_heartbeat_health.md`](./analysis_heartbeat_health.md)     | Health polling, MCP health loop, failure threshold, wake channel                             | 21KB    |
| [`analysis_store_sqlite.md`](./analysis_store_sqlite.md)             | Per-connection pragmas, schema versioning, dynamic UPDATE, nullable helpers                  | 4KB     |
| [`analysis_error_handling.md`](./analysis_error_handling.md)         | Error classification, HTTPError, RetryDo[T], sentinel errors, user-facing formatting         | 6KB     |
| [`analysis_observability.md`](./analysis_observability.md)           | Span batching, token counting, cost calc, event dedup, OTel export                           | 8KB     |

---

## Top 25 Padrões Extraídos (por impacto para AGH)

### Tier 1 — Fundação de Robustez (copiar/adaptar esta semana)

#### 1. `safego.Recover()` — Panic Recovery Universal

**Source:** `internal/safego/recover.go` (30 LOC)

```go
func Recover(onPanic func(v any), attrs ...any) {
    r := recover()
    if r == nil { return }
    buf := make([]byte, 8192)
    n := runtime.Stack(buf, false)
    slog.Error("goroutine panicked",
        append(attrs, "panic", fmt.Sprint(r), "stack", string(buf[:n]))...)
    if onPanic != nil { onPanic(r) }
}
```

**Todo `go func()` no AGH deveria ter `defer safego.Recover(nil, "component", name)`.**

#### 2. Per-Connection PRAGMA Wrapper para SQLite

**Source:** `internal/store/sqlitestore/pool.go`

AGH aplica pragmas via DSN params — mas novas conexões do pool podem não recebê-las. Sob carga, isso causa deadlocks. O wrapper `pragmaConnector` garante WAL/busy_timeout em **toda** conexão via `sql.OpenDB()`.

#### 3. `RetryDo[T]` — Retry Genérico com Backoff + Jitter

**Source:** `internal/providers/retry.go`

```go
type RetryConfig struct {
    Attempts int           // 3
    MinDelay time.Duration // 100ms
    MaxDelay time.Duration // 30s
    Jitter   float64       // 0.1
}
func RetryDo[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error)
func IsRetryableError(err error, statusCode int) bool
```

Respeita `Retry-After` header. Útil para ACP calls, DB writes, qualquer op transiente.

#### 4. Error Classification Enum + Retryable Flag

**Source:** `internal/providers/error_classify.go`

```go
type FailoverReason string // "auth", "rate_limit", "timeout", "billing", "overloaded", ...
type FailoverClassification struct {
    Reason    FailoverReason
    Retryable bool
}
type ErrorClassifier interface {
    Classify(err error, statusCode int, body string) FailoverClassification
}
```

Separa detecção (o que deu errado) de handling (o que fazer). AGH não tem isso — erros são strings.

#### 5. `context.WithoutCancel()` para Must-Complete Ops

**Source:** `internal/tracing/collector.go`, `internal/pipeline/finalize_stage.go`

Quando sessão é cancelada mas precisa persistir estado final (span completion, session state, memory flush):

```go
detached := context.WithoutCancel(ctx) // preserva valores, remove deadline
opCtx, cancel := context.WithTimeout(detached, 5*time.Second)
defer cancel()
```

Usado no FinalizeStage do pipeline — roda com `context.WithoutCancel` para garantir cleanup.

#### 6. Atomic File Writes (temp + rename)

**Source:** `internal/sessions/manager.go`

```go
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, perm); err != nil { return err }
    return os.Rename(tmp, path) // atomic no mesmo filesystem
}
```

AGH usa file I/O para state mas não faz atomic write — crash durante write = data loss.

---

### Tier 2 — Arquitetura do Agent Loop

#### 7. 8-Stage Pipeline Execution

**Source:** `internal/pipeline/`

```
Setup:     ContextStage (identity, scope, workspace, system prompt, L0 memory inject)
Iteration: ThinkStage → PruneStage → ToolStage → ObserveStage → CheckpointStage
Finalize:  FinalizeStage (sanitize, NO_REPLY detection, atomic persist)
```

- Cada stage é stateless — mutable state vive em `RunState`
- Exit control via 3 sinais: `Continue`, `BreakLoop`, `AbortRun`
- 3-tier message buffer: system / history / pending
- ~50 callback injection points via `PipelineDeps` struct
- Tool execution: parallel I/O + sequential state mutation

**AGH pode adaptar:** A separação em stages dá testabilidade individual. O `RunState` mutável + stages puros é mais limpo que um loop monolítico.

#### 8. Dual Identity (UUID + Key)

**Source:** `internal/agent/loop_context.go`

```go
agentUUID := store.WithAgentID(ctx, l.agentUUID)   // DB PKs, foreign keys
agentKey  := store.WithAgentKey(ctx, l.id)          // logs, paths, filesystem
```

UUID para DB, key humano para logs/paths/UI. Previne scope leaks silenciosos.

#### 9. Two-Pass Context Pruning

**Source:** `internal/agent/pruning.go`, `internal/pipeline/prune_stage.go`

1. **Phase 1 (70% budget):** Soft prune — remove tool results antigos, trunca outputs grandes
2. **Phase 2 (100% budget):** Memory flush + LLM compaction (summarize first 70%, keep last 30%)
3. **Cache-TTL gate:** Per-session, provider-aware — não compacta se cache ainda é válido

```go
if tokenRatio > 0.7 {
    pruneMessages(messages, tokenBudget)  // soft
}
if tokenRatio > 1.0 {
    flushMemory(ctx, session)             // extract memories before losing them
    compactHistory(messages, keepLast: 4) // LLM summarization
}
```

#### 10. 3-Level Tool Loop Detection

**Source:** `internal/agent/toolloop.go`

Previne loops infinitos onde o agent repete as mesmas tool calls:

1. **Same args detection:** Hash determinístico dos argumentos — se tool call idêntica 3x, injeta warning
2. **Read-only streak:** Se últimas N calls são todas read-only (file reads, searches), força break
3. **Same result detection:** Se output idêntico nas últimas 2 calls, break

```go
type loopDetector struct {
    callHashes map[uint64]int  // hash → count
    readOnlyStreak int
    lastResultHash uint64
}
```

#### 11. Input Guard — Injection Detection

**Source:** `internal/agent/input_guard.go`

Valida input do usuário antes de processar:

- Detecta tentativas de prompt injection
- Trunca mensagens excessivamente longas
- Sanitiza caracteres de controle

#### 12. Output Sanitization Pipeline (8 stages)

**Source:** `internal/agent/sanitize.go`

Pipeline de sanitização do output do agent antes de enviar ao usuário:

1. Config leak prevention (remove API keys, tokens do output)
2. Thinking block removal (extended thinking não vai pro user)
3. Directive stripping (system prompt fragments que vazam)
4. Unicode normalization
5. Content truncation
6. Format cleanup

---

### Tier 3 — Provider & Protocol Patterns

#### 13. Minimal Provider Interface (4 métodos)

**Source:** `internal/providers/types.go`

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
    DefaultModel() string
    Name() string
}
```

20+ providers implementam essa interface. Optional capability interfaces (`ThinkingCapable`, etc.) para features específicas.

**AGH:** O `AgentDriver` interface é similar mas pode se beneficiar do `ChatRequest.Options map[string]any` para extensibilidade sem breaking changes.

#### 14. Protocol Frame Demultiplexing

**Source:** `pkg/protocol/frames.go`

```go
type Frame struct {
    Type string          `json:"type"` // "request", "response", "event"
    // Deferred unmarshaling — body parsed only when needed
}
```

3 frame types com unmarshaling adiado. 100+ RPC method constants organizados por priority phases. Structured error with `Retryable` + `RetryAfterMs`.

#### 15. Two-Bus Architecture

**Source:** `internal/bus/`, `internal/eventbus/`

- **MessageBus:** Channel routing (inbound/outbound messages, real-time)
- **DomainEventBus:** Consolidation pipeline (session.completed → episodic → semantic → dedup)

Separação clara entre mensagens de real-time (chat) e eventos de domínio (lifecycle).

#### 16. RPC Method Router com Permission Checks

**Source:** `internal/gateway/methods/`

```go
type MethodRouter struct {
    methods map[string]MethodHandler
}
func (r *MethodRouter) Register(name string, handler MethodHandler, roles ...Role)
```

Permission checks no dispatcher (não no handler). Role-based access + session ownership + team membership.

---

### Tier 4 — MCP, Tools & Skills

#### 17. MCP Dual-Pointer para Reconnect Race-Safe

**Source:** `internal/mcp/manager.go`

```go
type serverState struct {
    client    *mcpclient.Client                // direct ref for health loop (single goroutine)
    clientPtr atomic.Pointer[mcpclient.Client] // shared with BridgeTools (atomic swap on reconnect)
}
```

BridgeTools fazem `clientPtr.Load()` em `Execute()` — race-safe durante reconnect sem locks.

#### 18. Lazy Tool Loading (Threshold-Based)

**Source:** `internal/mcp/manager.go`, `internal/mcp/registry.go`

- < 40 tools: inline (enviados na request ao LLM)
- > = 40 tools: search mode (deferred, ativados on-demand via callbacks)
- 3-phase locking para prevenir deadlock entre Manager e Registry durante ativação

#### 19. Tool Parameter Cleaning

**Source:** `internal/mcp/bridge_tool.go`

LLMs enviam placeholder values nos params ("optional", "null", all-caps):

```go
func cleanParams(params map[string]any) map[string]any {
    for k, v := range params {
        if isPlaceholder(v) { delete(params, k) }
    }
    return params
}
```

#### 20. Skill Catalog com Hot-Reload

**Source:** `internal/skills/`

- 5-tier priority hierarchy para skill matching
- Frontmatter parsing (JSON + YAML) com `{baseDir}` substitution
- BM25 search com optional vector embeddings (hybrid)
- Version tracking por millisecond precision — sem filesystem polling

---

### Tier 5 — Memory, Testing & Cross-Cutting

#### 21. 3-Tier Memory Model

**Source:** `internal/memory/`, `internal/consolidation/`

Working (session) → Episodic (summaries) → Semantic (knowledge graph)

Event-driven pipeline:

1. `SessionCompleted` → episodic worker (summarize)
2. `EpisodicCreated` → semantic worker (extract KG)
3. `EntityUpserted` → dedup worker (merge)

AGH já tem memory — pode adotar o pipeline event-driven para consolidation.

#### 22. Extractive Memory Fallback (Regex)

**Source:** `internal/agent/extractive_memory.go`

Quando LLM memory flush falha/timeout, regex extrai:

- Decisions: "decided to", "agreed on", "we'll use"
- Preferences: "I prefer", "don't do", "always", "never"
- Facts: URLs, file paths, dates, "API is", "version is"

Cheap insurance — 50 LOC que salva memória mesmo quando LLM não coopera.

#### 23. Generic Cache[V] com TTL + Lazy Eviction

**Source:** `internal/cache/`

```go
type Cache[V any] interface {
    Get(ctx context.Context, key string) (V, bool)
    Set(ctx context.Context, key string, value V, ttl time.Duration)
    Delete(ctx context.Context, key string)
    DeleteByPrefix(ctx context.Context, prefix string)
}
```

`sync.Map` backed, lazy eviction on Get, optional periodic sweep, size cap com oldest-first eviction (20%).

#### 24. Test Context Builders (sem DB)

**Source:** `internal/testutil/`

```go
func TenantCtx(tenantID uuid.UUID) context.Context
func UserCtx(tenantID uuid.UUID, userID string) context.Context
func AgentCtx(tenantID, agentID uuid.UUID) context.Context
func FullCtx(tenantID uuid.UUID, userID string, agentID uuid.UUID) context.Context
```

Leves, sem DB, composable. AGH `internal/testutil` pode adotar.

#### 25. Hooks System (Lifecycle Events)

**Source:** `internal/hooks/`

7 lifecycle events: `session_start`, `user_prompt_submit`, `pre_tool_use`, `post_tool_use`, `stop`, `subagent_start`, `subagent_stop`

3 handler types: command, http, prompt. **Fail-closed** (blocking event timeout → block).
Circuit breaker: auto-disable hook após N falhas consecutivas.

---

## Roadmap de Adoção Priorizado

### Fase 1 — Robustez Core (~1 semana, ~200 LOC)

| #   | Item                                             | Esforço | Impacto |
| --- | ------------------------------------------------ | ------- | ------- |
| 1   | `safego.Recover()` em todo `go func()`           | 1h      | Crítico |
| 2   | `pragmaConnector` per-connection para SQLite     | 2h      | Crítico |
| 3   | Atomic file writes (temp + rename)               | 1h      | Alto    |
| 4   | Sentinel errors + `errors.Is()` unificados       | 2h      | Alto    |
| 5   | `context.WithoutCancel()` para must-complete ops | 1h      | Alto    |

### Fase 2 — Error Handling & Retry (~1 semana, ~300 LOC)

| #   | Item                                               | Esforço | Impacto |
| --- | -------------------------------------------------- | ------- | ------- |
| 6   | ErrorKind enum + classificação retryable/permanent | 3h      | Alto    |
| 7   | `RetryDo[T]` genérico com backoff + jitter         | 3h      | Alto    |
| 8   | `HTTPError` custom type com `errors.As()`          | 1h      | Médio   |
| 9   | User-facing error formatter                        | 2h      | Médio   |
| 10  | `containsAny()` helper                             | 0.5h    | Baixo   |

### Fase 3 — Agent Loop Hardening (~2 semanas, ~500 LOC)

| #   | Item                                                 | Esforço | Impacto |
| --- | ---------------------------------------------------- | ------- | ------- |
| 11  | Two-pass context pruning (soft + hard)               | 8h      | Alto    |
| 12  | Tool loop detection (3 levels)                       | 4h      | Alto    |
| 13  | Output sanitization pipeline (config leak, thinking) | 4h      | Alto    |
| 14  | Input guard (injection detection, truncation)        | 3h      | Médio   |
| 15  | Dedup set com TTL para eventos                       | 2h      | Médio   |

### Fase 4 — Observability & Memory (~2 semanas)

| #   | Item                                                           | Esforço | Impacto |
| --- | -------------------------------------------------------------- | ------- | ------- |
| 16  | Token counter interface (fallback rune/3 primeiro, BPE depois) | 4h      | Alto    |
| 17  | Cost calculation com reasoning token split                     | 2h      | Médio   |
| 18  | Event-driven memory consolidation pipeline                     | 8h      | Médio   |
| 19  | Extractive memory fallback (regex)                             | 2h      | Médio   |
| 20  | Wake channel pattern para polling services                     | 1h      | Baixo   |

### Fase 5 — Architecture Refinement (futuro)

| #   | Item                                             | Esforço | Impacto |
| --- | ------------------------------------------------ | ------- | ------- |
| 21  | Staged pipeline (RunState + stateless stages)    | 16h     | Alto    |
| 22  | Generic `Cache[V]` com TTL                       | 3h      | Médio   |
| 23  | Test context builders (TenantCtx, UserCtx, etc.) | 2h      | Médio   |
| 24  | Hooks system com circuit breaker                 | 8h      | Médio   |
| 25  | MCP dual-pointer para reconnect race-safe        | 4h      | Baixo   |

### NÃO adaptar agora

- Multi-provider failover (2-tier) — AGH não precisa de fallback entre providers
- Full event bus com worker pool — AGH usa Notifier pattern, suficiente pro alpha
- OTel export — prematuro (build-tag gating é bom pattern mas não é prioridade)
- Sandbox Docker — AGH não executa código arbitrário (agents fazem isso)
- i18n system — premature
- Knowledge graph extraction — Phase 2+ do AGH
- Feature edition gating — premature
- Connection pool multi-tenant — AGH é single-tenant local-first

---

## Conclusão

O GoClaw é um sistema maduro (~1500 Go files, multi-tenant, production) que compartilha DNA com o AGH. Os padrões mais valiosos dividem-se em duas categorias:

**Infraestrutura de segurança** (Fases 1-2): `safego.Recover`, pragmaConnector, retry genérico, error classification, atomic writes. ~500 LOC total, impacto desproporcional na robustez.

**Hardening do agent loop** (Fases 3-4): Two-pass pruning, loop detection, sanitization, token counting. ~500 LOC total, previne classes inteiras de bugs (loops infinitos, context overflow, data leaks).

**Princípio guia: copiar a infraestrutura de segurança e os guardrails do loop, não as features.**
