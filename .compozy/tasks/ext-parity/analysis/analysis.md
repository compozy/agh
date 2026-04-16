# Architectural Analysis: AGH vs Compozy Extensibility

Date: 2026-04-15
Scope: compare the shipped extensibility surface in `/Users/pedronauck/Dev/compozy/agh` against `/Users/pedronauck/Dev/compozy/compozy`, with emphasis on real runtime wiring rather than planned architecture.

## Executive conclusion

`compozy` is currently more extensible at the user-facing workflow/runtime layer.

Users can extend `compozy` through first-class resources such as tools, agents, workflows, tasks, MCPs, webhooks, schemas, memories, models, knowledge bases, and project-level config, and those resources are indexed, watched, and reconciled through a shared runtime graph. The system is broad, declarative, and reaches most of the execution flow.

`agh` is currently more advanced as a formal extension host.

It already has a stronger extension runtime contract: manifests, a registry, managed install/update/enable/disable flows, subprocess lifecycle supervision, capability-gated Host API access, bridge adapters, external skills, and a typed hook taxonomy. The architecture is serious.

The gap is that `agh`'s shipped authoring/runtime surface is narrower than its own contract suggests, and narrower than `compozy`'s day-to-day extension breadth. In plain terms: `agh` has the better plugin skeleton, but `compozy` currently gives users more places to actually extend behavior.

## What `compozy` already ships

### 1. A broad resource graph, not isolated extension features

`compozy` exposes a shared `ResourceStore` with `Put`, `Get`, `Delete`, `List`, `Watch`, and batched list operations, covering `workflow`, `task`, `agent`, `tool`, `mcp`, `memory`, `schema`, `model`, `knowledge_base`, `embedder`, and `vector_db`. This is a real runtime composition layer, not just config parsing.

Evidence:

- [engine/resources/store.go](/Users/pedronauck/Dev/compozy/compozy/engine/resources/store.go)
- [engine/autoload/registry.go](/Users/pedronauck/Dev/compozy/compozy/engine/autoload/registry.go)

### 2. Project and workflow resources are published into that graph

Project configs publish tools, memories, schemas, embedders, vector DBs, knowledge bases, and models into the store. Workflow configs publish workflow-local workflows, agents, tools, schemas, MCPs, and knowledge bases.

Evidence:

- [engine/project/indexer.go](/Users/pedronauck/Dev/compozy/compozy/engine/project/indexer.go)
- [engine/workflow/indexer.go](/Users/pedronauck/Dev/compozy/compozy/engine/workflow/indexer.go)

### 3. Live reconcile/watch loop exists

The reconciler watches store events for workflows, agents, tools, schemas, models, MCPs, and knowledge bases, recompiles impacted workflows, and swaps live state.

Evidence:

- [engine/infra/server/reconciler/reconciler.go](/Users/pedronauck/Dev/compozy/compozy/engine/infra/server/reconciler/reconciler.go)

### 4. Tools are first-class user extensions

Custom tools are a core surface in `compozy`, with schemas, timeout, env, config, and CWD, plus runtime execution through the Bun worker.

Evidence:

- [engine/tool/config.go](/Users/pedronauck/Dev/compozy/compozy/engine/tool/config.go)
- [engine/runtime/bun/worker.tpl.ts](/Users/pedronauck/Dev/compozy/compozy/engine/runtime/bun/worker.tpl.ts)
- [engine/agent/config.go](/Users/pedronauck/Dev/compozy/compozy/engine/agent/config.go)

### 5. Extensibility reaches entrypoints and orchestration flow

`compozy` lets users extend the system at webhook ingress, workflow/task DSL, tool runtime, MCP integration, and resource hot-reload.

Evidence:

- [engine/webhook/config.go](/Users/pedronauck/Dev/compozy/compozy/engine/webhook/config.go)
- [engine/webhook/service.go](/Users/pedronauck/Dev/compozy/compozy/engine/webhook/service.go)
- [schemas/workflow.json](/Users/pedronauck/Dev/compozy/compozy/schemas/workflow.json)
- [schemas/task.json](/Users/pedronauck/Dev/compozy/compozy/schemas/task.json)

## What `agh` already ships well

### 1. A real extension host

`agh` has a formal extension manifest, persistent extension registry, managed lifecycle, subprocess supervision, health checks, restart logic, capability grants, and a negotiated Host API.

Evidence:

- [internal/extension/manifest.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/manifest.go)
- [internal/extension/registry.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/registry.go)
- [internal/extension/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/manager.go)
- [internal/subprocess/handshake.go](/Users/pedronauck/Dev/compozy/agh/internal/subprocess/handshake.go)
- [internal/extension/capability.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/capability.go)

### 2. Hook taxonomy is already broad on paper

`agh` defines 33 hook events across session, input, prompt, event, automation, agent, turn, message, tool, permission, and context families.

Evidence:

- [internal/hooks/events.go](/Users/pedronauck/Dev/compozy/agh/internal/hooks/events.go)
- [internal/hooks/introspection.go](/Users/pedronauck/Dev/compozy/agh/internal/hooks/introspection.go)

### 3. Extension packaging and governance are stronger than in `compozy`

`agh` already has operator-grade install/list/enable/disable/update surfaces and source-tier security controls. That part is ahead of `compozy`, which is more config/runtime-centric than plugin-host-centric.

Evidence:

- [internal/cli/extension.go](/Users/pedronauck/Dev/compozy/agh/internal/cli/extension.go)
- [internal/api/udsapi/extensions.go](/Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/extensions.go)
- [internal/extension/install_managed.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/install_managed.go)

## The concrete gaps in `agh`

These are the real gaps if the goal is "highly extensible like `compozy`, but with stronger plugin architecture".

### Gap 1. Tool and permission hooks are defined but not wired into the main runtime

`agh` defines `tool.pre_call`, `tool.post_call`, `tool.post_error`, `permission.request`, `permission.resolved`, and `permission.denied` in the hook taxonomy. But the production search only finds the dispatch method definitions themselves, not runtime callers outside tests. The session hook interfaces also omit tool and permission groups entirely.

This is the clearest contract-vs-runtime mismatch in the system.

Evidence:

- [internal/hooks/dispatch.go](/Users/pedronauck/Dev/compozy/agh/internal/hooks/dispatch.go)
- [internal/session/hooks.go](/Users/pedronauck/Dev/compozy/agh/internal/session/hooks.go)

### Gap 2. Extension-provided tools exist in protocol/SDK shape, but not in daemon negotiation

The subprocess handshake supports `provide_tools`, and the TypeScript SDK can implement it. But the daemon only negotiates `execute_hook`, `health_check`, and `shutdown`. That means the runtime does not actually ask extensions to provide tools, even though the protocol and SDK imply that it can.

Evidence:

- [internal/subprocess/handshake.go](/Users/pedronauck/Dev/compozy/agh/internal/subprocess/handshake.go)
- [sdk/typescript/src/extension.ts](/Users/pedronauck/Dev/compozy/agh/sdk/typescript/src/extension.ts)
- [internal/extension/manager.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/manager.go)

### Gap 3. Manifest resources do not have first-class tool contribution

Extension manifests can contribute `skills`, `agents`, `bundles`, `hooks`, and `mcp_servers`, but not `tools`. Compared with `compozy`, where tools are a primary resource, this keeps `agh` from exposing tools as a first-class extension unit.

Evidence:

- [internal/extension/manifest.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/manifest.go)
- [engine/tool/config.go](/Users/pedronauck/Dev/compozy/compozy/engine/tool/config.go)

### Gap 4. Host API lacks a generic resource graph

`agh` Host API currently covers sessions, memory, observe, skills, automation, tasks, and bridges. It does not expose a generalized resource registry analogous to `compozy`'s `ResourceStore`, and it has no `tools/*` Host API family even though `tool.read` appears in the marketplace security ceiling.

That means extension authors cannot compose new runtime resources through one unified surface.

Evidence:

- [internal/extension/protocol/host_api.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/protocol/host_api.go)
- [internal/extension/capability.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/capability.go)
- [engine/resources/store.go](/Users/pedronauck/Dev/compozy/compozy/engine/resources/store.go)

### Gap 5. The live extensibility surface is concentrated in sessions and automation

The hook bridge and session wiring cover session lifecycle, prompt/input, event persistence, agent lifecycle, turn/message streaming, compaction, and automation fire/run hooks. That is substantial, but it is not yet "almost every part of the process" in the way `compozy` reaches config, resources, tools, workflows, tasks, MCPs, and webhooks.

Evidence:

- [internal/daemon/hooks_bridge.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/hooks_bridge.go)
- [internal/session/manager_hooks.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_hooks.go)
- [internal/automation/dispatch.go](/Users/pedronauck/Dev/compozy/agh/internal/automation/dispatch.go)

### Gap 6. Authoring DX is behind the runtime design

There is a standalone extension scaffolder package with two templates, `hook-subprocess` and `memory-backend`, but this still feels narrower than the runtime ambition and is not part of the daemon CLI/operator flow. The SDK/runtime story is stronger than nothing, but still thinner than the platform needs.

Evidence:

- [sdk/create-extension/src/index.ts](/Users/pedronauck/Dev/compozy/agh/sdk/create-extension/src/index.ts)
- [sdk/typescript/src/index.ts](/Users/pedronauck/Dev/compozy/agh/sdk/typescript/src/index.ts)

## The correct conclusion

If the benchmark is "does `agh` already have a serious extensibility architecture?", the answer is yes.

If the benchmark is "is `agh` already as extensible in practice as `compozy` is for end users?", the answer is no.

The main reason is not lack of architectural direction. The main reason is incomplete shipping of the surfaces that `agh` already gestures toward:

1. wire the missing tool and permission hooks into production paths;
2. make extension-provided tools real, not only negotiated in SDK/tests;
3. decide whether `agh` wants a first-class resource graph analogous to `compozy` or whether extensions will stay manifest-and-hook centric;
4. expand the extension authoring story around the runtime that already exists.

## Recommended priority order

### P0: close contract/runtime gaps

- Wire `tool.*` and `permission.*` hooks into real production execution.
- Either ship `provide_tools` end to end or remove/de-scope it from the public extension contract until ready.
- Add first-class extension manifest support for tools if tool extensibility is meant to be a core promise.

### P1: add a unified extensibility model

- Decide whether `agh` should gain a resource graph with watch/reconcile semantics, or keep a narrower extension model centered on manifests plus Host API.
- If the answer is yes, introduce a real resource ownership model instead of adding one-off APIs per capability.

### P2: improve creator DX

- Promote extension scaffolding to a first-class workflow.
- Add more templates than `hook-subprocess` and `memory-backend`.
- Make packaging, validation, and local iteration paths as strong as operator install/enable/status paths.

## Bottom line

`compozy` is broader today.

`agh` is architecturally sharper today.

To reach the goal of a highly extensible AGH, the next work should not start with another round of abstract extension concepts. It should start by finishing the surfaces already claimed by the protocol and hook taxonomy, then deciding whether to adopt a true resource-graph model comparable to `compozy`.
