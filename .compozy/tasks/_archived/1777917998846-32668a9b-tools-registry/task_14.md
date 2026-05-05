---
status: completed
title: Site Documentation and Generated References
type: docs
complexity: high
dependencies:
    - task_13
---

# Task 14: Site Documentation and Generated References

## Overview

Ship the documentation and generated references required for operators, agents, and extension authors to use the Tool Registry correctly. This task updates Fumadocs content, CLI reference output, API reference links, config documentation, extension authoring docs, and MCP/hosted MCP threat-model guidance.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, `packages/site/CLAUDE.md`, and completed tasks 01-13 before writing docs
- DO NOT document speculative controls, hidden flags, or behavior not backed by merged daemon/web contracts
- DO NOT hand-author generated CLI reference content; regenerate it from Cobra output
- TESTS REQUIRED: site source generation, typecheck, build, and link/reference checks must pass
</critical>

<requirements>
1. MUST document the Tool Registry model, canonical ToolID, backend kinds, operator/session visibility, policy gates, and result redaction.
2. MUST document `config.toml` keys for `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]`, agent `tools`, agent `toolsets`, and agent `deny_tools`.
3. MUST document TypeScript `extension.tool(...)`, public Go extension SDK authoring, manifest-authoritative descriptors, and runtime reconciliation.
4. MUST document external MCP call-through, existing `agh mcp auth` management, redacted auth diagnostics, and hosted AGH MCP session exposure.
5. MUST regenerate CLI docs for `agh tool`, `agh toolsets`, and related command changes.
6. MUST update API/reference docs or generated references for new HTTP/UDS surfaces.
</requirements>

## Subtasks
- [ ] 14.1 Update registry, policy, ToolID, and visibility documentation
- [ ] 14.2 Update config docs and examples for tool policy and hosted MCP
- [ ] 14.3 Update extension authoring docs for TypeScript and Go executable tools
- [ ] 14.4 Update MCP docs for call-through, auth diagnostics, hosted MCP, and approval bridge behavior
- [ ] 14.5 Regenerate CLI reference and API references
- [ ] 14.6 Add docs tests/build verification and remove any obsolete descriptor-only wording

## Implementation Details

Use TechSpec "Impact Analysis", "Config Lifecycle", "Extensibility Plan", and ADRs 001-010. Docs must distinguish cold manifest resources from executable extension-host tools to prevent repeating the rejected descriptor-only design.

### Relevant Files
- `packages/site/content/runtime/core/extensions/develop.mdx` - extension authoring docs
- `packages/site/content/runtime/core/configuration/config-toml.mdx` - tools config keys and examples
- `packages/site/content/runtime/core/configuration/mcp-json.mdx` - MCP config/auth behavior
- `packages/site/content/runtime/core/sessions/permissions.mdx` - approval bridge and hosted MCP behavior
- `packages/site/content/runtime/api-reference/index.mdx` - API reference entry point
- `packages/site/content/runtime/cli-reference/**` - generated CLI docs

### Dependent Files
- `internal/cli/**` - source for generated CLI docs
- `openapi/agh.json` - source for API reference docs
- `web/src/systems/tools/**` - source of truth for any UI screenshots or operator-surface descriptions
- `.compozy/tasks/tools-registry/adrs/**` - decision evidence to cite

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - docs must explain native vs extension-host execution
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - docs must explain hosted MCP session exposure
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - docs must explain approval and policy layering
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - docs must explain reconciliation
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - docs must explain Go SDK authoring
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - docs must explain external MCP call-through

### Web/Docs Impact
- `web/`: no code changes unless docs need screenshots or examples from the actual task_13 UI; checked `web/src/systems/tools/**`.
- `packages/site`: updates runtime core docs, config docs, MCP docs, sessions/permissions docs, API reference, generated CLI reference, and navigation metadata as needed.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: documents extension manifests, TypeScript SDK, Go SDK, MCP backend tools, hosted MCP, hooks, and tool resources.
- Agent manageability: documents CLI, HTTP, UDS, structured output, session projections, deterministic errors, and approval behavior.
- Config lifecycle: documents all new/changed tools config keys, defaults, examples, validation semantics, and removed descriptor-only assumptions.

## Deliverables
- Updated Fumadocs pages for Tool Registry runtime, policy, extensions, MCP, sessions, and configuration
- Regenerated CLI reference pages for new commands
- Updated API reference entry points for tool endpoints
- Documentation tests and build verification
- Unit/documentation checks with 80%+ relevant coverage where applicable **(REQUIRED)**
- Integration/site build tests **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Docs mention canonical `ToolID` and do not mention dotted aliases or descriptor-only callable extensions
  - [ ] Config docs list defaults and validation behavior for every new tools key
  - [ ] Extension docs include both TypeScript and Go function-based tool authoring
  - [ ] MCP docs distinguish external MCP backend call-through from hosted AGH MCP exposure
- Integration tests:
  - [ ] `make cli-docs` regenerates command pages with `agh tool` and `agh toolsets`
  - [ ] `cd packages/site && bun run source:generate` passes
  - [ ] `cd packages/site && bun run typecheck` passes
  - [ ] `cd packages/site && bun run build` passes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Operators, agents, and extension authors can follow docs without relying on internal code knowledge
- Generated CLI/API references match the implemented daemon surfaces
