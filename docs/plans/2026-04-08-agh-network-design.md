# Design: AGH Network v1

**Date:** 2026-04-08
**Status:** Approved
**Authors:** Pedro Nauck + Codex

## Summary

`AGH Network` is the approved direction for the project's agent network protocol. It replaces the earlier `AGORA` naming and formalizes a layered design that other projects can implement without depending on AGH, while still preserving AGH's operational advantage through the reference runtime, SDKs, observability, and ergonomics.

The approved design is intentionally opinionated, but not captive:

- the semantic core is transport-agnostic
- `NATS` is the first and normative transport profile
- verified interoperability is defined by a normative baseline trust profile
- AGH remains the strongest implementation, not the mandatory control plane

## Problem

The earlier `network` drafts established a strong conceptual direction, but they mixed several distinct concerns inside the same body of text:

1. Protocol semantics
2. NATS transport mapping
3. Artifact definitions such as `recipe` and `trace`
4. SDK and runtime ergonomics
5. Product-specific AGH advantages

That was useful for exploration, but not sufficient for a protocol RFC intended to be implemented by third parties.

At the same time, the protocol should not be flattened into a generic commodity spec. The product strategy is to keep AGH materially better through:

- the best Go SDK and NATS integration
- stronger runtime defaults
- richer observability and replay
- memory and handoff quality
- better operational packaging

## Approved Decisions

| Topic                    | Approved decision                                                                                                        |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| Protocol name            | `AGH Network`                                                                                                            |
| RFC packaging            | One RFC for v1 with internal normative sections for `Core`, `NATS Profile`, and `Baseline Trust Profile`                 |
| Core posture             | Small semantic core, transport-agnostic                                                                                  |
| First transport          | `AGH Network over NATS` as the normative v1 profile                                                                      |
| Lifecycle model          | Lightweight task lifecycle, not a workflow engine                                                                        |
| Identity/discovery       | Canonical identity plus minimal `whois/capabilities` in the core                                                         |
| Artifact model           | `recipe` remains a first-class core artifact, but execution stays outside the core                                       |
| Observability            | Minimal mandatory observability in core: correlation, lineage, `receipt`, and light `trace`                              |
| Delivery model           | Core defines semantic expectations; concrete retries, ack behavior, replay, and timeouts belong to the transport profile |
| Trust model              | Core models claimed vs verified identity; verified interoperability is defined by a normative baseline trust profile     |
| Trust algorithm strategy | A single MTI algorithm in v1 for verified-mode interoperability                                                          |
| Product moat             | AGH wins on runtime, SDK, observability, and DX, not by making the wire protocol AGH-only                                |

## Approved Architecture

### 1. AGH Network Core

The core defines the protocol semantics that should survive across transports:

- canonical envelope
- canonical identity and sender/receiver fields
- `whois/capabilities` discovery minimum
- interaction model and lightweight lifecycle
- normative message kinds
- first-class `recipe` artifact
- correlation, lineage, `receipt`, and `trace`
- semantic delivery rules
- extension and profile negotiation
- trust semantics at the level of claimed vs verified identity

The core does not define transport-specific addressing, runtime defaults, or broker topology.

### 2. AGH Network over NATS

The first official transport profile is a normative NATS binding that defines:

- subject mapping
- broadcast and direct routing
- request/reply behavior
- NATS-specific delivery expectations
- timeout, retry, and replay posture
- operational constraints specific to the binding

This keeps the protocol open while giving AGH a natural first-class fit with Go and NATS.

### 3. AGH Network Baseline Trust Profile

Verified interoperability is not left to ad hoc implementation choices. The v1 RFC includes a normative baseline trust profile that defines:

- the MTI signature algorithm
- canonicalization rules for signed messages
- how public key material is represented
- how `verified`, `unverified`, and `rejected` are interpreted

The core therefore carries trust semantics, while the concrete mechanics of verification remain fixed by the baseline profile.

## Approved Core Semantic Model

The core is intentionally small, but not vague. The main protocol primitives are:

- `Envelope`
- `Interaction`
- `Peer Card`
- `Recipe`

### Envelope

Every message travels in a canonical envelope with:

- protocol version
- sender identity
- intended target or scope
- timestamps
- correlation data
- optional proof material
- kind-specific payload
- extension surface

### Interaction

The protocol keeps a lightweight lifecycle instead of a heavy orchestration model:

- `submitted`
- `working`
- `needs_input`
- `completed`
- `failed`
- `canceled`

This is sufficient for operational handoff and progress tracking without turning the protocol into an enterprise workflow engine.

### Message kinds

The approved normative kinds are:

- `greet`
- `whois`
- `say`
- `direct`
- `receipt`
- `trace`
- `recipe`

These retain the chat-first and artifact-aware character of the earlier drafts.

### Capabilities and conformance

Peers advertise:

- supported protocol profiles
- supported trust modes
- supported artifacts
- capabilities exposed by the peer

The RFC defines explicit conformance classes so implementations can be partially compliant without ambiguity:

- `Core Sender`
- `Core Receiver`
- `Core Peer`
- `Verified Peer`
- `NATS Peer`

## Approved Profile and Interoperability Model

The approved conformance model separates syntax, operations, and trust:

- `Core` guarantees shared semantics
- `NATS Profile` guarantees transport interoperability on NATS
- `Baseline Trust Profile` guarantees verified interoperability

This means:

- third-party projects can implement the core without adopting AGH
- projects that want practical v1 interoperability implement the NATS profile
- projects that want verified-mode interoperability implement the baseline trust profile

## v1 Scope

The RFC v1 must cover:

- `AGH Network Core`
- `AGH Network over NATS`
- `AGH Network Baseline Trust Profile`
- envelope semantics
- lifecycle semantics
- message kinds
- `recipe` artifact semantics
- observability minimums
- conformance classes
- semantic delivery model
- security considerations

## v1 Non-Goals

The RFC v1 explicitly does not try to fully solve:

- global federation between organizations
- rich distributed registries
- advanced governance or policy systems
- fine-grained authorization and delegation frameworks
- workflow engines or schedulers
- execution sandbox standards
- storage and retention backends
- multiple official transports beyond NATS

These belong in future RFCs, extension profiles, or implementation-specific documentation.

## RFC Packaging

The approved primary RFC path is:

- `docs/rfcs/agh-network-v1.md`

The existing earlier draft in `docs/rfcs/agh-network.md` should no longer act as the main RFC and should instead point readers to the v1 document.

Recommended RFC structure:

1. Overview
2. Goals and Non-Goals
3. Terminology
4. Architecture and Profiles
5. Core Protocol
6. Core Message and Artifact Model
7. Core Lifecycle and Observability
8. Core Identity and Capabilities
9. NATS Profile
10. Baseline Trust Profile
11. Conformance
12. Security Considerations
13. Extensions and Future Work
14. Normative References
15. Informative References
16. Research Corpus Consulted
17. Traceability Appendix

## Research Inputs

The RFC is grounded in three input categories:

### Local project drafts

- `docs/rfcs/ideas/network/agora-spec-v0.2.md`
- `docs/rfcs/ideas/network/agora-spec-v0.1.md`
- `docs/rfcs/ideas/network/draft_1.md`
- `docs/rfcs/ideas/network/draft_2.md`
- `docs/rfcs/ideas/network/draft_3.md`
- `docs/rfcs/ideas/network/draft_4.md`
- `docs/rfcs/ideas/network/draft_5.md`
- `docs/rfcs/ideas/network/agora-recipe-design.md`
- `docs/rfcs/ideas/network/agora-council_round1.md`
- `docs/rfcs/ideas/network/agora-council_round2.md`

### Consulted knowledge-base notes from `~/dev/knowledge/agent-networks`

- `~/dev/knowledge/agent-networks/wiki/index/Concept Index.md`
- `~/dev/knowledge/agent-networks/wiki/index/Source Index.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/The A2A Protocol.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent-to-Agent Protocol Landscape.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent Network Protocol.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/AGNTCY and the Internet of Agents.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent Discovery and Registries.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent Identity and Verifiable Credentials.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent Observability and Distributed Tracing.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/Agent Capability Negotiation and Binding.md`
- `~/dev/knowledge/agent-networks/wiki/concepts/The MCP-A2A Composition Pattern.md`

### Consulted knowledge-base notes from `~/dev/knowledge/ai-harness`

- `~/dev/knowledge/ai-harness/wiki/index/Concept Index.md`
- `~/dev/knowledge/ai-harness/wiki/index/Source Index.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/The Agent Harness.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Model Context Protocol.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Agent Communication Protocols.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Agent Orchestration.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Memory Systems for Agents.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/LLMOps and Observability.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Context Engineering.md`
- `~/dev/knowledge/ai-harness/wiki/concepts/Open Source Agent Frameworks.md`

### Consulted ai-harness outputs

- `~/dev/knowledge/ai-harness/outputs/briefings/State of AI Agent Harnesses 2025-2026.md`
- `~/dev/knowledge/ai-harness/outputs/queries/2026-04-04 Key Open Questions.md`
- `~/dev/knowledge/ai-harness/outputs/queries/2026-04-06 Skill Systems Comparison Across Six Harnesses.md`
- `~/dev/knowledge/ai-harness/outputs/queries/2026-04-06 Workspace and Directory Access Across Six Harnesses.md`

## Outcome

The approved design preserves the project's three core goals:

- the protocol remains implementable outside AGH
- AGH keeps a meaningful competitive advantage
- the protocol remains distinctive instead of collapsing into either generic RPC or pure transport documentation

The correct next step is to write the main RFC in `docs/rfcs/agh-network-v1.md` and make it the authoritative specification for v1.
