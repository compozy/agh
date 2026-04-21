---
status: completed
title: Capability Catalog Loader and Validation
type: backend
complexity: medium
dependencies: []
---

# Task 01: Capability Catalog Loader and Validation

## Overview

Implement the local capability catalog model and loader in `internal/config` so agent directories gain a real, validated source of truth for capabilities. This task establishes the runtime contract for single-file and directory modes, strict validation, and normalized capability data that later tasks will project into network discovery.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", and "Testing Approach"
- KEEP CAPABILITY FILE PARSING INSIDE `internal/config` - do not move local filesystem concerns into `internal/network`
- TESTS REQUIRED - this task must land with concrete unit and integration coverage, not only parser smoke checks
- GREENFIELD: nao aceitar compat layers, merge semantics implicitas, ou inferencia a partir de tools/MCP/prompt
</critical>

<requirements>
- MUST implement the normalized runtime model for `CapabilityDef`, `CapabilityCatalog`, and `CapabilityBrief` in `internal/config` or an adjacent config-owned package
- MUST support exactly one local storage mode per agent directory: `capabilities.toml`, `capabilities.json`, `capabilities/*.toml`, or `capabilities/*.json`
- MUST reject mixed-mode and mixed-format layouts with hard validation errors that identify the conflicting files
- MUST validate required fields (`id`, `summary`, `outcome`), duplicate IDs, basename-without-extension to `id` mismatches, and directory file selection rules from the TechSpec
- MUST treat missing catalogs as optional and return a deterministic "no capabilities" result without failing agent load
- SHOULD reuse the existing sidecar-loading discipline from `mcp.json` where it helps, but MUST keep capability-specific validation and shape rules separate from MCP parsing
</requirements>

## Subtasks
- [x] 1.1 Add the capability catalog structs and loader entrypoints under `internal/config`
- [x] 1.2 Implement single-file TOML/JSON parsing plus directory enumeration with explicit format selection rules
- [x] 1.3 Enforce catalog validation rules, including duplicate IDs, required fields, basename matching, and mixed-layout rejection
- [x] 1.4 Integrate optional capability loading into agent-directory loading without breaking existing `AGENT.md` and `mcp.json` behavior
- [x] 1.5 Add precise unit and integration coverage for valid layouts, invalid layouts, and optional-catalog behavior

## Implementation Details

See TechSpec "Data Models", "Projection rules", and "Testing Approach". Reuse the sidecar-loading approach from `mergeAgentMCPSidecar` and `mcpjson.go` for optional file discovery, but keep the capability catalog as its own strongly typed surface rather than as raw `json.RawMessage` blobs.

### Relevant Files
- `internal/config/agent.go` - current agent-directory load path where capability sidecars should be discovered and attached
- `internal/config/agent_test.go` - existing AGENT.md and sidecar coverage to extend with capability-aware cases
- `internal/config/mcpjson.go` - reference pattern for optional sidecar parsing and strict JSON handling
- `internal/config/mcpjson_test.go` - reference tests for optional files, strict decode, and descriptive validation errors
- `internal/config/agent_resource.go` - normalize exported `AgentDef` resource behavior if the runtime agent struct gains capability data

### Dependent Files
- `internal/session/manager_start.go` - later tasks need the loaded capability catalog during session activation
- `internal/session/interfaces.go` - the runtime/network boundary will consume the normalized catalog from this task
- `internal/network/manager.go` - downstream projection will rely on the catalog loaded here rather than re-reading agent files
- `internal/config/agent_resource_test.go` - may need updates if serialized agent resources expose capability metadata

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - establishes explicit local catalogs as the source of truth
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) - governs file-vs-directory and TOML-vs-JSON rules
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - defines required and optional capability fields

## Deliverables
- Capability catalog loader and validator under `internal/config`
- Optional capability attachment during agent-directory loading without regressing `AGENT.md` and `mcp.json`
- Descriptive validation errors for unsupported layouts and malformed capability definitions
- Updated `internal/config` unit tests covering valid and invalid capability layouts **(REQUIRED)**
- Integration coverage proving real agent directories load capability catalogs correctly **(REQUIRED)**
- Test coverage >=80% for the touched `internal/config` package(s) **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Loading `capabilities.toml` with multiple valid entries returns a normalized catalog and preserves optional fields such as `context_needed` and `execution_outline`
  - [x] Loading `capabilities.json` accepts the same schema, rejects unknown JSON fields, and rejects trailing JSON data
  - [x] Directory mode loads only regular files of the selected format and ignores dotfiles plus files with other extensions
  - [x] `capabilities.toml` plus `capabilities/` returns a hard validation error instead of merging
  - [x] `capabilities.toml` plus `capabilities.json` in the same agent directory returns a hard validation error
  - [x] Mixed `.toml` and `.json` files under `capabilities/` returns a hard validation error
  - [x] Duplicate capability IDs across directory entries return an error that identifies the normalized duplicated ID
  - [x] Missing `id`, missing `summary`, and missing `outcome` each return field-specific validation errors
  - [x] Directory entries whose basename without extension does not match `id` return a validation error
  - [x] Missing capability catalog returns a deterministic no-catalog result instead of an error
- Integration tests:
  - [x] `LoadAgentDefFile()` on a real agent directory loads `AGENT.md`, merges `mcp.json`, and attaches a capability catalog from a sibling capability sidecar
  - [x] `LoadWorkspaceAgentDefs()` preserves root/additional/global precedence while carrying capability catalogs only for the winning agent definition
  - [x] Agents with no capability catalog still load successfully through the same workspace discovery path
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agent directories can declare capabilities through one supported local layout with deterministic validation
- The runtime exposes a normalized capability catalog without forcing later tasks to parse files again
