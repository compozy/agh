# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Write Task 09 runtime agent docs: definitions, providers, spawning, and sidebar metadata under `packages/site/content/runtime/agents/`.
- Completion requires site build, browser QA on `/runtime/agents/*`, final verification evidence, task tracking updates, and a local commit only if the full gate is clean.

## Important Decisions
- Treat RFC 001 as design context but document current runtime behavior first. RFC fields not accepted by `internal/config` must be called out as draft/not implemented rather than shown as usable frontmatter.
- Use workspace agent paths from implementation: `<workspace>/.agh/agents/<name>/AGENT.md` and `<additional>/.agh/agents/<name>/AGENT.md`; RFC-style `.agents/<name>/AGENT.md` is not what the current resolver scans.
- Use the existing custom `<Mermaid />` component for the spawn sequence diagram because fenced Mermaid blocks are not rendered natively.

## Learnings
- Current `AGENT.md` parser supports `name`, `provider`, `command`, `model`, `tools`, `permissions`, `mcp_servers`, and `hooks`. It rejects unknown YAML/TOML frontmatter fields.
- RFC 001 also proposes `description`, `skills.*`, and `memory.*`, but these are not implemented in `internal/config/agent.go`.
- Provider resolution is `agent.provider` -> `defaults.provider`; `agent.command` overrides provider command; `agent.model` falls back to provider `default_model`.
- Built-in providers live in `internal/config/provider.go`; there is no current `internal/acp/providers.go`.
- `api_key_env` is provider metadata only in current spawn flow; the ACP subprocess inherits the daemon environment rather than having that variable loaded/injected by AGH.
- `default_model` resolves metadata but is not sent in the current ACP `session/new` or `session/load` payload.
- The task's literal build selector `turbo run build --filter=packages/site` is stale because the package name is `@agh/site`; `bunx turbo run build --filter=@agh/site` is the working site build command.

## Files / Surfaces
- Source context: `docs/rfcs/001_agent-md-with-skills-memory.md`, `internal/config/agent.go`, `internal/config/provider.go`, `internal/config/mcpjson.go`, `internal/acp/client.go`, `internal/acp/handlers.go`, `internal/session/manager_start.go`, `internal/workspace/scanner.go`.
- Docs written: `packages/site/content/runtime/agents/definitions.mdx`, `packages/site/content/runtime/agents/providers.mdx`, `packages/site/content/runtime/agents/spawning.mdx`, `packages/site/content/runtime/agents/meta.json`, `packages/site/content/runtime/meta.json`.

## Errors / Corrections
- QMD `agh-docs`/`agh-compozy` did not return useful indexed results for this repo; local `rg` over `.compozy/tasks/_archived`, `.codex/ledger`, and `.codex/plans` provided the relevant prior context.
- Existing getting-started agent docs contain stale `/runtime/getting-started/*` links and `.agents` workspace examples; Task 09 should avoid copying those paths.
- Browser QA caught escaped inline-code backticks in the definitions resolution table; corrected the MDX so `agh install` renders normally.
- Browser QA passed for `/runtime/agents/definitions/`, `/runtime/agents/providers/`, and `/runtime/agents/spawning/`; sidebar links between touched routes worked, and the spawning Mermaid diagram rendered as an SVG.
- `bunx turbo run build --filter=@agh/site` passed after the browser-found MDX fix and exported 142 static pages including `/runtime/agents/definitions` and `/runtime/agents/providers`.
- Final `make verify` still fails outside this task in `web/src/styles.test.ts`: 3 assertions expect old neutral token values while the current token CSS contains `#141312`, `#1e1c1b`, and `#2e2c2b`.

## Ready for Next Run
- Task docs are implemented and task-specific build/browser validation passed.
- Do not mark task tracking complete or create the automatic commit until the full `make verify` gate is clean.
