# Pi-Mono Extension Ecosystem Analysis

**Framework**: Pi-Mono (formerly "pi") by Mario Zechner (@badlogic)
**Type**: TypeScript AI coding agent framework (monorepo)
**Repository**: [github.com/badlogic/pi-mono](https://github.com/badlogic/pi-mono)
**Website**: [shittycodingagent.ai](https://shittycodingagent.ai/)
**Community list**: [qualisero/awesome-pi-agent](https://github.com/qualisero/awesome-pi-agent)
**Date researched**: 2026-04-11

---

## Overview of Findings

Pi-Mono is a TypeScript monorepo that provides a minimal AI coding agent with an aggressively extensible architecture. Its philosophy is "what you leave out matters more than what you put in" -- the core ships with only 4 tools (read, write, edit, bash) and no built-in MCP, sub-agents, plan mode, or permission popups. Instead, all of these are buildable via extensions.

Pi's extension model has four dimensions:

1. **Extensions** -- TypeScript modules that hook into lifecycle events, register tools, add commands, customize UI
2. **Skills** -- Directory-based capability packages with a `SKILL.md` file, loaded on-demand (progressive disclosure)
3. **Prompt Templates** -- Reusable prompt shortcuts stored as Markdown
4. **Themes** -- JSON files customizing TUI appearance

All four can be bundled into **Pi Packages** and distributed via npm (keyword `pi-package`) or git.

The ecosystem is young but active, with 30+ community extensions, a curated awesome-list, and adoption by OpenClaw (the most-starred GitHub repo). The extension patterns map remarkably well to AGH's three-dimensional model.

---

## Extension Catalog

### Official Example Extensions (packages/coding-agent/examples/extensions/)

| Name                | Category          | Description                                                                                                                               | AGH Mapping                                     |
| ------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| `hello.ts`          | Tool + Event      | Minimal example: registers a "greet" tool, subscribes to `tool_call` events to block dangerous `rm -rf` commands, adds a `/hello` command | Resource (hook) + Capability (content.validate) |
| `todo.ts`           | Stateful Tool     | Persistent to-do list with state management across sessions                                                                               | Resource (skill) + Action (session state)       |
| `tool-override.ts`  | Tool Override     | Overrides the built-in `read` tool with custom behavior while inheriting default rendering                                                | Capability (agent.driver tool override)         |
| `dynamic-tools.ts`  | Dynamic Tools     | Registers/unregisters tools at runtime via `pi.setActiveTools()`                                                                          | Capability (agent.driver dynamic registration)  |
| `question.ts`       | Interactive Tool  | Tool that prompts the user for input via `ctx.ui`                                                                                         | Action (session interaction)                    |
| `questionnaire.ts`  | Multi-step Wizard | Multi-step interactive tool with sequential user prompts                                                                                  | Action (session interaction)                    |
| `truncated-tool.ts` | Output Control    | Demonstrates output truncation (50KB / 2000 lines limit)                                                                                  | Capability (content.validate)                   |
| `doom-overlay.ts`   | TUI Overlay       | Plays DOOM as a WebAssembly overlay in the terminal at 35 FPS                                                                             | Resource (UI extension)                         |

### Community Extensions (from awesome-pi-agent + npm)

| Name                      | Author        | Category           | Description                                                                                                                                       | AGH Mapping                                                                     |
| ------------------------- | ------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **filter-output**         | community     | Security           | Redacts sensitive data (API keys, tokens, passwords) from tool results before the LLM sees them                                                   | Capability: **content.validate** -- sanitize output before it reaches the model |
| **security**              | community     | Permission Gate    | Blocks dangerous bash commands and protects sensitive file paths from writes                                                                      | Capability: **permission.gate** -- pre-execution validation                     |
| **safe-git**              | community     | Permission Gate    | Requires user approval before dangerous git operations (force push, reset --hard, etc.)                                                           | Capability: **permission.gate** -- git-specific guard                           |
| **plan-mode**             | community     | Tool Override      | Read-only exploration mode that restricts the agent to non-destructive tools only                                                                 | Capability: **permission.gate** + **agent.driver** tool filtering               |
| **oracle**                | community     | Multi-Model        | Gets a second opinion from an alternative AI model without switching context                                                                      | Capability: **agent.driver** (multi-model dispatch)                             |
| **handoff**               | jayshah5696   | Session Management | Transfers context to a new focused session with editable handoff prompt                                                                           | Action: **session** (fork/spawn with context transfer)                          |
| **memory-mode**           | community     | Persistent Memory  | Saves instructions to AGENTS.md with AI-assisted integration                                                                                      | Capability: **memory.backend** -- instruction persistence                       |
| **cost-tracker**          | hjanuschka    | Observability      | Session spending analysis from pi logs                                                                                                            | Capability: **observe.exporter** -- cost metrics                                |
| **usage-bar**             | hjanuschka    | Observability      | Persistent provider status indicator showing token/cost stats in footer                                                                           | Capability: **observe.exporter** -- real-time metrics display                   |
| **pi-rewind**             | arpagon       | Checkpointing      | Git-based file checkpoints per turn, `/rewind` command with diff preview, redo stack                                                              | Action: **session** (checkpoint/restore) + Resource (hook)                      |
| **pi-powerline-footer**   | nicobailon    | UI Customization   | Powerline-style status bar with git integration, model switcher, editor stash (Alt+S), welcome overlay, "working vibes" (themed loading messages) | Resource (UI extension)                                                         |
| **pi-canvas**             | community     | UI Component       | Interactive TUI canvases (calendar, document, flights) rendered inline                                                                            | Resource (UI extension)                                                         |
| **background-notify**     | community     | Notifications      | Audio beep + terminal focus when tasks complete                                                                                                   | Capability: **observe.exporter** (notification channel)                         |
| **session-emoji**         | community     | UI                 | AI-powered emoji in footer representing conversation context                                                                                      | Resource (UI extension)                                                         |
| **session-color**         | community     | UI                 | Colored band in footer to distinguish active sessions                                                                                             | Resource (UI extension)                                                         |
| **pi-ssh-remote**         | community     | Remote Execution   | Redirects all file operations and commands to a remote host via SSH                                                                               | Capability: **agent.driver** (remote execution backend)                         |
| **pi-dcp**                | community     | Context Management | Dynamic context pruning for intelligent conversation optimization                                                                                 | Capability: **message.transform**                                               |
| **pi-rtk**                | sherif-fanous | Token Optimization | Routes bash commands through rtk for LLM token savings                                                                                            | Capability: **message.transform** (output compression)                          |
| **ultrathink**            | community     | UI Effect          | Rainbow animated effect with Knight Rider shimmer during thinking                                                                                 | Resource (UI extension / theme)                                                 |
| **pi-gui**                | community     | UI                 | GUI extension providing visual interface for the pi agent                                                                                         | Resource (UI extension)                                                         |
| **pi-screenshots-picker** | community     | Tool               | Screenshot picker extension for better screenshot selections                                                                                      | Resource (tool)                                                                 |
| **pi-super-curl**         | community     | Tool               | Empowers curl requests with coding agent capabilities                                                                                             | Resource (tool)                                                                 |
| **go-to-bed**             | mitsuhiko     | Safety Guard       | Late-night safety guard with explicit confirmation after midnight                                                                                 | Capability: **permission.gate** (time-based)                                    |
| **multi-edit**            | mitsuhiko     | Tool Override      | Replaces built-in edit tool with batch multi-edits and Codex-style patch support with preflight validation                                        | Capability: **agent.driver** (tool replacement)                                 |
| **loop**                  | mitsuhiko     | Workflow           | Prompt loop for rapid iterative coding with optional auto-continue                                                                                | Action: **session** (iteration control)                                         |
| **context**               | mitsuhiko     | Observability      | Context breakdown showing extensions, skills, AGENTS.md/CLAUDE.md with token usage                                                                | Capability: **observe.exporter**                                                |
| **files**                 | mitsuhiko     | Tool               | Unified file browser with git status, session references, reveal/open/edit/diff actions                                                           | Resource (tool)                                                                 |

### Orchestration Tools (built on Pi SDK)

| Name             | Category            | Description                                                                                                                | AGH Mapping                                           |
| ---------------- | ------------------- | -------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| **Grove**        | Multi-Agent         | Reads a structured markdown plan, parses work streams with dependencies, orchestrates parallel AI agents via web dashboard | Action: **session** (multi-agent orchestration)       |
| **PiSwarm**      | Multi-Agent         | Parallel GitHub issue and PR processing using pi agent and git worktrees                                                   | Action: **session** (parallel agent dispatch)         |
| **task-factory** | Queue Orchestration | Queue-first work orchestrator with planning, execution skills, and web UI                                                  | Action: **session** + **skills** (task queue)         |
| **Gondolin**     | Sandboxing          | Linux micro-VM sandbox with programmable network/filesystem and Pi integration                                             | Capability: **permission.gate** (sandboxed execution) |
| **pi-mobile**    | Mobile Client       | Android client for Pi coding agent with session management over Tailscale                                                  | Resource (client transport)                           |

### Skills Packages

| Name                      | Author                     | Skills Included                                                                                                                                                           | AGH Mapping                              |
| ------------------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| **agent-stuff (mitsupi)** | mitsuhiko (Armin Ronacher) | commit, changelog, github, web-browser, tmux, sentry, ghidra, google-workspace, mermaid, native-web-search, openscad, pi-share, summarize, uv, frontend-design, librarian | Resource: **skills** catalog             |
| **pi-amplike**            | community                  | Web search and webpage extraction via Jina APIs                                                                                                                           | Resource: **skills** (search/extraction) |

### Themes

| Name             | Description                                  | AGH Mapping                              |
| ---------------- | -------------------------------------------- | ---------------------------------------- |
| **pi-rose-pine** | Rose Pine themes (main, moon, dawn variants) | Resource: theme (if AGH adds UI theming) |

---

## Detailed Analysis: Most Impactful Extensions for AGH

### 1. filter-output -- Content Sanitization Before LLM

**What it does**: Intercepts tool results and redacts sensitive data (API keys, tokens, passwords) before the LLM sees them. Uses regex patterns to detect and mask secrets.

**Why it matters**: Prevents accidental leakage of credentials into LLM context (and therefore into API provider logs, training data, or displayed output). This is a critical security layer.

**AGH mapping**: Capability: `content.validate` -- register a validator that inspects outbound tool results. Could also be `message.transform` since it modifies content before model consumption.

**Implementation idea**: AGH could ship a built-in `content.validate` capability that scans for common secret patterns (AWS keys, GitHub tokens, JWT tokens, private keys) and either redacts or blocks them. Make the pattern list configurable via TOML.

---

### 2. security + safe-git -- Permission Gates

**What they do**: `security` blocks dangerous bash commands (rm -rf /, sudo, etc.) and protects sensitive paths (.env, node_modules). `safe-git` specifically guards destructive git operations (force push, reset --hard, branch -D).

**Why it matters**: Pi runs in "YOLO mode" with no built-in permission checks. These extensions are the community's answer to that -- granular, composable permission gates rather than a one-size-fits-all confirmation dialog.

**AGH mapping**: Capability: `permission.gate` -- this is a direct match. AGH can define a `PermissionGate` interface that receives a tool call and returns allow/deny/prompt-user.

**Implementation ideas**:

- Ship a default `permission.gate` with configurable deny-lists for bash patterns and file paths
- Allow stacking multiple gates (security + safe-git both fire)
- Gates should be orderable (priority field) so more specific gates can override general ones

---

### 3. OpenClaw's Context Pruning + Compaction Safeguards

**What it does**: OpenClaw uses two Pi extensions for context management:

- `context-pruning.ts`: Cache-TTL based pruning that trims old tool results using a time-decay model (recent = full, old = head+tail, ancient = removed)
- `compaction-safeguard.ts`: Multi-stage compaction pipeline that preserves file operation history and tool failure data, with adaptive token budgeting

The four-layer strategy: message count limit, token count limit, TTL time decay, smart compaction.

**Why it matters**: Context management is the #1 challenge for long-running agent sessions. Pi's `session_before_compact` and `context` events let extensions implement sophisticated strategies without modifying the core.

**AGH mapping**: Capability: `message.transform` (for the `context` event equivalent) + a new compaction strategy extension point.

**Implementation ideas**:

- Define a `CompactionStrategy` interface in AGH that receives the message history and returns a compacted version
- Ship a default strategy but allow override via configuration
- The `context` event pattern (rewrite messages before each LLM call) maps to `message.transform`
- Consider a "memory flush before compaction" pattern: before compacting, give the agent a turn to save important state to persistent memory

---

### 4. handoff -- Session Context Transfer

**What it does**: Extracts the current conversation context, opens an editor for the user to review/refine the handoff prompt, then spawns a new focused session with that context. Shows a loader while context is extracted.

**Why it matters**: Long sessions accumulate noise. Handoff lets users distill the important context and start fresh without losing progress. This is particularly valuable when switching between exploration and implementation phases.

**AGH mapping**: Action: `session` -- AGH already has session management. Handoff would be a session fork operation with context summarization.

**Implementation idea**: Add a `session.fork` action that accepts a context summary, creates a new session with that summary as the initial context, and optionally archives the parent session.

---

### 5. oracle -- Multi-Model Second Opinion

**What it does**: Sends the current conversation context to an alternative AI model and returns its response, without switching the active model or disrupting the session.

**Why it matters**: Different models have different strengths. Getting a second opinion on a tricky architectural decision or bug diagnosis without context-switching is valuable.

**AGH mapping**: Capability: `agent.driver` -- AGH already supports multiple agent drivers. Oracle would dispatch the current context to a different driver and inject the response.

**Implementation idea**: Add an `agent.consult` action that sends context to a specified agent driver and returns the response as an injected message. Could be triggered via CLI command or API endpoint.

---

### 6. pi-rewind -- Git-Based Checkpointing

**What it does**: Creates automatic git-based snapshots (stored as refs) after write/edit/bash tools -- one checkpoint per turn. Provides a `/rewind` command with a checkpoint browser, diff preview, and safe restore. Includes a redo stack and Esc+Esc quick rewind shortcut.

**Why it matters**: AI agents make mistakes. The ability to cleanly rewind file changes while preserving conversation state is essential for confidence in agent-assisted coding.

**AGH mapping**: Resource: `hook` (on tool completion) + Action: `session` (checkpoint/restore).

**Implementation ideas**:

- AGH hook that fires after file-modifying tools complete
- Store checkpoints as git stash entries or lightweight refs
- Expose checkpoint browse/restore via UDS API for CLI access
- Consider per-session checkpoint namespacing to avoid conflicts

---

### 7. multi-edit (mitsuhiko) -- Batch Edit Tool

**What it does**: Replaces Pi's built-in edit tool with one that supports batch multi-edits (multiple changes in a single tool call) and Codex-style unified diff patch format. Includes preflight validation to catch errors before applying.

**Why it matters**: Single-edit-per-call is expensive in tokens and turns. Batch edits reduce round trips and give the model a more natural way to express multi-site refactors.

**AGH mapping**: Capability: `agent.driver` -- tool override/replacement.

**Implementation idea**: AGH could allow agent drivers to register tool variants. A "batch edit" tool would be a configuration option on the ACP driver, letting the agent express multiple edits in one tool call.

---

### 8. plan-mode -- Read-Only Exploration

**What it does**: Restricts the agent to read-only tools (read, grep, find, ls) so it can explore and plan without making any changes. The user can review the plan and then switch back to full mode.

**Why it matters**: Separating exploration from execution reduces risk and gives users confidence to let the agent explore freely.

**AGH mapping**: Capability: `permission.gate` (restricts tool set) + Action: `session` (mode switching).

**Implementation idea**: AGH session modes -- define a "readonly" mode that filters the available tools to non-destructive ones only. Could be toggled via CLI or API.

---

### 9. context (mitsuhiko) -- Context Breakdown + Token Usage

**What it does**: Shows a breakdown of what's consuming context: loaded extensions, skills, AGENTS.md/CLAUDE.md content, and token usage per component. Highlights which skills are currently loaded.

**Why it matters**: Context windows are the scarcest resource. Understanding what's consuming tokens helps users optimize their setup.

**AGH mapping**: Capability: `observe.exporter` -- expose context composition metrics.

**Implementation idea**: AGH observe package could track token allocation across system prompt components (agent instructions, skills, memory, conversation history) and expose it via the HTTP API for the web UI.

---

### 10. Gondolin -- Micro-VM Sandboxing

**What it does**: Provides Linux micro-VMs with programmable network/filesystem as execution sandboxes for Pi agents. The TypeScript control plane manages VM lifecycle.

**Why it matters**: Running untrusted agent commands in isolation is the gold standard for security. VMs provide stronger isolation than containers.

**AGH mapping**: Capability: `permission.gate` (sandboxed execution environment) -- or more accurately, an alternative `agent.driver` execution backend.

**Implementation idea**: AGH could support pluggable execution backends: direct subprocess (default), container, or VM. The agent driver would delegate tool execution to the configured backend.

---

## Extension Patterns Summary

### Pattern 1: Event-Driven Lifecycle Hooks

Pi's event system provides these hook points:

- `session_start` / `session_switch` / `session_shutdown` -- session lifecycle
- `before_agent_start` -- inject/modify system prompt before each agent turn
- `tool_call` -- intercept, gate, or modify tool calls before execution
- `context` -- rewrite the message array before each LLM call
- `session_before_compact` -- customize or cancel compaction
- `session_before_fork` -- control session forking behavior
- `turn_start` / `turn_end` -- per-turn lifecycle
- `input` -- transform user input before skill/template expansion

**AGH equivalent**: These map to AGH's hook system. The most valuable hooks for AGH:

1. Pre-tool-execution gate (permission.gate)
2. Pre-LLM message transform (message.transform)
3. Pre-compaction override
4. Session lifecycle (start, fork, shutdown)

### Pattern 2: Tool Override with Rendering Independence

Pi allows overriding built-in tools while inheriting the default rendering. Execution and rendering are decoupled -- you can wrap a tool for logging or access control without reimplementing its UI.

**AGH equivalent**: Since AGH separates the agent driver from the UI (HTTP/SSE), tool overrides should focus on execution behavior. The web UI rendering is already decoupled.

### Pattern 3: Progressive Disclosure via Skills

Pi skills are directories with a `SKILL.md` frontmatter file. Only skill descriptions are always in context; full instructions load on-demand via the `read` tool. This preserves prompt cache efficiency while providing deep capability.

**AGH equivalent**: AGH's skills system already follows this pattern. Key insight: the skill description should be tightly written for the prompt cache, with full instructions loaded lazily.

### Pattern 4: Composable Package Distribution

Pi packages bundle multiple extension types (extensions, skills, prompts, themes) into a single installable unit via npm or git. The `pi` key in `package.json` declares what's included.

**AGH equivalent**: Since AGH is a Go binary, the distribution mechanism would differ. Options: Git-based skill/hook repos, Go plugin system, or WASM-based extensions. The "bundle multiple types in one package" pattern is worth preserving.

### Pattern 5: Context as Mutable Pipeline

Pi's `context` event treats the message history as a mutable pipeline before each LLM call. Extensions can filter, transform, inject, or prune messages non-destructively (the original history stays on disk).

**AGH equivalent**: Capability: `message.transform`. This is one of the most powerful patterns -- it enables RAG injection, context pruning, memory injection, and token optimization without modifying the persisted event stream.

---

## Key Takeaways for AGH Extension Ideas

### High-Priority Extensions to Build or Enable

1. **Secret Redaction** (`content.validate`): Scan tool outputs for API keys, tokens, and credentials before they reach the LLM. Ship as a built-in capability with configurable patterns.

2. **Permission Gates** (`permission.gate`): Composable, stackable gates for bash commands, file paths, and git operations. Ship a sensible default set, allow user override via TOML.

3. **Context Pruning** (`message.transform`): TTL-based and token-budget-based pruning of old tool results. Essential for long-running sessions.

4. **Custom Compaction Strategies**: Allow plugins to override the default compaction with domain-specific summarization (e.g., preserve file operation history, tool failures, architectural decisions).

5. **Session Checkpointing**: Automatic git-based snapshots after file-modifying operations, with rewind/restore via CLI.

6. **Session Handoff/Fork**: Distill context and spawn focused sub-sessions.

7. **Multi-Model Consultation**: Send context to an alternative agent for a second opinion without disrupting the active session.

8. **Cost/Token Observability**: Real-time tracking of token consumption, cost per session, and context composition breakdown.

### Design Principles Learned from Pi

1. **Core minimalism, extension maximalism**: Ship the smallest possible core, make everything else extensible. Pi proves that even sub-agents, plan mode, and MCP can be extensions.

2. **Events over configuration**: Rather than adding config flags for every behavior, expose lifecycle events and let extensions implement the behavior. This is more composable and maintainable.

3. **Execution and rendering are independent concerns**: Tool overrides should not require reimplementing the display. AGH's architecture (Go backend, React frontend) already enforces this naturally.

4. **Progressive disclosure for skills**: Only inject skill descriptions into the system prompt; load full instructions on-demand. This preserves prompt cache efficiency.

5. **Distribution matters**: Pi's npm/git package system with a single `pi install` command drove community adoption. AGH needs an equally frictionless extension distribution story.

6. **Security is opt-in, not opt-out**: Pi chose no built-in permission gates (full YOLO mode) and lets extensions add security. AGH should consider shipping sensible defaults but making them overridable.

---

## Sources

- [Pi-Mono GitHub Repository](https://github.com/badlogic/pi-mono)
- [Pi Extensions Documentation](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/extensions.md)
- [Pi Packages Documentation](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/packages.md)
- [Pi Skills Documentation](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/skills.md)
- [Pi Extension Examples](https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent/examples/extensions)
- [Awesome Pi Agent](https://github.com/qualisero/awesome-pi-agent)
- [Pi Coding Agent Website](https://shittycodingagent.ai/)
- [Pi Packages Page](https://shittycodingagent.ai/packages)
- [agent-stuff by mitsuhiko](https://github.com/mitsuhiko/agent-stuff)
- [pi-rewind](https://github.com/arpagon/pi-rewind)
- [pi-powerline-footer](https://github.com/nicobailon/pi-powerline-footer)
- [pi-agent-extensions by jayshah5696](https://github.com/jayshah5696/pi-agent-extensions)
- [shitty-extensions by hjanuschka](https://github.com/hjanuschka/shitty-extensions)
- [How to Build a Custom Agent Framework with PI (Nader Dabit)](https://nader.substack.com/p/how-to-build-a-custom-agent-framework)
- [What I Learned Building an Opinionated and Minimal Coding Agent (Mario Zechner)](https://mariozechner.at/posts/2025-11-30-pi-coding-agent/)
- [Pi-Mono DeepWiki](https://deepwiki.com/badlogic/pi-mono)
- [npm: keywords:pi-package](https://www.npmjs.com/search?q=keywords:pi-package)
- [OpenClaw Compaction Docs](https://docs.openclaw.ai/concepts/compaction)
