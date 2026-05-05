# TC-FUNC-057 — Site docs use canonical `ToolID` and never reference dotted IDs / descriptor-only callable extensions

- **Priority:** P1
- **Type:** Documentation / docs grammar
- **Trace:** Task 14, ADR-007, ADR-008, TechSpec Delete Targets

## Objective

Prove `packages/site/content/runtime/**` content uses canonical `ToolID` examples (with `__`) and never references:

- dotted IDs (`agh.skill.view`),
- descriptor-only callable extensions,
- legacy `tool_name`/`tool_namespace` matchers in registry-owned hooks,
- silent `sse → http` rewrites,
- `["*"]` agent tools default,
- `MemoryTokenStore`/library OAuth helpers as authority.

## Test Steps

1. `grep -REn 'agh\.[a-z_]+\.[a-z_]+' packages/site/content/runtime/`.
   - **Expected:** Zero matches (or only matches that explicitly mark the form as forbidden).
2. `grep -REn 'descriptor-only.*callable' packages/site/content/runtime/`.
   - **Expected:** Zero matches outside historical/deprecated callouts.
3. Confirm config docs list defaults and validation behavior for every `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]` key.
4. Extension docs include both TypeScript `extension.tool(...)` and Go SDK `aghsdk.Tool[T]` authoring examples.
5. MCP docs distinguish external MCP backend call-through from hosted AGH MCP exposure.

## Automation

- **Target:** Manual + grep checks
- **Status:** Manual
- **Command/Spec:** Doc grep checks during Task 16; reviewer signs off on copy.
