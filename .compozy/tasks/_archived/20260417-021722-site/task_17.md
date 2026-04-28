---
status: completed
title: "Docs: Configuration Reference (TOML, AGENT.md, SKILL.md, env vars)"
type: docs
complexity: medium
dependencies:
    - task_03
---

# Task 17: Docs: Configuration Reference (TOML, AGENT.md, SKILL.md, env vars)

## Overview

Write the six configuration reference pages providing complete schema documentation for all AGH configuration surfaces. This is Diataxis **Reference** type — exhaustive, accurate, and structured for lookup rather than learning. Every configuration field, environment variable, and file format is documented here with defaults, valid values, and examples. This is the section users will consult most frequently after initial setup.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Configuration"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- EVERY configuration field MUST have: name, type, default, valid values, description
- Content MUST be accurate to `internal/config/` — read the source for all config types
- Reference docs must be COMPLETE — omitting a field is a documentation bug
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/runtime/reference/config-toml.mdx` — complete `config.toml` schema: every section (`[daemon]`, `[defaults]`, `[observe]`, `[memory]`, `[network]`, `[sandboxes.*]`, etc.), every field with type, default, valid values, and description
- MUST create `packages/site/content/runtime/reference/agent-md.mdx` — complete AGENT.md frontmatter schema: all fields, types, defaults, valid values; markdown body conventions; examples for each provider
- MUST create `packages/site/content/runtime/reference/skill-md.mdx` — complete SKILL.md frontmatter schema: all fields, types, defaults; procedural instruction body conventions; MCP sidecar config
- MUST create `packages/site/content/runtime/reference/mcp-json.mdx` — mcp.json configuration format for MCP server definitions, tool mappings, sidecar lifecycle
- MUST create `packages/site/content/runtime/reference/env-vars.mdx` — all environment variables AGH reads (AGH_HOME, AGH_CONFIG, AGH_LOG_LEVEL, etc.) with precedence rules
- MUST create `packages/site/content/runtime/reference/file-locations.mdx` — all file paths AGH uses: home directory (~/.agh/), workspace directory (.agh/), config file locations, database locations, log locations, socket paths
- MUST create `packages/site/content/runtime/reference/meta.json` — sidebar ordering
- MUST read ALL config struct definitions in `internal/config/` to ensure completeness
- MUST cross-reference with RFCs 001 (AGENT.md) and 002 (SKILL.md) for schema accuracy
- MUST include a complete annotated example for config.toml, AGENT.md, and SKILL.md
- SHOULD organize config.toml documentation by TOML section
- SHOULD include a "quick reference" table at the top of each page for scanability
</requirements>

## Subtasks

- [ ] 17.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 17.2 Read `internal/config/config.go` exhaustively — enumerate ALL config structs and fields
- [ ] 17.3 Read `docs/rfcs/001_agent-md-with-skills-memory.md` for AGENT.md schema
- [ ] 17.4 Read `docs/rfcs/002_skills-system-final.md` for SKILL.md schema
- [ ] 17.5 Read `internal/config/` for environment variable handling and file path resolution
- [ ] 17.6 Write `config-toml.mdx` — complete config.toml reference
- [ ] 17.7 Write `agent-md.mdx` — complete AGENT.md reference
- [ ] 17.8 Write `skill-md.mdx` — complete SKILL.md reference
- [ ] 17.9 Write `mcp-json.mdx` — MCP configuration reference
- [ ] 17.10 Write `env-vars.mdx` — environment variables reference
- [ ] 17.11 Write `file-locations.mdx` — file paths reference
- [ ] 17.12 Create `meta.json` for sidebar ordering
- [ ] 17.13 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Configuration".

Configuration reference is the most frequently consulted section after onboarding. Every field must be documented with its type, default value, valid values, and a clear description. The pages should be structured for quick lookup — users arrive with a specific question ("what does this field do?") and need to find the answer fast. Tables and code examples are the primary content format.

### Relevant Files

- `internal/config/config.go` — Main config type definitions (EVERY struct and field)
- `internal/config/merge.go` — Config merge rules, overlay semantics
- `internal/config/validation.go` — Config validation rules, valid values
- `internal/config/paths.go` — File path resolution, home directory
- `internal/config/agent.go` — Agent-specific config (if exists)
- `docs/rfcs/001_agent-md-with-skills-memory.md` — AGENT.md format specification
- `docs/rfcs/002_skills-system-final.md` — SKILL.md format specification
- `internal/daemon/daemon.go` — Environment variable reading

### Dependent Files

- `packages/site/content/runtime/reference/` — Output directory
- All other doc tasks — config reference is cross-linked from everywhere
- Agents (task_09), Skills (task_11), Workspaces (task_12) — format details live here

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Pages live in `runtime` collection

## Deliverables

- `packages/site/content/runtime/reference/config-toml.mdx` — config.toml reference
- `packages/site/content/runtime/reference/agent-md.mdx` — AGENT.md reference
- `packages/site/content/runtime/reference/skill-md.mdx` — SKILL.md reference
- `packages/site/content/runtime/reference/mcp-json.mdx` — mcp.json reference
- `packages/site/content/runtime/reference/env-vars.mdx` — Environment variables reference
- `packages/site/content/runtime/reference/file-locations.mdx` — File locations reference
- `packages/site/content/runtime/reference/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All six pages render at their expected URLs under `/runtime/reference/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] config.toml reference covers ALL sections and fields from `internal/config/`
  - [ ] AGENT.md reference covers ALL frontmatter fields from RFC 001
  - [ ] SKILL.md reference covers ALL frontmatter fields from RFC 002
  - [ ] Every field has type, default, valid values, and description
  - [ ] Complete annotated examples included for config.toml, AGENT.md, SKILL.md
  - [ ] Environment variables are exhaustively documented
  - [ ] File locations match actual path resolution in source code
  - [ ] No broken internal links

## Success Criteria

- All six MDX pages build and render correctly
- Content follows Diataxis Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Every configuration field is documented (completeness verified against source code)
- Annotated examples are included for all major formats
- Environment variables and file paths are exhaustively documented
- Pages are structured for quick lookup (tables, code blocks, clear headings)
- Zero build warnings from Fumadocs
