# TechSpec: Unified Capabilities for AGH Runtime and Network

## Executive Summary

This TechSpec defines the replacement of the current `capabilities + recipes` split with a single concept: `capability`. A capability becomes the only authored delegation artifact, the only rich discovery artifact, and the only transferable procedural artifact on the wire. The current `recipe` protocol kind and runtime model are removed.

The implementation keeps the strongest parts of the current branch unchanged: local-first capability authoring in `capabilities.toml`, `capabilities.json`, `capabilities/*.toml`, or `capabilities/*.json`; brief capability discovery through `greet`; rich capability discovery through explicit `whois`. The main technical trade-off is preserving a structured, field-based capability model instead of moving to a body-centric artifact format. This keeps authoring simple and semantically explicit, but requires canonicalization rules so the runtime can compute stable digests for transferable capabilities. No `_prd.md` exists for this feature; this TechSpec is based on the current branch architecture, existing `agent-capabilities` artifacts, and the approved design decisions for this follow-up.

## System Architecture

### Component Overview

`internal/config` remains the source of truth for local capability authoring, discovery, parsing, normalization, and validation. It continues to load capability catalogs from the agent directory and now owns the unified capability schema.

`internal/session` remains the boundary that projects config-owned capabilities into runtime-owned network capability values. It must not parse local files or depend on network wire rules.

`internal/network` remains responsible for three distinct concerns built from the same normalized capability model:
- brief capability discovery in `greet`
- rich capability discovery in explicit `whois`
- transferable capability envelopes through `kind: "capability"`

`docs/rfcs/003_agh-network-v0.md` must be rewritten so the protocol no longer speaks about `recipe` as a first-class artifact. `docs/rfcs/005_capability-catalogs-agent-directories.md` remains the runtime-facing authoring guide, but it must be updated to document the unified schema and transfer semantics.

Data flow:
- Runtime loads `AGENT.md` and optional capability catalogs from the agent directory.
- Runtime normalizes one canonical `CapabilityCatalog`.
- Session start projects that catalog into runtime-owned `NetworkPeerCapability` values.
- Network join uses that projection to derive `PeerCard.capabilities`, `agh.capabilities_brief`, and the local rich capability catalog state.
- Peers may explicitly exchange a full capability artifact through `kind: "capability"` without introducing a second concept.

## Implementation Design

### Core Interfaces

```go
type CapabilityDef struct {
	ID                string
	Summary           string
	Outcome           string
	Version           string
	ContextNeeded     []string
	ArtifactsExpected []string
	ExecutionOutline  []string
	Constraints       []string
	Examples          []string
	Requirements      []string
}
```

```go
type CapabilityProjector interface {
	CapabilityIDs(catalog *CapabilityCatalog) []string
	CapabilityBrief(catalog *CapabilityCatalog) []CapabilityBrief
	FilterCatalog(catalog *CapabilityCatalog, ids []string) *CapabilityCatalog
	CanonicalDigest(cap CapabilityDef) (string, error)
}
```

```go
type CapabilityBody struct {
	Capability CapabilityEnvelopePayload `json:"capability"`
}

type CapabilityEnvelopePayload struct {
	ID           string   `json:"id"`
	Version      string   `json:"version,omitempty"`
	Digest       string   `json:"digest"`
	Summary      string   `json:"summary"`
	Outcome      string   `json:"outcome"`
	Requirements []string `json:"requirements,omitempty"`
}
```

Error handling conventions:
- invalid capability catalogs remain hard validation failures in `internal/config`
- duplicate capability IDs remain hard validation failures
- malformed `kind:"capability"` envelopes remain protocol validation failures in `internal/network`
- digest mismatches between canonicalized runtime content and transmitted payload are hard verification failures for the received envelope

### Data Models

The unified capability model keeps the current field-based authored shape and extends it minimally.

Required authored fields:
- `id`
- `summary`
- `outcome`

Optional authored fields:
- `version`
- `context_needed`
- `artifacts_expected`
- `execution_outline`
- `constraints`
- `examples`
- `requirements`

Derived runtime fields:
- `digest`

Key rules:
- `digest` is not authored; the runtime computes it from a canonical capability representation
- `version` is optional and authored when the capability needs explicit evolution semantics
- `requirements` references other `capability.id` values
- the canonical capability document is the structured object itself; there is no primary `inline`, `uri`, or `content_type` body surface

Filesystem layouts remain unchanged:

Single-file mode:
- `agents/<name>/capabilities.toml`
- `agents/<name>/capabilities.json`

Directory mode:
- `agents/<name>/capabilities/*.toml`
- `agents/<name>/capabilities/*.json`

Validation rules remain, with unified-schema extensions:
- exactly one storage mode may be used
- exactly one serialization format may be used inside one directory-mode catalog
- directory filename basenames must match `id`
- `id` must remain unique within one agent catalog
- `requirements` entries must be normalized and unique
- local validation does not require every `requirements` target to exist in the same local catalog unless the feature explicitly models local composition that way
- runtime and protocol docs must define whether unresolved `requirements` are allowed as remote dependencies or rejected locally

Network projections stay split by purpose:
- `greet` carries `peer_card.capabilities` plus optional `agh.capabilities_brief`
- `whois` returns rich capability data only when explicitly requested
- `kind:"capability"` carries a transferable full capability artifact and opens interactions like current `recipe`

### API Endpoints

No new daemon HTTP or UDS endpoints are required in this phase.

This change is implemented inside the existing runtime and network surfaces:
- agent-directory capability loading in `internal/config`
- session-to-network capability projection in `internal/session`
- `greet`, `whois`, and transferable artifact routing in `internal/network`

If daemon APIs later expose unified capabilities directly, they must reuse the normalized runtime capability model instead of re-parsing agent files.

## Integration Points

No external services are introduced.

Internal integration points:
- `internal/config/capabilities.go`
- `internal/config/agent.go`
- `internal/session/network_peer.go`
- `internal/network/envelope.go`
- `internal/network/validate.go`
- `internal/network/router.go`
- `internal/network/lifecycle.go`
- `docs/rfcs/003_agh-network-v0.md`
- `docs/rfcs/005_capability-catalogs-agent-directories.md`

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `internal/config` | modified | Extends capability schema with unified fields and canonical digest rules. Medium risk due to normalization and backward design changes. | Update schema, validation, and loader tests |
| `internal/session` | modified | Continues projecting capabilities, but must include unified fields required by network transfer and discovery. Low risk. | Extend runtime projection structs and tests |
| `internal/network/envelope` | modified | Replaces `KindRecipe` and `RecipeBody` with `KindCapability` and capability envelope payloads. High risk because it changes the protocol surface. | Rewrite kinds, body types, and decode paths |
| `internal/network/validate` | modified | Removes recipe validation and adds capability envelope validation and digest verification rules. High risk. | Replace validation logic and regression tests |
| `internal/network/router` | modified | Keeps current recipe operational behavior under the new capability kind. High risk because routing and lifecycle depend on the kind model. | Update dispatch, delivery summaries, and tests |
| `internal/network/lifecycle` | modified | `capability` must inherit current recipe interaction semantics. Medium risk. | Replace recipe references and preserve lifecycle invariants |
| RFC and runtime docs | modified | Protocol and authoring docs must stop describing a split model. Medium risk due to conceptual clarity. | Rewrite RFC 003 sections and capability guide |
| `.compozy/tasks/agent-capabilities` artifacts | unchanged | Historical record of the previous design. Low risk. | Leave in place as prior branch context |
| `.compozy/tasks/unified-capabilities` | new | New authoritative follow-up TechSpec and ADR set. Low risk. | Save approved TechSpec and continue with task breakdown |

## Testing Approach

### Unit Tests

Test `internal/config` for:
- schema validation with `version` and `requirements`
- canonical digest stability across equivalent normalized input
- digest changes when meaningful capability content changes
- unchanged support for file mode and directory mode
- unchanged failure behavior for mixed layout, mixed format, duplicate IDs, and filename mismatch

Test `internal/session` for:
- correct projection of unified capability fields from `CapabilityCatalog` into `NetworkPeerCapability`
- preserved cloning semantics so network does not share mutable config state

Test `internal/network` for:
- envelope decoding and validation of `kind:"capability"`
- replacement of `recipe` summary extraction and helper text
- rich discovery still deriving from the same normalized capability source as brief discovery
- `requirements` filtering and projection behavior
- digest mismatch rejection for transferred capabilities if digest verification is enforced at receive time

### Integration Tests

Test `internal/network` integration for:
- local peer join still publishing brief capability discovery in initial and reconnect greets
- explicit `whois` still returning rich capability catalogs through envelope `ext`
- `kind:"capability"` broadcast delivery across peers
- `kind:"capability"` directed delivery and interaction opening
- lifecycle behavior for capability interactions after terminal states
- interoperability between `direct` and `capability` in the same interaction flow if the protocol permits it

Test runtime integration for:
- agent-directory capability catalogs loading into the unified model without network parsing local files
- local discovery and transferable capability envelopes using the same canonical capability content
- envelope size checks still applying to rich discovery and transferred capability payloads

Environment dependencies:
- filesystem only for config/session tests
- local in-process network transport for network integration tests
- no external services

## Development Sequencing

### Build Order

1. Define the unified capability schema and canonicalization rules in `internal/config` and supporting tests. This step has no dependencies.
2. Extend session-owned runtime capability projection to carry every unified field needed by discovery and transfer. This step depends on step 1.
3. Replace `KindRecipe` with `KindCapability` in `internal/network/envelope.go` and update body types and decode registration. This step depends on steps 1 and 2.
4. Replace recipe validation with capability envelope validation and digest verification in `internal/network/validate.go`. This step depends on step 3.
5. Update router dispatch, lifecycle helpers, delivery summaries, and peer registry handling so transferred capabilities preserve current recipe behavior. This step depends on steps 3 and 4.
6. Rewrite brief and rich discovery code only where the unified schema changed, while preserving the existing `greet` and `whois` split. This step depends on steps 1, 2, and 5.
7. Rewrite RFC 003 and runtime capability docs to remove the split model and describe the unified semantics. This step depends on steps 3 through 6.
8. Add and run full unit and integration coverage across config, session, and network. This step depends on steps 1 through 7.

### Technical Dependencies

Blocking dependencies:
- a precise canonicalization algorithm for digest computation
- an explicit rule for whether unresolved `requirements` are valid remote references or local validation errors
- final RFC wording for the replacement of `recipe` examples and terminology

## Monitoring and Observability

Operational visibility should remain simple and local-first:
- log capability catalog load failures with agent directory path and validation reason
- log capability digest computation failures with capability ID and source file path
- log invalid received `kind:"capability"` envelopes with reason code and sender peer
- log digest verification failures with `peer_id`, `capability.id`, and transmitted digest
- keep existing discovery-related observability for greet/whois projection paths

No new alerting system is required in this phase.

## Technical Considerations

### Key Decisions

- Decision: replace the `capabilities + recipes` split with a single `capability` concept
  - Rationale: the current split creates overlapping author and protocol semantics
  - Trade-offs: requires a real RFC/runtime rewrite rather than a local cleanup
  - Alternatives rejected: preserve both concepts; introduce a third replacement name

- Decision: keep the current local capability catalog layouts
  - Rationale: local authoring is already good and aligned with AGH agent directories
  - Trade-offs: the unified model must fit into the current schema discipline
  - Alternatives rejected: new artifact-centric authoring format; separate local and wire shapes

- Decision: keep a structured field-based capability model as canonical content
  - Rationale: the current capability shape is semantically explicit and avoids burying behavior in an opaque body
  - Trade-offs: canonical digest rules become more important
  - Alternatives rejected: body-centric `inline|uri|content_type`; hybrid dual-center model

- Decision: compute `digest` in the runtime and keep `version` optional and authored
  - Rationale: digest stability belongs to canonicalization logic, not author discipline
  - Trade-offs: canonicalization must be specified and tested carefully
  - Alternatives rejected: authoring digest directly; removing version entirely

- Decision: replace `kind:"recipe"` with `kind:"capability"` and preserve current recipe interaction semantics
  - Rationale: the protocol needs a dedicated transferable capability artifact without keeping a second concept
  - Trade-offs: wire-level rename plus router/lifecycle rewrites
  - Alternatives rejected: tunnel transfer through `direct`; keep the `recipe` wire name

### Known Risks

- Risk: capability documents become too broad and start collecting unrelated semantics
  - Mitigation: keep the model structured and outcome-oriented; add fields only when they serve discovery, transfer, or composition directly

- Risk: digest instability from inconsistent canonicalization
  - Mitigation: define one normalization algorithm and cover it with focused deterministic tests

- Risk: confusion between passive discovery metadata and transferable capability envelopes
  - Mitigation: keep the protocol distinction explicit: `greet` and `whois` discover capabilities, `kind:"capability"` transfers one

- Risk: unresolved `requirements` semantics create inconsistent validation between local and remote cases
  - Mitigation: define one clear rule in the final spec and encode it in both config validation and network tests

## Architecture Decision Records

- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) — Replaces the `capabilities + recipes` split with one unified `capability` concept.
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) — Preserves current TOML/JSON catalog ergonomics and makes the structured capability object the canonical digest source.
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) — Swaps `recipe` for `capability` on the wire without losing current transfer and lifecycle behavior.
