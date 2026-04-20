# Architectural Analysis Report

**Date**: 2026-04-18
**Scope**: Agent-declared capabilities across AGENT.md, config parsing, and AGH Network peer advertising

---

## Executive Summary

There is a real architectural gap between the network protocol, the agent-definition RFC, and the current runtime:

- `docs/rfcs/003_agh-network-v0.md` requires `Peer Card.capabilities` and treats them as the discovery surface for `greet`/`whois`.
- `docs/rfcs/001_agent-md-with-skills-memory.md` says an `AGENT.md` definition could generate an Agent Card / discovery metadata, but it never defines the field shape for capabilities.
- `internal/config/agent.go` has no capabilities field in `AgentDef`.
- `internal/network` currently creates local peer cards with `Capabilities: []string{}` by default.

The result is not just a documentation omission. It is a broken contract chain:

1. the protocol expects advertised capabilities
2. the authoring format cannot declare them
3. the runtime has no canonical source to project into `PeerCard`

This is a root-cause design issue, not a small config bug.

---

## Findings

### 1. Protocol requires capability advertisement, but the runtime has no source of truth

**Evidence**

- `docs/rfcs/003_agh-network-v0.md:277-285` defines `Peer Card.capabilities` as a required field.
- `docs/rfcs/003_agh-network-v0.md:456-475` makes `greet` and `whois` the capability-advertising and lookup path.
- `internal/network/peer.go:102-107` builds the default local card with empty `Capabilities`.
- `internal/network/manager.go:416-420` joins a local peer using only `DefaultPeerCard(request.peerID)`.

**Impact**

- Local peers can join the network without any declared capabilities.
- Discovery exists structurally, but carries no useful agent-specific capability data.
- Adding capability strings elsewhere later would create drift unless the projection path is defined.

**Severity**: HIGH

### 2. AGENT.md RFC implies derivation to discovery metadata, but never specifies the mapping

**Evidence**

- `docs/rfcs/001_agent-md-with-skills-memory.md:316-318` says Agent Cards could be generated from `AGENT.md`.
- `docs/rfcs/001_agent-md-with-skills-memory.md:439` explicitly raises the open question of whether provider-specific settings should evolve into a capabilities model.

**Impact**

- The intended architecture already points to `AGENT.md -> discovery card`, but the spec stops before defining how.
- That missing mapping is the reason `internal/config/agent.go` never got a capabilities shape.

**Severity**: HIGH

### 3. Agent definition parsing is shape-closed today

**Evidence**

- `internal/config/agent.go:17-26` and `internal/config/agent.go:29-37` define the full parsed/frontmatter shape.
- `internal/config/agent.go:210-218` copies only existing fields into `AgentDef`.
- Current fields cover provider/model/tools/permissions/mcp_servers/hooks/prompt only.

**Impact**

- There is no extension slot for network-facing capability metadata.
- Any capability proposal affects parsing, validation, cloning, resource codecs, and API payloads.

**Severity**: HIGH

### 4. Capability semantics are advisory in the protocol, but runtime capabilities elsewhere are often enforcement-oriented

**Evidence**

- `docs/rfcs/003_agh-network-v0.md:301-309` says capability strings are opaque and implementation-defined.
- `docs/rfcs/003_agh-network-v0.md:840` says advertised capabilities are advisory until behavior is verified.
- Extension/runtime capability systems in `internal/extension/capability.go` are enforcement-oriented and security-sensitive.

**Impact**

- Reusing one undifferentiated `capabilities` field across all AGH subsystems would mix two different semantics:
  - advisory discovery claims for the network
  - enforced authorization grants for runtime/host APIs
- That would be a modeling mistake and likely create future confusion.

**Severity**: HIGH

### 5. Existing cloned/serialized agent surfaces would drift if the field is added informally

**Evidence**

- `internal/workspace/clone.go:145-163` deep-copies `AgentDef` fields explicitly.
- `internal/config/agent_resource.go:25-44` normalizes `AgentDef` for resource sync.
- `internal/api/core/conversions.go:118-145` serializes `AgentDef` into API payloads.
- `internal/api/contract/contract.go:88-96` exposes an API `AgentPayload` that currently omits capability metadata.

**Impact**

- Any new field must be propagated deliberately across config, clone, resources, and API.
- Otherwise different runtime surfaces will disagree on the same agent.

**Severity**: MEDIUM

---

## Architectural Risks

### Risk 1: Single flat `capabilities: []string` becomes overloaded

If one flat list is used to mean:

- network discovery claims
- tool/runtime authorization
- provider feature support
- hooks/resources availability

then AGH will accumulate one ambiguous field with incompatible semantics.

### Risk 2: Capabilities become config-only decoration

If the field is added to `AGENT.md` without defining:

- normalization rules
- validation rules
- projection into `PeerCard`
- projection or exclusion from API/resource surfaces

then the system will still not solve the actual integration gap.

### Risk 3: Provider-specific behavior gets hidden inside opaque capability strings

If provider/runtime launch features like `permissions`, ACP modes, or model support are moved wholesale into generic capability strings, the spec may lose clarity rather than gain portability.

---

## Recommended Direction

### Recommendation 1: Separate declared agent capabilities from enforced runtime grants

Use agent-declared capabilities as **authoring/discovery metadata**, not as the security boundary.

That keeps alignment with RFC 003, which treats peer capabilities as advisory.

### Recommendation 2: Define a projection contract, not just a frontmatter field

The spec should answer:

1. Where capabilities are declared in `AGENT.md`
2. How they are normalized and validated
3. Which subset is projected into `network.PeerCard`
4. Which parts, if any, appear in API/resource payloads
5. Which runtime layer owns the derivation

### Recommendation 3: Prefer a structured capability block over a single overloaded list

The analysis suggests a structure closer to:

```yaml
capabilities:
  declare:
    - workspace.patch.apply
    - artifact.recipe.consume
  network:
    advertise:
      - workspace.patch.apply
      - artifact.recipe.consume
```

or an equivalent minimal variant with an explicit projection rule.

This keeps room for future non-network capability metadata without collapsing everything into one ambiguous flat list.

---

## Conclusion

The gap is architectural and spans three layers:

- **RFC 003** already requires capability signaling.
- **RFC 001** already hints that AGENT.md should generate discovery metadata.
- **Current code** has no field and no projection path, so peer cards default to empty capabilities.

The correct fix is to define a capability declaration model in the agent spec together with a daemon-owned derivation path into `PeerCard`. The wrong fix would be to add an unscoped `[]string` field with no semantics beyond "maybe used later".
