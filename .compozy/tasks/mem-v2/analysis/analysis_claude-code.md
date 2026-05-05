# Claude Code — Memory & Context Analysis

> Source corpus: `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/` (full TS source)
> Read-only analysis written for AGH `mem-v2` redesign.

---

## 1. TL;DR (200 words)

Claude Code's memory system has **two cleanly separated layers**.

**Layer A — File-loaded instructions** (`utils/claudemd.ts`): hierarchical discovery of `CLAUDE.md` + `.claude/CLAUDE.md` + `.claude/rules/*.md` + `CLAUDE.local.md` walked from CWD up to root, plus managed (`/etc/...`) and user (`~/.claude/CLAUDE.md`) layers. Loaded reverse-priority (root → CWD), supports `@include`, has setting-source gating, and is pasted into the system prompt via a fixed preamble.

**Layer B — Auto-memory ("memdir")** (`memdir/*`): per-project directory at `~/.claude/projects/<sanitized-git-root>/memory/` containing a `MEMORY.md` index plus topic files with YAML frontmatter (`name`, `description`, `type ∈ {user,feedback,project,reference}`). Index is always loaded into the system prompt, capped at 200 lines / 25 KB. Topic files are recalled on-demand via a Sonnet side-query that ranks frontmatter descriptions against the user's input, then injected as `<system-reminder>` attachments per turn (top 5, with age-based freshness caveats).

**Writes** happen two ways: the main agent writes via Write/Edit (using the same memory-prompt instructions), or, if it didn't, a forked Sonnet "extract-memories" subagent runs at the end of every Stop turn — same prompt cache, restricted toolset, max 5 turns.

**Compaction** is a separate concern — pure prose summarization with `<analysis>`/`<summary>` blocks; PreCompact/PostCompact hooks; `setCwd`-style cleanup. Sessions persist as `.jsonl` per session id under the per-project dir; `/clear` regenerates session id and runs `SessionEnd`/`SessionStart` hooks.

---

## 2. memdir architecture (file-by-file)

All paths below are relative to `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/memdir/`.

### `memoryTypes.ts` — taxonomy + prompt sections

- `MEMORY_TYPES = ['user','feedback','project','reference']` (closed set, `as const`); `MemoryType = (typeof MEMORY_TYPES)[number]`.
- `parseMemoryType(raw): MemoryType | undefined` — graceful degradation for legacy/unknown.
- Two big prompt-section constants:
  - `TYPES_SECTION_INDIVIDUAL` (auto-only mode) — XML-style `<types><type><name>...<description>...<when_to_save>...<how_to_use>...<examples>...</type>...` with the four types and "[saves user memory: …]" example syntax (`memoryTypes.ts:113-178`).
  - `TYPES_SECTION_COMBINED` (auto+team mode) — same shape but each `<type>` adds `<scope>always private | default to private … | private or team but bias team | usually team</scope>` and `<body_structure>` for `feedback`/`project` types (`memoryTypes.ts:37-106`).
  - The two are kept duplicated, intentionally — comment: "intentionally duplicated rather than generated from a shared spec — keeping them flat makes per-mode edits trivial without reasoning through a helper's conditional rendering" (`memoryTypes.ts:8-12`).
- `WHAT_NOT_TO_SAVE_SECTION` (`memoryTypes.ts:183-195`) — code patterns, git history, CLAUDE.md content, ephemeral state — explicitly forbidden even on user request: "If they ask you to save a PR list or activity summary, ask what was *surprising* or *non-obvious* about it — that is the part worth keeping."
- `MEMORY_DRIFT_CAVEAT` (`memoryTypes.ts:201-202`) — single recall-side bullet: "Memory records can become stale … verify that the memory is still correct and up-to-date by reading the current state."
- `WHEN_TO_ACCESS_SECTION` (`:216-222`) — ignore-on-explicit-ask rule; H6 eval-validated.
- `TRUSTING_RECALL_SECTION` (`:240-256`) — header is **"## Before recommending from memory"** (action-cue, eval-validated 3/3 vs 0/3 for "Trusting what you recall"). Body: "If the memory names a file path: check the file exists. If the memory names a function or flag: grep for it." Plus snapshot guidance.
- `MEMORY_FRONTMATTER_EXAMPLE` (`:261-271`):

```markdown
---
name: {{memory name}}
description: {{one-line description — used to decide relevance in future conversations, so be specific}}
type: {{user, feedback, project, reference}}
---

{{memory content — for feedback/project types, structure as: rule/fact, then **Why:** and **How to apply:** lines}}
```

### `memoryScan.ts` — scan + manifest

```ts
export type MemoryHeader = {
  filename: string
  filePath: string
  mtimeMs: number
  description: string | null
  type: MemoryType | undefined
}

const MAX_MEMORY_FILES = 200
const FRONTMATTER_MAX_LINES = 30
```

- `scanMemoryFiles(memoryDir, signal)` (`:35-77`): `readdir(..., {recursive: true})`, filter `.md` excluding `MEMORY.md`, parallel `readFileInRange(path, 0, 30, undefined, signal)` (each call `stat`s + reads first 30 lines), `parseFrontmatter`, sort newest-first by mtime, slice to 200. Single-pass: "readFileInRange stats internally and returns mtimeMs, so we read-then-sort rather than stat-sort-read."
- `formatMemoryManifest(memories)` (`:84-94`): one line per file as `- [type] filename (ISO-timestamp): description`. Used by both the recall-time selector AND the extraction-agent prompt (the extractor doesn't waste a turn on `ls`).

### `memoryAge.ts` — freshness rendering

- `memoryAgeDays(mtimeMs)` floor-rounds; `memoryAge(mtimeMs)` → `"today" | "yesterday" | "{n} days ago"` (models reason poorly about ISO arithmetic).
- `memoryFreshnessText(mtimeMs)` returns staleness caveat for >1 day: "This memory is N days old. Memories are point-in-time observations, not live state — claims about code behavior or file:line citations may be outdated. Verify against current code before asserting as fact." Returns `''` for fresh memories — warning there is noise.
- `memoryFreshnessNote(mtimeMs)` wraps in `<system-reminder>...</system-reminder>\n` for callers without their own wrapper.

### `paths.ts` — path resolution + gating

- `isAutoMemoryEnabled()` (`:30-55`) priority chain (first defined wins):
  1. `CLAUDE_CODE_DISABLE_AUTO_MEMORY` env (truthy → off, falsy → on)
  2. `CLAUDE_CODE_SIMPLE` (`--bare`) → off
  3. `CLAUDE_CODE_REMOTE` without `CLAUDE_CODE_REMOTE_MEMORY_DIR` → off
  4. `settings.autoMemoryEnabled`
  5. default: enabled
- `isExtractModeActive()` (`:69-77`) gates the *forked extraction agent* — gated on `tengu_passport_quail` GB flag + (interactive OR `tengu_slate_thimble`).
- `getMemoryBaseDir()`: `CLAUDE_CODE_REMOTE_MEMORY_DIR` env override → else `~/.claude`.
- `validateMemoryPath(raw, expandTilde)` (`:109-150`) — security: rejects relative, root-near (<3 chars), drive root, UNC, null bytes; expands `~/`; rejects bare `~`/`~/.`/`~/..`; NFC-normalizes.
- `getAutoMemPath` (memoized on `getProjectRoot()`): resolution order
  1. `CLAUDE_COWORK_MEMORY_PATH_OVERRIDE` env (Cowork SDK)
  2. `autoMemoryDirectory` setting from policy/flag/local/user (NOT projectSettings — security: malicious repo could redirect to `~/.ssh`)
  3. `<base>/projects/<sanitizePath(canonical-git-root)>/memory/` — uses `findCanonicalGitRoot` so all worktrees of the same repo share one memory dir (anthropics/claude-code#24382).
- `getAutoMemDailyLogPath(date)`: `<auto>/logs/YYYY/MM/YYYY-MM-DD.md` for KAIROS daily-log mode.
- `isAutoMemPath(path)` — used by the FileWriteTool carve-out and tool-permission gate.

### `teamMemPaths.ts` — team scope security

- `isTeamMemoryEnabled()` requires both `isAutoMemoryEnabled()` AND `tengu_herring_clock` GB flag.
- `getTeamMemPath()` = `<autoMem>/team/`.
- Heavy security focus (PSR M22186): `validateTeamMemWritePath`, `validateTeamMemKey` — null-byte / URL-encoded / NFKC-normalized / backslash / absolute / dangling-symlink / symlink-loop checks via `realpathDeepestExisting()` walk-up. Two-pass: string-level prefix check then real-path containment.

### `teamMemPrompts.ts` — combined prompt builder

`buildCombinedMemoryPrompt(extraGuidelines?, skipIndex)` (`:22-100`) — two-directory header (private + team), `Memory scope` section explaining the split, `TYPES_SECTION_COMBINED`, two-step save (write file in chosen dir, point at *that* dir's `MEMORY.md`), explicit "MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials."

### `memdir.ts` — orchestration + truncation + injection point

Most-loaded module — exported by the memory section of the system prompt builder.

- `ENTRYPOINT_NAME = 'MEMORY.md'`
- `MAX_ENTRYPOINT_LINES = 200`
- `MAX_ENTRYPOINT_BYTES = 25_000` ("~125 chars/line at 200 lines. At p97 today; catches long-line indexes that slip past the line cap (p100 observed: 197KB under 200 lines)")
- `truncateEntrypointContent(raw)` (`:57-103`) — line-truncate first (natural boundary) then byte-truncate at last newline before cap. Appends a `> WARNING: MEMORY.md is N lines (limit 200) — index entries are too long. Only part of it was loaded. Keep index entries to one line under ~200 chars; move detail into topic files.`
- `ensureMemoryDirExists(memoryDir)` (`:129-147`) — idempotent recursive `mkdir`; harness guarantees existence so the model can `Write` without checking.
- `DIR_EXISTS_GUIDANCE = 'This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence).'` ("Shipped because Claude was burning turns on `ls`/`mkdir -p` before writing.")
- `buildMemoryLines(displayName, memoryDir, extraGuidelines?, skipIndex)` (`:199-266`) — assembles the system-prompt section: header + dir-exists guidance + intent paragraph + remember/forget rule + `TYPES_SECTION_INDIVIDUAL` + `WHAT_NOT_TO_SAVE` + `## How to save memories` (two-step: write file with frontmatter; pointer in `MEMORY.md`) + `WHEN_TO_ACCESS` + `TRUSTING_RECALL` + persistence-vs-plans-vs-tasks distinction + `buildSearchingPastContextSection`.
- `buildSearchingPastContextSection(autoMemDir)` (`:375-407`) — gated on `tengu_coral_fern`. Tells the model to grep its own memory dir before grepping session transcripts: `Grep with pattern="<term>" path="<memDir>" glob="*.md"` then session jsonl as last resort.
- `loadMemoryPrompt()` (`:419-507`) — top-level dispatcher:
  1. KAIROS-active → `buildAssistantDailyLogPrompt(skipIndex)` (append-only daily log; nightly `/dream` distills into `MEMORY.md`).
  2. TEAMMEM enabled → `ensureMemoryDirExists(teamDir)` then `buildCombinedMemoryPrompt(...)`.
  3. auto enabled → `ensureMemoryDirExists(autoDir)` then `buildMemoryLines('auto memory', ...)`.
  4. else → log `tengu_memdir_disabled` + return null.
- `CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES` env var lets Cowork inject extra guideline lines into all builders.

### `findRelevantMemories.ts` — recall-time selector

The hottest read path. Core anchor `:18-141`.

```ts
export type RelevantMemory = { path: string; mtimeMs: number }

const SELECT_MEMORIES_SYSTEM_PROMPT = `You are selecting memories that will be useful to Claude Code as it processes a user's query. You will be given the user's query and a list of available memory files with their filenames and descriptions.

Return a list of filenames for the memories that will clearly be useful to Claude Code as it processes the user's query (up to 5). Only include memories that you are certain will be helpful based on their name and description.
- If you are unsure if a memory will be useful in processing the user's query, then do not include it in your list. Be selective and discerning.
- If there are no memories in the list that would clearly be useful, feel free to return an empty list.
- If a list of recently-used tools is provided, do not select memories that are usage reference or API documentation for those tools (Claude Code is already exercising them). DO still select memories containing warnings, gotchas, or known issues about those tools — active use is exactly when those matter.
`

export async function findRelevantMemories(
  query: string,
  memoryDir: string,
  signal: AbortSignal,
  recentTools: readonly string[] = [],
  alreadySurfaced: ReadonlySet<string> = new Set(),
): Promise<RelevantMemory[]>
```

Algorithm (`findRelevantMemories.ts:39-141`):
1. `scanMemoryFiles(memoryDir, signal)` returns up to 200 most-recent headers.
2. Filter out `alreadySurfaced` paths (caller passes a Set of paths injected in earlier turns).
3. If 0 memories → return `[]`.
4. `selectRelevantMemories` builds the manifest text and calls `sideQuery({ model: getDefaultSonnetModel(), system: SELECT_MEMORIES_SYSTEM_PROMPT, skipSystemPromptPrefix: true, max_tokens: 256, output_format: { type: 'json_schema', schema: { selected_memories: array<string> } }, querySource: 'memdir_relevance' })`.
5. Parse `selected_memories`, intersect with `validFilenames`, return top entries (up to 5) as `{ path, mtimeMs }[]`.
6. Telemetry under `MEMORY_SHAPE_TELEMETRY` feature flag — `-1` ages distinguish "ran, picked nothing" from "never ran" (`:65-72`).

`recentTools` is critical: comment block at `:86-95` says "When Claude Code is actively using a tool (e.g. mcp__X__spawn), surfacing that tool's reference docs is noise — the conversation already contains working usage. The selector otherwise matches on keyword overlap … false positive." The system prompt explicitly tells Sonnet to keep warnings/gotchas about those tools but drop docs.

---

## 3. Auto-memory write path

### Trigger

`stopHooks.ts:148-149` — `void extractMemoriesModule!.executeExtractMemories(context, appendSystemMessage)` fired at the end of every query loop where the assistant produced a final response with no tool calls (i.e. a "Stop" event). Fire-and-forget; awaited later by `cli/print.ts:968` via `drainPendingExtraction()` before graceful shutdown (60 s soft timeout).

### Two write paths (mutually exclusive per turn)

The main agent's system prompt **always** contains `buildMemoryLines(...)` (the same save instructions). So:

- **Path 1 — main agent writes**: it calls `Write`/`Edit` on a path inside `getAutoMemPath()`. `extractMemories.hasMemoryWritesSince(messages, lastMemoryMessageUuid)` (`extractMemories.ts:121-148`) detects this by scanning `tool_use` blocks for `FILE_EDIT_TOOL_NAME`/`FILE_WRITE_TOOL_NAME` with `file_path` inside `isAutoMemPath`. If so, the forked extractor is **skipped** and the cursor advances past these messages.
- **Path 2 — forked extractor**: if the main agent didn't write, `runExtraction()` (`extractMemories.ts:329-523`) spawns a forked Sonnet subagent.

### Forked extractor mechanics

- Fork via `runForkedAgent({ promptMessages, cacheSafeParams, canUseTool, querySource: 'extract_memories', forkLabel: 'extract_memories', skipTranscript: true, maxTurns: 5 })` — "perfect fork of the main conversation that shares the parent's prompt cache" (`:9-10`). `cacheSafeParams = createCacheSafeParams(context)` keeps the prompt prefix identical so cache hits.
- `canUseTool = createAutoMemCanUseTool(memoryDir)` (`:171-222`) — locked-down toolset:
  - `Read`, `Grep`, `Glob` allowed unrestricted.
  - `Bash` allowed only if `BashTool.isReadOnly(parsed)` (ls/find/grep/cat/stat/wc/head/tail and similar). `rm` denied. Comment: "All other tools — MCP, Agent, write-capable Bash, etc — will be denied."
  - `Edit`/`Write` allowed only when `isAutoMemPath(input.file_path)` — i.e. inside the memory dir.
  - `REPL` allowed (re-invokes canUseTool per inner primitive).
- Skip transcript writing to avoid race with main thread.

### Throttle / coalescing

- `tengu_bramble_lintel` (default 1) controls "every N eligible turns". `runExtraction` increments `turnsSinceLastExtraction` and short-circuits until the threshold (`:374-386`).
- `inProgress` flag prevents overlapping runs. If a call lands while one is running, the latest `{context, appendSystemMessage}` is stashed in `pendingContext`; the trailing run executes after the current finishes (`:319-321`, `:557-565`).
- `inFlightExtractions = Set<Promise<void>>` (`:303`) tracks every promise; `drainPendingExtraction(timeoutMs=60_000)` (`:579-586`) is awaited before exit.
- Closure-scoped state (cursor `lastMemoryMessageUuid`, overlap guard, pending context) — fresh per `initExtractMemories()` call (testable, no module-level mutation).
- Subagent gate: `executeExtractMemories` early-returns if `context.toolUseContext.agentId` is set (`:532` — only the main agent extracts).
- Remote-mode skip (`getIsRemoteMode()` returns true → no extraction).

### Extraction prompt (`services/extractMemories/prompts.ts`)

`opener(newMessageCount, existingMemories)` (`:29-44`):

```
You are now acting as the memory extraction subagent. Analyze the most recent ~{N} messages above and use them to update your persistent memory systems.

Available tools: Read, Grep, Glob, read-only Bash (ls/find/cat/stat/wc/head/tail and similar), and Edit/Write for paths inside the memory directory only. Bash rm is not permitted. All other tools — MCP, Agent, write-capable Bash, etc — will be denied.

You have a limited turn budget. Edit requires a prior Read of the same file, so the efficient strategy is: turn 1 — issue all Read calls in parallel for every file you might update; turn 2 — issue all Write/Edit calls in parallel. Do not interleave reads and writes across multiple turns.

You MUST only use content from the last ~{N} messages to update your persistent memories. Do not waste any turns attempting to investigate or verify that content further — no grepping source files, no reading code to confirm a pattern exists, no git commands.

## Existing memory files

- [feedback] feedback_ci_no_cron.md (2026-04-15T...): Never propose schedule/cron workflows
- ...

Check this list before writing — update an existing file rather than creating a duplicate.
```

After the opener: `TYPES_SECTION_INDIVIDUAL` (or `_COMBINED`), `WHAT_NOT_TO_SAVE`, two-step save instructions. Combined variant adds: `- You MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials.`

### Persistence + dedup

- File-system based — `Write`/`Edit` directly. No database.
- Dedup: `existingMemories` manifest is pre-injected into the prompt; the extractor is told "Check this list before writing — update an existing file rather than creating a duplicate."
- Index update: prompted as Step 2 — "add a pointer to that file in `MEMORY.md`. `MEMORY.md` is an index, not a memory — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. It has no frontmatter. Never write memory content directly into `MEMORY.md`." Index updates are `extractWrittenPaths`-detected but excluded from "memories saved" count: `memoryPaths = writtenPaths.filter(p => basename(p) !== 'MEMORY.md')` (`:464-467`).
- Telemetry: `tengu_extract_memories_extraction` event with token counts, `files_written`, `memories_saved`, `team_memories_saved`, `duration_ms`.
- UI surfacing: `createMemorySavedMessage(memoryPaths)` is appended via `appendSystemMessage?.(msg)` so the user sees what was saved.

### Cursor management

- `lastMemoryMessageUuid` advances to `messages.at(-1).uuid` only after a successful run (so errors retry on next turn).
- Compaction-tolerant: if `sinceUuid` was lost (compaction removed it), `countModelVisibleMessagesSince` falls back to counting all model-visible messages rather than returning 0 (which would permanently disable extraction post-compact).

---

## 4. Auto-memory read path

`findRelevantMemories` is called from `utils/attachments.ts:2196-2242` in `getRelevantMemoryAttachments`:

```ts
async function getRelevantMemoryAttachments(
  input: string,
  agents: AgentDefinition[],
  readFileState: FileStateCache,
  recentTools: readonly string[],
  signal: AbortSignal,
  alreadySurfaced: ReadonlySet<string>,
): Promise<Attachment[]>
```

Algorithm:
1. `extractAgentMentions(input)` — if user `@`-mentions an agent (e.g. `@agent-debugger`), search **only that agent's memory dir** (isolation). Otherwise search the auto-memory dir.
2. Per-dir, call `findRelevantMemories(input, dir, signal, recentTools, alreadySurfaced).catch(() => [])`.
3. Flatten + filter by `readFileState` (model already read it via FileReadTool) AND `alreadySurfaced` (belt-and-suspenders) → top 5.
4. `readMemoriesForSurfacing(selected, signal)` — reads with `MAX_MEMORY_LINES`/`MAX_MEMORY_BYTES` truncation via `readFileInRange`'s `truncateOnByteLimit` (truncate-with-note rather than drop — frontmatter+opening is worth surfacing).
5. Returns `[{ type: 'relevant_memories', memories: [{ path, content, mtimeMs, header?, limit? }, ...] }]`.

**Injection** — `utils/messages.ts:3708-3722`:

```ts
case 'relevant_memories': {
  return wrapMessagesInSystemReminder(
    attachment.memories.map(m => {
      const header = m.header ?? memoryHeader(m.path, m.mtimeMs)
      return createUserMessage({
        content: `${header}\n\n${m.content}`,
        isMeta: true,
      })
    }),
  )
}
```

The header is **pre-computed at attachment-creation time** so the rendered bytes are stable across turns (otherwise `memoryAge(mtimeMs)` calls `Date.now()` and "saved 3 days ago" → "saved 4 days ago" busts the prompt cache).

**Surfaced-paths bookkeeping** — `collectSurfacedMemories(messages)` (`attachments.ts:2251-2266`) walks all message attachments, collects `{paths: Set<string>, totalBytes: number}`. Compaction naturally resets both because old attachments aren't in the compacted transcript.

**Throttling** — only top-5 per turn after dedup; `alreadySurfaced` is filtered *inside* `findRelevantMemories` before the Sonnet call so the 5-slot budget goes to fresh candidates, not re-picks.

### Read-time freshness

Each rendered memory injection **already lives inside `wrapMessagesInSystemReminder`** so the system-reminder wrapping is provided by the caller. `memoryFreshnessText(mtimeMs)` is used elsewhere (FileReadTool output) where the consumer doesn't add its own wrapper. The recall-side prompt also includes `MEMORY_DRIFT_CAVEAT` at the bottom of `WHEN_TO_ACCESS` — "Memory records can become stale … verify against the current state."

### `appendSystemPrompt` for high-confidence guidance

The `TRUSTING_RECALL_SECTION` (header "## Before recommending from memory") is eval-validated to be more effective when delivered via `appendSystemPrompt` rather than as a bullet — position matters. AGH should consider an analogous "tail-injected reminder" hook for high-priority recall guidance.

---

## 5. File conventions (CLAUDE.md / MEMORY.md / agent-md)

### Static instruction layers (`utils/claudemd.ts`)

Loaded **bottom-up** (lowest → highest priority, last wins):
1. **Managed** — e.g. `/etc/claude-code/CLAUDE.md` plus `/etc/claude-code/.claude/rules/*.md` (policy-pinned, always loaded).
2. **User** — `~/.claude/CLAUDE.md` + `~/.claude/rules/*.md` (gated on `userSettings`).
3. **Project** — walk from CWD up to filesystem root, collect `<dir>/CLAUDE.md`, `<dir>/.claude/CLAUDE.md`, `<dir>/.claude/rules/*.md` (gated on `projectSettings`). Deeper-CWD dirs load **last** (highest priority).
4. **Local** — `<dir>/CLAUDE.local.md` (gated on `localSettings`, gitignored).
5. **Additional dirs** — `--add-dir` directories if `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD` truthy.
6. **Memdir entrypoint** — `getAutoMemEntrypoint()` (i.e. `MEMORY.md` from auto-memory) appended last.

System-prompt preamble: `MEMORY_INSTRUCTION_PROMPT = 'Codebase and user instructions are shown below. Be sure to adhere to these instructions. IMPORTANT: These instructions OVERRIDE any default behavior and you MUST follow them exactly as written.'`

Key invariants:
- `MAX_MEMORY_CHARACTER_COUNT = 40000` per file (recommended cap).
- `@include` directive supported in leaf text nodes (`@path`, `@./`, `@~/`, `@/abs`); recursion guard prevents cycles.
- Worktree handling: `findCanonicalGitRoot` deduplicates parent-repo + worktree CLAUDE.md (otherwise both load).
- `claudeMdExcludes` setting (only from non-project sources for security) skips listed files.
- Cache: `getMemoryFiles = memoize(...)` with explicit `clearMemoryFileCaches()`; reset triggers include `/clear`, `/compact`, `/memory`.

### Memdir entrypoint (`MEMORY.md`)

- Always loaded into the system prompt via `loadMemoryPrompt()`.
- Hard caps: **200 lines** (`MAX_ENTRYPOINT_LINES`) **or 25 KB** (`MAX_ENTRYPOINT_BYTES`), whichever fires first.
- On overflow: line-truncate, then byte-truncate at last newline before cap, append warning that names which cap fired and tells the model to keep entries to ~200-char one-liners.
- **Format**: pure markdown bulleted index, NO frontmatter:
  ```markdown
  - [Title](file.md) — one-line hook
  - [User Role](user_role.md) — Pedro is a senior Go engineer working on AGH
  ```
- Topic files (referenced from MEMORY.md) DO have YAML frontmatter (`name`, `description`, `type`).

### Hot reload

- Memory files are NOT hot-reloaded mid-conversation. `clearMemoryFileCaches()` is called explicitly on `/memory` (after edit) and on `/compact`.
- The memdir is re-scanned every turn at recall time (`scanMemoryFiles` is uncached) — so write→read in the same session sees fresh state.

### `AGENTS.md`

I did not find any `AGENTS.md` discovery in `claudemd.ts` — Claude Code is `CLAUDE.md`-only. Subagents have a separate `.claude/agents/<name>.md` system, but those are agent definitions (frontmatter + system prompt) loaded via `loadAgentsDir.ts`, not memory files.

---

## 6. Compaction

### `/compact` flow (`commands/compact/compact.ts`)

1. Drop messages before any prior `compactBoundary` (`getMessagesAfterCompactBoundary` — REPL keeps snipped messages for scrollback but they shouldn't reach the compactor).
2. **Try session memory compaction first** if no custom instructions — `trySessionMemoryCompaction(messages, agentId)` (`services/SessionMemory/`). This is a separate, gentler path that prunes around an already-maintained running summary. If it succeeds, skip prose summarization entirely.
3. **Reactive-only mode** (`tengu_cobalt_raccoon` GB flag, `feature('REACTIVE_COMPACT')`) — route through `services/compact/reactiveCompact.ts` which only fires when the API returns `prompt_too_long`. Otherwise:
4. **Microcompact** (`microcompactMessages`) — first reduces tokens by stripping low-value content before summarization.
5. **`compactConversation(messages, context, cacheSafeParams, suppressUserQuestions=false, customInstructions, isAutoCompact=false)`** — full prose summary (see prompt below).
6. Post-success: `setLastSummarizedMessageId(undefined)`, `suppressCompactWarning()`, `getUserContext.cache.clear()`, `runPostCompactCleanup()`.

### Compaction prompt (`services/compact/prompt.ts`)

Three variants:
- `BASE_COMPACT_PROMPT` — full conversation; 9 sections (Primary Request and Intent / Key Technical Concepts / Files and Code Sections / Errors and fixes / Problem Solving / All user messages / Pending Tasks / Current Work / Optional Next Step).
- `PARTIAL_COMPACT_PROMPT` — recent portion only (used when earlier context is preserved verbatim).
- `PARTIAL_COMPACT_UP_TO_PROMPT` — for "up_to" direction: "summary will be placed at the start of a continuing session; newer messages that build on this context will follow after your summary." Replaces "Optional Next Step" with "Context for Continuing Work".

All variants are wrapped with:
- **`NO_TOOLS_PREAMBLE`** (first): "CRITICAL: Respond with TEXT ONLY. Do NOT call any tools. … Tool calls will be REJECTED and will waste your only turn — you will fail the task. Your entire response must be plain text: an `<analysis>` block followed by a `<summary>` block."
- **`NO_TOOLS_TRAILER`**: "REMINDER: Do NOT call any tools. Respond with plain text only — an `<analysis>` block followed by a `<summary>` block. Tool calls will be rejected and you will fail the task."

The cache-sharing fork inherits the parent's full tool set (required for cache-key match), and on Sonnet 4.6+ adaptive-thinking models the model sometimes attempts a tool call despite the trailer alone — putting NO_TOOLS first cut tool-call rate from 2.79% on 4.6 to 0.01% on 4.5.

### Output format

`<analysis>...drafting scratchpad...</analysis><summary>...</summary>` — `formatCompactSummary` strips `<analysis>` (drafting scratchpad with no informational value) and replaces `<summary>` tags with a `Summary:` header.

### Continuation message (`getCompactUserSummaryMessage`)

```
This session is being continued from a previous conversation that ran out of context. The summary below covers the earlier portion of the conversation.

{formattedSummary}

If you need specific details from before compaction (like exact code snippets, error messages, or content you generated), read the full transcript at: {transcriptPath}

Recent messages are preserved verbatim.

[if suppressFollowUpQuestions:]
Continue the conversation from where it left off without asking the user any further questions. Resume directly — do not acknowledge the summary, do not recap what was happening, do not preface with "I'll continue" or similar. Pick up the last task as if the break never happened.
```

If proactive/KAIROS active: extra paragraph "You are running in autonomous/proactive mode. This is NOT a first wake-up — you were already working autonomously before compaction. Continue your work loop …".

### What survives compaction

- The compact summary message (above), as a synthetic user message.
- Any messages preserved by `up_to`/`from` partial-compact directives (preserved **verbatim** — that's literally the design).
- `compactBoundary` system-message marker (`createCompactBoundaryMessage`) at the boundary.
- File state cache (`readFileState` is restored selectively via `POST_COMPACT_TOKEN_BUDGET = 50_000`, `POST_COMPACT_MAX_FILES_TO_RESTORE = 5`, `POST_COMPACT_MAX_TOKENS_PER_FILE = 5_000`).
- Skills are re-injected up to `POST_COMPACT_MAX_TOKENS_PER_SKILL = 5_000` × `POST_COMPACT_SKILLS_TOKEN_BUDGET = 25_000`.

### What's discarded

- Image blocks are stripped before compaction (`stripImagesFromMessages`) — replaced with text marker — to keep the compactor itself from blowing the context limit.
- `relevant_memories` attachments — lost when the messages are summarized; that's why `collectSurfacedMemories` resetting on compact is by design ("re-surfacing is valid again").

### Auto-compact (`services/compact/autoCompact.ts`)

- `getAutoCompactThreshold(model) = effectiveContextWindow - AUTOCOMPACT_BUFFER_TOKENS (13_000)`.
- `MAX_OUTPUT_TOKENS_FOR_SUMMARY = 20_000` (p99.99 observed = 17,387).
- `MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES = 3` — circuit breaker after 3 consecutive failures (one BQ caught 1,279 sessions hammering the API ~250K calls/day).
- Recursion guards: skip if `querySource === 'session_memory'` or `'compact'` or `'marble_origami'` (the ctx-agent — autocompact during it would corrupt module-level commit state).

### Hooks

- **`PreCompact`** hook (`utils/hooks.ts:3961` `executePreCompactHooks`) — gets `{ trigger: 'manual' | 'auto', customInstructions: string | null }`. Hook can return `newCustomInstructions` (merged via `mergeHookInstructions`) and a `userDisplayMessage`. Reactive-mode runs PreCompact concurrently with `getCacheSharingParams`.
- **`PostCompact`** hook — `executePostCompactHooks`, runs after summary is in place.

---

## 7. Session persistence

### Storage

- Per-project dir: `<projectDir> = getProjectDir(getOriginalCwd())` — sanitized path under `~/.claude/projects/<sanitized-cwd>/`.
- Session transcript: `<projectDir>/<sessionId>.jsonl` (`getTranscriptPath`).
- Per-agent transcript: `<projectDir>/<sessionId>/agent-<agentId>.jsonl` and `.meta.json`.
- Worktree state: `saveWorktreeState(...)`.
- Mode: `saveMode('coordinator' | 'normal')`.

### `/clear` (`commands/clear/conversation.ts`)

Steps:
1. `executeSessionEndHooks('clear', {timeoutMs: getSessionEndHookTimeoutMs()})` — env `CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS`, default 1.5 s.
2. `setMessages(() => [])`.
3. `clearSessionCaches(preservedAgentIds)` — but preserves background tasks (`isBackgrounded !== false`).
4. `setCwd(getOriginalCwd())`, `readFileState.clear()`, `loadedNestedMemoryPaths?.clear()`.
5. `clearAllPlanSlugs()`, `clearSessionMetadata()`.
6. `regenerateSessionId({setCurrentAsParent: true})` — new session id, old as parent (lineage tracking).
7. `resetSessionFilePointer()`, `saveWorktreeState`, `saveMode`.
8. `processSessionStartHooks('clear')` — fires `SessionStart` hook; resulting messages become the first messages of the new session.

### `/resume` (`commands/resume/resume.tsx`)

Opens a session selector UI (lists `.jsonl` files under the project dir). Chosen session id reroutes `getTranscriptPath` to that file. Session state is rehydrated from the transcript (`hydrateSessionFromRemote` for remote, equivalent local logic).

### `/rewind` (`commands/rewind/rewind.ts`)

Trivial wrapper — calls `context.openMessageSelector()` which opens a UI for picking a message to rewind to. The selector trims the transcript to that point. Returns `{ type: 'skip' }` so no system message is appended.

---

## 8. Sub-agent memory propagation

Two distinct categories:

### Task-tool subagents (general-purpose / agent-* ones)

- `services/extractMemories/extractMemories.ts:532` — `if (context.toolUseContext.agentId) return` — **subagents do not extract memories**. Only the main agent.
- Subagents inherit the parent's system prompt (cache-share invariant). Auto-memory's `MEMORY.md` index is in that system prompt, so subagents **read** the index but don't write.
- For relevant-memory recall: `getRelevantMemoryAttachments` is called from the main agent's user-prompt-handling path. Subagents get whatever attachments their spawn included — they don't run their own per-turn recall.

### Persistent agent memory (`tools/AgentTool/agentMemory.ts`)

When an agent definition declares `memory: 'user' | 'project' | 'local'`, that agent gets its OWN persistent memory directory:

- `'user'` → `<memoryBase>/agent-memory/<agentType>/`
- `'project'` → `<cwd>/.claude/agent-memory/<agentType>/`
- `'local'` → `<cwd>/.claude/agent-memory-local/<agentType>/` (or remote equivalent)

`loadAgentMemoryPrompt(agentType, scope)` (`agentMemory.ts:138-177`) calls `buildMemoryPrompt({ displayName: 'Persistent Agent Memory', memoryDir, extraGuidelines: [scopeNote, ...] })` — exactly the same prompt machinery as auto-memory, but at a different directory. The scope note differs per scope ("user-scope keep learnings general", "project-scope shared via VCS tailor to project", "local-scope not in VCS").

`@`-mention isolation: `getRelevantMemoryAttachments` (`attachments.ts:2196-2225`) scans the user input for `@agent-<type>` mentions; if found and that agent has `memory`, it searches **only that agent's** memory dir (not the auto-memory dir).

---

## 9. Hooks integration

Hook events (from `types/hooks.ts:73-160`):

| Hook | When |
|---|---|
| `PreToolUse` | before each tool call |
| `PostToolUse` / `PostToolUseFailure` | after tool call success / failure |
| `UserPromptSubmit` | when user submits |
| `SessionStart` / `SessionEnd` | session lifecycle (also `/clear`) |
| `Setup` | one-time setup |
| `SubagentStart` | spawning a subagent |
| `PermissionDenied` / `PermissionRequest` | tool gating |
| `Notification` | toast |
| `Elicitation` / `ElicitationResult` | inline question |
| `CwdChanged` / `FileChanged` / `WorktreeCreate` | filesystem events |
| `PreCompact` | before compaction (manual or auto) |
| (no `PostCompact` enum — but `executePostCompactHooks` exists in code; may be a separate codepath) |

**Memory-relevant hooks**:
- `SessionStart` runs on resume + on clear (`processSessionStartHooks`). Returned messages prepend the new session — a place to inject "You are continuing X" reminders.
- `SessionEnd` fires before clear/exit; bounded to ~1.5 s.
- `PreCompact` can rewrite custom-instructions and add a user-display message; runs concurrently with cache-param build for performance.
- `instructionsLoaded` (NOT in the public hook enum but referenced by `claudemd.ts:executeInstructionsLoadedHooks`) — fires when CLAUDE.md/rules are loaded.

---

## 10. Slash commands related to memory

| Command | Behavior |
|---|---|
| `/memory` (`commands/memory/memory.tsx`) | Opens a `MemoryFileSelector` dialog; user picks a CLAUDE.md / .claude/CLAUDE.md / CLAUDE.local.md / MEMORY.md file; opens it in `$VISUAL`/`$EDITOR`. Calls `clearMemoryFileCaches()` + `getMemoryFiles()` first to prime the selector. |
| `/clear` | Wipe conversation, regenerate session id, run SessionEnd → SessionStart hooks. See §7. |
| `/compact [instructions]` | Three-stage cascade: session-memory compact → reactive (if mode on) → traditional summarize. See §6. |
| `/rewind` | Open message selector to truncate transcript to a chosen point. |
| `/resume` | Open session selector to switch to a different `.jsonl`. |
| `/remember` (skill, ANT-only) | "Review auto-memory entries and propose promotions to CLAUDE.md, CLAUDE.local.md, or shared memory. Also detects outdated, conflicting, and duplicate entries across memory layers." (`_prompts/skill-remember.md`) — explicitly a *housekeeping* skill, not a write skill. |
| `/dream` (KAIROS, in `services/autoDream/`) | Nightly distillation of daily logs into topic files + MEMORY.md index. |

---

## 11. Privacy / redaction

- **Team memory secret scanning** (`services/teamMemorySync/secretScanner.ts`) — client-side gitleaks-derived regex bank scans content **before** upload to team sync. Curated subset of high-confidence rules with distinctive prefixes (AWS, GCP, DigitalOcean, Anthropic API key — assembled at runtime so the literal `sk-ant-api` string isn't in the bundle). Generic keyword-context rules omitted. Mode is "fail-closed": match → block upload + surface label like "anthropic-api-key".
- **Auto-memory itself has no scanning** — it's local-only. The `WHAT_NOT_TO_SAVE_SECTION` and the prompt's "MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials" are the only prevention.
- **Path security** (`paths.ts:validateMemoryPath`, `teamMemPaths.ts:validate*`):
  - Reject relative, root-near, drive-root, UNC, null bytes.
  - `.claude/settings.json` (project-level, committed) explicitly **excluded** from `autoMemoryDirectory` setting sources — a malicious repo could otherwise redirect to `~/.ssh`. Only `policySettings`/`flagSettings`/`localSettings`/`userSettings` honored.
  - Symlink-aware containment via `realpathDeepestExisting()` walk-up — catches dangling symlinks (attack vector — `writeFile` follows the link to write outside the dir).
- **Cowork mode**: `CLAUDE_COWORK_MEMORY_PATH_OVERRIDE` env redirects to a space-scoped mount; `hasAutoMemPathOverride()` flags this so the filesystem write carve-out is gated correctly.
- **No PII in telemetry**: analytics metadata uses `AnalyticsMetadata_I_VERIFIED_THIS_IS_NOT_CODE_OR_FILEPATHS` typed wrapper to gate non-PII-safe fields.

---

## 12. Failure modes / open issues / TODOs

- **Memory drift** is the single biggest acknowledged failure mode. Two prompt-side defenses + one read-time defense:
  - `MEMORY_DRIFT_CAVEAT` in `WHEN_TO_ACCESS`: "Memory records can become stale … verify before asserting."
  - `TRUSTING_RECALL_SECTION` (header "## Before recommending from memory") with grep/check-file rules — eval-validated 3/3 only when `appendSystemPrompt`-injected.
  - `memoryFreshnessText` per-memory at-render: ">N days old. Memories are point-in-time observations, not live state."
- **Known gap** (called out in code comment `memoryTypes.ts:237-238`): "H1 doesn't cover slash-command claims (0/3 on the /fork case — slash commands aren't files or functions in the model's ontology)."
- **Compaction-tolerance** of the extract cursor — `countModelVisibleMessagesSince` falls back to counting all messages when sinceUuid is missing, otherwise extraction would silently die forever after the first compact.
- **Prompt-cache stability** is load-bearing: `header` field on `relevant_memories` attachment is precomputed once because `memoryAge(mtimeMs)` calls `Date.now()` (would silently bust the cache one day later).
- **`tengu_bramble_lintel` throttle** default 1 — every turn extracts. Production may dial this up for cost.
- **Subagent extraction blackout**: subagents never write memory. A long-running subagent can produce extractable feedback that's lost when it returns. The main agent's prompt covers it next turn IF the subagent return surfaces enough text.
- **Memory dir creation at every session start**: `ensureMemoryDirExists` is called from `loadMemoryPrompt` (cached via `systemPromptSection('memory', ...)`) so it's once per session. Errors logged at `debug` level.
- **No user-side delete UI**: forgetting a memory is "ask the assistant to forget" → it Edit/Writes to remove the index entry + deletes the file. No GUI. `/memory` only opens for editing.
- **Index drift**: nothing reconciles `MEMORY.md` against actual files in the dir. If the agent forgets to update the index, the entry persists. Telemetry would catch via `tengu_extract_memories_extraction.files_written - memories_saved` mismatch.

---

## 13. Notably good / notably bad — takeaways for AGH

### Good (steal these)

1. **Closed type taxonomy** (`user`/`feedback`/`project`/`reference`) **with explicit non-goals** ("don't save things derivable from code/git/CLAUDE.md"). The `WHAT_NOT_TO_SAVE_SECTION` is stronger than the "what to save" section — it's the boundary that keeps the system from devolving into a noise log.
2. **Index + topic files separation**. `MEMORY.md` (no frontmatter, one-line bullets, hard-capped at 200 lines / 25 KB) is always loaded; topic files (frontmatter, full content) are recalled on-demand. Prevents context-window overflow without sacrificing breadth.
3. **Sonnet-side recall ranker** with frontmatter-only manifest. Bounds the ranker's prompt to ~200 lines × ~150 chars = ~30 KB regardless of memory size, returns top 5 via JSON-schema output. `recentTools` filtering prevents tool-doc-noise false positives.
4. **`alreadySurfaced` set, threaded through**. Belt-and-suspenders: filter inside the ranker (efficient slot use) AND inside the caller (multi-dir results). Reset by compaction (because attachments are gone from the transcript), not separately.
5. **Forked extraction agent with cache-share**. Same prompt prefix → near-zero cache miss; locked tool perms (read-only Bash, Edit/Write only inside memdir); strict turn budget (`maxTurns: 5`); pre-injected manifest so it doesn't waste a turn on `ls`.
6. **Mutual exclusion main vs forked**. `hasMemoryWritesSince` cleanly delegates to whichever agent did the work. Cursor advances regardless so the forked extractor can't double-process.
7. **Per-message freshness header**, precomputed at attachment-creation. Simple but load-bearing for cache hit rate.
8. **Eval-validated prompt section placement**. Comment trail in `memoryTypes.ts` explicitly cites pass-rates for hypothesis variants (H1, H5, H6) — proper science. Header wording matters: "Before recommending from memory" 3/3 vs "Trusting what you recall" 0/3.
9. **`appendSystemPrompt` for high-priority recall guidance**. Some rules only work at the tail of the system prompt — adopt this pattern for any AGH "be careful with X" surface.
10. **Path security depth**. NFKC-normalize / null-byte / URL-encode / dangling-symlink / symlink-loop checks. Settings-source whitelist excludes project-level for memory dir.

### Bad / questionable (avoid)

1. **`TYPES_SECTION_INDIVIDUAL` and `TYPES_SECTION_COMBINED` are duplicated**. Comment defends this ("flat makes per-mode edits trivial"), but it's a smell — adding a 5th type means editing two giant string blocks. AGH should have a single source of truth + a thin formatter.
2. **No `AGENTS.md` discovery** — only `CLAUDE.md`. AGH's existing taxonomy needs to bridge this.
3. **No first-class delete/forget UX**. Relies on the assistant remembering to remove both the index line AND the topic file.
4. **Index is the source of truth from the model's POV, but the directory listing is the actual source of truth**. Drift is possible. AGH could either (a) generate the index from the dir on every load, or (b) keep the index but add a periodic reconciler.
5. **Subagents have no extraction**. A 30-minute task-tool agent can produce feedback the main agent never sees because the return summary is too short.
6. **No semantic search** — purely description-string ranking via Sonnet. Works because descriptions are forced to be specific by the prompt, but stale descriptions drag the ranker badly. AGH's vector-or-FTS index would close this gap.
7. **No memory expiration / TTL**. `memoryAge` only annotates; nothing actually retires old memories. Project memories explicitly "decay fast" in the prompt but aren't garbage-collected.
8. **Throttle is global per-turn**, not per-class. A user can produce 10 turns of test fixtures and hit the extractor 10 times.
9. **`/memory` UX is an editor jump-off** — no in-UI list/search/edit/delete. Power users only.
10. **KAIROS daily-log mode + nightly `/dream`** is a parallel codepath rather than a generalization. AGH should pick ONE persistence model from day one (live index OR append-then-distill) rather than supporting both.

### Recommended for AGH `mem-v2`

- Adopt the **closed-taxonomy + WHAT_NOT_TO_SAVE** structure, but make types extensible at config-time (not literal-only) — AGH's plurality of agents will need types like `runtime-decision`, `provider-quirk`, etc.
- Adopt the **MEMORY.md index + topic files** layout but **regenerate the index from the directory on every load** (drift-free, index is just rendered cache).
- Adopt the **Sonnet-side recall ranker** but also add a deterministic substring-or-FTS pre-filter so the LLM call is bounded even at 10K memories (Claude Code caps at 200 — too low for AGH's lifetime).
- Adopt **forked extraction with cache-share + locked tools + manifest pre-injection + cursor-with-fallback**.
- Adopt **per-memory freshness header precomputed once + age-based caveat injection**.
- Add **TTL/decay** (Claude Code doesn't) — types should declare default lifetimes.
- Add a **memory-edit/inspect TUI/CLI** (Claude Code's `/memory` is weak).
- Make **`/forget`** a first-class command that touches both the index and the file (Claude Code expects the agent to do this).
- **Subagent memory write-back** — a subagent should be able to durably record a feedback memory at exit (without polluting peers).
