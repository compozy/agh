# AI-Harness Memory & Context Analysis (Source: ~/dev/knowledge/ai-harness)

**Topic:** AI & Agent Harnesses (`~/dev/knowledge/ai-harness`)
**Scope:** memory, context, persistence, compaction, file conventions, hooks, sub-agent isolation, learning loops, failure modes
**Cross-cutting source repos read:** `claude-code` (live source under `agh/.resources/claude-code/memdir/`), `codex` Rust core (`agh/.resources/codex/codex-rs/core/src/compact.rs`, `context/user_instructions.rs`), `opencode` (`agh/.resources/opencode/specs/v2/session-concepts-gap.md`, `packages/opencode/AGENTS.md`), wikis for `claude-code` / `hermes` / `openclaw` / `openfang` / `goclaw` under `~/dev/knowledge/<topic>/wiki/concepts/`.
**Date:** 2026-05-04

---

## TL;DR (200 words)

Production agent harnesses converge on a **layered memory stack** with five recurring tiers: (1) **system-prompt static layer** (~12-30 % of window), (2) **scoped instruction files** (`CLAUDE.md` / `AGENTS.md` / `.cursorrules`) loaded at session start with directory-deep precedence, (3) **session transcript** (JSONL on disk or SQLite+FTS5), (4) **cross-session auto-memory** (markdown files, four-type taxonomy, written by a forked extraction subagent, surfaced via top-K relevance side-query), and (5) **structured stores** (vector/pgvector or RDF-style triple stores for entity-relation memory).

The dominant patterns: **harness-managed memory not model-managed** (format enforced, deletion gated, staleness banners injected); **multi-stage compaction cascades** (free → cheap → expensive: tool-result-budget → snip → microcompact → context-collapse → autocompact, with circuit breaker after 3 failures); **content-external FTS5 / semantic + KG hybrid recall**; **sub-agent forks share parent prompt cache but are sandboxed by `isAutoMemPath`**; **filesystem-as-mutex** for cross-process consolidation; **lifecycle hooks** at `pre-/post-tool`, `session.start/end`, `context.assemble`, `before/after_compaction`. Universal failure modes: hash-collision memory bleed, MEMORY.md silent truncation, lost coalesced extractions, stale-fact poisoning, contradictory memories. The field's open problem is **memory lifecycle governance** (decay, contradiction resolution, consolidation at scale).

---

## 1. The five-layer memory taxonomy (the field's de-facto standard)

Drawn from `~/dev/knowledge/ai-harness/wiki/concepts/Memory Systems for Agents.md:30-84` and reinforced by every harness-specific wiki:

| Tier | Lifetime | Implementation in the wild |
| --- | --- | --- |
| **Sensory / immediate** | single inference step | current turn user message + freshly-returned tool result |
| **Short-term / conversation** | current session | in-context history; must be compacted before window fills |
| **Medium-term / session state** | across related sessions | session-summary file, scratchpad, todo list |
| **Long-term / persistent** | indefinite | files, SQLite, vector store, KG triples |
| **Procedural / skills** | indefinite, reusable | packaged markdown how-tos invoked on trigger |

Claude Code, Hermes, OpenClaw, OpenFang, GoClaw all align on this taxonomy. Hermes layer naming (`hermes/wiki/concepts/Learning Loop and Curated Memory.md:33-45`) is the cleanest:

> | Layer | Storage | Consumer | Scope |
> | --- | --- | --- | --- |
> | Session memory | SQLite + FTS5 | session_search_tool | per-session |
> | Persistent memory | `~/.hermes/memory.md` | system prompt every turn | global |
> | Skills | `~/.hermes/skills/*/INDEX.md` | injected on `/skill-name` | global |
> | User profile (Honcho) | external Honcho API | memory provider plugin | per-user |

Claude Code expands the same taxonomy into **five storage surfaces** (`claude-code/wiki/concepts/Memory and Session Persistence.md:35-45`):

> | Layer | Purpose | Storage | Consumer |
> | --- | --- | --- | --- |
> | 1. Conversation history | transient transcript | `.jsonl` per session, project-scoped | user (`/export`, `/resume`) |
> | 2. Session memory | live summary maintained during conversation | `~/.claude/projects/<slug>/.claude/session_memory` | SM-Compact strategy |
> | 3. CLAUDE.md | human-controlled instructions | hierarchical `.md` in repo | main model every turn |
> | 4. Auto-memory | facts Claude learns | `~/.claude/projects/<slug>/memory/` | main model via semantic recall |
> | 5. Team memory | shared across users on the project | `<memoryBase>/projects/<root>/memory/team/` | gated by `tengu_herring_clock` flag |

The cross-harness invariant is that **each layer has a different write protocol, reader, lifetime, and compaction policy**. AGH's mem-v2 should explicitly model these as orthogonal axes rather than collapsing into "the memory system".

---

## 2. File-based instruction & rules conventions

### CLAUDE.md / AGENTS.md hierarchical scope

The most load-bearing file convention. Claude Code (`Settings System` / "Scoped Instruction Hierarchy") implements a **nine-level precedence** that merges via `lodash.mergeWith` with array dedup:

> managed enterprise policy → MDM profiles → CLI flags → project-local overrides → shared project config → user-global preferences → directory-scoped CLAUDE.md → auto-generated → team-shared
> — `claude-code/wiki/concepts/Memory and Session Persistence.md:43`

Codex implements the **same pattern for AGENTS.md** with a deeper-overrides-shallower rule (`agh/.resources/codex/codex-rs/core/hierarchical_agents_message.md:1-8`):

> Files called AGENTS.md commonly appear in many places... at "/", in "~", deep within git repositories... Each AGENTS.md governs the entire directory that contains it and every child directory beneath that point. When two AGENTS.md files disagree, the one located deeper in the directory structure overrides the higher-level file, while instructions given directly in the prompt by the system, developer, or user outrank any AGENTS.md content.

Codex's wrapping in the user-instructions message uses an explicit start/end marker per directory (`agh/.resources/codex/codex-rs/core/src/context/user_instructions.rs:11-16`):

```rust
const START_MARKER: &'static str = "# AGENTS.md instructions for ";
const END_MARKER: &'static str = "</INSTRUCTIONS>";
fn body(&self) -> String {
    format!("{}\n\n<INSTRUCTIONS>\n{}\n", self.directory, self.text)
}
```

Cursor uses `.cursorrules` in repo root (`ai-harness/wiki/concepts/Coding Agents Deep Dive.md:236-249`) — the same pattern with a single file. The DAIR.AI cite at the bottom of `karpathy-llm-knowledge-bases.md:232-236` references the **AGENTbench** finding: *"Human-written files help (+4%), LLM-generated files hurt (-2%), and all files add 20%+ cost."* The implication: cost amortization (prompt caching) is mandatory for these files.

### Bytes / lines caps

Claude Code's `MEMORY.md` index has hard caps (live source `agh/.resources/claude-code/memdir/memdir.ts:34-38`):

```ts
export const ENTRYPOINT_NAME = 'MEMORY.md'
export const MAX_ENTRYPOINT_LINES = 200
export const MAX_ENTRYPOINT_BYTES = 25_000
// ~125 chars/line at 200 lines. At p97 today; catches long-line indexes
// that slip past the line cap (p100 observed: 197KB under 200 lines).
```

Truncation appends a warning line (`memdir.ts:78-103`) — but the `200 + 1`-th entry is **silently invisible to the model** until the index shrinks. This is one of the canonical failure modes (catalogued in §10).

### Sanitized project paths

`~/.claude/projects/<sanitized-path>/` — `claude-code/wiki/concepts/Memory and Session Persistence.md:49-60`:

```ts
export function sanitizePath(name: string): string {
  const sanitized = name.replace(/[^a-zA-Z0-9]/g, '-')
  if (sanitized.length <= MAX_SANITIZED_LENGTH) return sanitized
  const hash = typeof Bun !== 'undefined'
    ? Bun.hash(name).toString(36)
    : simpleHash(name)
  return `${sanitized.slice(0, MAX_SANITIZED_LENGTH)}-${hash}`
}
```

Failure mode: long-project-path hash collisions cause **memory bleed across unrelated projects**, and worktrees of the same repo deliberately share memory (a feature for resume, but a bug when feature-branch facts contradict main).

---

## 3. Cross-session auto-memory: extraction + recall pipeline

### Four memory types (Claude Code's load-bearing taxonomy)

`agh/.resources/claude-code/memdir/memoryTypes.ts:14-19`:

```ts
export const MEMORY_TYPES = ['user', 'feedback', 'project', 'reference'] as const
```

Each type has full prompt-engineered guidance for **when to save / when not to / how to structure / examples** (lines 113-178). Verbatim discriminators:

- **`user`**: role, expertise, preferences. Always private scope.
- **`feedback`**: corrections AND validated approaches. Lead with rule, then `**Why:**` line (the user-given rationale, often a prior incident), then `**How to apply:**`. Save on confirmations not just corrections — "if you only save corrections you avoid past mistakes but drift away from approaches the user has already validated, and may grow overly cautious" (line 60).
- **`project`**: ongoing work, deadlines, motivations not in code/git. Same body structure (Why / How to apply). Convert relative dates to absolute.
- **`reference`**: pointers to external systems (Linear, Slack, Grafana). Usually team-scoped.

Hard invariant in `memoryTypes.ts:183-194` — **NEVER save**:

> - Code patterns, conventions, architecture, file paths, project structure (derivable from project state)
> - Git history, recent changes (git log/blame is authoritative)
> - Debugging solutions / fix recipes (the fix is in the code; commit message has the context)
> - Anything already documented in CLAUDE.md
> - Ephemeral task details
>
> *These exclusions apply even when the user explicitly asks you to save. If they ask you to save a PR list or activity summary, ask what was surprising or non-obvious about it — that is the part worth keeping.*

This explicit-save gate was eval-validated: "memory-prompt-iteration case 3, 0/2 → 3/3: prevents 'save this week's PR list' → activity-log noise" (line 192).

### The forked Sonnet extraction subagent

`claude-code/wiki/concepts/Memory and Session Persistence.md:160-205`. After every main-agent response a **forked Sonnet** (cache-shared with the parent so most input tokens are cache hits) runs with a sandboxed `CanUseToolFn`:

```ts
export function createAutoMemCanUseTool(memoryDir: string): CanUseToolFn {
  return async (tool, input) => {
    if (tool.name === FILE_READ_TOOL_NAME ||
        tool.name === GREP_TOOL_NAME ||
        tool.name === GLOB_TOOL_NAME) {
      return { behavior: 'allow', updatedInput: input }
    }
    if (tool.name === BASH_TOOL_NAME) {
      const parsed = tool.inputSchema.safeParse(input)
      if (parsed.success && tool.isReadOnly(parsed.data)) {
        return { behavior: 'allow', updatedInput: input }
      }
      return denyAutoMemTool(tool, 'Only read-only shell commands allowed')
    }
    if ((tool.name === FILE_EDIT_TOOL_NAME ||
         tool.name === FILE_WRITE_TOOL_NAME) && 'file_path' in input) {
      if (typeof input.file_path === 'string' && isAutoMemPath(input.file_path)) {
        return { behavior: 'allow', updatedInput: input }
      }
    }
    return denyAutoMemTool(tool, 'only Read, Grep, Glob, read-only Bash, ' +
      `and Edit/Write within ${memoryDir} are allowed`)
  }
}
```

Capped at 5 turns. Prompt explicitly forbids verification grepping ("Do not waste turns attempting to verify content. No grepping source files, no reading code.") to keep extraction cheap. MCP tools and the Agent (subagent-spawning) tool are denied.

`scanMemoryFiles` pre-scans up to `MAX_MEMORY_FILES = 200` headers sorted newest-first (`Memory and Session Persistence.md:204`). Terminal `catch { return [] }` is a **silent failure mode** if the directory is unreadable.

### Race-resolution between main agent and extraction subagent

`Memory and Session Persistence.md:213-227`. Both can write to the memory dir. Doubles are prevented by `hasMemoryWritesSince(messages, sinceUuid)` which scans for any `file_path` writes into the memory directory after the last extraction point. If the main agent already wrote, extraction skips. Coalescing handles overlapping rapid messages with a **single-slot pending context** (not a queue):

```ts
} finally {
  inProgress = false
  const trailing = pendingContext
  pendingContext = undefined
  if (trailing) await runExtraction({ ...trailing, isTrailingRun: true })
}
```

If messages 2 and 3 both arrive while message 1's extraction runs, message 2's context is overwritten — **message 2's extraction window is silently lost**. Deliberate simplification.

### Semantic recall via Sonnet side-query

The recall path: every turn, MEMORY.md (the index) is loaded into the system prompt, but topic files are not. A non-blocking Sonnet side-query selects the top-5 (`agh/.resources/claude-code/memdir/findRelevantMemories.ts:77-141`).

System prompt (`findRelevantMemories.ts:18-24`):

```
You are selecting memories that will be useful to Claude Code as it processes a user's query...
Return a list of filenames for the memories that will clearly be useful to Claude Code as it processes the user's query (up to 5). Only include memories that you are certain will be helpful based on their name and description.
- If you are unsure if a memory will be useful in processing the user's query, then do not include it in your list. Be selective and discerning.
- If there are no memories in the list that would clearly be useful, feel free to return an empty list.
- If a list of recently-used tools is provided, do not select memories that are usage reference or API documentation for those tools (Claude Code is already exercising them). DO still select memories containing warnings, gotchas, or known issues about those tools — active use is exactly when those matter.
```

Critical design choices:

- Selector runs **Sonnet even when the main model is Opus** — cheap relevance filter.
- **Explicit precision-over-recall bias**: "if unsure, do not include." False negatives preferred to false positives because irrelevant context is more damaging than missed recall.
- Three additional filters: (1) top-5 budget keeps surfaced budget under ~20KB/turn, (2) skip docs for tools already in active use but keep warnings/gotchas, (3) exclude files shown in prior turns to spend the 5-slot budget on fresh candidates.
- If Sonnet fails or returns garbage, returns `[]` silently — user loses all memory context with no UI hint.

`json_schema` output format with `selected_memories: string[]` and `additionalProperties: false` (lines 109-119) — typed extraction not regex parsing.

### Staleness ban

`Memory and Session Persistence.md:393-403`:

```ts
export function memoryFreshnessText(mtimeMs: number): string {
  const d = memoryAgeDays(mtimeMs)
  if (d <= 1) return ''
  return `This memory is ${d} days old. Memories are point-in-time observations, ` +
    `not live state — claims about code behavior or file:line citations may be ` +
    `outdated. Verify against current code before asserting as fact.`
}
```

Plus a `TRUSTING_RECALL_SECTION` (`memoryTypes.ts:240-256`) that reinforces "before recommending from memory: if it names a file path, check it exists; if it names a function or flag, grep for it; if the user is about to act, verify first." Header phrasing matters — eval-validated: "Before recommending from memory" (action cue) tested 3/3 vs "Trusting what you recall" (abstract) at 0/3.

### Per-turn memory cost in production

`Memory and Session Persistence.md:443-455`:

| Component | Typical size |
| --- | --- |
| Memory prompt instructions | 500-1,000 tokens |
| MEMORY.md content | up to 25KB (~5-6K tokens) |
| CLAUDE.md files | up to 40KB per file (~8-10K tokens) |
| Surfaced memories (top 5) | up to 20KB per turn (~4K tokens) |
| **Cumulative session cap** | **60KB (~12K tokens)** |
| **Anthropic-observed average** | **26K tokens/session — 13 % of 200K window** |

The memory block is **static across turns** and sits at the top of the system prompt for cache-friendliness, but it's not free — prompt-cache hit makes input cheap, but it still consumes the window's effective budget.

---

## 4. Compaction cascades (reducing footprint without losing critical info)

### Claude Code's five-layer cascade

`claude-code/wiki/concepts/Token Budget and Context Compaction.md:55-94`. Runs at the **start of every turn** before model sampling:

```
Layer 1: Tool Result Budget         applyToolResultBudget()         per-message size cap (persist large results to disk)
Layer 2: Snip Compaction            snipCompactIfNeeded()           clear old tool result CONTENTS only (preserve tool_use/tool_result pairs)   [HISTORY_SNIP]
Layer 3: Microcompaction            microcompactMessages()          remove old tool results via cache_edits API (preserve prompt cache!)        per-turn
Layer 4: Context Collapse           applyCollapsesIfNeeded()        model-side semantic compression with commit log                              [CONTEXT_COLLAPSE]
Layer 5: Autocompact                autoCompactIfNeeded()           SM-Compact then Full Compaction                                              threshold-based
```

Threshold constants (line 99-112):

| Constant | Value | Purpose |
| --- | --- | --- |
| `AUTOCOMPACT_BUFFER_TOKENS` | 13,000 | trigger at `effectiveWindow - 13K` |
| `WARNING_THRESHOLD_BUFFER_TOKENS` | 20,000 | user-visible warning |
| `MANUAL_COMPACT_BUFFER_TOKENS` | 3,000 | hard limit triggering REPL warning |
| `MAX_OUTPUT_TOKENS_FOR_SUMMARY` | 20,000 | reserved for compaction call itself |
| `MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES` | 3 | circuit breaker |
| SM-Compact min/max preserved | 10K / 40K | floor + ceiling for kept tail |
| SM-Compact min text-block messages | 5 | minimum messages preserved |

**Critical invariants** (lines 207-223):

- `adjustStartIndexForToolPairs` — never split a `tool_use` / `tool_result` pair (API 400). Never orphan thinking blocks. Never drop below the last `compact_boundary_message`.
- Hybrid token counting: API authoritative when available, ~1 token/4 chars heuristic between calls.
- **20K reserve held back from the compaction call itself** so compaction doesn't fail with `prompt_too_long` while compacting — there's no recovery path if it does.
- Cost ordering of layers: Snip (free, no API) → Cached Microcompact (free, piggybacked on `cache_edits`) → Time-based microcompact (cache already cold) → Context Collapse (amortized) → SM-Compact (free, summary on disk) → Full Compaction (one forked-agent API call).

### Two strategies inside autocompact

**SM-Compact** (cheap path, line 195-225): boundary-aware reorganize. Replaces old messages with the **already-on-disk session-memory summary** — no new API call. Aborts if it can't free enough tokens, falls through to:

**Full Compaction** (expensive fallback, line 229-258): spawns a **forked summarization agent** with a structured prompt:

```
1. Strip images (replace with [image] placeholders)
2. Group messages by API round
3. Send to forked compaction agent with structured prompt:
   - Primary Request and Intent
   - Key Technical Concepts
   - Files and Code Sections (with snippets)
   - Errors and Fixes
   - Problem Solving
   - All User Messages (verbatim)
   - Pending Tasks
   - Current Work
   - Optional Next Step
4. Agent returns <analysis> + <summary>
5. Build [CompactBoundary] + [Summary] + [Attachments] + [KeptMessages]
6. Re-attachments:
   - Top 5 recently-used files (50K token budget)
   - Top 5 invoked skills (25K budget)
   - Plan mode state if active
   - Notify about running async agents
```

**All user messages preserved verbatim** — user instructions are irreplaceable ground truth. Custom user instructions accepted via `/compact [instructions]`.

### Codex's compaction (cleaner, simpler)

`agh/.resources/codex/codex-rs/core/src/compact.rs:42-44, 56-59, 92-114`:

```rust
pub const SUMMARIZATION_PROMPT: &str = include_str!("../templates/compact/prompt.md");
pub const SUMMARY_PREFIX: &str = include_str!("../templates/compact/summary_prefix.md");
const COMPACT_USER_MESSAGE_MAX_TOKENS: usize = 20_000;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(crate) enum InitialContextInjection {
    BeforeLastUserMessage,  // mid-turn: model trained to see summary as last item
    DoNotInject,            // pre-turn / manual: clears reference_context_item; next regular turn fully reinjects initial context
}
```

Codex prompt (`codex-rs/core/templates/compact/prompt.md`):

> You are performing a CONTEXT CHECKPOINT COMPACTION. Create a handoff summary for another LLM that will resume the task.
> Include:
> - Current progress and key decisions made
> - Important context, constraints, or user preferences
> - What remains to be done (clear next steps)
> - Any critical data, examples, or references needed to continue
> Be concise, structured, and focused on helping the next LLM seamlessly continue the work.

Note the "for another LLM" framing — Codex treats compaction explicitly as an **inter-LLM handoff** rather than a self-summary. Reinforced by `summary_prefix.md`:

> Another language model started to solve this problem and produced a summary of its thinking process. You also have access to the state of the tools that were used by that language model. Use this to build on the work that has already been done and avoid duplicating work.

This framing matters: it tells the receiving model to **trust the summary as evidence**, not as one of its own thoughts.

### OpenClaw's compaction lifecycle (5 phases)

`openclaw/wiki/concepts/Context Compaction.md:30-90`:

1. **Overflow detection** — `evaluateContextWindowGuard` at the beginning of each turn, before LLM call.
2. **Preparation + locking** — `acquireSessionWriteLock`, `flushPendingToolResults` (drain async tool buffers before truncating), run `before_compaction` hooks.
3. **Execution** — `compactEmbeddedPiSessionDirect`. **Independent model selection** — `agents.defaults.compaction.model` lets the operator point compaction at a cheaper model (e.g., Haiku) while the main conversation uses Opus.
4. **Truncation** — atomic JSONL rewrite. `SessionToolResultGuard` validates message-role alternation invariants (Anthropic: strict user/assistant alternation with tool-use/tool-result blocks).
5. **Post-sync** — run `after_compaction` hooks, refresh memory search index, release lock, emit `session` event.

Thresholds (line 95-103):

| Threshold | Token Count | Behavior |
| --- | --- | --- |
| Hard Min | 8,192 | minimum window; below this no compaction |
| Warning (soft) | 32,768 | trigger asynchronous compaction |

`compactWithSafetyTimeout` enforces wall-clock duration via `EMBEDDED_COMPACTION_TIMEOUT_MS`. Retry attempt limits prevent infinite recursion (a summary that grows large enough to need another compaction). Compression ratios observed: 3:1 to 10:1 (32K-token session → 3-10K).

### Hermes context compression

`hermes/wiki/concepts/Session Store and FTS5 Recall.md:302-333` — compaction creates a **new session** with the parent linked via `parent_session_id`, forming a chain rather than mutating in place:

```python
# Original session fills up at 50% of context limit
session_id_old = "sess_abc..."
compressed_msgs = compressor.compress(messages)
session_id_new = "sess_xyz..."
add_session({
    "id": session_id_new,
    "parent_session_id": session_id_old,
    "message_count": len(compressed_msgs),
})

def get_session_chain(session_id):
    chain, current = [], session_id
    while current:
        s = get_session(current); chain.append(s); current = s.get("parent_session_id")
    return chain
```

Cost is summed across the chain, FTS5 search hits any ancestor. The chain preserves the full historical trail; the trade-off is that aggregate queries must walk the chain.

---

## 5. Persistence backends — what each harness chose

### Claude Code: filesystem-only, no DB

`Memory and Session Persistence.md:25-45`. The entire system is `fs.writeFile()`-backed. No database, index server, vector store. JSONL session transcripts in `~/.claude/projects/<sanitized-path>/`. Markdown files for memory.

**Filesystem-as-mutex** (`Memory and Session Persistence.md:262-300`) for cross-process consolidation:

```ts
const LOCK_FILE = '.consolidate-lock'
const HOLDER_STALE_MS = 60 * 60 * 1000  // PID reuse guard

export async function tryAcquireConsolidationLock(): Promise<number | null> {
  const path = lockPath()
  const [s, raw] = await Promise.all([stat(path), readFile(path, 'utf8')])
  // mtime IS lastConsolidatedAt; body IS holder PID
  if (Date.now() - s.mtimeMs < HOLDER_STALE_MS && isProcessRunning(holderPid)) {
    return null
  }
  await writeFile(path, String(process.pid))
  // Two reclaimers both write → last wins. Loser bails on re-read.
  if (parseInt(await readFile(path, 'utf8'), 10) !== process.pid) return null
  return mtimeMs ?? 0
}
```

The `.mtime`-as-`lastConsolidatedAt` trick is elegant: rollback uses `utimes()` to rewind. No `flock`, no advisory locks, no DB transactions. The cost: a crashed process holds the lock for up to an hour.

### Hermes: SQLite + FTS5 + write-retry

`hermes/wiki/concepts/Session Store and FTS5 Recall.md:31-43`. Single `~/.hermes/state.db`. Three tables: `sessions`, `messages`, `messages_fts` (virtual, content-external). Triggers keep FTS in sync automatically — application code never touches the FTS index.

```sql
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    content=messages,
    content_rowid=id
);
CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
END;
-- + delete + update triggers
```

Multi-writer (CLI + 8 gateway adapters + worktree subagents + cron) coordinated via **WAL mode + application-level retries with random jitter**:

```python
_WRITE_MAX_RETRIES = 15
_WRITE_RETRY_MIN_S = 0.020       # 20 ms base
_WRITE_RETRY_MAX_S = 0.150       # 150 ms max
_CHECKPOINT_EVERY_N_WRITES = 50  # passive WAL checkpoint
```

The jitter is **load-bearing**: without it, retries synchronize on tick boundaries and deadlock; with it, contention decoheres. Worst case: 15 × 150 ms = 2.25 s; typical: first or second attempt.

PRAGMAs: `journal_mode=WAL`, `synchronous=NORMAL`, `timeout=1.0` — trade theoretical durability for 10-100× throughput. Observed: <100 ms FTS5 search at 10K messages.

Two-stage recall (line 270-298):

```
User asks: "what was that deployment bug from last week?"
  ↓
session_search(query="deployment bug")
  ↓
db.fts_search("deployment bug", limit=20)        # FTS5 BM25 ranking
  ↓
group by session_id
  ↓
auxiliary_client.summarize_session(meta, msgs, query)   # cheap LLM (gpt-4o-mini)
  ↓
return compact summaries
```

LLM is a **relevance filter and compactor, not a ranker** — FTS5 handles ranking, LLM compresses. This pattern (BM25 → LLM compress) is cheaper and more controllable than vector-only retrieval.

### OpenClaw: JSONL on disk + DM scope policy

`openclaw/wiki/concepts/Sessions, DMs and Memory.md:75-113`. Per-agent storage:

```
~/.openclaw/agents/<agentId>/sessions/
├── sessions.json                  ← metadata index
└── <sessionId>.jsonl              ← transcript, one message per line
```

JSONL chosen for: append-only writes are atomic; trivially inspectable with `tail`/`jq`; recovery after crash is "truncate at last complete line." For very long transcripts, OpenClaw writes a **dual file**: full JSONL + `.summary.json` (compacted summary for fast list rendering).

**Crucial: DM scope policy** (lines 117-141) — when one assistant fronts multiple chat platforms, naive routing leaks Bob's messages to Alice. Four scopes:

| Scope | Behavior |
| --- | --- |
| `main` | all DMs → one session (single user only) |
| `per-peer` | separate per sender across channels |
| `per-channel-peer` | separate per (channel + sender) — DEFAULT, multi-user safe |
| `per-account-channel-peer` | separate per (account + channel + sender) |

This is a memory architecture decision masquerading as a routing decision. AGH should explicitly model the **scoping axis** distinct from the **persistence axis**.

### OpenFang: SQLite triple store (RDF-style KG)

`openfang/wiki/concepts/Knowledge Graph Engine.md:30-87`. Three tables: `entities`, `relations` (entity-to-entity), `facts` (entity-to-literal). Every row carries `confidence: f64 [0.0, 1.0]` and `source` for provenance. Confidence by source:

| Source | Typical Confidence |
| --- | --- |
| Direct user input | 0.95 |
| Official Wikipedia | 0.90 |
| Peer-reviewed article | 0.85 |
| Web search result | 0.70 |
| Indirect inference | 0.50 |

Hybrid recall via `recall_with_kg_context` (lines 350-367): semantic search + entity extraction + KG lookup → combined context blocks. The KG contradiction-detection pattern (lines 332-343) is novel — query for `competes_with` and `partners_with` on the same entity, flag conflicts for re-verification.

### GoClaw: pgvector + LLM-extracted KG

`goclaw/wiki/concepts/Memory and Knowledge Graph.md:30-78, 117-194`. Memory chunks embedded into PostgreSQL with pgvector, IVFFLAT index. Knowledge graph extracted via **LLM call per document** with `minConfidence = 0.75` filter. Strict multi-tenant `team_id` scoping. JSONB `properties` so per-entity-type metadata can be added without schema migrations.

Key implementation decision: extraction is **async/deferred, not in the critical path**. The agent doesn't block on KG extraction.

---

## 6. Hooks & lifecycle events for memory

### Claude Code's hooks

`claude-code/wiki/concepts/Hook System` referenced from `Memory and Session Persistence.md`. 25+ lifecycle events:

```
Pre-tool: PreToolUse, before specific tool types, before shell
Post-tool: PostToolUse, after file modifications, after test runs
Notification: SessionStart, SessionEnd, errors, output patterns
Stop: review gates, policy enforcement
PermissionRequest, UserPromptSubmit, CwdChanged, ...
```

Hook output is a **structured JSON protocol** that feeds back into the query loop — hooks can block or modify actions. MDM-enforced for enterprises (tamper-resistant).

### OpenClaw's compaction hooks

`openclaw/wiki/concepts/Context Compaction.md:171-183`:

| Hook | Timing | Use Case |
| --- | --- | --- |
| `before_compaction` | before summarization | inspect / cancel / inject domain context |
| `after_compaction` | after truncation | metrics / external indexes / notifications |
| `session` (event) | post-truncation | downstream subscribers |

Plus `context.assemble` (`openclaw/wiki/concepts/Sessions, DMs and Memory.md:476-489`):

```json5
{
  hooks: {
    internal: {
      entries: [
        {
          id: "inject-current-time",
          event: "context.assemble",
          handler: "exec:///usr/local/bin/inject-time.sh"
        }
      ]
    }
  }
}
```

The hook receives the prompt bundle (system context, history, tools), modifies it, returns it. Use cases: inject real-time data, redact PII, log to external system.

### Skills as on-demand context injection

Highlighted in `theo-inject-dynamic-context-pattern-claude-code-skills.md`:

> The "inject dynamic context" pattern in Claude Code skills is so useful... should be part of the "skills standard" and included in tools like Codex CLI, Pi, Cursor etc.

Skills shift from "static prompts" to "context-routing primitives" — this is exactly the boundary AGH's mem-v2 should formalize: skills are not just packaged prompts, they're **dynamic memory loaders** that assemble the right context for the trigger.

---

## 7. Sub-agent / multi-agent memory sharing & isolation

### Claude Code: three execution models

`Memory and Session Persistence.md` + `Agent Swarm and Subagents` referenced throughout. Three patterns:

1. **Fork (same process, cache-shared clone)** — used by extraction agent and session-memory agent. Shares parent prompt cache → cheap. Sandboxed by `CanUseToolFn` (read-only or scoped to memory dir).
2. **Teammate (separate tmux pane, file-based mailbox)** — separate context window, scoped system prompt, restricted tool set selected by `subagent_type`.
3. **Worktree (git isolation)** — `.claude/worktrees/`, new branch from HEAD, full filesystem isolation, parent's prompt cache reused. Best-of-N variant: same task across different models, parallel reviewers, merge winning branch.

The architectural insight: worktrees serve as both **isolation primitive** AND **experimental control surface** for evaluation inside the harness.

### Hermes & GoClaw: shared SQLite across all agents

Hermes `~/.hermes/state.db` is the **cross-platform identity layer** — a Telegram bot session can `session_search` a CLI conversation. Cross-agent shared substrate. The trade-off: write contention requires retry/jitter discipline.

GoClaw uses Postgres with strict `team_id` scoping for multi-tenancy. Cross-agent queries forbidden via API; possible only via direct SQL.

### OpenFang: agent-scoped KG by default

`Knowledge Graph Engine.md:88-89, 274-296`:

> Every row is scoped by `agent_id` — each agent has its own knowledge graph. Cross-agent graph sharing happens via the consolidation pipeline, not by direct multi-agent writes.

`consolidate_kg(from_agent_id, to_agent_id)` returns a `ConsolidationReport` listing conflicts where `from` and `to` had contradictory facts. This is the **operator-reviewed merge** pattern — automated dedup with human-reviewed conflicts.

---

## 8. User feedback → durable rules conversion

The clearest pattern in the corpus is Claude Code's `feedback` memory type (`memoryTypes.ts:60-74`):

> **`<when_to_save>`**: Any time the user corrects your approach ("no not that", "don't", "stop doing X") OR confirms a non-obvious approach worked ("yes exactly", "perfect, keep doing that", accepting an unusual choice without pushback). Corrections are easy to notice; confirmations are quieter — watch for them. In both cases, save what is applicable to future conversations, especially if surprising or not obvious from the code. Include *why* so you can judge edge cases later.

> **`<body_structure>`**: Lead with the rule itself, then a `**Why:**` line (the reason the user gave — often a past incident or strong preference) and a `**How to apply:**` line (when/where this guidance kicks in). Knowing *why* lets you judge edge cases instead of blindly following the rule.

Examples (verbatim, line 65-72):

```
user: don't mock the database in these tests — we got burned last quarter when
      mocked tests passed but the prod migration failed
assistant: [saves team feedback memory: integration tests must hit a real
      database, not mocks. Reason: prior incident where mock/prod divergence
      masked a broken migration. Team scope: this is a project testing policy,
      not a personal preference]

user: stop summarizing what you just did at the end of every response, I can
      read the diff
assistant: [saves private feedback memory: this user wants terse responses with
      no trailing summaries. Private because it's a communication preference,
      not a project convention]

user: yeah the single bundled PR was the right call here, splitting this one
      would've just been churn
assistant: [saves private feedback memory: for refactors in this area, user
      prefers one bundled PR over many small ones. Confirmed after I chose this
      approach — a validated judgment call, not a correction]
```

The third example is the differentiator from the field: **save validations not just corrections**. The rationale (line 60): "if you only save corrections, you will avoid past mistakes but drift away from approaches the user has already validated, and may grow overly cautious."

Memory-as-policy framing (`Memory Systems for Agents.md:198-234`, MemFactory): treat memory management as a learned decision-making process. Extractor / Updater / Retriever modules trained via GRPO. 4B-parameter Qwen3 + 250 RL steps yields 7-14 % improvement on memory benchmarks, with **out-of-distribution gains** at 4B (not at 1.7B), suggesting a capability threshold for general memory policy.

---

## 9. Self-maintaining wikis (the Karpathy pattern)

`Memory Systems for Agents.md:304-346` and `karpathy-llm-knowledge-bases.md`. Four-phase cycle:

1. **Ingest** — Obsidian Web Clipper, papers, repos → `raw/` directory.
2. **Compile** — LLM reads `raw/`, builds `wiki/` with index files (always consulted first), concept articles (~100 articles, ~400K words), backlinks/cross-links auto-generated.
3. **Query & enhance** — Q&A agent renders complex queries as markdown / Marp slides / matplotlib charts, **filed back into the wiki**.
4. **Lint & maintain** — health checks, find missing connections, suggest new article candidates. Returns to phase 2.

Properties (line 339-345):

- **Human-readable** — plain markdown, editable in any editor.
- **Version-controlled** — git history, diffs, rollback.
- **Cross-linked** — wikilinks form a navigable graph.
- **Scriptable** — standard file operations enable automation.
- **Compounding** — each cycle adds knowledge AND connections.

LLM-Wiki-v2 extension (`heynavtoor-karpathy-llm-wiki-extended-memory-lifecycle.md`):

> Memory lifecycle. Confidence scoring. Knowledge graphs. Automated hooks. Forgetting curves.

The forgetting-curve direction is missing from current Claude Code / Hermes / OpenClaw — they all rely on staleness banners + `mtime` heuristics rather than time-decay weighting.

---

## 10. Failure modes (catalog from the wild)

### Silent degradation modes (Claude Code, `Memory and Session Persistence.md:457-468`)

> - **Unreadable memory directory** → `scanMemoryFiles` returns `[]` silently
> - **Sonnet side-query fails** → `selectRelevantMemories` returns `[]` silently; user loses all memory context with no UI hint
> - **Memory directory creation fails** (EACCES, EPERM, EROFS, ENOSPC) → `ensureMemoryDirExists` catches, logs at debug level, swallows. System prompt still tells the model the directory exists; the write fails as a tool error to the user later.
> - **AutoDream gate fails** → silent skip
> - **Coalesced extraction window** → middle message's extraction silently lost when pending-context slot is overwritten
> - **MEMORY.md truncation** → entry 201+ invisible to the model

### Memory poisoning / instruction collision

- **Hash collisions on long project paths** cause memory bleed across unrelated projects (`sanitizePath`).
- **Worktrees deliberately share memory** but feature-branch facts contradict main.
- **Stale facts** referencing renamed files / removed functions / unmerged PRs — the staleness banner helps but is advisory, not enforcing.
- **Contradictory memories** (Hermes design tension §4 in `Learning Loop and Curated Memory.md:412-415`): "as memory.md grows past a few hundred lines, the model starts seeing contradictions." Hermes has no auto-dedup; users must edit by hand.
- **Skill explosion** (Hermes design tension §3): `skill_manage(action="propose")` can generate dozens of one-off skills from ephemeral tasks. Without curation, skill index grows unbounded.

### Anthropic message-protocol invariants violated by naive truncation

Key invariant from Claude Code SM-Compact (`Token Budget and Context Compaction.md:213-217`) and OpenClaw `SessionToolResultGuard` (`Context Compaction.md:145-148`):

> never truncate the middle of a `tool_use` / `tool_result` pair — doing so triggers an API 400 error
> never orphan thinking blocks (same `message.id`)
> never drop below the last `compact_boundary_message`

These are not optional. Any compaction that violates them fails closed in a way that breaks the next turn entirely. The implication for AGH: compaction must be **role-alternation-aware** (Anthropic) and **tool-pair-aware** (all providers).

### Path security

`Memory and Session Persistence.md:407-411`. Three defences for the auto-memory sandbox:

1. `sanitizePathKey` rejects null bytes, URL-encoded traversals, Unicode normalization attacks (PSR M22187).
2. `realpathDeepestExisting` resolves symlinks for the deepest existing ancestor to prevent symlink-escape (PSR M22186).
3. `autoMemoryDirectory` setting is **global-only** — cannot be set in project-level `.claude/settings.json` because a malicious repo could point memory at `~/.ssh`.

The general principle: treat user-controllable extension points as untrusted in contexts where the user has not explicitly assented.

---

## 11. Anthropic / OpenAI prescriptive guidance

### Anthropic's three core principles for harness builders

`anthropic-building-effective-agents.md:175-180`:

> 1. Maintain **simplicity** in your agent's design.
> 2. Prioritize **transparency** by explicitly showing the agent's planning steps.
> 3. Carefully craft your agent-computer interface (ACI) through thorough tool documentation and testing.

Key memory-relevant guidance:

> "We suggest that developers start by using LLM APIs directly: many patterns can be implemented in a few lines of code. If you do use a framework, ensure you understand the underlying code. Incorrect assumptions about what's under the hood are a common source of customer error."

Tools are the augmented LLM's **primary memory write path** (lines 64-72). Tool definitions get the same prompt-engineering attention as the main prompt. Always require absolute file paths (line 232 — example of tool-fix that improved SWE-bench performance more than prompt-fixes).

### Claude Agent SDK's memory abstraction

`anthropic-claude-agent-sdk.md:380-385`. Default file-based config:

| Feature | Description | Location |
| --- | --- | --- |
| Skills | Specialized capabilities | `.claude/skills/*/SKILL.md` |
| Slash commands | Custom commands | `.claude/commands/*.md` |
| Memory | Project context/instructions | `CLAUDE.md` or `.claude/CLAUDE.md` |
| Plugins | Extensions | programmatic via `plugins` option |

Anthropic now exposes a **dedicated Memory tool** for agent-managed memory: `https://platform.claude.com/docs/en/agents-and-tools/tool-use/memory-tool` (referenced in `anthropic-claude-agent-sdk.md:53`). Plus a separate **context-management section** with: context windows, compaction, context editing, prompt caching, token counting (line 60-62).

### OpenAI guardrails framing

`openai-practical-guide-building-agents.md:460-510`. Layered defense:

> Relevance classifier → Safety classifier → PII filter → Moderation → Tool safeguards (low/med/high risk rating) → Rules-based protections (regex/blocklist/length) → Output validation

Two human-intervention triggers (line 657-661):

> 1. **Exceeding failure thresholds** — set retry limits; escalate when exceeded
> 2. **High-risk actions** — sensitive, irreversible, high-stakes actions trigger human oversight until confidence in agent reliability grows

These map onto memory-write decisions: should the agent be allowed to autonomously write a memory that contradicts an existing one? AGH's answer should be operator-configurable per memory type.

### Aider's repo map

`aider-ai-pair-programming.md:53`:

> Aider's key innovation is the "repo map" — a condensed representation of the codebase using tree-sitter to extract class/function signatures and relationships. This lets Aider intelligently select relevant context for each task rather than loading entire files.

This is a **derived structural memory** pattern — the repo itself is the source, the map is a continually-recomputed projection. Distinct from the file-based memory patterns (which store new facts) and the KG patterns (which extract entities). AGH could expose this as a dedicated `derived` memory type with TTL = `mtime(source_files)`.

---

## 12. Context engineering: the 8-phase model & ACE framework

### The 8-phase model (Claude Code anatomy, `AI Bookmarks Knowledge Base and Memory.md:201-229`)

Verbatim from @ellen_in_sf's reverse-engineered phase walkthrough:

> Phase 1: session init registers hooks, warms the memory cache, and kicks off async directory walks before the first render
> Phase 2: memory is discovered in priority order — managed enterprise policy → user global → project VCS → local per-directory → auto-generated → team shared
> Phase 3: three parallel pipelines merge into every API call: system prompt + memory section + user context. relevance prefetch selects up to 5 memory files via sonnet side-call
> Phase 4: the model can directly read/write memory files using FileReadTool, FileWriteTool, FileEditTool. background extractor and model writes are mutually exclusive
> Phase 5: after EVERY response, three background agents fire — extractMemories, sessionMemory, and autoDream. extractMemories is a forked agent that runs in parallel, capped at 200 lines / 25kb
> Phase 6: when context fills up, compaction summarizes old messages using a skipped summarizer, preserving min 10k tokens / 5 text-block messages
> Phase 7: memory lives across ~/.claude/, project root, sessions/, and agent-memory/ — auto memory is git-ignored, team memory is VCS-tracked
> Phase 8: self-improving loop across sessions — within-turn writes + end-of-turn extracts + session memory + auto-dream consolidations every 24h+

### ACE framework (`Context Engineering.md:154-202`)

Four concerns:

1. **Selection** — what tools, docs, KB are accessible
2. **Formatting** — XML / JSON / markdown delimiters, ordering
3. **Timing** — pre-loaded vs on-demand vs deferred (load schemas only when tool is invoked)
4. **Lifecycle** — compaction triggers, externalization, session-to-session persistence

### Token budget allocation table (production benchmark)

From `Context Engineering.md:253-263`. For a 200K window:

| Layer | Token Budget | Percentage |
| --- | --- | --- |
| System prompt | 24,000 | 12 % |
| Project docs (CLAUDE.md) | 8,000 | 4 % |
| Tool definitions | 12,000 | 6 % |
| Conversation history (compacted) | 40,000 | 20 % |
| Tool results (current step) | 60,000 | 30 % |
| Retrieved context (RAG) | 40,000 | 20 % |
| User message + scratch | 16,000 | 8 % |

Dynamic rebalancing per task type (line 287-298):
- code review: tool results 45 %, history 10 %
- architecture discussion: history 35 %, tool results 15 %
- retrieval-heavy: retrieved 35 %, tool results 15 %

### Context rot — empirical: more tokens is worse

`Context Engineering.md:44-56` (Chroma research, ICML/NeurIPS):

> Increasing the number of tokens in the context window can actively degrade LLM performance, even when the additional tokens contain relevant information... attention becomes diluted; relevant signals get buried under a rising noise floor.

Implications:
- Naive context stuffing is anti-pattern.
- 32K curated > 128K dumped.
- Compression and eviction are **requirements, not optimizations**.

### ctx-zip pattern

`Memory Systems for Agents.md:618` + bookmarks. Compacts tool-call spillover to files, references them rather than inlining. Specifically:

> 1. Removing redundant structural elements (repeated headers, boilerplate)
> 2. Extracting only the fields relevant to the current task
> 3. Applying format-aware compression (e.g., collapsing JSON structures)

Frees an 8K budget to cover what would otherwise need 32K of raw output.

### Submodular optimization for curation

`Context Engineering.md:206-247`. Greedy submodular selection picks context items that maximize **marginal information gain** within token budget. JinaAI applied this to RAG. The diminishing-returns property maps to "the third snippet from the same file adds noise that outweighs its information value."

---

## 13. Slash commands, /clear, /compact, /memory

### Universal command surface

| Command | Purpose | Implementations |
| --- | --- | --- |
| `/compact [instructions]` | force compaction, optionally with custom focus | Claude Code |
| `/resume <id>` | restore session by UUID | Claude Code |
| `/rewind` | scroll prior `UserMessage` entries, pick restoration point | Claude Code (`MessageSelector`) |
| `/feedback` | export transcript with redaction | Claude Code |
| `/export` | dump transcript | Claude Code |
| `/new`, `/new <model>` | fresh transcript, optional model switch | OpenClaw |
| `/sessions list / current / reset / archive` | session management | OpenClaw |
| `/context list / add <file>` | inspect or add context files | OpenClaw |
| `/skills list / delete` | curate skill index | Hermes |
| `/insights [days]` | aggregate tool usage, learning velocity, memory growth | Hermes |
| `/usage` | per-session cost (walks compression chain) | Hermes |
| `/history` | recent sessions | Hermes |
| `/<skill-name>` | activate packaged procedure | Hermes, Claude Code |

The `/memory` command is conspicuously absent in Claude Code itself — memory is managed by the model + harness, not the user. Direct manipulation is via filesystem.

### `/rewind` checkpoint / restore semantics

`Memory and Session Persistence.md:96-103`. Selecting a session mounts `ResumeConversation`:

> 1. **Worktree sync** — `restoreWorktreeForResume` aligns filesystem with session's git state
> 2. **Cost restore** — `restoreCostStateForSession` continues cumulative cost
> 3. **Metadata recovery** — `restoreSessionMetadata` restores configurations
> 4. **Agent state** — `restoreAgentFromSession` re-hydrates subagents and memory snapshots

`MessageSelector` powers `/rewind`: scroll prior `UserMessage` entries, pick restoration point, optionally restore code (`fileHistory` enabled), or "Summarize up to here" to compress prefix while keeping history.

### Redaction at export

`Memory and Session Persistence.md:104-106`. `redactSensitiveInfo` masks Anthropic/AWS/GCP keys, Bearer tokens, env vars before payload leaves the machine. `loadAllSubagentTranscriptsFromDisk` collects subagent transcripts. `getSanitizedErrorLogs` scrubs traces.

This is a **memory-export pipeline** more than a debug utility. AGH should treat memory-export as a first-class operator-controllable surface.

---

## 14. Synthesis: design tensions every harness wrestles with

Cross-cutting tensions extracted from Claude Code §11 (`Memory and Session Persistence.md:471-477`), Hermes §10 (`Learning Loop and Curated Memory.md:412-420`), OpenFang `Knowledge Graph Engine.md`:

1. **Trust the model vs trust the harness.** Claude Code consistently chooses the harness: format enforced, retrieval uses a separate (cheaper) model, deletion has no automatic trigger, staleness warnings injected, path security validated outside the model. Hermes leans on `MEMORY_GUIDANCE` — trusts the model more, accepts unsaved-context loss.

2. **Filesystem simplicity vs database correctness.** Filesystem-as-mutex, MEMORY.md 200-line cap, hash collisions, no transactional multi-document updates — these are the *price* of a markdown-on-disk design. SQLite gets atomic transactions and FTS5 but pays write-contention complexity (Hermes 15-retry jitter dance).

3. **Recall precision vs recall recall.** Sonnet selector biased to skip uncertain memories, top-5 hard budget, exclude prior turns. False negatives accepted as the price of avoiding false-positive clutter.

4. **Trust the model to save vs force structured extraction.** Hermes nudges; Claude Code forks an extraction subagent. Both have failure modes — Hermes loses unsaved context, Claude Code loses coalesced extraction windows.

5. **Flat markdown vs structured DB.** Hermes' `memory.md` is human-editable but has no indexing, no dedup, no staleness. Past a few hundred lines, contradictions accumulate.

6. **Skill explosion vs skill curation.** Auto-propose skills from completed tasks → unbounded growth. Manual review (`/skills list / delete`) is the only check.

7. **Per-tenant embeddings cost vs sharing.** Strict tenant isolation makes the same public document re-embedded per tenant. Shared embedding caches violate isolation.

8. **IVFFLAT vs HNSW.** Lower build cost and faster inserts vs slightly worse top-K recall. GoClaw chose IVFFLAT.

9. **LLM extraction vs rule-based NER.** LLM captures nuanced relations but costs an LLM call per document. NER + dependency parsing is cheaper but narrower. GoClaw chose LLM with `minConfidence=0.75` filter.

10. **Static workflow graph vs code-first composition.** OpenAI Agents SDK is code-first ("Agents.run(agent, [UserMessage(...)])") rather than declarative graphs. The trade-off: no upfront visualization, but full flexibility.

11. **Per-session settings vs global.** OpenClaw exposes per-session `model`, `thinkingLevel`, `verboseLevel`, `dmScope` — every session picks its own.

12. **Daily reset vs indefinite session.** OpenClaw's default 04:00 daily reset prevents unbounded transcript growth in always-on bots. Claude Code has no auto-reset; SM-Compact runs continuously.

---

## 15. What this means for AGH mem-v2 (synthesis only — no recommendations)

The patterns the harness corpus crystallizes around (in priority order of evidence-density):

### A. Five-tier memory taxonomy as orthogonal axes

Don't collapse "memory" into a single store. Distinct tiers (static project rules, session transcript, session summary, cross-session auto-memory, structured knowledge), each with its own:
- write protocol (manual / model-driven / forked-extractor / hook-injected)
- read protocol (always / on-demand / relevance-filtered / compaction-only)
- lifetime (per-step / per-session / per-day / indefinite)
- compaction policy (verbatim / summarize / evict / archive)
- scope (operator / user / project / team / agent / shared)

### B. Hierarchical scoped instruction files with explicit precedence

`AGENT.md` / `CLAUDE.md` / equivalent at: managed → user-global → project-shared → directory-deep → auto. Deeper overrides shallower; system/developer prompt outranks all. Bytes & lines caps with explicit truncation warnings (avoid silent loss). Path-security: only writable from operator-controlled scope.

### C. Compaction cascade with cost ordering

Free-first: tool-result-budget (per-message cap, persist large to disk) → snip (clear contents, keep tool_use/tool_result shape) → cached-microcompact (cache_edits piggyback) → time-based microcompact (cache cold anyway) → context-collapse (semantic, model-side) → SM-Compact (use pre-built session summary) → full-compaction (forked agent). Circuit breaker after 3 consecutive failures. 20K reserve for the compaction call itself. **Role-alternation + tool-pair invariants are non-optional.**

### D. Sub-agent extraction with sandboxed tool surface

Forked extractor with cache-shared parent, scoped `CanUseToolFn`, capped turns (5), explicit "do not verify" prompt, race-protection via `hasMemoryWritesSince`, single-slot pending context (acceptable loss), `MAX_MEMORY_FILES` pre-scan.

### E. Two-stage recall: cheap retrieval + LLM compress

FTS5 / pgvector / KG returns top-N → LLM (cheap model, e.g., Sonnet/Haiku/gpt-4o-mini) selects top-5 with `json_schema` typed output → caller materializes. Exclude already-surfaced files. Skip-if-uncertain bias. Failure → empty list, log debug, continue.

### F. Fixed memory-type taxonomy

`user`, `feedback`, `project`, `reference` (Claude Code's exact set) — minimum viable categorization with strong priors:
- save validations not just corrections
- include `**Why:**` and `**How to apply:**` body lines for `feedback` and `project`
- exclude code patterns / git history / debugging fixes / CLAUDE.md duplicates / ephemeral state — the explicit-save gate eval-validated this delta

### G. Lifecycle hooks at every memory-touching boundary

`session.start`, `session.end`, `pre-/post-tool-use`, `cwd-changed`, `before-/after-compaction`, `context.assemble`, `permission.request`. Structured JSON protocol that can block or modify. MDM-enforceable for enterprises.

### H. Filesystem-as-mutex for cross-process consolidation

Lock-file with PID body, mtime as `lastConsolidatedAt`, stale-after-N-minutes guard, write-then-re-read verification, rollback via `utimes()`. No `flock` dependency. Acceptable cost: crashed process holds lock briefly.

### I. Persistence-backend pluggability

Markdown + SQLite + FTS5 + (optional) vector + (optional) KG triple. Each pluggable per agent in `config.toml`. Default to inspectable plain-files; upgrade to SQLite where multi-writer concurrency demands it; KG triples for entity-rich domains.

### J. Truth-first staleness over auto-decay

`mtime`-based staleness banner injected with the surfaced memory ("This memory is N days old. Verify before acting."). Optional eval-time: H1 (verify file/function claims via grep) and H5 (treat snapshot memories as frozen-in-time). No automatic decay weighting yet — the field hasn't solved this.

### K. Compaction chains, not in-place mutation

Hermes pattern: new session with `parent_session_id` link. Preserves audit trail; cost-walk across chain; FTS5 hits any ancestor. Trade-off accepted: aggregate queries walk the chain.

### L. Failure modes as explicit features

Document the silent-failure surfaces (memory dir unreadable, side-query returns garbage, MEMORY.md truncation, race-coalescing loss). Surface them via `/insights`-style operator dashboards with confirmed-deficit metrics, not just alerts.

---

## 16. Source map (what was read, where it lives)

### Primary wiki articles consumed in full
- `~/dev/knowledge/ai-harness/wiki/concepts/Memory Systems for Agents.md` (640 lines) — taxonomy, persistence backends, compaction, lifecycle, self-healing, RAG vs memory boundary
- `~/dev/knowledge/ai-harness/wiki/concepts/Context Engineering.md` (448 lines) — context rot, layered architecture, ACE framework, ctx-zip, token budgeting, submodular curation
- `~/dev/knowledge/ai-harness/wiki/concepts/The Agent Harness.md` (501 lines) — agentic loop, tool dispatch, permissions, memory tiers, hooks, skills, ACI, configuration-as-behavior
- `~/dev/knowledge/ai-harness/wiki/concepts/Coding Agents Deep Dive.md` (558 lines) — Claude Code 8-phase memory, Cursor codebase indexing, Devin VM sandbox, Copilot, Windsurf SWE-1, MCP integration
- `~/dev/knowledge/ai-harness/outputs/briefings/State of AI Agent Harnesses 2025-2026.md` (240 lines) — three major shifts, MCP adoption, framework consolidation, infrastructure maturation

### Implementation-level wikis
- `~/dev/knowledge/claude-code/wiki/concepts/Memory and Session Persistence.md` (498 lines) — five-layer architecture, sanitizePath, scanMemoryFiles, createAutoMemCanUseTool, hasMemoryWritesSince, tryAcquireConsolidationLock, four-phase autoDream, Sonnet selector, staleness, silent-failure catalogue, context cost
- `~/dev/knowledge/claude-code/wiki/concepts/Token Budget and Context Compaction.md` (423 lines) — five-layer cascade, threshold constants, SM-Compact vs Full Compaction, post-compact cleanup, circuit breaker, lifecycle walkthrough
- `~/dev/knowledge/claude-code/wiki/concepts/Agentic Harness Design Patterns.md` (189 lines) — 12 reusable patterns across Memory & Context / Workflow & Orchestration / Tools & Permissions / Automation
- `~/dev/knowledge/hermes/wiki/concepts/Learning Loop and Curated Memory.md` (440 lines) — four-layer model, MemoryManager + provider plugins, MEMORY_GUIDANCE, session_search_tool, skill creation loop, /insights, prompt-cache integration
- `~/dev/knowledge/hermes/wiki/concepts/Session Store and FTS5 Recall.md` (456 lines) — schema, SessionDB, write contention with jitter, FTS5 search, two-stage recall, compression chains, performance envelope
- `~/dev/knowledge/openclaw/wiki/concepts/Sessions, DMs and Memory.md` (587 lines) — session lifecycle, DM scope (4 policies), JSONL on-disk, three memory backends, dual-write summary, crash recovery
- `~/dev/knowledge/openclaw/wiki/concepts/Context Compaction.md` (234 lines) — five-phase lifecycle, safeguard mode, model selection, JSONL rewrite, SessionToolResultGuard, before/after_compaction hooks
- `~/dev/knowledge/openfang/wiki/concepts/Knowledge Graph Engine.md` (525 lines) — RDF triple store, confidence scoring, BFS traversal, consolidation, contradiction detection, semantic + KG hybrid recall
- `~/dev/knowledge/goclaw/wiki/concepts/Memory and Knowledge Graph.md` (337 lines) — pgvector embeddings, IVFFLAT indexing, LLM-extracted KG, multi-tenant scoping, async deferred extraction

### Source code (primary truth, wiki may lag)
- `~/dev/compozy/agh/.resources/claude-code/memdir/memdir.ts` — `ENTRYPOINT_NAME`, `MAX_ENTRYPOINT_LINES = 200`, `MAX_ENTRYPOINT_BYTES = 25_000`, `truncateEntrypointContent`
- `~/dev/compozy/agh/.resources/claude-code/memdir/memoryTypes.ts` — full prompt-engineering of four memory types (`user` / `feedback` / `project` / `reference`), `WHAT_NOT_TO_SAVE_SECTION`, `MEMORY_DRIFT_CAVEAT`, `TRUSTING_RECALL_SECTION`, `MEMORY_FRONTMATTER_EXAMPLE`
- `~/dev/compozy/agh/.resources/claude-code/memdir/findRelevantMemories.ts` — `SELECT_MEMORIES_SYSTEM_PROMPT`, `selectRelevantMemories`, `json_schema` typed output, recent-tools filter
- `~/dev/compozy/agh/.resources/codex/codex-rs/core/src/compact.rs` — `SUMMARIZATION_PROMPT`, `SUMMARY_PREFIX`, `COMPACT_USER_MESSAGE_MAX_TOKENS = 20_000`, `InitialContextInjection` enum (`BeforeLastUserMessage` vs `DoNotInject`)
- `~/dev/compozy/agh/.resources/codex/codex-rs/core/templates/compact/prompt.md` — context-checkpoint compaction prompt
- `~/dev/compozy/agh/.resources/codex/codex-rs/core/templates/compact/summary_prefix.md` — handoff framing for receiving model
- `~/dev/compozy/agh/.resources/codex/codex-rs/core/src/context/user_instructions.rs` — `START_MARKER: "# AGENTS.md instructions for "`, `END_MARKER: "</INSTRUCTIONS>"`, per-directory wrapping
- `~/dev/compozy/agh/.resources/codex/codex-rs/core/hierarchical_agents_message.md` — AGENTS.md scope rules and override precedence
- `~/dev/compozy/agh/.resources/opencode/specs/v2/session-concepts-gap.md` — V1→V2 session model gap analysis: snapshots/patches, step boundaries, compaction (auto/overflow/tail_start_id), retries as parts vs aggregate, history filtering with retained tails

### Article corpus consumed
- `~/dev/knowledge/ai-harness/raw/articles/anthropic-building-effective-agents.md` — workflow vs agent distinction, 5 patterns (prompt-chaining / routing / parallelization / orchestrator-workers / evaluator-optimizer), ACI principles, poka-yoke, absolute-paths win
- `~/dev/knowledge/ai-harness/raw/articles/anthropic-claude-agent-sdk.md` — built-in tools (Read/Write/Edit/Bash/Glob/Grep/WebSearch/WebFetch/AskUserQuestion), file-based config (Skills/Commands/Memory/Plugins), session sources project flag
- `~/dev/knowledge/ai-harness/raw/articles/openai-practical-guide-building-agents.md` — three components (Model/Tools/Instructions), single vs multi-agent, manager vs decentralized, declarative vs code-first, layered guardrails, human-intervention triggers
- `~/dev/knowledge/ai-harness/raw/articles/aider-ai-pair-programming.md` — repo map via tree-sitter AST extraction, multi-language, auto-git-commit, 88% dogfood-generated
- `~/dev/knowledge/ai-harness/raw/articles/karpathy-llm-knowledge-bases.md` — 4-phase wiki cycle, no-RAG file-based KB, ~100 articles ~400K words
- `~/dev/knowledge/ai-harness/raw/articles/philschmid-agentic-context-engineering.md` — context-engineering definition, magical vs cheap-demo example
- `~/dev/knowledge/ai-harness/raw/articles/heynavtoor-karpathy-llm-wiki-extended-memory-lifecycle.md` — LLM-Wiki-v2 forgetting curves
- `~/dev/knowledge/ai-harness/raw/articles/vtrivedy10-harness-memory-context-fragments-bitter-lesson.md` — harness × memory × context-fragments × bitter-lesson intersection
- `~/dev/knowledge/ai-harness/raw/articles/theo-inject-dynamic-context-pattern-claude-code-skills.md` — skill standard for dynamic-context injection

### Bookmark cluster digests
- `~/dev/knowledge/ai-harness/raw/bookmarks/AI Bookmarks Agent Memory Deep.md` — ctx-zip, Memory 2.0, Memobase, mem0 multi-agent shared memory, MemFactory policy framing
- `~/dev/knowledge/ai-harness/raw/bookmarks/AI Bookmarks Knowledge Base and Memory.md` — Karpathy pattern (DataChaz), pluggable backends (Teknium/Hermes), memory-as-policy (rryssf_/MemFactory), Claude Code 8-phase anatomy (ellen_in_sf)
- `~/dev/knowledge/ai-harness/raw/bookmarks/AI Bookmarks Claude Code Architecture.md` — skills/hooks/subagents extensibility, Conductor parallel CC, taskmaster MCP sampling, reverse-engineering, Claude Code on web

### What was NOT read in full but indexed
- The full ~150 articles in `~/dev/knowledge/ai-harness/raw/articles/` — sampled by topic relevance, full reads on 8 of the most directly memory/context-relevant
- Sub-wikis under each topic (`hermes/wiki/concepts/`, `openclaw/wiki/concepts/`, etc.) — read the memory + compaction articles per topic; skipped tools / channels / authentication / scheduling articles unless they referenced memory mechanics
- `~/dev/knowledge/ai-harness/outputs/queries/` — confirmed Workspace, Skill systems comparisons exist; read briefing instead for the consolidated synthesis
