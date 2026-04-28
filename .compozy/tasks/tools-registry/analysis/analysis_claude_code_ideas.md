# Local Ideas Cross-Reference: docs/ideas/from-claude-code

## Overview

The `docs/ideas/from-claude-code` folder contains prior Claude Code analyses and a filtered recommendation document. The most important point is a tension:

- Earlier filtering said Claude Code Tool Search and streaming tool execution were "not relevant" because AGH manages external ACP runtimes and does not make LLM API calls directly.
- A Tool Registry feature changes part of that conclusion. If AGH introduces daemon-owned, agent-callable tools, then search, progressive schema disclosure, permission ordering, result budgets, and tool metadata become relevant again.

The TechSpec should be explicit about this boundary. AGH should not pretend it controls every provider's internal tool loop, but it can own a registry for AGH-native tools and extension-provided tools exposed through AGH surfaces.

## Relevant Ideas

### Canonical tool contract

`analysis_tool_system.md` proposes a Go `Tool` shape with name, description, schema, permission checks, execution, classifier input, and result limits. This aligns with the competitor research, but AGH should avoid making security classifier behavior mandatory in the first registry layer.

The stronger AGH version should split:

- descriptor fields used for discovery and schema projection,
- policy metadata used for authorization,
- availability checks used for live health,
- handler/adapters used for dispatch,
- result policy used for truncation, persistence, redaction, and telemetry.

### Tool Search and deferred loading

The document proposes an `eager`, `deferred`, and `discovered` registry with `Search(query)` returning tool references. That is useful if AGH has large tool universes from MCP, extensions, skills, network peers, and built-ins.

AGH should adapt this provider-neutrally:

- `agh__tool_search` returns descriptors and optional schema handles.
- CLI/HTTP/UDS search returns the same data.
- ACP/model-specific `tool_reference` integration is optional and provider-dependent.
- The registry can still compute deltas and persist "discovered in this session" state later.

### Result persistence and budgets

The Claude Code analysis highlights per-tool max result sizes and disk persistence for large outputs. AGH needs the same concept because tool results may be delivered into session transcripts, HTTP responses, UDS clients, or agent-visible messages.

The registry should define:

- `MaxResultBytes` or a default by risk/source class,
- preview strategy,
- artifact persistence target,
- redaction path,
- telemetry fields for result bytes and persisted artifact id.

### Permission and security validators

The local idea files discuss command-specific validators, classifier input projection, dangerous pattern registries, and bash-specific semantics. Those are important for shell tools, but they should not block the Tool Registry foundation.

For MVP, the registry should provide a hook point:

- a tool can declare `RiskClass`, `OpenWorld`, `Destructive`, and `RequiresUserInteraction`;
- a policy engine can decide allow/deny/ask;
- specialized tools such as shell/url/browser can later plug in validators.

### Prompt and catalog deltas

`analysis_prompt_architecture.md` discusses enabled-tools-aware prompt sections and delta attachment patterns. The relevant AGH takeaway is that catalog changes should be incremental and explicit. For ACP runtimes that cannot accept live tool deltas, AGH should clearly fall back to "visible on next session."

### Streaming executor and concurrency

`analysis_query_engine.md` includes concurrency-safe vs exclusive tool execution. AGH should keep the metadata and enforce it at registry dispatch, but not copy a direct model streaming executor unless AGH owns a provider's query loop.

### Plugin system references

`analysis_services_infra.md` describes plugin refresh, availability, hooks, and plugin error taxonomy. This supports an AGH registry model where extension tools have lifecycle state and refresh reasons. AGH should convert extension sidecar and MCP health into availability reasons rather than exposing raw plugin errors to agents.

## Filtered Recommendation Reversal

`filtered_recommendations.md` says Tool Search, streaming execution, bash classifiers, and API tool loops are not relevant because AGH does not make LLM API calls. That remains true for driver-internal tools.

However, the Tool Registry feature is not about controlling Claude Code's own tools. It is about creating AGH-owned tools that are:

- discoverable through AGH,
- governed by AGH,
- executable through AGH,
- extensible by AGH extensions,
- visible to agents regardless of ACP runtime when an adapter exists.

Therefore:

- Do not copy Claude Code's provider API request mechanics into the MVP.
- Do copy the registry/search/permission/result architecture where AGH owns the tool surface.

## Relevant Code / Document Paths

- `docs/ideas/from-claude-code/analysis_tool_system.md:450-620`: deferred registry, security validator pipeline, classifier input projection, dangerous pattern registry, and key file references.
- `docs/ideas/from-claude-code/filtered_recommendations.md:1-38`: architectural warning that AGH is an orchestration kernel, not a direct LLM API loop.
- `docs/ideas/from-claude-code/analysis_prompt_architecture.md`: prompt sections and tool-aware catalog deltas.
- `docs/ideas/from-claude-code/analysis_query_engine.md`: tool execution concurrency and exclusive scheduling references.
- `docs/ideas/from-claude-code/analysis_services_infra.md`: plugin refresh and availability ideas.

## Transferable Patterns

1. Build a provider-neutral registry search API first.
2. Add deferred schema loading as an AGH catalog behavior, not as an Anthropic-only assumption.
3. Track per-tool result budgets and persisted artifacts.
4. Keep security validators pluggable by tool family.
5. Model concurrency metadata even before advanced scheduling.
6. Treat live catalog deltas as optional driver capabilities with a fallback.

## Risks / Mismatches

AGH should not duplicate the ACP runtime's internal shell/browser/file tools unless there is a clear cross-runtime AGH-owned reason.

AGH should not depend on Claude Code-only `tool_reference` wire formats for the core registry contract.

AGH should not make shell command classifiers part of the foundation unless the MVP includes an AGH-owned shell tool.

AGH should not over-inject tool catalogs into prompts. Progressive disclosure and search should be preferred once the tool universe grows.

## Open Questions

1. Should AGH's first registry search surface be `agh__tool_search` as an agent-callable tool, `agh tool search` as CLI, or both?
2. Should catalog delta support be designed now even if only a subset of drivers implement it?
3. Should result persistence share AGH's session event/artifact store or get a dedicated tool-result artifact store?

## Evidence

- `docs/ideas/from-claude-code/analysis_tool_system.md:450-620`: local implementation sketches for deferred registry, validators, result storage, and key Claude Code paths.
- `docs/ideas/from-claude-code/filtered_recommendations.md:1-38`: explicit "AGH is not the LLM API loop" constraint.
- `docs/ideas/from-claude-code/filtered_recommendations.md:96-134`: skills activation and prompt assembly ideas that intersect with registry progressive disclosure.
