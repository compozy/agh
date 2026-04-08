# Prompt Architecture Analysis

## How It Works (in Claude Code)

Claude Code uses a layered, modular prompt assembly system that splits the system prompt into independently managed sections with aggressive caching optimization. The architecture can be summarized as a **7-layer composition pipeline** that builds a `string[]` (not a single string), where each array element represents a distinct section that can be cached, swapped, or omitted independently.

### The 7-Layer Prompt Structure

The `getSystemPrompt()` function in `constants/prompts.ts` returns an array of strings composed from these layers:

1. **Identity/Intro** -- Static identity framing ("You are Claude Code...") plus security guardrails (cyber risk instruction). Produced by `getSimpleIntroSection()`.

2. **System Behavior** -- Rules about tool execution, permission modes, system-reminder tags, hooks, and auto-compression. Produced by `getSimpleSystemSection()`.

3. **Task Instructions** -- How to do software engineering tasks, code style rules, what NOT to do (no gold-plating, no premature abstractions). Produced by `getSimpleDoingTasksSection()`. Conditionally omitted when a custom output style replaces coding instructions.

4. **Safety/Actions** -- The "executing actions with care" section about reversibility, blast radius, and risky operations. Produced by `getActionsSection()`.

5. **Tool Usage** -- Instructions on using dedicated tools over Bash, task management, agent delegation, parallel tool calls. Produced by `getUsingYourToolsSection()`. This section is parameterized by `enabledTools: Set<string>` so it only mentions tools that are actually available.

6. **Tone/Style + Output Efficiency** -- Formatting rules (no emojis unless asked, be concise, use file_path:line_number references). Two functions: `getSimpleToneAndStyleSection()` and `getOutputEfficiencySection()`.

7. **Dynamic Boundary + Registry-Managed Sections** -- Everything after `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` is session-specific and cannot use global cache scope. These are resolved via the `systemPromptSection()` registry pattern.

The actual return value looks like:

```typescript
return [
  // --- Static content (cacheable globally) ---
  getSimpleIntroSection(outputStyleConfig),
  getSimpleSystemSection(),
  getSimpleDoingTasksSection(),
  getActionsSection(),
  getUsingYourToolsSection(enabledTools),
  getSimpleToneAndStyleSection(),
  getOutputEfficiencySection(),
  // === BOUNDARY MARKER ===
  ...(shouldUseGlobalCacheScope() ? [SYSTEM_PROMPT_DYNAMIC_BOUNDARY] : []),
  // --- Dynamic content (registry-managed) ---
  ...resolvedDynamicSections,
].filter(s => s !== null);
```

### Cache Splitting with `__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__`

The `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` constant is a sentinel string inserted into the prompt array. The `splitSysPromptPrefix()` function in `utils/api.ts` uses it to split the prompt into blocks with different cache scopes:

- **Before boundary**: Static content that is identical across all users/orgs. Tagged with `cacheScope: 'global'` so Anthropic's API can cache it across organizations (massive cost savings).
- **After boundary**: Dynamic content (environment info, CLAUDE.md, MCP instructions, language preferences, scratchpad paths) tagged with `cacheScope: null` (no cross-org caching).

The split produces up to 4 blocks:

```typescript
type SystemPromptBlock = {
  text: string;
  cacheScope: "global" | "org" | null;
};
```

1. Attribution header (cacheScope=null)
2. System prompt prefix (cacheScope=null)
3. Static content before boundary (cacheScope='global')
4. Dynamic content after boundary (cacheScope=null)

When MCP tools are present, the system falls back to `org`-level caching (no global cache on system prompt) to avoid cache fragmentation from varying MCP server instructions.

### The Section Registry Pattern

Dynamic sections use a registry abstraction (`systemPromptSections.ts`) with two constructors:

```typescript
// Computed once per session, cached until /clear or /compact
systemPromptSection("memory", () => loadMemoryPrompt());

// Recomputed every turn -- explicitly labeled DANGEROUS because it breaks cache
DANGEROUS_uncachedSystemPromptSection(
  "mcp_instructions",
  () => getMcpInstructionsSection(mcpClients),
  "MCP servers connect/disconnect between turns" // required reason
);
```

The resolver checks a cache map first, only calling `compute()` on cache miss. The `DANGEROUS_` prefix is a deliberate code-review friction device -- it forces developers to document why a section must break the cache.

### System Context vs User Context

Two separate context channels exist:

1. **System Context** (`getSystemContext()` in `context.ts`): Appended to the system prompt array. Contains git status, cache breaker injection. Assembled via `appendSystemContext()`.

2. **User Context** (`getUserContext()` in `context.ts`): Prepended as a synthetic user message wrapped in `<system-reminder>` tags. Contains CLAUDE.md content and current date. Assembled via `prependUserContext()`.

The user context message includes an explicit disclaimer:

```
<system-reminder>
As you answer the user's questions, you can use the following context:
# claudeMd
[contents]
# currentDate
Today's date is 2026-04-01.

IMPORTANT: this context may or may not be relevant to your tasks.
You should not respond to this context unless it is highly relevant to your task.
</system-reminder>
```

### System Reminders as an Injection Channel

`<system-reminder>` tags are a first-class concept used throughout the codebase as a way to inject runtime information into user-role messages without confusing the model. Key properties:

- The system prompt explicitly teaches the model: "Tool results and user messages may include `<system-reminder>` tags. Tags contain information from the system."
- `wrapInSystemReminder()` utility wraps any string in the tags.
- Attachment messages (memory files, MCP instructions, skill discovery results, team context, task notifications) are wrapped in system-reminder tags before being injected.
- A post-processing pass (`smooshSystemReminderSiblings`) consolidates adjacent system-reminder text blocks into the nearest tool_result to avoid creating extra Human: boundaries.

### Per-Turn Attachment System

Beyond the static system prompt, Claude Code injects **attachments** per turn:

- **Memory files** (CLAUDE.md at various levels)
- **Skill discovery** results (surfaced skills matching the current task)
- **MCP instructions** (can be injected as persisted delta attachments)
- **Task notifications** from subagents
- **Team context** for swarm/teammate mode
- **IDE selections** (code selections from connected IDEs)
- **Diagnostic files** (file state tracking)

These attachments are assembled by `getAttachmentMessages()` which produces `AttachmentMessage[]` that get interleaved with the conversation.

### Multi-Persona Prompt Selection

The `buildEffectiveSystemPrompt()` function in `utils/systemPrompt.ts` implements a priority chain:

```
0. Override system prompt (replaces everything, e.g., /loop mode)
1. Coordinator mode (orchestrates workers)
2. Agent system prompt:
   - Proactive mode: APPENDED to default prompt
   - Otherwise: REPLACES default prompt
3. Custom system prompt (--system-prompt flag)
4. Default system prompt (standard Claude Code)
+ appendSystemPrompt is always added at the end
```

### Conditional Composition Based on Feature Flags

The codebase uses `feature()` (a Bun bundler compile-time function) and runtime GrowthBook feature flags extensively:

- `feature('PROACTIVE')` / `feature('KAIROS')` -- autonomous/proactive mode
- `feature('COORDINATOR_MODE')` -- multi-worker orchestration
- `feature('CACHED_MICROCOMPACT')` -- function result clearing
- `feature('TOKEN_BUDGET')` -- token budget tracking
- `feature('EXPERIMENTAL_SKILL_SEARCH')` -- skill discovery
- `feature('VERIFICATION_AGENT')` -- adversarial verification

Feature flags gate entire prompt sections. The `feature()` call enables dead-code elimination at build time -- external builds strip internal-only sections entirely.

Another conditional axis is `process.env.USER_TYPE === 'ant'` which adds Anthropic-internal prompt sections (more detailed comment writing rules, false-claims mitigation, numeric length anchors, verbose communicating-with-user section).

### Prompt Templates for Different Agents

The `_prompts/` directory contains markdown templates for:

- **System prompts**: main, coordinator, proactive, cyber-risk, teammate
- **Agent prompts**: general-purpose, explore, plan, verification, fork, default, claude-code-guide, statusline-setup
- **Tool prompts**: One per tool (bash, edit, read, grep, glob, agent, skill, etc.)
- **Service prompts**: compact, memory extraction, dream consolidation, session memory, team memory
- **Skill prompts**: simplify, claude-api, update-config, loop, batch, debug, verify, remember, schedule, skillify, keybindings

Each tool prompt is loaded via the tool's `prompt()` method which can be async and context-dependent (receiving the available tools, agents, and permission context).

## Key Patterns Worth Adopting

### 1. Array-Based Prompt Composition (Not String Concatenation)

The system prompt is `readonly string[]` (branded type `SystemPrompt`), not a single string. Each element is an independently manageable section. This enables:

- Per-section caching
- Per-section conditional inclusion
- Per-section token counting for context analysis
- Clean boundary for cache splitting

```typescript
// From systemPromptType.ts
export type SystemPrompt = readonly string[] & {
  readonly __brand: "SystemPrompt";
};

export function asSystemPrompt(value: readonly string[]): SystemPrompt {
  return value as SystemPrompt;
}
```

### 2. Explicit Cache Boundary Marker

A sentinel value separates globally-cacheable static content from per-session dynamic content:

```typescript
export const SYSTEM_PROMPT_DYNAMIC_BOUNDARY = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__";
```

The `splitSysPromptPrefix()` function finds this marker and assigns different cache scopes to blocks before/after it. This is critical for cost optimization -- the static prefix can be cached across all users.

### 3. Section Registry with Dangerous/Safe Split

```typescript
// Safe: computed once, cached for session
systemPromptSection("memory", () => loadMemoryPrompt());

// Dangerous: recomputed every turn, requires written justification
DANGEROUS_uncachedSystemPromptSection(
  "mcp_instructions",
  () => getMcpInstructionsSection(mcpClients),
  "MCP servers connect/disconnect between turns"
);
```

This pattern forces developers to be explicit about cache-busting behavior and document why it's necessary.

### 4. Tool-Aware Prompt Generation

The `getUsingYourToolsSection(enabledTools: Set<string>)` function only mentions tools that are actually enabled. If the agent doesn't have Glob/Grep (because embedded search is used), those instructions are omitted. If no task tool exists, task management guidance is skipped.

```typescript
function getUsingYourToolsSection(enabledTools: Set<string>): string {
  const taskToolName = [TASK_CREATE_TOOL_NAME, TODO_WRITE_TOOL_NAME].find(n => enabledTools.has(n));
  // ... only include guidance for available tools
}
```

### 5. System Reminders as a Typed Injection Channel

Runtime context is injected via `<system-reminder>` XML tags in user-role messages. The model is taught about these tags in the system prompt. This allows injecting context without polluting the system prompt (which would break the cache).

```typescript
export function wrapInSystemReminder(content: string): string {
  return `<system-reminder>\n${content}\n</system-reminder>`;
}
```

### 6. Dual Context Channels (System vs User)

- **System context** (git status, cache breaker) -- appended to system prompt array
- **User context** (CLAUDE.md, date) -- prepended as a synthetic user message with system-reminder tags

This separation allows the CLAUDE.md content to be updated without invalidating the system prompt cache.

### 7. Feature-Flag-Gated Sections with Dead-Code Elimination

```typescript
// Build-time gate: entire section is stripped from external builds
...(feature('TOKEN_BUDGET')
  ? [systemPromptSection('token_budget', () => '...')]
  : []),

// Runtime gate: section is conditionally included
...(process.env.USER_TYPE === 'ant'
  ? [systemPromptSection('numeric_length_anchors', () => '...')]
  : []),
```

### 8. Prompt Priority Chain for Multi-Mode Support

The `buildEffectiveSystemPrompt()` function implements a clean priority chain:

```typescript
if (overrideSystemPrompt) return [overrideSystemPrompt]
if (coordinatorMode) return [coordinatorPrompt, ...append]
if (agentPrompt && proactiveMode) return [...default, agentPrompt, ...append]
return [...(agentPrompt ?? customPrompt ?? default), ...append]
```

### 9. Memoized Context Factories

```typescript
export const getSystemContext = memoize(async () => {
  const gitStatus = await getGitStatus();
  return { ...(gitStatus && { gitStatus }) };
});
```

Context computation is async and memoized -- computed once per session, cleared on `/clear` or `/compact`.

### 10. The Compaction/Summary System

When context grows too large, the system compacts messages into a summary. The compact prompt itself is a carefully designed template with:

- A `NO_TOOLS_PREAMBLE` that prevents the compaction model from calling tools
- Structured analysis in `<analysis>` tags (stripped before use)
- Detailed instructions about what to preserve (file names, code snippets, user feedback)

## Ideas for Our System

### 1. Adopt Array-Based Prompt Composition in Go

Replace our current `strings.Join(sections, "\n\n")` with a typed prompt array:

```go
// PromptSection represents a named, independently-cacheable section
type PromptSection struct {
    Name     string
    Content  string
    Dynamic  bool   // if true, recomputed every turn
    Reason   string // required when Dynamic is true
}

// SystemPrompt is an ordered collection of sections
type SystemPrompt struct {
    sections []PromptSection
}

func (sp *SystemPrompt) Add(s PromptSection) {
    sp.sections = append(sp.sections, s)
}

func (sp *SystemPrompt) Render() string {
    // Join non-empty sections
}

func (sp *SystemPrompt) StaticPrefix() string {
    // Everything before first dynamic section -- for caching
}

func (sp *SystemPrompt) DynamicSuffix() string {
    // Everything from first dynamic section onward
}
```

### 2. Implement a Section Registry

Create a registry that memoizes section computation:

```go
type SectionRegistry struct {
    mu    sync.RWMutex
    cache map[string]string
}

func (r *SectionRegistry) Section(name string, compute func() string) PromptSection {
    r.mu.RLock()
    if v, ok := r.cache[name]; ok {
        r.mu.RUnlock()
        return PromptSection{Name: name, Content: v}
    }
    r.mu.RUnlock()

    value := compute()
    r.mu.Lock()
    r.cache[name] = value
    r.mu.Unlock()
    return PromptSection{Name: name, Content: value}
}

func (r *SectionRegistry) DangerousUncachedSection(name, reason string, compute func() string) PromptSection {
    return PromptSection{Name: name, Content: compute(), Dynamic: true, Reason: reason}
}

func (r *SectionRegistry) Clear() {
    r.mu.Lock()
    r.cache = make(map[string]string)
    r.mu.Unlock()
}
```

### 3. Add System Reminder Injection Support

Implement the `<system-reminder>` pattern for runtime context injection:

```go
func WrapInSystemReminder(content string) string {
    return fmt.Sprintf("<system-reminder>\n%s\n</system-reminder>", content)
}

// Inject CLAUDE.md and date as a synthetic user message
func BuildUserContextMessage(claudeMD, date string) Message {
    parts := []string{}
    if claudeMD != "" {
        parts = append(parts, "# claudeMd\n"+claudeMD)
    }
    parts = append(parts, "# currentDate\n"+date)

    content := WrapInSystemReminder(
        "As you answer the user's questions, you can use the following context:\n" +
            strings.Join(parts, "\n") +
            "\n\nIMPORTANT: this context may or may not be relevant to your tasks.",
    )
    return UserMessage{Content: content, IsMeta: true}
}
```

### 4. Separate Static vs Dynamic Prompt Boundary

Add an explicit marker in our prompt assembly:

```go
const DynamicBoundary = "__DYNAMIC_BOUNDARY__"

func (a *Assembler) Build() *SystemPrompt {
    sp := &SystemPrompt{}

    // Static layers (safe to cache globally)
    sp.Add(a.identitySection())
    sp.Add(a.systemBehaviorSection())
    sp.Add(a.taskInstructionsSection())
    sp.Add(a.safetySection())
    sp.Add(a.toolUsageSection())
    sp.Add(a.toneSection())

    sp.AddBoundary()  // marks the cache split point

    // Dynamic layers (session-specific)
    sp.Add(a.environmentSection())
    sp.Add(a.memorySection())
    sp.Add(a.skillsSection())
    sp.Add(a.roleSpecializationSection())

    return sp
}
```

### 5. Tool-Aware Prompt Generation

Make the tool instructions section aware of which tools are actually enabled:

```go
func (a *Assembler) toolUsageSection(enabledTools map[string]bool) PromptSection {
    var items []string
    if enabledTools["bash"] {
        items = append(items, "Use Bash for shell commands...")
    }
    if enabledTools["read"] {
        items = append(items, "Use Read instead of cat...")
    }
    // Only mention tools that exist
    return PromptSection{Name: "tool_usage", Content: renderBullets(items)}
}
```

### 6. Multi-Mode Prompt Selection

Implement a priority-based prompt selection similar to `buildEffectiveSystemPrompt()`:

```go
func (k *Kernel) buildPrompt(session *Session) (*SystemPrompt, error) {
    // Priority chain:
    // 1. Override prompt (e.g., loop mode)
    if session.OverridePrompt != "" {
        return NewSystemPrompt(session.OverridePrompt), nil
    }

    // 2. Supervisor/coordinator mode
    if session.Mode == ModeSupervisor {
        return k.supervisorPrompt(session), nil
    }

    // 3. Agent-specific prompt
    if session.AgentDef != nil {
        base := k.defaultPrompt(session)
        if session.Mode == ModeProactive {
            return base.Append(session.AgentDef.SystemPrompt), nil
        }
        return NewSystemPrompt(session.AgentDef.SystemPrompt), nil
    }

    // 4. Custom prompt (flag/config)
    if session.CustomPrompt != "" {
        return NewSystemPrompt(session.CustomPrompt), nil
    }

    // 5. Default prompt
    return k.defaultPrompt(session), nil
}
```

### 7. Per-Turn Attachment Pipeline

Instead of putting everything in the system prompt, add a per-turn attachment pipeline that injects context as user messages:

```go
type Attachment struct {
    Type    string // "memory", "skill_discovery", "task_notification", etc.
    Content string
    Meta    bool   // true = synthetic message, not real user input
}

func (k *Kernel) getAttachments(session *Session, turn int) []Attachment {
    var attachments []Attachment

    // Memory files (CLAUDE.md equivalent)
    if mem := k.loadMemory(session); mem != "" {
        attachments = append(attachments, Attachment{
            Type: "memory", Content: WrapInSystemReminder(mem), Meta: true,
        })
    }

    // Skill discovery
    if skills := k.discoverRelevantSkills(session, turn); skills != "" {
        attachments = append(attachments, Attachment{
            Type: "skill_discovery", Content: WrapInSystemReminder(skills), Meta: true,
        })
    }

    return attachments
}
```

### 8. Context Window Analysis

Port the context analysis pattern for debugging/observability:

```go
type ContextBreakdown struct {
    SystemPromptTokens int
    ToolTokens         int
    MemoryTokens       int
    MessageTokens      int
    FreeTokens         int
    MaxTokens          int
}

func (k *Kernel) AnalyzeContext(session *Session) (*ContextBreakdown, error) {
    // Count tokens per category for /context command
}
```

## Key Files Reference

| File                                    | Description                                                                                                                                                                                                                                                                                                                     |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `constants/prompts.ts`                  | **Core prompt assembly engine.** Contains `getSystemPrompt()` which builds the 7-layer prompt array. Contains all section-generating functions, the `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` constant, and feature-flag-gated conditional sections. (~915 lines)                                                                        |
| `constants/systemPromptSections.ts`     | **Section registry pattern.** Defines `systemPromptSection()` (cached) and `DANGEROUS_uncachedSystemPromptSection()` (volatile, requires reason). Implements `resolveSystemPromptSections()` with cache lookup. (~70 lines)                                                                                                     |
| `utils/systemPromptType.ts`             | **Branded type for system prompt.** Defines `SystemPrompt = readonly string[] & { __brand }` and `asSystemPrompt()` constructor. Intentionally dependency-free. (~15 lines)                                                                                                                                                     |
| `utils/systemPrompt.ts`                 | **Multi-mode prompt selection.** `buildEffectiveSystemPrompt()` implements the priority chain: override > coordinator > agent > custom > default. (~125 lines)                                                                                                                                                                  |
| `context.ts`                            | **Context factories.** Memoized `getSystemContext()` (git status, cache breaker) and `getUserContext()` (CLAUDE.md, date). Both return `{[k: string]: string}` dictionaries. (~190 lines)                                                                                                                                       |
| `utils/api.ts`                          | **Cache splitting and context injection.** `splitSysPromptPrefix()` splits prompt by boundary marker into `SystemPromptBlock[]` with cache scopes. `appendSystemContext()` adds system context to prompt array. `prependUserContext()` injects user context as `<system-reminder>`-wrapped synthetic user message. (~720 lines) |
| `utils/messages.ts`                     | **System reminder utilities.** `wrapInSystemReminder()`, `wrapMessagesInSystemReminder()`, `ensureSystemReminderWrap()`, `smooshSystemReminderSiblings()`. Handles merging, deduplication, and cleanup of system-reminder-tagged content. (~3500 lines)                                                                         |
| `utils/attachments.ts`                  | **Per-turn attachment pipeline.** `getAttachmentMessages()` assembles memory files, skill discovery, MCP instructions, task notifications, IDE selections, and diagnostic files as `AttachmentMessage[]`. (~3500 lines)                                                                                                         |
| `utils/analyzeContext.ts`               | **Context window analysis.** `analyzeContextUsage()` counts tokens per category (system prompt, tools, memory, MCP, agents, skills, messages) for the `/context` command. (~1400 lines)                                                                                                                                         |
| `query.ts`                              | **Main query loop.** Orchestrates per-turn assembly: builds full system prompt, auto-compacts if needed, prepends user context, streams API call, runs tools, collects attachments for next turn. (~1700 lines)                                                                                                                 |
| `setup.ts`                              | **Session initialization.** Sets up CWD, hooks, worktrees, plugins, session memory, attribution. Not directly involved in prompt assembly but initializes the environment that context factories read. (~480 lines)                                                                                                             |
| `_prompts/README.md`                    | **Prompt template inventory.** Lists all system prompts, agent prompts, tool prompts, service prompts, and skill prompts with their source locations.                                                                                                                                                                           |
| `_prompts/system-prompt-main.md`        | **Main prompt documentation.** The full composite system prompt in the order it appears, with all 7 layers documented with their source code.                                                                                                                                                                                   |
| `_prompts/system-prompt-coordinator.md` | **Coordinator mode prompt.** Multi-worker orchestration: phases (research/synthesis/implementation/verification), worker prompt guidelines, concurrency management.                                                                                                                                                             |
| `_prompts/system-prompt-proactive.md`   | **Autonomous mode prompt.** Tick-driven autonomous operation: pacing with Sleep, first wake-up behavior, terminal focus awareness, bias toward action.                                                                                                                                                                          |
| `services/compact/prompt.ts`            | **Compaction prompt.** Templates for summarizing conversation when context grows too large. Includes NO_TOOLS_PREAMBLE, analysis/summary structure, and partial-compact variants.                                                                                                                                               |
| `services/SessionMemory/prompts.ts`     | **Session memory template.** Structured markdown template for session notes (title, current state, task spec, files, workflow, errors, learnings, worklog).                                                                                                                                                                     |
