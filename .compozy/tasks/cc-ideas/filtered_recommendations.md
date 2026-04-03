# Claude Code Ideas: Filtered for AGH

## Key Architectural Difference

AGH is a **multi-agent orchestration kernel** — it manages external agent processes
(Claude Code, Codex, OpenCode, Pi) via PTY-backed drivers. It does NOT make LLM API
calls directly. This means most of Claude Code's internal harness patterns (query loop,
context compression, streaming, tool execution, caching, cost tracking) happen INSIDE
the driver processes, not in our kernel.

Our kernel's job: session lifecycle, workgroup hierarchy, inter-agent messaging,
state management, prompt assembly, resilience, and dashboard.

---

## NOT Relevant (skip these)

| CC Pattern                                                  | Why It Doesn't Fit                           |
| ----------------------------------------------------------- | -------------------------------------------- |
| Generator-based query loop                                  | We don't make API calls; drivers do          |
| 5-tier context compression (autoCompact, snipCompact, etc.) | Context window is driver-internal            |
| Streaming tool execution                                    | Agents execute their own tools               |
| Bash AST parser / permission classifier                     | Agents handle their own security             |
| Deferred tool loading / ToolSearch                          | Our tools are CLI commands, not API-injected |
| Cost tracking / token budgets                               | Drivers manage their own budgets             |
| Fork-as-a-primitive                                         | We have workgroups for parallelism           |
| Prompt cache boundary marker                                | We don't send cache-control to APIs          |
| BUDDY Tamagotchi                                            | Not relevant for orchestration kernel        |
| VCR test fixtures                                           | We don't make LLM API calls from kernel      |
| Rate limit state machine                                    | Driver-internal concern                      |
| Withhold-recover error pattern                              | Streaming is driver-internal                 |
| Prefetch during streaming                                   | No streaming in our kernel                   |
| Two-stage ML classifier                                     | Permission is driver-internal                |

---

## RELEVANT: High Priority

### 1. Memory System Across Sessions (memdir pattern)

**What CC does**: File-based persistent memory with YAML frontmatter, 4-type taxonomy
(user/feedback/project/reference), MEMORY.md index always in context, staleness warnings,
LLM-powered relevance selection.

**Why it fits us**: Our blackboard is session-scoped and ephemeral (SQLite, dies with session).
We have NO cross-session persistent learning. The meta-learning system exists in config but is
disabled and basic (auto-approve drafts). A memdir-like system would give agents institutional
memory across sessions.

**How to implement**:

- Add `internal/kernel/memdir/` package
- Store at `~/.agh/memory/` (global) and `<workspace>/.agh/memory/` (project)
- MEMORY.md index injected into prompts via assembler
- Staleness warnings on recalled memories (mtime-based)
- Memory types: `user`, `feedback`, `project`, `reference`
- Agents write via `agh memory write` CLI command
- Master agents get full memory context; workers get filtered subset

### 2. Dream Consolidation for Cross-Session Learning

**What CC does**: Background 4-phase cycle (orient/gather/consolidate/prune) that
synthesizes session transcripts into durable memory. Lock file with mtime-as-state.

**Why it fits us**: We already have session transcripts (SQLite events table), ring
buffer output, and blackboard entries. Dream consolidation would distill these into
persistent memory across sessions. This IS our meta-learning system done right.

**How to implement**:

- Add `internal/kernel/dream/` package
- Run after session stop (or as periodic background task in daemon)
- 3-gate triggering: time since last consolidation > N hours, sessions since last > M, lock acquired
- Phase 1: Read existing memories + recent session events/blackboard
- Phase 2: Extract patterns, recurring issues, resolved problems
- Phase 3: Update/create memory files with YAML frontmatter
- Phase 4: Prune outdated memories, keep index under 200 lines
- Lock file at `~/.agh/memory/.consolidate-lock` with mtime-as-state pattern

### 3. Coordinator Prompt: Synthesize Before Delegating

**What CC does**: Coordinator is explicitly forbidden from lazy delegation like
"based on your findings, fix the bug". Must synthesize findings with file paths,
line numbers, and specific instructions before directing workers.

**Why it fits us**: Our master agent prompts already have delegation protocol, but
they don't enforce synthesis. Adding this principle to `internal/prompt/templates/master.md`
would improve master-worker communication quality.

**How to implement**:

- Add to master template: "NEVER delegate with vague instructions. After receiving
  worker results, YOU MUST synthesize the findings into a specific implementation
  spec with file paths, line numbers, and exact changes before directing follow-up work."
- Add anti-pattern examples (bad: "fix the auth bug", good: "fix the null pointer in
  src/auth/validate.go:42, the user field is undefined when sessions expire")

### 4. Structured Task System with Dependencies

**What CC does**: Tasks with id, subject, description, status (pending/running/completed/failed),
owner, blocks/blockedBy fields. Workers claim tasks, coordinator tracks progress.

**Why it fits us**: Our blackboard is unstructured key-value. Status entries track agent
state but not task decomposition. A structured task system would let masters decompose
work and track progress with dependency ordering.

**How to implement**:

- Add `tasks` table to SQLite state store (alongside blackboard/status/events)
- Fields: id, subject, description, status, owner_agent, blocks, blocked_by, created_at
- New CLI commands: `agh task create/update/list/get`
- Master prompt section: "Use `agh task create` to decompose work into trackable units"
- Workers claim tasks via `agh task update --owner <self>`

### 5. Skills Conditional Activation via Path Patterns

**What CC does**: Skills with `paths: ["src/**/*.ts"]` stay dormant until file operations
touch matching paths. Two execution modes: inline (inject into context) vs fork (isolated sub-agent).

**Why it fits us**: Our skills system already has `SKILL.md` with frontmatter but no
conditional activation. Adding path-based triggers and fork mode would make skills
context-aware and prevent bloating the prompt with irrelevant skills.

**How to implement**:

- Add `paths` field to `SkillMeta` frontmatter
- Add `context` field (`inline` | `fork`) to `SkillMeta`
- In skill catalog builder, filter skills by path relevance when workspace context is known
- Fork mode: skill content sent as a separate prompt to a worker agent, results returned

---

## RELEVANT: Medium Priority

### 6. Prompt Assembly: Section Registry Pattern

**What CC does**: Sections registered with `systemPromptSection(name, computeFn)` are
memoized. `DANGEROUS_uncachedSection(name, reason, computeFn)` forces developers to
justify cache-breaking sections.

**Why it fits us**: Our `prompt.Assemble()` re-renders everything on each call. Templates
are cached via `sync.Once` but catalogs (skills, roles, playbooks) are rebuilt every time.
A section registry would add structure and memoization.

**How to implement**:

- Add `SectionRegistry` to prompt package
- Sections: template, specialization, skills_catalog, roles_catalog, playbooks_catalog, context
- Memoize static sections (template, roles catalog) per session
- Mark volatile sections (context, historical state) as dynamic with reason

### 7. Cron Scheduler for Daemon

**What CC does**: Production-grade cron scheduler with per-project locking, deterministic
jitter, missed-task recovery, file watcher.

**Why it fits us**: Our daemon runs persistently but has no scheduling capabilities.
Adding a cron scheduler would enable periodic tasks: health reports, memory consolidation,
workspace scanning, status summaries.

**How to implement**:

- Add `internal/kernel/cron/` package
- Store tasks in `~/.agh/scheduled_tasks.json`
- Per-session and per-daemon task scoping
- Support: `agh schedule create "0 9 * * *" "run morning health check"`
- Integration point: daemon ticker fires cron evaluation on each tick

### 8. Sink Pattern for Analytics/Events

**What CC does**: Events queued in memory until sink attached during init. Decouples
producers from transport layer.

**Why it fits us**: We log events to SQLite but have no analytics/metrics pipeline.
A sink pattern would allow emitting structured events during boot (before logger is
ready) and forwarding to multiple backends (SQLite, file, future remote).

**How to implement**:

- Add `internal/kernel/events/` with `EventBus` + `AttachSink()` pattern
- Default sink: SQLite events table (existing)
- Future sinks: file-based audit log, OpenTelemetry, dashboard WebSocket

### 9. System Reminders for Dynamic Context Injection

**What CC does**: `<system-reminder>` XML tags in user-role messages carry runtime
context without polluting the system prompt.

**Why it fits us**: Currently our assembler puts everything in the system prompt string.
Introducing a separate "context injection" channel via message-level reminders would
allow injecting dynamic state (agent topology, recent events, workspace changes) without
rebuilding the full prompt.

**How to implement**:

- Add `WrapInSystemReminder(content string) string` to prompt package
- Agents receive topology updates, task notifications, and memory recalls as
  system-reminder-tagged messages via `agh send`
- Kernel injects these when forwarding messages through drivers

### 10. Workspace-Level Prompt Template Overrides

**What CC does**: Prompts loaded from multiple sources with override hierarchy.
Custom system prompts per-mode.

**Why it fits us**: Our templates are embedded at compile time (`sync.Once`). Users
can't customize agent behavior per-workspace. Allowing `.agh/prompts/master.md` etc.
would enable project-specific agent tuning.

**How to implement**:

- Check `<workspace>/.agh/prompts/<type>.md` before falling back to embedded template
- Check `~/.agh/prompts/<type>.md` for global user customization
- Resolution: workspace > user home > embedded default

---

## RELEVANT: Low Priority (nice-to-have)

### 11. Team Memory (shared across agents within session)

Our blackboard already serves this role. Could enhance with frontmatter metadata and
type taxonomy to make it more structured.

### 12. Lock File with mtime-as-State

Useful pattern for daemon lock, dream consolidation lock, and any cross-process
coordination. Our daemon lock is flock-based already; the mtime trick is complementary.

### 13. Reactive State Store with onChange

Our dashboard uses WebSocket hub for streaming. A reactive store pattern could
centralize state change notifications and simplify dashboard adapter code.

### 14. Richer Hook Types (prompt, agent, HTTP)

Our hooks are command-only (shell commands forwarded via NATS). Adding prompt-type
hooks (LLM evaluation) and HTTP hooks (webhooks) would enable richer automation.

### 15. BoundedUUIDSet for Message Dedup

Useful if we add bridge/remote communication. Not needed for current NATS-based messaging
which has built-in dedup.
