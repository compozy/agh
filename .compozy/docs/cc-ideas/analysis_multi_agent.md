# Multi-Agent Orchestration & Skills Analysis

## How It Works (in Claude Code)

### 1. COORDINATOR_MODE: The Orchestration Layer

Claude Code has a dedicated **coordinator mode** (`coordinator/coordinatorMode.ts`) that transforms the main agent from a tool-using worker into a pure orchestrator. When enabled (via `CLAUDE_CODE_COORDINATOR_MODE` env var), the agent loses direct tool access and instead operates exclusively through worker delegation.

**Core mechanics:**

- The coordinator gets a specialized system prompt (`getCoordinatorSystemPrompt()`) that defines its role as a task orchestrator rather than a tool user.
- Its only tools are: `Agent` (spawn workers), `SendMessage` (continue workers), `TaskStop` (kill workers), and optionally PR subscription tools.
- Workers are spawned via the `Agent` tool with `subagent_type: "worker"` and run asynchronously. Results arrive back as `<task-notification>` XML blocks injected as user-role messages.
- The coordinator is explicitly instructed to **synthesize findings** from workers before delegating follow-up work -- it must never use "based on your findings" lazy delegation.

**Task workflow phases:**

1. **Research** -- workers investigate in parallel
2. **Synthesis** -- coordinator reads findings, crafts specific implementation specs with file paths and line numbers
3. **Implementation** -- workers make targeted changes per spec
4. **Verification** -- separate workers test changes with fresh eyes

**Key design constraint:** Coordinator mode and fork mode are mutually exclusive -- the coordinator already owns orchestration, so forking (which lets agents autonomously decide to parallelize) would conflict.

**Scratchpad directory:** When the `tengu_scratch` feature gate is enabled, the coordinator exposes a shared scratchpad directory to workers. Workers can read/write there without permission prompts. The coordinator prompt instructs: "Use this for durable cross-worker knowledge -- structure files however fits the work."

### 2. Dual-Worker Architecture: In-Process vs Process-Based

Claude Code supports **three distinct execution backends** for multi-agent work:

#### A. Process-Based Teammates (tmux / iTerm2)

These run as completely separate Claude Code processes, each in their own terminal pane:

- **TmuxBackend**: Creates tmux panes within a dedicated swarm session/window. Uses external tmux socket to isolate from user's sessions.
- **ITermBackend**: Uses iTerm2's native split panes via the `it2` CLI.

Each teammate gets:

- Its own process with `CLAUDE_CODE_AGENT_ID` env var
- Its own message inbox (file-based)
- Communication via `SendMessage` tool writing to inbox files
- Visual pane with colored borders and title showing agent name

#### B. In-Process Teammates

Defined in `.compozy/tasks/InProcessTeammateTask/` and spawned via `utils/swarm/spawnInProcess.ts`:

- Run in the **same Node.js process** using `AsyncLocalStorage` for context isolation
- Have a `TeammateIdentity` (agentId formatted as `name@teamName`)
- Support plan mode approval flow (teammates must get approval before implementing)
- Can be idle (waiting for work) or active (processing)
- Messages delivered via `pendingUserMessages` queue in AppState
- Capped at 50 messages in UI state (`TEAMMATE_MESSAGES_UI_CAP`) to prevent memory bloat

The `InProcessBackend` (`utils/swarm/backends/InProcessBackend.ts`) implements the `TeammateExecutor` interface:

```typescript
type TeammateExecutor = {
  type: BackendType;
  isAvailable(): Promise<boolean>;
  spawn(config: TeammateSpawnConfig): Promise<TeammateSpawnResult>;
  sendMessage(agentId: string, message: TeammateMessage): Promise<void>;
  terminate(agentId: string, reason?: string): Promise<boolean>;
  kill(agentId: string): Promise<boolean>;
  isActive(agentId: string): Promise<boolean>;
};
```

#### C. Background Agents (LocalAgentTask)

Standard subagents spawned via the `Agent` tool that run as async background tasks. Results are delivered via `<task-notification>` XML. These are the workhorse for coordinator mode.

### 3. Scratchpad / Shared Memory for Inter-Agent Communication

Claude Code uses **multiple mechanisms** for inter-agent knowledge sharing:

#### A. Scratchpad Directory

- Gated behind `tengu_scratch` feature flag
- Path injected via dependency injection from `QueryEngine.ts`
- Workers get no-permission-prompt access to this directory
- Coordinator prompt says: "Use this for durable cross-worker knowledge"
- Simple filesystem-based -- no structured protocol, just files

#### B. Team Memory (`memdir/teamMemPaths.ts`, `memdir/teamMemPrompts.ts`)

- A persistent, file-based memory system with **two directories**: private (per-user) and team (shared)
- Team memories stored at `<memoryBase>/projects/<project-root>/memory/team/`
- Each memory file has YAML frontmatter (name, description, type) and markdown body
- An entrypoint `MEMORY.md` index file lists all memories (loaded into context)
- Four memory types: user, feedback, project, reference -- each with scope guidance
- Extensive security: path traversal protection, symlink resolution, null byte checks

#### C. Team Config File

- At `~/.claude/teams/{team-name}/config.json`
- Contains `members` array with name, agentId, agentType, model, cwd
- Teammates read this to discover peers
- Shared task list directory at `~/.claude/tasks/{team-name}/`

#### D. SendMessage Tool

- Direct message passing between agents by name
- Supports broadcast (`to: "*"`) to all teammates
- Cross-session messaging via UDS sockets or bridge sessions
- Protocol messages for shutdown requests/responses and plan approval

### 4. Skills System: Registration, Discovery, Invocation, Prompt Expansion

The skills system is a sophisticated mechanism for injecting domain-specific knowledge and capabilities into the agent:

#### Registration Sources

Skills are loaded from multiple sources with a priority hierarchy:

1. **Bundled skills** (`skills/bundled/`) -- compiled into the CLI binary, registered via `registerBundledSkill()` at startup
2. **Managed skills** -- from policy settings at `<managed-path>/.claude/skills/`
3. **User skills** -- from `~/.claude/skills/`
4. **Project skills** -- from `.claude/skills/` in project directories, walking up to home
5. **Plugin skills** -- from installed plugins
6. **MCP skills** -- from connected MCP servers, registered via `mcpSkillBuilders.ts`
7. **Remote skills** -- experimental, loaded from AKI/GCS with local caching

#### Skill Format

Each skill is a directory containing `SKILL.md` with YAML frontmatter:

```yaml
---
name: my-skill
description: What this skill does
when_to_use: Trigger conditions for the model
allowed-tools: [Bash, Read, Write]
model: opus
context: fork # or inline
agent: code-reviewer # optional agent type
effort: high
paths: ["src/**/*.ts"] # conditional activation
hooks:
  PreToolUse:
    - matcher: Bash
      hooks:
        - command: "echo 'before bash'"
---
Skill content with ${CLAUDE_SKILL_DIR} and ${CLAUDE_SESSION_ID} variables...
```

#### Invocation Modes

The SkillTool (`tools/SkillTool/SkillTool.ts`) supports two execution modes:

1. **Inline** (default): Skill content is processed and injected as user messages into the current conversation. The skill can modify context (allowed tools, model override, effort level) via `contextModifier`.

2. **Forked** (`context: fork`): Skill runs in an isolated sub-agent via `runAgent()`. The sub-agent gets its own token budget. Results are extracted and returned as tool output. This prevents large skill executions from consuming the parent's context window.

#### Dynamic Discovery

- Skills with `paths` frontmatter are **conditional** -- stored in a pending map and only activated when file operations touch matching paths (gitignore-style matching)
- `discoverSkillDirsForPaths()` walks up from file paths to find nested `.claude/skills/` directories
- A signal system (`onDynamicSkillsLoaded`) notifies consumers when new skills are discovered

#### Budget Management

The skill listing in the prompt gets 1% of the context window budget. Descriptions are truncated to fit, with bundled skills getting priority for full descriptions.

### 5. Fork-as-a-Primitive: Autonomous Forking Decisions

The fork system (`tools/AgentTool/forkSubagent.ts`) enables agents to autonomously decide when to parallelize work:

#### How It Works

- When `subagent_type` is **omitted** from the Agent tool call, a fork is triggered
- The fork child **inherits the parent's full conversation context** and system prompt
- Fork children get the parent's exact tool pool for cache-identical API prefixes
- `permissionMode: 'bubble'` surfaces permission prompts to the parent terminal
- `model: 'inherit'` keeps the parent's model for context length parity

#### Cache Optimization

The fork system is heavily optimized for **prompt cache sharing**:

```typescript
// All fork children produce byte-identical API request prefixes:
// [...history, assistant(all_tool_uses), user(placeholder_results..., directive)]
// Only the final text block differs per child, maximizing cache hits.
```

- All tool_result blocks use identical placeholder text: "Fork started -- processing in background"
- The assistant message is cloned verbatim (all tool_use blocks, thinking, text)
- Only the per-child directive differs

#### Anti-Recursion Guard

Fork children are prevented from forking again by detecting the `<fork-boilerplate>` tag in conversation history. The Agent tool stays in their tool pool (for cache-identical definitions) but fork attempts are rejected at call time.

#### Fork Child Protocol

Each fork child receives strict instructions:

- Must begin response with "Scope:" -- no preamble
- Keep report under 500 words
- Structured output format: Scope, Result, Key files, Files changed, Issues
- Must commit changes before reporting
- Must not spawn sub-agents

#### Worktree Isolation

Fork children can optionally run in a **git worktree** (`isolation: "worktree"`):

- Gets an isolated copy of the repository at a temporary worktree path
- Changes stay in the worktree and don't affect parent's files
- Worktree is auto-cleaned if no changes are made
- A `buildWorktreeNotice()` tells the child to translate inherited paths

#### Decision Heuristic

The agent is prompted to fork based on a qualitative criterion: "will I need this output again?"

- **Research**: fork open-ended questions; if breakable into independent questions, launch parallel forks
- **Implementation**: prefer to fork work requiring more than a couple of edits

### 6. Task System for Progress Tracking

The task system (`Task.ts`, `tasks.ts`, and `tools/Task*Tool/`) provides structured progress tracking:

#### Task Types

```typescript
type TaskType =
  | "local_bash" // Shell command execution
  | "local_agent" // Background agent (or backgrounded main session)
  | "remote_agent" // Remote agent execution
  | "in_process_teammate" // In-process teammate
  | "local_workflow" // Workflow scripts
  | "monitor_mcp" // MCP monitoring
  | "dream"; // Dream/background processing
```

#### Task State Machine

```
pending -> running -> completed
                   -> failed
                   -> killed
```

Terminal states are checked via `isTerminalTaskStatus()` to guard against injecting messages into dead tasks.

#### Task ID Scheme

Each task type has a unique prefix for instant identification:

- `b` = local_bash, `a` = local_agent, `r` = remote_agent
- `t` = in_process_teammate, `w` = local_workflow, `m` = monitor_mcp, `d` = dream
- IDs use 8 random characters from a 36-char alphabet (2.8 trillion combinations)

#### Task List Tools

For team coordination, structured task management is exposed via:

- **TaskCreate**: Creates tasks with subject, description, activeForm, metadata
- **TaskUpdate**: Updates status, subject, description, owner, dependencies (blocks/blockedBy)
- **TaskList**: Lists all tasks with filtering for available work
- **TaskGet**: Gets full task details
- **TaskStop**: Stops a running task
- **TaskOutput**: Reads task output (supports blocking reads)

#### Team Task Coordination

When agent swarms are enabled, the task system integrates with team management:

- Teams have a 1:1 correspondence with task lists (Team = TaskList)
- Tasks are stored at `~/.claude/tasks/{team-name}/`
- Teammates claim tasks by setting `owner` via TaskUpdate
- Dependencies via `blocks`/`blockedBy` fields prevent premature starts
- Teammates are instructed to prefer tasks in ID order (lowest first)

## Key Patterns Worth Adopting

### 1. Coordinator as Pure Orchestrator (No Direct Tools)

The coordinator mode pattern cleanly separates orchestration from execution. The coordinator cannot edit files or run commands -- it can only spawn workers, send messages, and stop tasks.

```typescript
// From coordinatorMode.ts - the coordinator's system prompt
"You are a **coordinator**. Your job is to:
- Help the user achieve their goal
- Direct workers to research, implement and verify code changes
- Synthesize results and communicate with the user
- Answer questions directly when possible"
```

**Key insight**: The coordinator must "synthesize, not delegate understanding." The prompt explicitly forbids lazy patterns like "based on your findings, fix the bug" and requires file paths, line numbers, and specific changes.

### 2. Task Notification via XML in User Messages

Worker results are delivered as user-role messages containing structured XML:

```xml
<task-notification>
  <task-id>{agentId}</task-id>
  <status>completed|failed|killed</status>
  <summary>{human-readable status summary}</summary>
  <result>{agent's final text response}</result>
  <usage>
    <total_tokens>N</total_tokens>
    <tool_uses>N</tool_uses>
    <duration_ms>N</duration_ms>
  </usage>
</task-notification>
```

This is elegant because it uses the existing message stream rather than requiring a separate communication channel.

### 3. Fork with Cache-Optimized Message Construction

The `buildForkedMessages()` function is a masterclass in optimization:

```typescript
// All fork children produce byte-identical API request prefixes
// 1. Keep the full parent assistant message
// 2. Build tool_results with identical placeholder text
// 3. Append per-child directive as the only varying content
// Result: massive prompt cache sharing across parallel forks
```

### 4. Dual Permission Models for Agents

Agents use different permission modes:

- **bubble**: Permission prompts surface to the parent terminal (for fork children)
- **plan**: Must enter plan mode and get approval before implementing
- **auto**: Automatic tool approval based on classifier
- **default**: Standard permission checking

### 5. Skill Conditional Activation via Path Patterns

Skills with `paths` frontmatter are stored dormant and only activated when file operations touch matching paths:

```typescript
// From loadSkillsDir.ts
export function activateConditionalSkillsForPaths(filePaths: string[], cwd: string): string[] {
  // Uses gitignore-style pattern matching
  const skillIgnore = ignore().add(skill.paths);
  if (skillIgnore.ignores(relativePath)) {
    dynamicSkills.set(name, skill);
    // Skill is now active and visible to the model
  }
}
```

### 6. Team Config as Discovery Mechanism

The simple `config.json` file at `~/.claude/teams/{team-name}/` serves as a lightweight service registry:

```json
{
  "name": "my-team",
  "leadAgentId": "team-lead@my-team",
  "members": [
    {
      "name": "team-lead",
      "agentId": "team-lead@my-team",
      "agentType": "team-lead"
    },
    {
      "name": "researcher",
      "agentId": "researcher@my-team",
      "agentType": "worker"
    }
  ]
}
```

Teammates discover each other by reading this file. The naming convention `name@teamName` provides both human-readable and machine-parseable identification.

### 7. Typed Task State with Discriminated Union

All task states derive from `TaskStateBase` and form a discriminated union:

```typescript
type TaskState =
  | LocalShellTaskState
  | LocalAgentTaskState
  | RemoteAgentTaskState
  | InProcessTeammateTaskState
  | LocalWorkflowTaskState
  | MonitorMcpTaskState
  | DreamTaskState;
```

This allows type-safe handling of different task types while maintaining a common interface.

### 8. Async Agent Lifecycle Pattern

The `runAsyncAgentLifecycle()` function in `agentToolUtils.ts` encapsulates the full lifecycle:

1. Create progress tracker and activity resolver
2. Stream messages, updating progress and AppState on each
3. On completion: finalize results, run handoff classifier, get worktree result
4. Enqueue notification (completed/failed/killed)
5. Always clean up (clear invoked skills, clear dump state)

Critical insight: **Mark task completed BEFORE running embellishment steps** (classifier, worktree cleanup). These can hang and should not gate the status transition.

### 9. In-Process Isolation via AsyncLocalStorage

In-process teammates achieve isolation without separate processes:

```typescript
// Each teammate gets its own context via AsyncLocalStorage
const teammateContext = createTeammateContext({
  agentId,
  agentName,
  teamName,
  color,
  planModeRequired,
  parentSessionId,
  abortController,
});
```

This allows concurrent agents with independent abort controllers, permission modes, and identity -- all in a single process.

### 10. Memory Capping for Long-Running Agents

The `TEAMMATE_MESSAGES_UI_CAP = 50` pattern prevents memory bloat in long-running agents:

```typescript
export function appendCappedMessage<T>(prev: readonly T[] | undefined, item: T): T[] {
  if (prev && prev.length >= TEAMMATE_MESSAGES_UI_CAP) {
    const next = prev.slice(-(TEAMMATE_MESSAGES_UI_CAP - 1));
    next.push(item);
    return next;
  }
  return [...prev, item];
}
```

## Ideas for Our System

### 1. Implement Coordinator Mode as a Kernel State

Our Go kernel already has session management. We could add a `CoordinatorMode` flag to the kernel that:

- Replaces the normal tool set with orchestration-only tools (SpawnWorker, SendMessage, StopWorker)
- Injects the coordinator system prompt
- Routes worker notifications back as user messages with XML wrapping
- Maintains a worker registry with task IDs and status

```go
type CoordinatorState struct {
    Enabled     bool
    Workers     map[string]*WorkerState
    ScratchDir  string
}

type WorkerState struct {
    TaskID      string
    Status      TaskStatus
    Description string
    StartTime   time.Time
    AgentID     string
}
```

### 2. Build a Skill Registry with YAML Frontmatter

Adopt the SKILL.md format with frontmatter for our skill system:

- Skills stored as directories under `.claude/skills/` with `SKILL.md` files
- Frontmatter fields: name, description, when_to_use, allowed-tools, context (inline/fork), paths
- Support `${CLAUDE_SKILL_DIR}` and `${CLAUDE_SESSION_ID}` variable substitution
- Implement conditional activation via path pattern matching
- Skills loaded from: bundled (compiled), user (~/.config), project (.claude/skills/)

```go
type SkillDefinition struct {
    Name           string
    Description    string
    WhenToUse      string
    AllowedTools   []string
    Context        string // "inline" or "fork"
    Paths          []string // gitignore-style patterns for conditional activation
    Content        string
    BaseDir        string
}
```

### 3. Implement Fork-as-a-Primitive with Context Inheritance

This is perhaps the most powerful pattern. In our Go system:

- When the Agent tool is called without specifying a subagent type, fork the current context
- The fork child inherits the full conversation history
- Optimize for cache sharing by making fork message prefixes byte-identical
- Guard against recursive forking by detecting fork marker in history

```go
type ForkConfig struct {
    Directive    string
    ParentCwd    string
    WorktreePath string // optional isolated worktree
    // Parent's rendered system prompt for byte-exact cache sharing
    SystemPrompt []byte
}
```

### 4. Dual Backend for Agents: Goroutine vs Process

Mirror the in-process vs process-based dichotomy:

- **Goroutine agents**: Use `context.Context` for cancellation and isolation (like AsyncLocalStorage). Cheap, share memory, good for lightweight workers.
- **Process agents**: Spawn separate CLI instances for heavyweight isolation. Communicate via filesystem or Unix sockets.

```go
type AgentBackend interface {
    Spawn(config AgentSpawnConfig) (*AgentHandle, error)
    SendMessage(agentID string, msg Message) error
    Kill(agentID string) error
    IsActive(agentID string) bool
}
```

### 5. Team Memory via Shared Filesystem

Implement the two-tier memory system:

- **Private memory**: Per-user, persists across sessions at `~/.config/agh/memory/`
- **Team memory**: Shared, project-scoped at `.agh/memory/team/`
- Each memory file has frontmatter (name, description, type)
- An entrypoint `MEMORY.md` index is loaded into context
- Apply path traversal protection (sanitize keys, resolve symlinks, check containment)

### 6. Structured Task Lists for Worker Coordination

Build a file-based task system:

- Tasks stored as individual files in `~/.config/agh/tasks/{team-name}/`
- Each task has: id, subject, description, status, owner, blocks, blockedBy
- Workers claim tasks by setting owner
- Dependencies via blocks/blockedBy prevent premature starts
- Expose via TaskCreate, TaskUpdate, TaskList, TaskGet tools

### 7. Notification Queue with XML Protocol

Use the `<task-notification>` XML pattern for worker results:

- Workers write results to a notification queue
- Main loop picks up notifications between turns
- XML structure provides: task-id, status, summary, result, usage metrics
- This avoids complex IPC by embedding structured data in the message stream

### 8. Permission Bubbling for Sub-Agents

When a sub-agent hits a permission check, bubble it up to the parent:

- Fork children use `permissionMode: "bubble"` to surface prompts to parent terminal
- Plan mode requires explicit approval before implementation
- Implement a permission bridge that serializes permission requests between agents

### 9. Skill Execution Budget

Adopt the 1% context window budget for skill listings:

- Truncate skill descriptions to fit
- Bundled/priority skills get full descriptions
- Per-entry hard cap of ~250 characters
- Use `formatCommandsWithinBudget()` pattern for dynamic truncation

### 10. Agent Summarization Service

Implement background summarization for long-running agents:

- Start summarization when cache-safe parameters are available
- Stop summarization when agent completes
- Use summaries for progress reporting and task notifications
- Keep summaries brief (token-efficient) for notification payloads

## Key Files Reference

### Coordinator / Orchestration

- `coordinator/coordinatorMode.ts` -- Coordinator mode toggle, system prompt, worker context injection, scratchpad integration
- `tools/AgentTool/AgentTool.tsx` -- Main agent spawning tool (233K, the largest file -- handles sync/async agents, fork path, worktree isolation)
- `tools/AgentTool/runAgent.ts` -- Core agent execution loop
- `tools/AgentTool/agentToolUtils.ts` -- Agent lifecycle management, tool filtering, async agent lifecycle orchestration
- `tools/SendMessageTool/SendMessageTool.ts` -- Inter-agent message delivery, broadcast support, cross-session messaging
- `tools/SendMessageTool/prompt.ts` -- Protocol for teammate communication, shutdown/approval responses

### Fork System

- `tools/AgentTool/forkSubagent.ts` -- Fork feature gate, anti-recursion guard, cache-optimized message construction, worktree notice
- `tools/AgentTool/prompt.ts` -- Agent tool prompt with fork-aware examples, when-to-fork heuristics, writing directive prompts

### Skills System

- `skills/loadSkillsDir.ts` -- Skill loading from filesystem (skills/ and legacy commands/), frontmatter parsing, deduplication, conditional activation, dynamic discovery
- `skills/bundledSkills.ts` -- Programmatic skill registration for compiled-in skills, file extraction, base directory injection
- `skills/bundled/index.ts` -- All bundled skill registrations (update-config, keybindings, verify, debug, lorem-ipsum, remember, simplify, etc.)
- `skills/mcpSkillBuilders.ts` -- Dependency-free registry bridge for MCP skill discovery
- `tools/SkillTool/SkillTool.ts` -- Skill invocation tool with inline and forked execution modes, permission checking, remote skill loading
- `tools/SkillTool/prompt.ts` -- Skill listing with budget management, description truncation

### Task System

- `Task.ts` -- Task types, status enum, ID generation with type prefixes, base state creation
- `tasks.ts` -- Task registry (getAllTasks, getTaskByType)
- `.compozy/tasks/types.ts` -- Discriminated union of all task states, background task detection
- `.compozy/tasks/stopTask.ts` -- Shared task stopping logic with proper notification handling
- `.compozy/tasks/LocalMainSessionTask.ts` -- Backgrounded main session support (Ctrl+B), foreground/background toggle
- `.compozy/tasks/InProcessTeammateTask/types.ts` -- In-process teammate state with identity, lifecycle, message cap
- `.compozy/tasks/InProcessTeammateTask/InProcessTeammateTask.tsx` -- Teammate lifecycle management (kill, append messages, inject user messages)

### Team / Swarm

- `tools/TeamCreateTool/TeamCreateTool.ts` -- Team creation with config file, task list, AppState registration
- `tools/TeamCreateTool/prompt.ts` -- Comprehensive team workflow documentation (spawn, assign, coordinate, shutdown)
- `tools/TeamDeleteTool/TeamDeleteTool.ts` -- Team teardown
- `utils/swarm/spawnInProcess.ts` -- In-process teammate spawning with AsyncLocalStorage isolation
- `utils/swarm/constants.ts` -- Team lead name, swarm session naming, env vars for teammate config
- `utils/swarm/backends/types.ts` -- TeammateExecutor interface, PaneBackend interface, spawn configs
- `utils/swarm/backends/registry.ts` -- Backend detection and caching (tmux vs iTerm2 vs in-process)
- `utils/swarm/inProcessRunner.ts` -- Core execution loop for in-process teammates (53K)
- `utils/swarm/teamHelpers.ts` -- Team file read/write, member management, session cleanup

### Task Management Tools

- `tools/TaskCreateTool/TaskCreateTool.ts` -- Task creation with hooks, auto-expand UI
- `tools/TaskCreateTool/prompt.ts` -- When to use tasks, field descriptions
- `tools/TaskUpdateTool/prompt.ts` -- Status workflow, dependency management, staleness handling
- `tools/TaskListTool/prompt.ts` -- Task listing with teammate workflow integration
- `tools/TaskGetTool/TaskGetTool.ts` -- Full task detail retrieval
- `tools/TaskOutputTool/TaskOutputTool.tsx` -- Task output reading with blocking support
- `tools/TaskStopTool/TaskStopTool.ts` -- Task termination

### Memory / Inter-Agent Communication

- `memdir/teamMemPaths.ts` -- Team memory path resolution with extensive security (symlink, traversal, null byte protection)
- `memdir/teamMemPrompts.ts` -- Combined memory prompt with private + team directories, four-type taxonomy
- `memdir/memdir.ts` -- Core memory system (21K)
- `memdir/memoryTypes.ts` -- Memory type definitions and taxonomy (23K)
