# AGH Runtime Capability Analysis

## Runtime product summary

AGH Runtime is the local daemon and operator surface for running AI agent sessions on one machine or server. In practical terms, it installs into a workspace, starts a background daemon, exposes the runtime through HTTP/SSE and a Unix domain socket, launches ACP-compatible agents as subprocesses, persists every session event, and gives the operator CLI and web UI surfaces to create, resume, inspect, and govern work.

It is broader than “chat with an agent.” The runtime also owns workspace-aware memory, skills, automation, bridges, extensions, hooks, and observability. Those are runtime capabilities, but they do not all deserve equal weight in homepage copy.

The AGH Network Protocol is intentionally out of scope for this analysis. It should be treated as a separate product surface with its own docs, terminology, and launch narrative.

## Capability map

| Area                        | What it does in practice                                                                                                    | Launch signal    |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| Daemon + transport          | Boots and stops the runtime, serves the web UI, and exposes the same runtime over HTTP/SSE and UDS.                         | Core runtime     |
| Sessions                    | Creates, lists, prompts, resumes, waits on, and stops agent sessions; streams live events; exposes history and transcripts. | Core runtime     |
| Persistence + observability | Stores per-session events and state, exposes daemon health, cross-session events, and reconciliation views.                 | Core runtime     |
| Memory                      | Manages global and workspace memory, supports read/write/delete, and runs dream consolidation.                              | Core runtime     |
| Skills                      | Lists, views, installs, updates, enables, and disables skills; loads workspace-scoped skills and skill-local MCP sidecars.  | Core runtime     |
| Workspaces                  | Registers, resolves, edits, and removes workspaces; overlays workspace config on top of global config.                      | Core runtime     |
| Automation                  | Manages jobs, triggers, runs, and webhook delivery for scheduled or event-driven work.                                      | Advanced runtime |
| Bridges                     | Creates and manages bridge instances, lists routes, and tests delivery targets.                                             | Advanced runtime |
| Extensions                  | Searches, installs, enables, disables, updates, and inspects extensions.                                                    | Supporting/admin |
| Hooks + tasks               | Surfaces hook catalog/runs/events and task graphs/runs for inspection and orchestration.                                    | Supporting/admin |

## What AGH Runtime is in practical terms

AGH Runtime is the control plane you install locally when you want to run AI agents without handing the workflow to a hosted service. The runtime handles:

- starting and stopping the daemon
- creating and resuming sessions
- sending prompts and receiving live output
- keeping a replayable event history and canonical transcript
- attaching workspace-specific memory and skills
- managing automation and integrations from the same product

This is the product that should anchor the main site. The network protocol should be a separate branch in the information architecture, not a subsection of runtime marketing copy.

## Launch-worthy differentiators

1. Single local daemon, no sidecars, no external services. The README states this directly, and the architecture centers the `daemon/` composition root.
2. Unified operator surface. The same runtime is reachable from CLI, HTTP/SSE, and UDS, which makes it useful both for human operators and automation.
3. Durable sessions. AGH can create, resume, stop, inspect, and replay sessions, and it exposes both raw events and grouped history.
4. Persistent memory with consolidation. Runtime memory is not an ephemeral scratchpad; it has global and workspace scope and supports dream-style consolidation.
5. Skills as first-class runtime assets. Skills are loaded from bundled, user, marketplace, and workspace sources, and they can contribute MCP servers to the runtime.
6. Workspace-aware operations. Runtime behavior changes with the active workspace, which gives AGH a real operator model instead of a flat global configuration store.
7. Advanced operations in one place. Automation and bridge management are built into the same runtime, so AGH can coordinate scheduled work and delivery targets without needing a second control plane.

These are the strongest candidates for homepage copy and the top-level runtime narrative.

## Secondary/internal capabilities

These capabilities are real, but they should stay lower in the runtime story:

- Hooks: powerful for inspection and governance, but too low-level to lead the homepage.
- Extensions: important for platform growth, but they read like an admin surface.
- Tasks: useful for orchestration and linked work, but they feel adjacent to the core runtime promise.
- Agent definitions and `whoami`: necessary baseline surfaces, not differentiators.
- Output formats (`human`, `json`, `toon`): useful for operators and scripts, not a product selling point.
- Network runtime / AGH Network Protocol: separate product surface, not runtime copy.

One terminology note: the codebase currently uses `memory` in the API and `knowledge` in the web route name. Runtime docs should standardize on one canonical term, most likely `Memory`, and mention the UI label only as an alias.

## Recommended runtime doc taxonomy

### 1. Home

The home page should do three things:

1. explain what AGH Runtime is
2. show the runtime/product split clearly
3. point to the right docs entry points fast

### 2. Getting Started

Required sections:

- install and bootstrap
- daemon start/stop/status
- first session
- first prompt
- first resume
- how to open the web UI
- where config files live

### 3. Concepts

Required sections:

- daemon and transport model
- sessions, event history, and transcript replay
- memory and consolidation
- skills and agent definitions
- workspaces and workspace overlay
- automation
- bridges
- extensions
- hooks
- observability and health

### 4. Operations

Required sections:

- daemon operations
- session operations
- memory operations
- skills operations
- workspace operations
- automation operations
- bridge operations
- extension operations
- troubleshooting

### 5. Integrations

Required sections:

- ACP-compatible agents
- agent definitions and `AGENT.md`
- MCP sidecars and skill-local MCP loading
- external providers and default models
- workspace-scoped runtime overlays

### 6. CLI / API / Reference

Required sections:

- CLI reference
- API reference
- output formats and flags
- config reference
- agent definition reference
- memory/reference schema
- session/event reference
- observability reference

### 7. Developer Guide

Required sections:

- architecture overview
- package map
- boot sequence
- persistence model
- runtime composition root
- observability pipeline
- how the web UI maps to runtime behavior

### 8. Separate Protocol Section

AGH Network Protocol should have its own documentation branch with:

- wire format
- message kinds
- transport binding
- trust / conformance model
- examples

That section should stand apart from runtime docs so the runtime story stays crisp.

## Evidence

- [README.md](/Users/pedronauck/Dev/compozy/agh/README.md) - product summary, CLI tree, config, agent definitions, web UI, project structure.
- [internal/cli/root.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/root.go) - runtime command tree and transport-neutral CLI entrypoints.
- [internal/api/spec/spec.go](/Users/pedronauck/Dev/compozy/agh/internal/api/spec/spec.go) - canonical runtime API for sessions, memory, skills, workspaces, automation, bridges, extensions, hooks, tasks, and daemon health.
- [internal/session/manager_lifecycle.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_lifecycle.go) - create/resume/stop lifecycle and session finalization.
- [internal/session/transcript.go](/Users/pedronauck/Dev/compozy/agh/internal/session/transcript.go) - canonical transcript assembly from persisted events.
- [internal/memory/consolidation/runtime.go](/Users/pedronauck/Dev/compozy/agh/internal/memory/consolidation/runtime.go) - dream consolidation runtime and background loop.
- [internal/daemon/boot.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/boot.go) - boot sequence and runtime composition order.
- [internal/daemon/daemon.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/daemon.go) - runtime dependency graph and server factories.
- [internal/cli/skill_commands.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/skill_commands.go) - skill list/view/create/install/update/remove workflows.
- [internal/cli/workspace.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/workspace.go) - workspace registration and overlay management.
- [internal/cli/observe.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/observe.go) - observability queries and event streaming.
- [web/src/routes/\_app/session.$id.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/session.$id.tsx) - live session chat, transcript hydration, resume/stop, permission prompts.
- [web/src/routes/\_app/knowledge.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/knowledge.tsx) - memory/knowledge UI with global/workspace scope.
- [web/src/routes/\_app/skills.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/skills.tsx) - installed skills and marketplace view.
- [web/src/routes/\_app/automation.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/automation.tsx) - automation jobs, triggers, and run history.
- [web/src/routes/\_app/bridges.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/bridges.tsx) - bridge creation, routes, and delivery testing.
- [web/src/routes/\_app/network.tsx](/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/network.tsx) - separate network runtime surface, useful as a boundary marker.
- [.resources/openclaw/docs/index.md](/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/docs/index.md) - hub-style docs with get started, concepts, learn-more, and troubleshooting entry points.
- [.resources/hermes/website/docs/index.md](/Users/pedronauck/Dev/compozy/agh/.resources/hermes/website/docs/index.md) - homepage links grouped by getting started, user guide, features, developer guide, and reference.
- [.resources/opencode/packages/docs/index.mdx](/Users/pedronauck/Dev/compozy/agh/.resources/opencode/packages/docs/index.mdx) - docs split into essentials, development, and AI-tool-specific pages.
- [.resources/goclaw/docs/00-architecture-overview.md](/Users/pedronauck/Dev/compozy/agh/.resources/goclaw/docs/00-architecture-overview.md) - architecture-first docs with module map and capability inventory.
- [.resources/acpx/conformance/README.md](/Users/pedronauck/Dev/compozy/agh/.resources/acpx/conformance/README.md) - spec/conformance-oriented docs with explicit scope and non-goals.
- [docs/rfcs/003_agh-network-v0.md](/Users/pedronauck/Dev/compozy/agh/docs/rfcs/003_agh-network-v0.md) - protocol boundary showing the separate AGH Network product surface.
