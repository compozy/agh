---
status: completed
title: Replace Recipe Wire Kind with Capability Envelopes
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Replace Recipe Wire Kind with Capability Envelopes

## Overview

Replace `kind:"recipe"` at the protocol layer with `kind:"capability"` so the wire model matches the new unified concept. This task owns the envelope types, decode/encode registry changes, payload validation rules, and digest verification required for capability transfer to become a first-class network behavior.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "Core Interfaces", "Data Models", "Testing Approach", and "Build Order"
- REMOVE `recipe` AS A PROTOCOL KIND - do not leave shadow registration, fallback parsing, or dual validation paths behind
- KEEP DIGEST VERIFICATION RULES COHERENT WITH TASK_01 - the wire payload must be derived from the canonical runtime capability model
- TESTS REQUIRED - envelope, validation, and integration coverage must prove the new kind end to end
- GREENFIELD: protocol clarity wins over compatibility shims or alias decoding
</critical>

<requirements>
- MUST replace `KindRecipe` and recipe-specific body structs with unified capability envelope types in `internal/network`
- MUST validate `kind:"capability"` payloads against the unified capability shape, including required metadata and digest integrity
- MUST reject malformed, incomplete, or digest-mismatched transferred capabilities with hard protocol validation errors
- MUST update envelope decoding, summary extraction, and helper text so the runtime no longer treats `recipe` as a supported kind
- MUST preserve the `greet` and `whois` discovery split while introducing `kind:"capability"` as the dedicated transfer artifact
- SHOULD keep the transferred capability payload narrow enough to avoid duplicating unrelated peer metadata in the envelope body
</requirements>

## Subtasks
- [x] 2.1 Replace recipe-specific envelope/body types with unified capability transfer types
- [x] 2.2 Update decode and encode registration so `kind:"capability"` becomes the only artifact-transfer kind
- [x] 2.3 Rewrite network validation rules for transferred capabilities, including digest verification behavior
- [x] 2.4 Update helper text, summaries, and any router-visible metadata derived from envelope kinds
- [x] 2.5 Add unit and integration coverage for valid and invalid capability envelopes

## Implementation Details

See TechSpec "Core Interfaces", "Data Models", and "Build Order" items 3-4. This task should leave one explicit network artifact kind for transferred capabilities while discovery remains where it already belongs: brief in `greet`, rich in `whois`, transfer in `kind:"capability"`.

### Relevant Files
- `internal/network/envelope.go` - primary home for message kinds, body structs, and envelope registry updates
- `internal/network/validate.go` - protocol validation rules that currently distinguish artifact kinds and payload shape
- `internal/network/validate_test.go` - precise validation cases for malformed envelopes, missing fields, and digest failures
- `internal/network/envelope_integration_test.go` - end-to-end encode/decode coverage for supported wire kinds
- `internal/network/helpers_test.go` - summary extraction and helper behavior coverage around envelope bodies

### Dependent Files
- `internal/network/router.go` - router dispatch will need the new kind and payload helpers from this task
- `internal/network/lifecycle.go` - interaction creation and terminal-state handling will consume the renamed kind
- `internal/network/delivery.go` - transfer delivery summaries and bookkeeping will depend on the new envelope type
- `internal/network/router_test.go` - router-level regressions will shift from recipe handling to capability handling
- `docs/rfcs/003_agh-network-v0.md` - documentation updates later depend on the final wire shape created here

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - defines the removal of recipe as a separate wire/runtime concept
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - defines the new transfer kind and its operational role

## Deliverables
- `kind:"capability"` envelope and payload structs replacing recipe-specific wire bodies
- Updated decode/encode registry and protocol validation logic for unified capability transfer
- Regression coverage for malformed envelopes, digest mismatches, and supported transfer payloads **(REQUIRED)**
- Integration tests proving the runtime can decode and validate capability transfers without recipe remnants **(REQUIRED)**
- Test coverage >=80% for the touched network packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Decoding a valid `kind:"capability"` envelope returns the expected payload shape and summary metadata
  - [x] Missing capability metadata such as `id`, `summary`, `outcome`, or `digest` yields a descriptive validation failure
  - [x] A transferred capability whose digest does not match the canonicalized payload is rejected
  - [x] Unknown `kind:"recipe"` envelopes are rejected once the new registry is in place
  - [x] Capability envelope helpers no longer emit recipe-specific labels or summaries
- Integration tests:
  - [x] Encode/decode round-trips for `kind:"capability"` succeed across the in-process network envelope path
  - [x] Invalid capability transfers fail before router delivery or lifecycle creation
  - [x] Discovery flows continue using `greet` and `whois` rather than the transfer body path
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `recipe` no longer exists as a supported network artifact kind
- The protocol exposes one explicit transferred artifact type: `kind:"capability"`
