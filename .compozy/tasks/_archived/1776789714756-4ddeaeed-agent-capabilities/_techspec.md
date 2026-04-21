# TechSpec: Agent Capability Catalog for AGH Runtime

## Executive Summary

This TechSpec defines how AGH runtime authors, loads, validates, and projects agent capability catalogs from self-contained agent directories. A capability is an outcome-oriented, structured delegation offer that other peers can discover and invoke without needing to know the agent's internal skills, prompt, or workflow wiring.

The runtime keeps capability authoring local-first and explicit: capabilities may be stored as a single catalog file or as one file per capability inside a directory, using either TOML or JSON. The primary trade-off is explicit authoring over automatic inference. This avoids brittle derivation from tools, MCPs, or prompt text, but requires agents to declare their offerings intentionally. No `_prd.md` exists for this feature; this TechSpec is based on the approved technical design discussion and current codebase architecture.

## System Architecture

### Component Overview

- `internal/config` owns local capability catalog discovery, parsing, and validation.
- `AGENT.md` remains the prose/identity surface for an agent, but the agent directory gains an optional capability sidecar/catalog.
- The runtime projects local capabilities into two network-facing views:
  - brief capability discovery for `PeerCard`
  - rich capability catalog for explicit `whois` discovery
- `internal/network` remains transport/runtime logic only; it consumes already-normalized capability data and does not parse capability files directly.

Data flow:
- Runtime discovers an agent directory.
- Runtime loads `AGENT.md` and the optional capability catalog from the same directory.
- Runtime validates and normalizes the local capability set.
- Runtime derives:
  - `PeerCard.capabilities []string`
  - `peer_card.ext["agh.capabilities_brief"]`
  - explicit rich `whois` responses from the same normalized catalog

## Implementation Design

### Core Interfaces

```go
type CapabilityDef struct {
	ID                string
	Summary           string
	Outcome           string
	ContextNeeded     []string
	ArtifactsExpected []string
	ExecutionOutline  []string
	Constraints       []string
	Examples          []string
}
```

```go
type CapabilityCatalog struct {
	Capabilities []CapabilityDef
}

type CapabilityBrief struct {
	ID      string
	Summary string
}

type CapabilityLoader interface {
	LoadAgentCapabilities(agentDir string) (*CapabilityCatalog, error)
}
```

```go
type CapabilityProjector interface {
	CapabilityIDs(catalog *CapabilityCatalog) []string
	CapabilityBrief(catalog *CapabilityCatalog) []CapabilityBrief
	FilteredCatalog(catalog *CapabilityCatalog, ids []string) *CapabilityCatalog
}
```

Error handling conventions:
- invalid catalog shape returns a hard validation error
- duplicate capability IDs return a hard validation error
- mixed file/directory mode returns a hard validation error
- mixed TOML/JSON within the same directory mode returns a hard validation error

### Data Models

`CapabilityDef` fields:

Required:
- `id`
- `summary`
- `outcome`

Optional:
- `context_needed`
- `artifacts_expected`
- `execution_outline`
- `constraints`
- `examples`

Projection shape:
- `CapabilityBrief` contains:
  - `id`
  - `summary`

Authoring guidance:
- `summary` should be a single short sentence because it is reused in periodic `greet` discovery
- summaries should target `<= 160` UTF-8 characters in v0

Runtime storage modes:

Single-file mode:
- `agents/<name>/capabilities.toml`
- `agents/<name>/capabilities.json`

Directory mode:
- `agents/<name>/capabilities/*.toml`
- `agents/<name>/capabilities/*.json`

Validation rules:
- exactly one storage mode may be used: file or directory
- in file mode, only one of `capabilities.toml` or `capabilities.json` may exist
- in directory mode, all capability files must use the same format
- in directory mode, regular files under `capabilities/` whose extension matches the selected format are loaded as capability definitions
- dotfiles and files with other extensions are ignored
- directory entries are self-contained; no `_catalog` file is required
- `id` must be unique within the agent
- in directory mode, the filename basename excluding `.toml` or `.json` must match capability `id`
- capability IDs use simple slug form such as `create-landing-page`
- capability IDs are only required to be unique within one agent directory
- each runtime peer projects that local catalog independently; effective network disambiguation is `peer_id + capability_id`

Projection rules:
- when no capability catalog exists, the runtime emits:
  - `peer_card.capabilities = []`
  - no `peer_card.ext["agh.capabilities_brief"]`
- when rich capability discovery is explicitly requested and no local catalog exists, the runtime returns `agh.capability_catalog.capabilities = []`
- when rich capability discovery is filtered by `capability_ids` and none match, the runtime returns `agh.capability_catalog.capabilities = []`
- `peer_card.capabilities` and `agh.capabilities_brief[*].id` are derived from the same normalized catalog and must agree exactly
- v0 does not introduce pagination or chunking for rich catalogs; requesters should prefer filtered `capability_ids` lookups for large peers
- the runtime must not emit a `whois` response that exceeds the protocol envelope size limit

### API Endpoints

No new daemon API endpoints are required in this phase.

This TechSpec relies on existing network discovery surfaces:
- brief discovery through `greet`
- rich discovery through explicit `whois`

Any daemon/API surface additions should reuse the normalized runtime catalog model rather than re-parse files outside `internal/config`.

## Integration Points

No external services are added.

Internal integration points:
- `internal/config/agent.go`
- agent directory discovery under runtime/workspace resolution
- `internal/network` peer registration and `whois` response assembly

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `internal/config` | modified | Adds capability catalog discovery, load, and validation. Medium risk due to directory-mode edge cases. | Implement loader and validation tests |
| agent directory contract | modified | Agent directories gain optional capability sidecars/catalogs. Low risk because the feature is additive. | Document authoring rules |
| `internal/network` | modified | Local peer cards stop advertising empty capabilities when a catalog exists. Medium risk because discovery output changes. | Add projection and response tests |
| runtime docs | modified | Must document capability authoring and projection behavior. Low risk. | Update runtime-facing docs |

## Testing Approach

### Unit Tests

- loader tests for single-file TOML
- loader tests for single-file JSON
- loader tests for directory TOML
- loader tests for directory JSON
- validation failures:
  - file and directory both present
  - TOML and JSON mixed in directory mode
  - duplicate IDs
  - missing required fields
  - filename/id mismatch
- projection tests:
  - capability IDs
  - brief summaries
  - filtered rich catalog selection

Mocks should be minimal and limited to filesystem boundaries where necessary. Prefer real temporary directories and real files.

### Integration Tests

- runtime agent discovery should load capabilities from real agent directories under `t.TempDir()`
- local peer registration should project loaded capabilities into `PeerCard`
- explicit rich discovery should return filtered or full catalogs from normalized runtime state

Environment dependencies:
- filesystem only
- no external network dependency required for runtime loading tests

## Development Sequencing

### Build Order

1. Add normalized capability model and loader under `internal/config` — no dependencies.
2. Add validation rules for storage mode, file format, uniqueness, and filename/id matching — depends on step 1.
3. Add projection helpers for IDs, brief summaries, and filtered rich catalogs — depends on step 1.
4. Wire capability loading into agent directory and runtime discovery — depends on steps 1 and 2.
5. Wire projection into network peer registration and explicit `whois` assembly — depends on steps 3 and 4.
6. Add unit and integration coverage for loading, validation, and projection — depends on steps 1-5.
7. Update runtime-facing documentation — depends on steps 4-6.

### Technical Dependencies

- existing agent directory discovery in `internal/config`
- existing `PeerCard` and `whois` handling in `internal/network`
- RFC 003 update defining brief and rich capability discovery on the wire

## Monitoring and Observability

- structured validation errors when capability catalog load fails
- debug logs when capability catalogs are discovered and projected
- explicit warning when an agent joins the network with no capability catalog
- the no-catalog warning is runtime-local observability only; it does not generate a protocol-level signal
- no alerting changes required in this phase

## Technical Considerations

### Key Decisions

- Decision: capabilities are explicitly authored, not inferred from tools, MCPs, or prompts
  - Rationale: inference would be ambiguous and unstable
  - Trade-offs: more authoring work, better semantic clarity
  - Alternatives rejected: derive capabilities from tools or prompt text

- Decision: runtime supports both single-file and directory catalog modes
  - Rationale: small agents and large agents need different authoring ergonomics
  - Trade-offs: loader complexity increases slightly
  - Alternatives rejected: file-only, directory-only, or merge/overlay between both

- Decision: directory mode has self-contained capability files and no required `_catalog`
  - Rationale: avoid ritual manifest files without operational meaning
  - Trade-offs: shared directory metadata is unavailable for now
  - Alternatives rejected: mandatory `_catalog` manifest

- Decision: capability contracts remain soft and outcome-oriented
  - Rationale: AGH is LLM-first and non-deterministic
  - Trade-offs: weaker machine rigidity, better agentic delegation
  - Alternatives rejected: strict RPC-style required input/output schemas

### Known Risks

- Risk: authors may create vague or overlapping capabilities
  - Mitigation: require `summary` and `outcome`; document authoring guidance

- Risk: directory-mode authoring may drift without strong validation
  - Mitigation: hard validation on duplicate IDs, filename/id mismatch, and mixed formats

- Risk: runtime and wire model could diverge over time
  - Mitigation: keep one normalized runtime model and project all network views from it

## Architecture Decision Records

- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) — Capabilities are authored explicitly as agent-local catalogs and are not inferred from tools, MCPs, or prompt text.
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) — Runtime supports single-file and directory catalog modes, but never both simultaneously and never with merge semantics.
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) — Capabilities are structured delegation offers for agentic execution, not rigid RPC contracts.
