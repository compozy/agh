# Memory & Autonomous Features Analysis

## How It Works (in Claude Code)

### 1. Memory System (memdir/)

The memory system is a **file-based, directory-scoped persistent memory** that survives across sessions. It is structured around a per-project directory at `~/.claude/projects/<sanitized-git-root>/memory/`.

**Core Architecture:**

- **MEMORY.md** is the entrypoint index file. It is always loaded into the system prompt context at session start. Capped at 200 lines / 25KB to prevent context bloat. Each line is a pointer to a topic file: `- [Title](file.md) -- one-line hook`.
- **Topic files** are individual markdown files with YAML frontmatter (`name`, `description`, `type`) containing the actual memory content. There is a closed four-type taxonomy:
  - `user` -- who the user is, their role, preferences, knowledge level
  - `feedback` -- guidance on how to approach work (corrections AND confirmations)
  - `project` -- ongoing work context not derivable from code/git
  - `reference` -- pointers to external systems (Linear, Grafana, Slack)
- **Team memory** (`memory/team/`) is a subdirectory for shared team memories, synced across users. Has its own `MEMORY.md` index. Scoping rules are per-type: `user` is always private, `feedback` defaults private, `project` biases team, `reference` is usually team.
- **Daily logs** (KAIROS mode): `memory/logs/YYYY/MM/YYYY-MM-DD.md` -- append-only timestamped bullets. A separate nightly dream process distills these into topic files.

**Staleness Handling (`memoryAge.ts`):**

Memories include staleness warnings based on file mtime. Memories older than 1 day get explicit caveats: "This memory is N days old. Claims about code behavior or file:line citations may be outdated. Verify against current code before asserting as fact." This addresses a critical real-world problem: stale memories being asserted as current truth.

**Relevance Selection (`findRelevantMemories.ts`):**

When a new user query arrives, a background Sonnet call selects up to 5 relevant memories from the full manifest (filename + frontmatter description). This avoids loading all memories into context -- only the relevant ones are injected. The system uses `sideQuery` (a lightweight parallel API call) to run the selection.

**Memory Scanning (`memoryScan.ts`):**

Scans the memory directory for .md files, reads their frontmatter (first 30 lines), and returns a sorted (newest-first) list capped at 200 files. Each file's `MemoryHeader` includes: filename, filePath, mtimeMs, description, type.

**Security:**

The team memory path system (`teamMemPaths.ts`) has extensive path traversal protection: null byte rejection, URL-encoded traversal detection, Unicode normalization attack prevention, symlink resolution via `realpathDeepestExisting()`, and dangling symlink detection. The `autoMemoryDirectory` setting is only accepted from trusted sources (user/local/policy settings, NOT project settings that a malicious repo could commit).

### 2. Memory Extraction (extractMemories/)

This is a **background subagent** that runs after every model response to automatically extract durable memories from the conversation.

**How it fires:**

- Registered as a "stop hook" -- runs after each complete query loop (when the model produces a final response with no tool calls)
- Fire-and-forget from `handleStopHooks` in `stopHooks.ts`
- Throttled by a configurable turn frequency (`tengu_bramble_lintel`, default every turn)
- Uses a cursor (`lastMemoryMessageUuid`) to only process new messages since last extraction
- Mutual exclusion: if the main agent already wrote memories, the extraction agent skips that range

**How it works:**

- Creates a "forked agent" -- a perfect fork of the main conversation that shares the parent's prompt cache
- The fork gets a restricted tool set: Read/Grep/Glob (unrestricted), read-only Bash, and Edit/Write only within the memory directory
- Limited to 5 turns max to prevent rabbit-holes
- Pre-injects the existing memory manifest so the agent doesn't waste a turn on `ls`
- Efficient strategy is prescribed: turn 1 = parallel reads, turn 2 = parallel writes

**Overlap Prevention:**

- `hasMemoryWritesSince()` checks if the main agent already wrote to memory in the current range
- `inProgress` flag prevents overlapping runs
- Stashed context pattern: if a new extraction request arrives during an in-progress run, the latest context is stashed and a "trailing extraction" runs after the current one completes

### 3. Dream Consolidation (autoDream/)

Dream is a **background memory consolidation process** that synthesizes daily logs and session transcripts into durable, well-organized topic files. It is conceptually a "nightly distillation" pass.

**Gate Order (cheapest first):**

1. **Time gate**: hours since `lastConsolidatedAt >= minHours` (default 24h, one `stat` call)
2. **Session gate**: count of session transcripts with mtime > lastConsolidatedAt >= minSessions (default 5)
3. **Lock gate**: no other process mid-consolidation

**The 4-Phase Consolidation Prompt:**

1. **Orient** -- `ls` memory directory, read MEMORY.md, skim existing topic files to avoid duplicates
2. **Gather** -- search daily logs, find drifted memories, grep JSONL transcripts for narrow terms
3. **Consolidate** -- merge new signal into existing topic files, convert relative dates to absolute, delete contradicted facts
4. **Prune** -- update MEMORY.md index (keep under 200 lines / 25KB), remove stale pointers, demote verbose entries, resolve contradictions

**Lock Mechanism (`consolidationLock.ts`):**

Uses a lock file (`.consolidate-lock`) whose mtime doubles as `lastConsolidatedAt`. The body is the holder's PID. Stale-PID detection: if the PID is dead or lock is older than 1 hour, it is reclaimed. Race condition handling: two reclaimers both write, then re-read to verify -- loser bails. On failure, the lock mtime is rolled back so the time gate passes again next session.

**Task UI (`DreamTask.ts`):**

The dream agent is visible in the footer pill and background tasks dialog. Tracks phases (starting -> updating), files touched, and assistant turns. Supports user-initiated kill with lock mtime rollback.

### 4. KAIROS / Proactive Mode

KAIROS is the **autonomous/daemon mode** where Claude Code runs continuously, receiving periodic `<tick>` prompts that keep it alive between user interactions.

**Tick-Based Architecture:**

- The system sends `<tick>` prompts periodically with the user's current local time
- Multiple ticks may be batched into a single message -- the agent processes only the latest
- On first wake-up: greet user briefly, ask what they want to work on (do NOT explore unprompted)
- On subsequent wake-ups: look for useful work, investigate, reduce risk, build understanding

**Sleep Tool:**

The Sleep tool controls pacing -- the agent sleeps when idle to avoid burning API calls, but the prompt cache expires after 5 minutes of inactivity, so there is a tradeoff. Key rule: "If you have nothing useful to do on a tick, you MUST call Sleep."

**Terminal Focus Awareness:**

- **Unfocused**: User is away -- lean into autonomous action, make decisions, explore, commit, push
- **Focused**: User is watching -- be more collaborative, surface choices, ask before large changes

**Cron Scheduler (`cronScheduler.ts`):**

A full cron scheduler with:

- Standard 5-field cron expressions in user's local timezone
- Both recurring and one-shot tasks
- Durable tasks persisted to `.claude/scheduled_tasks.json` (survive restarts)
- Session-only tasks that die with the process
- Per-project scheduler lock to prevent double-firing when multiple sessions share a cwd
- Deterministic jitter to distribute load off the :00 mark
- Recurring tasks auto-expire after 7 days (configurable)
- Missed task detection on startup with user confirmation before execution
- File watcher (chokidar) for live task list changes

**Loop Skill:**

The `/loop` command creates recurring cron jobs from human-friendly intervals (e.g., `/loop 5m /foo`). Gated behind `AGENT_TRIGGERS + isKairosCronEnabled()`.

### 5. Session Memory (`SessionMemory/`)

A separate system from the persistent memory (memdir). Session memory maintains a markdown file with notes about the **current conversation** -- a running scratchpad updated periodically.

**Trigger Conditions:**

- Minimum token threshold to initialize (avoids extraction on trivially short conversations)
- Both token growth AND tool call count thresholds must be met between updates
- OR: token threshold met AND no tool calls in last turn (natural conversation break)

**Integration with Auto-Compact:**

Session memory feeds into context compaction -- when the conversation grows too long and needs summarization, the session memory provides a high-quality summary that was built incrementally.

### 6. BUDDY Gamification System (buddy/)

A **Tamagotchi-like companion** that lives in the CLI. Feature-gated behind `BUDDY`.

**Deterministic Character Generation (`companion.ts`):**

- Uses a seeded PRNG (Mulberry32) with `hash(userId + salt)` as the seed
- The same user always gets the same companion (species, eyes, hat, stats, rarity)
- 18 species: duck, goose, blob, cat, dragon, octopus, owl, penguin, turtle, snail, ghost, axolotl, capybara, cactus, robot, rabbit, mushroom, chonk
- 6 eye styles, 8 hat types (common rarity = no hat)
- 5 rarity tiers with weighted distribution: common(60), uncommon(25), rare(10), epic(4), legendary(1)
- 5 stats: DEBUGGING, PATIENCE, CHAOS, WISDOM, SNARK -- with one peak stat and one dump stat
- 1% chance of "shiny" variant

**Bones vs Soul Architecture:**

- **Bones** (deterministic): rarity, species, eye, hat, shiny, stats -- regenerated from `hash(userId)` on every read, never persisted. This means species renames can't break stored companions and users can't fake a rarity by editing config.
- **Soul** (model-generated): name, personality -- stored in config after first "hatch"
- The stored config is `CompanionSoul & { hatchedAt: number }`, bones are merged at read time

**Visual System (`sprites.ts`):**

Each species has 3 ASCII art frames (5 lines x 12 chars) for idle fidget animation. Frame sequence: mostly rest, occasional fidget, rare blink. Hats render on the top line when the frame allows it.

**Speech Bubble System (`CompanionSprite.tsx`):**

- Bubbles show for ~10 seconds (20 ticks at 500ms), with a 3-second fade window
- A `renderFace()` function generates inline emoji-style faces for compact contexts
- The companion "sits beside the user's input box and occasionally comments"
- When the user addresses the companion by name, the bubble answers (the main agent stays out of the way)

**Engagement Mechanics:**

- Teaser window: April 1-7, 2026 -- rainbow `/buddy` notification on startup when no companion hatched
- `/buddy pet` interaction with floating heart animation (2.5 seconds)
- Companion is always present once hatched, creating a persistent relationship

### 7. Migrations (migrations/)

Simple, focused migration functions that run at startup:

- Model version migrations (Sonnet 4.5 -> 4.6, Opus 4.0 -> 4.6, etc.)
- Settings migrations (auto-updates preference, bypass permissions)
- Each migration is idempotent and guards against re-running
- Pattern: check current state, skip if already migrated, transform, log event

---

## Key Patterns Worth Adopting

### 1. File-Based Memory with YAML Frontmatter and Index

```typescript
// Memory directory structure:
// ~/.claude/projects/<slug>/memory/
//   MEMORY.md           -- index (always in context, max 200 lines)
//   user_role.md         -- topic file with frontmatter
//   feedback_testing.md  -- topic file with frontmatter
//   project_auth.md      -- topic file with frontmatter

// Frontmatter format:
// ---
// name: User Role
// description: Senior Go engineer, new to frontend
// type: user
// ---
// Content here...
```

This is brilliant because: (a) MEMORY.md is the only file always loaded, keeping context small; (b) topic files are demand-loaded based on relevance; (c) frontmatter enables machine-readable metadata for selection; (d) files can be grepped, edited, and version-controlled.

### 2. Staleness-Aware Memory Recall

```typescript
export function memoryFreshnessText(mtimeMs: number): string {
  const d = memoryAgeDays(mtimeMs);
  if (d <= 1) return "";
  return (
    `This memory is ${d} days old. ` +
    `Memories are point-in-time observations, not live state — ` +
    `claims about code behavior or file:line citations may be outdated. ` +
    `Verify against current code before asserting as fact.`
  );
}
```

Every recalled memory older than 1 day gets an explicit warning. This prevents the model from asserting stale information as fact -- a critical real-world failure mode.

### 3. Forked Subagent Pattern for Background Work

```typescript
// extractMemories and autoDream both use this pattern:
const result = await runForkedAgent({
  promptMessages: [createUserMessage({ content: prompt })],
  cacheSafeParams: createCacheSafeParams(context), // shares parent's prompt cache
  canUseTool: createAutoMemCanUseTool(memoryDir), // restricted tool access
  querySource: "extract_memories",
  forkLabel: "extract_memories",
  skipTranscript: true, // don't pollute main transcript
  maxTurns: 5, // hard cap on agent work
});
```

The "perfect fork" shares the parent conversation's prompt cache (massive cost savings) but has restricted tool permissions and runs independently. This is the pattern for all background autonomous work.

### 4. Three-Gate Triggering for Background Tasks

```typescript
// autoDream uses cheapest-first gate ordering:
// Gate 1: Time check (one stat call)
const hoursSince = (Date.now() - lastAt) / 3_600_000;
if (hoursSince < cfg.minHours) return;

// Gate 2: Session count (directory scan, throttled)
const sessionIds = await listSessionsTouchedSince(lastAt);
if (sessionIds.length < cfg.minSessions) return;

// Gate 3: Lock acquisition (filesystem atomic)
const priorMtime = await tryAcquireConsolidationLock();
if (priorMtime === null) return;
```

Gates are ordered from cheapest to most expensive. The scan is throttled separately (10-minute cooldown) to avoid repeated directory scans when the time gate passes but session gate doesn't.

### 5. Lock File with mtime-as-State

```typescript
// The lock file's mtime IS the lastConsolidatedAt timestamp
// The body IS the holder's PID
const LOCK_FILE = ".consolidate-lock";

// Stale-PID detection with race-condition handling:
async function tryAcquireConsolidationLock(): Promise<number | null> {
  // Read stat + body atomically
  // If PID is dead or lock is >1h old, reclaim
  // Write our PID
  // Re-read to verify we won the race (CAS pattern)
}

// Rollback on failure: rewind mtime to pre-acquire
async function rollbackConsolidationLock(priorMtime: number): Promise<void> {
  await writeFile(path, ""); // clear PID
  const t = priorMtime / 1000;
  await utimes(path, t, t); // rewind mtime
}
```

This is an elegant multi-purpose single file: state storage (mtime), lock ownership (PID body), and stale detection (PID liveness + age).

### 6. Mutual Exclusion Between Main Agent and Background Agent

```typescript
// If the main agent already wrote memories, skip extraction:
function hasMemoryWritesSince(messages, sinceUuid): boolean {
  // Scan assistant messages for Edit/Write tool_use blocks targeting auto-memory paths
}

// In runExtraction:
if (hasMemoryWritesSince(messages, lastMemoryMessageUuid)) {
  // Advance cursor past this range, don't fork
  return;
}
```

The main agent and the background extraction agent are mutually exclusive per turn. This prevents duplicate work and conflicts.

### 7. Coalesced Trailing Runs

```typescript
// If extraction is already running, stash context for a trailing run:
if (inProgress) {
  pendingContext = { context, appendSystemMessage };
  return; // fast return, promise resolves quickly
}

// In finally block of runExtraction:
const trailing = pendingContext;
pendingContext = undefined;
if (trailing) {
  await runExtraction({ ...trailing, isTrailingRun: true });
}
```

This prevents overlapping runs while ensuring no work is lost. Only the latest stashed context matters (overwrites previous) since it has the most messages.

### 8. Closed Type Taxonomy with Explicit Exclusions

```typescript
export const MEMORY_TYPES = ["user", "feedback", "project", "reference"] as const;

// Explicit "what NOT to save" section prevents noise:
// - Code patterns, conventions, architecture (derivable from code)
// - Git history (git log is authoritative)
// - Debugging solutions (the fix is in the code)
// - Anything in CLAUDE.md
// - Ephemeral task details

// Even explicit user requests are filtered:
// "These exclusions apply even when the user explicitly asks you to save.
//  If they ask you to save a PR list or activity summary, ask what was
//  *surprising* or *non-obvious* about it -- that is the part worth keeping."
```

### 9. Deterministic Companion Generation from User ID

```typescript
// Bones are NEVER persisted -- regenerated from hash every time
function roll(userId: string): Roll {
  const rng = mulberry32(hashString(userId + SALT));
  const rarity = rollRarity(rng);
  const bones: CompanionBones = {
    rarity,
    species: pick(rng, SPECIES),
    eye: pick(rng, EYES),
    hat: rarity === "common" ? "none" : pick(rng, HATS),
    shiny: rng() < 0.01,
    stats: rollStats(rng, rarity),
  };
  return { bones, inspirationSeed: Math.floor(rng() * 1e9) };
}
```

This is anti-cheat by design: the user can't edit their way to a legendary because the rarity is derived from their user ID, not stored.

### 10. Cron Scheduler with Lock, Jitter, and Missed-Task Recovery

The cron scheduler is production-grade:

- Per-project lock prevents double-firing across sessions
- Deterministic jitter distributes load off :00
- Missed one-shot tasks detected on startup with user confirmation
- File watcher for live updates
- Session-only vs durable task separation
- Recurring task aging and auto-expiry

---

## Ideas for Our System

### 1. Go-Based Memory Directory System

Implement a `memdir` package in our Go kernel with the same architecture:

```go
// internal/kernel/memdir/memdir.go
type MemoryStore struct {
    baseDir     string          // ~/.agh/projects/<slug>/memory/
    indexFile   string          // MEMORY.md
    maxLines    int             // 200
    maxBytes    int             // 25000
}

type MemoryHeader struct {
    Filename    string
    FilePath    string
    ModTime     time.Time
    Description string
    Type        MemoryType      // user | feedback | project | reference
}

type MemoryType string
const (
    MemoryTypeUser      MemoryType = "user"
    MemoryTypeFeedback  MemoryType = "feedback"
    MemoryTypeProject   MemoryType = "project"
    MemoryTypeReference MemoryType = "reference"
)
```

Key: use YAML frontmatter for machine-readable metadata, keep MEMORY.md as a lightweight index always loaded into context, and add mtime-based staleness warnings to recalled memories.

### 2. Background Memory Extraction via Goroutine

Instead of the "forked agent" pattern (which requires API call sharing), we can use a goroutine-based background worker:

```go
// internal/kernel/memory_extractor.go
type MemoryExtractor struct {
    mu              sync.Mutex
    lastCursorUUID  string
    inFlight        atomic.Bool
    pendingCtx      *ExtractionContext  // coalesced trailing run
    memDir          string
    sessionManager  *SessionManager
}

func (e *MemoryExtractor) OnTurnComplete(ctx context.Context, messages []Message) {
    if e.inFlight.Load() {
        e.mu.Lock()
        e.pendingCtx = &ExtractionContext{Messages: messages}
        e.mu.Unlock()
        return
    }
    go e.runExtraction(ctx, messages)
}
```

This follows the same mutual exclusion and coalesced trailing run patterns from Claude Code.

### 3. Dream Consolidation as a Periodic Kernel Service

Implement the 4-phase dream cycle as a background service in our daemon:

```go
// internal/kernel/dream.go
type DreamService struct {
    memDir       string
    lockPath     string
    minHours     float64    // 24
    minSessions  int        // 5
    scanInterval time.Duration
}

func (d *DreamService) ShouldRun(ctx context.Context) bool {
    // Gate 1: Time (cheapest)
    lastAt, _ := d.readLastConsolidatedAt()
    if time.Since(lastAt).Hours() < d.minHours { return false }

    // Gate 2: Session count (throttled scan)
    sessions := d.listSessionsSince(lastAt)
    if len(sessions) < d.minSessions { return false }

    // Gate 3: Lock
    return d.tryAcquireLock()
}
```

The consolidation prompt itself can be injected into a background agent session with restricted tool access (read-only bash, write only to memory dir).

### 4. Tick-Based Daemon Mode in Our Existing Daemon

Our `internal/cli/daemon.go` already has the daemon concept. Extend it with tick-based autonomous behavior:

```go
// internal/kernel/daemon_ticker.go
type DaemonTicker struct {
    interval    time.Duration
    kernel      *Kernel
    scheduler   *CronScheduler
    focus       TerminalFocus   // focused | unfocused
}

func (t *DaemonTicker) Run(ctx context.Context) {
    ticker := time.NewTicker(t.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case now := <-ticker.C:
            t.kernel.InjectTick(now)
        }
    }
}
```

The Sleep tool equivalent maps to our prompt assembler -- when the agent has nothing to do, it signals "sleep N seconds" and the ticker adjusts.

### 5. Cron Scheduler for Go

Port the cron scheduler with the same lock/jitter/recovery model:

```go
// internal/kernel/cron.go
type CronScheduler struct {
    mu          sync.Mutex
    tasks       []CronTask
    nextFireAt  map[string]time.Time
    lockPath    string
    isOwner     bool
    watcher     *fsnotify.Watcher
}

type CronTask struct {
    ID        string    `json:"id"`
    Cron      string    `json:"cron"`
    Prompt    string    `json:"prompt"`
    Recurring bool      `json:"recurring"`
    Durable   bool      `json:"durable"`
    CreatedAt int64     `json:"createdAt"`
}
```

Store tasks in `.agh/scheduled_tasks.json`. Use file-based locking to prevent double-firing across processes. Add deterministic jitter via a hash of the task ID.

### 6. Companion/Gamification System

A lightweight engagement system could live in our CLI:

```go
// internal/cli/companion.go
type Companion struct {
    Species    string
    Name       string         // model-generated
    Personality string        // model-generated
    Rarity     Rarity
    HatchedAt  time.Time
    Stats      map[string]int // DEBUGGING, PATIENCE, etc.
}

// Deterministic generation from user ID
func RollCompanion(userID string) CompanionBones {
    rng := mulberry32(fnvHash(userID + salt))
    // Same deterministic roll as Claude Code
}
```

The key insight: bones are never stored, only the "soul" (name + personality). This prevents cheating and allows species roster updates without breaking existing companions.

### 7. Memory Relevance Selection via Lightweight LLM Call

When loading context for a new session, use a fast/cheap model to select relevant memories:

```go
// internal/kernel/memory_recall.go
func (s *MemoryStore) FindRelevant(ctx context.Context, query string, limit int) ([]Memory, error) {
    headers := s.ScanHeaders(ctx)
    manifest := formatManifest(headers)

    // Use a fast model (Haiku/Sonnet-equivalent) to select
    selected, err := s.sideQuery(ctx, selectMemoriesPrompt, manifest, query)
    if err != nil {
        return nil, err
    }

    return s.loadMemories(selected, limit)
}
```

### 8. Lock File with mtime-as-State Pattern

This is a generally useful pattern for any coordination between multiple processes sharing a project directory:

```go
// internal/kernel/lockfile.go
type MtimeLock struct {
    path       string
    stalePIDMs int64  // 1 hour
}

func (l *MtimeLock) TryAcquire() (priorMtime time.Time, ok bool, err error) {
    // Read stat + PID
    // If PID dead or lock stale, reclaim
    // Write our PID
    // Re-read to verify we won the race
}

func (l *MtimeLock) Rollback(priorMtime time.Time) error {
    // Clear PID, rewind mtime
}
```

### 9. Explicit "What NOT to Remember" in System Prompt

Include explicit exclusion lists in our prompt assembler to prevent memory pollution:

- Do NOT save code patterns, architecture, file paths (derivable from project)
- Do NOT save git history (git log is authoritative)
- Do NOT save debugging solutions (fix is in the code)
- Do NOT save anything in CLAUDE.md / configuration files
- Even when user says "remember this", filter for the non-obvious part

### 10. Structured Migration Pattern

```go
// internal/kernel/migrations.go
type Migration struct {
    Name    string
    Run     func(config *Config) error
}

var migrations = []Migration{
    {Name: "v1_to_v2_model_alias", Run: migrateModelAlias},
    {Name: "v2_settings_format", Run: migrateSettingsFormat},
}

func RunMigrations(config *Config) {
    for _, m := range migrations {
        if config.CompletedMigrations[m.Name] { continue }
        if err := m.Run(config); err != nil {
            slog.Warn("migration failed", "name", m.Name, "err", err)
            continue
        }
        config.CompletedMigrations[m.Name] = true
    }
}
```

---

## Key Files Reference

### Memory System (memdir/)

- `memdir/memdir.ts` -- Core memory prompt builder, MEMORY.md loading/truncation, directory management. Contains `buildMemoryLines()`, `buildMemoryPrompt()`, `buildAssistantDailyLogPrompt()`, `loadMemoryPrompt()`.
- `memdir/paths.ts` -- Memory directory path resolution, auto-memory enable/disable logic, path validation with security checks. Contains `getAutoMemPath()`, `isAutoMemoryEnabled()`, `isExtractModeActive()`, `getAutoMemDailyLogPath()`.
- `memdir/memoryTypes.ts` -- The closed four-type taxonomy (user/feedback/project/reference). Contains type definitions, `TYPES_SECTION_INDIVIDUAL`, `TYPES_SECTION_COMBINED`, `WHAT_NOT_TO_SAVE_SECTION`, `WHEN_TO_ACCESS_SECTION`, `TRUSTING_RECALL_SECTION`, frontmatter examples.
- `memdir/memoryAge.ts` -- Staleness calculation and human-readable age strings. Contains `memoryAgeDays()`, `memoryAge()`, `memoryFreshnessText()`, `memoryFreshnessNote()`.
- `memdir/memoryScan.ts` -- Directory scanning, frontmatter parsing, manifest formatting. Contains `scanMemoryFiles()`, `formatMemoryManifest()`.
- `memdir/findRelevantMemories.ts` -- LLM-powered relevance selection for memory recall. Contains `findRelevantMemories()`, `selectRelevantMemories()` with Sonnet-based side query.
- `memdir/teamMemPaths.ts` -- Team memory path resolution and security (path traversal protection, symlink resolution). Contains `getTeamMemPath()`, `isTeamMemPath()`, `validateTeamMemWritePath()`, `validateTeamMemKey()`.
- `memdir/teamMemPrompts.ts` -- Combined (private+team) memory prompt builder. Contains `buildCombinedMemoryPrompt()`.

### Dream Consolidation (services/autoDream/)

- `services/autoDream/autoDream.ts` -- Main auto-dream orchestrator. Three-gate triggering (time/sessions/lock), forked agent execution, DreamTask registration, progress watching. Contains `initAutoDream()`, `executeAutoDream()`.
- `services/autoDream/config.ts` -- Feature gate and threshold configuration. Contains `isAutoDreamEnabled()`.
- `services/autoDream/consolidationLock.ts` -- Lock file management with mtime-as-state. Contains `readLastConsolidatedAt()`, `tryAcquireConsolidationLock()`, `rollbackConsolidationLock()`, `listSessionsTouchedSince()`, `recordConsolidation()`.
- `services/autoDream/consolidationPrompt.ts` -- The 4-phase dream prompt (orient/gather/consolidate/prune). Contains `buildConsolidationPrompt()`.

### Memory Extraction (services/extractMemories/)

- `services/extractMemories/extractMemories.ts` -- Background extraction subagent. Cursor tracking, mutual exclusion with main agent, coalesced trailing runs, restricted tool permissions. Contains `initExtractMemories()`, `executeExtractMemories()`, `drainPendingExtraction()`, `createAutoMemCanUseTool()`.
- `services/extractMemories/prompts.ts` -- Extraction prompt templates for auto-only and combined (auto+team) modes. Contains `buildExtractAutoOnlyPrompt()`, `buildExtractCombinedPrompt()`.

### Session Memory

- `services/SessionMemory/sessionMemory.ts` -- Per-session running notes system, threshold-based extraction, integration with auto-compact. Contains `shouldExtractMemory()`, `initSessionMemory()`, `manuallyExtractSessionMemory()`.

### KAIROS / Autonomous Mode

- `_prompts/system-prompt-proactive.md` -- The autonomous mode system prompt with tick handling, pacing rules, terminal focus awareness, bias toward action.
- `_prompts/tool-sleep.md` -- Sleep tool description for pacing autonomous work.
- `_prompts/tool-cron.md` -- CronCreate/CronDelete/CronList tool descriptions.
- `_prompts/skill-loop.md` -- Loop skill for recurring interval tasks.
- `_prompts/skill-schedule.md` -- Schedule skill for remote agent triggers.
- `utils/cronScheduler.ts` -- Full cron scheduler core with lock, jitter, missed-task recovery, file watcher. Contains `createCronScheduler()`, `isRecurringTaskAged()`, `buildMissedTaskNotification()`.
- `hooks/useScheduledTasks.ts` -- React hook wrapper for the cron scheduler in REPL mode.

### Dream Task UI

- `.compozy/tasks/DreamTask/DreamTask.ts` -- Background task entry for dream consolidation UI. Phase tracking, kill support with lock rollback. Contains `registerDreamTask()`, `addDreamTurn()`, `completeDreamTask()`, `failDreamTask()`.

### Coordinator Mode

- `_prompts/system-prompt-coordinator.md` -- Orchestrator system prompt for directing parallel workers (research/synthesis/implementation/verification phases).

### Buddy / Companion

- `buddy/companion.ts` -- Deterministic companion generation from user ID hash. Seeded PRNG, rarity rolling, stat generation. Contains `roll()`, `getCompanion()`, `companionUserId()`.
- `buddy/types.ts` -- Type definitions: species, eyes, hats, stats, rarity weights/colors, CompanionBones/CompanionSoul/Companion types.
- `buddy/sprites.ts` -- ASCII art sprite system with 3 animation frames per species (18 species), hat overlay, face rendering. Contains `renderSprite()`, `renderFace()`.
- `buddy/prompt.ts` -- Companion introduction text for the system prompt. Contains `companionIntroText()`, `getCompanionIntroAttachment()`.
- `buddy/useBuddyNotification.tsx` -- Teaser notification hook (rainbow /buddy on startup), teaser window logic (April 1-7 2026).
- `buddy/CompanionSprite.tsx` -- Full sprite rendering component with speech bubbles, idle animation, pet interaction, fade effects.

### Infrastructure

- `utils/backgroundHousekeeping.ts` -- Startup orchestrator that initializes extractMemories, autoDream, magic docs, skill improvement, plugin auto-update.
- `query/stopHooks.ts` -- Post-turn hook dispatcher that fires extractMemories, autoDream, and prompt suggestions after each model response.
- `projectOnboardingState.ts` -- Simple onboarding state tracker (CLAUDE.md exists, workspace not empty).
- `_prompts/service-dream-consolidation.md` -- Extracted dream consolidation prompt documentation.
- `_prompts/service-memory-extraction.md` -- Extracted memory extraction prompt documentation.

### Migrations

- `migrations/migrateLegacyOpusToCurrent.ts` -- Model version migration pattern (idempotent, event-logged).
- `migrations/migrateAutoUpdatesToSettings.ts` -- Settings format migration pattern.
