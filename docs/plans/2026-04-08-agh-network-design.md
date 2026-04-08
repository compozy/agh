# Design: AGH Network

**Date:** 2026-04-08
**Status:** Approved
**Author:** Pedro Nauck + Codex

## Problem

The current `network` drafts successfully identify a differentiated direction for agent networking, but they still mix five separate concerns inside one document:

1. **Protocol semantics** — what a message means
2. **Transport binding** — how a message moves across a transport
3. **Artifact model** — recipes, traces, and related content-addressed objects
4. **SDK/runtime ergonomics** — helper APIs, embedded broker, local defaults
5. **Product posture** — AGH-specific zero-config and operational advantages

This is enough to prototype quickly, but it is not yet strong enough as an open protocol other projects can implement without implicitly adopting AGH's runtime model.

At the same time, the project should not give away its strategic advantage. AGH should remain the best implementation of the protocol through superior SDKs, runtime behavior, observability, memory, and operational profiles.

## Decisions

| Decision               | Choice                                                                                            |
| ---------------------- | ------------------------------------------------------------------------------------------------- |
| Protocol name          | `AGH Network`                                                                                     |
| Core architecture      | Small transport-agnostic semantic core                                                            |
| First transport        | Normative `NATS Profile` in the same RFC                                                          |
| Runtime/product layer  | Explicitly non-core; AGH remains the best implementation                                          |
| Wire model             | Canonical signed JSON envelope with content-addressed artifacts                                   |
| Identity model         | Self-certified handles with Ed25519 + canonical signing                                           |
| Core interaction model | Chat-first messaging plus first-class artifacts                                                   |
| Discovery model        | Minimal peer card and capability surface in core; richer registries as future profiles            |
| Observability model    | Correlation and trace identifiers in core; exporters and telemetry pipelines in profiles/runtimes |
| Extension strategy     | Namespaced extensions and explicit profile negotiation                                            |

## Why NATS Is Not The Entire Protocol

There is no major technical obstacle to making `NATS` the mandatory transport of the whole protocol. The issue is architectural and strategic:

- If NATS is mandatory in the core, the protocol becomes "an agent protocol on top of NATS", not a reusable network protocol with a first-class NATS implementation.
- Subject grammar, wildcard behavior, broker topology, and account/ACL guidance start leaking into the semantic layer.
- Third parties who want browser-first, HTTP-first, edge, or embedded non-broker deployments must bridge into NATS before they can even claim conformance.
- The protocol becomes harder to evolve independently of transport assumptions.

The approved design keeps the protocol small and open while still preserving AGH's advantage:

- **AGH Network Core** defines the meaning of communication.
- **AGH Network NATS Profile** defines the canonical mapping to NATS subjects, request/reply, and pub/sub.
- **AGH Embedded Profile** remains an implementation advantage of AGH rather than a requirement of the protocol.

This gives AGH a strong moat in implementation quality without turning the protocol into a thin wrapper around one product stack.

## Architecture

### 1. Core Semantics

The core defines only invariants that should survive across transports:

- Canonical signed envelope
- Identity, sender verification, anti-replay, and expiration semantics
- Core message kinds and artifact kinds
- Logical namespace (`space`) and correlation fields
- Minimal peer capability advertisement
- Delivery/error semantics at the abstract level
- Extension and profile negotiation rules

The core does **not** define:

- NATS subjects
- HTTP routes
- Broker topology
- JetStream or persistence rules
- Embedded broker behavior
- SDK helper ergonomics
- AGH daemon lifecycle

### 2. Transport Profiles

Profiles map the core onto concrete transports. A profile is allowed to define:

- Addressing grammar
- Subscription model
- Request/reply mapping
- Delivery guarantees and constraints specific to the transport
- Authentication hooks exposed by that transport
- Profile-specific operational guidance

The first official profile is:

- **AGH Network NATS Profile v1**

Future profiles may include HTTP/SSE, WebSocket, gRPC, or federated transport profiles, but those remain out of scope for the first RFC.

### 3. Runtime Implementations

Runtimes are where product differentiation lives. AGH should compete here, not by closing the protocol:

- Best Go SDK
- Best NATS binding and embedded story
- Best sandbox and worktree orchestration
- Best trace/replay/debug tooling
- Best memory and compaction behavior

This follows the pattern surfaced in the `ai-harness` knowledge base: open protocol, strong runtime moat.

## Core Model

The approved protocol shape keeps the strongest parts of the current drafts:

- **Chat-first messaging** stays central
- **`recipe`** remains a teaching artifact, not a deterministic workflow engine
- **`trace`** remains a first-class verifiable execution artifact
- **`payment`** remains an optional hook, not a settlement rail
- **self-certified identity** remains the default core identity model

Canonical core kinds:

- `greet`
- `say`
- `direct`
- `recipe`
- `whois`

Canonical optional kinds:

- `receipt`
- `echo`
- `revoke`
- `trace`

These names stay transport-neutral. The `NATS Profile` is responsible for mapping them onto subjects.

## Scope Boundaries

### Core RFC must contain

- Protocol goals, non-goals, and design principles
- Envelope schema
- Identity model
- Core and optional message kinds
- Peer card / capability advertisement
- Artifact model for recipes and traces
- Delivery and error model
- Security considerations
- Extension and profile model
- Conformance classes
- Normative NATS profile

### Core RFC must not contain

- AGH CLI UX
- Embedded NATS startup UX
- Reference SDK API surface
- Full config file stories
- AGH daemon deployment topologies
- Product marketing claims

Those belong in companion docs or implementation-specific material.

## Conformance Strategy

The RFC should define conformance classes early to avoid ambiguity:

- **Core Sender**
- **Core Receiver**
- **Core Peer**
- **NATS Profile Peer**
- **NATS Profile Broker-Compatible Deployment**

AGH can then claim:

- Core conformance
- NATS profile conformance
- AGH embedded runtime profile support

## Open Items For The RFC

The design is approved, but the RFC still needs to make explicit choices on:

1. Exact envelope field names and canonicalization format
2. Final shape of the peer card
3. Minimal error code registry
4. Required versus optional discovery surface
5. Required correlation fields for tracing and handoff
6. Extension advertisement and incompatibility behavior

These should be resolved inside the RFC itself, not deferred back into product docs.

## Implementation Direction

After the RFC is written, the project should proceed in this order:

1. Finalize the `AGH Network Core + NATS Profile` RFC
2. Add a reference conformance section and canonical examples
3. Build the Go reference SDK around the RFC, not around AGH internals
4. Build AGH's embedded/runtime story as the best implementation profile
5. Add conformance tests that third parties can run without AGH

## Outcome

This design preserves all three strategic goals:

- The protocol remains implementable outside AGH
- AGH retains a strong competitive advantage
- The system stays ambitious without collapsing into overengineering

`AGH Network` should therefore be written as an open protocol with a small semantic core and a first-class normative NATS profile, not as a NATS-only product protocol.
