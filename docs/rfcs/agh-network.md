# RFC: AGH Network Core and NATS Profile

- **Status:** Draft
- **Authors:** AGH Core Team
- **Created:** 2026-04-08
- **Relates to:** NATS, MCP, A2A, OpenTelemetry, AgentSkills

---

## Abstract

`AGH Network` is an open protocol for communication between software agents. It is designed to be:

- **open to third-party implementations**
- **small enough to implement without adopting AGH**
- **chat-first and artifact-aware**
- **compatible with stronger runtime profiles built by AGH**

The protocol is split into two layers within this RFC:

1. **AGH Network Core** — the transport-agnostic semantic protocol
2. **AGH Network NATS Profile** — the normative mapping of that core onto NATS

This separation is deliberate. The protocol must not require the AGH daemon, AGH CLI, or AGH runtime model to exist. At the same time, AGH should remain the best implementation of the protocol through superior SDKs, runtime ergonomics, observability, memory, and operational profiles.

---

## 1. Problem Statement

### 1.1 The Current Gap

The agent ecosystem now has credible standards for several adjacent layers:

- **MCP** for agent-to-tool communication
- **A2A** for task-oriented agent-to-agent interaction
- **OpenTelemetry** and GenAI observability conventions for trace pipelines
- **AgentSkills** for reusable operational instructions

What remains under-specified is a lean, open, implementation-friendly protocol for agent networking that is:

- conversational rather than workflow-heavy
- transport-aware without being transport-locked
- identity-bearing without requiring a centralized trust authority
- artifact-aware without becoming a workflow engine
- useful for both local and distributed agent fabrics

### 1.2 Why Not Just Use Existing Protocols

This RFC does not argue that existing protocols are wrong. It argues they solve adjacent problems.

- **MCP** is the tool connectivity layer, not the network between autonomous agents.
- **A2A** is strong at enterprise task lifecycle and richer agent cards, but it optimizes for a different interaction shape than lightweight chat-first peer exchange.
- **NATS** is a transport and messaging system, not an agent protocol.

`AGH Network` occupies a narrower, deliberate position:

- it standardizes the meaning of peer-to-peer and peer-to-space agent communication
- it standardizes lightweight artifact sharing
- it keeps transport mapping separate enough that the protocol can survive its first binding

### 1.3 Requirements

The protocol must:

1. Be implementable without AGH
2. Preserve a first-class NATS story
3. Support both 1:1 and 1:many communication
4. Carry signed, verifiable identities
5. Support lightweight discovery and capability advertisement
6. Treat recipes and traces as first-class artifacts
7. Support correlation and observability across hops
8. Leave room for future transport, trust, and federation profiles

---

## 2. Goals

1. **Open implementation surface.** Any project should be able to implement the protocol with no dependency on AGH.
2. **Transport discipline.** The protocol should define semantic invariants first and transport mappings second.
3. **Chat-first interaction.** Broadcast and direct messaging should be native.
4. **Artifact awareness.** Agents should be able to exchange reusable procedures and execution traces.
5. **Pragmatic security.** The protocol should provide identity integrity, anti-replay, and room for stronger trust profiles.
6. **Operational clarity.** Correlation, expiry, deduplication, and error behavior should be explicit.

---

## 3. Non-Goals

1. `AGH Network` is **not** a replacement for MCP.
2. It is **not** a full enterprise task orchestration framework.
3. It is **not** a workflow engine or deterministic automation runtime.
4. It is **not** a payment rail or settlement protocol.
5. It is **not** a global reputation protocol.
6. It is **not** a federation protocol in v1.
7. It is **not** tied to AGH's embedded runtime or daemon lifecycle.

---

## 4. Design Principles

1. **Protocol first, product second.** The wire semantics must stand on their own.
2. **Small semantic core.** Only invariants belong in the core.
3. **Profiles over forks.** Transport and trust specialization should appear as explicit profiles.
4. **Chat-first.** Broadcast and direct exchange are native, not emulated.
5. **Artifact-native.** `recipe` and `trace` are first-class protocol citizens.
6. **Least-trust default.** Self-certified identity is part of the core; stronger trust models layer on top.
7. **Operationally observable.** Correlation metadata is not optional hand-waving.

---

## 5. Layer Model

### 5.1 Layer 1: AGH Network Core

The core defines:

- canonical envelope
- identity model
- core message kinds
- peer capability advertisement
- artifact model
- delivery semantics
- error model
- extension and profile negotiation

### 5.2 Layer 2: Transport Profiles

Profiles define:

- address mapping
- request/reply mapping
- broadcast mapping
- profile-specific delivery guarantees
- transport-specific authentication hooks

This RFC defines one normative profile:

- **AGH Network NATS Profile v1**

### 5.3 Layer 3: Runtime Implementations

Runtimes implement the core and one or more profiles. They may provide:

- helper SDKs
- retries and reconnect strategies
- persistence and caches
- telemetry exporters
- sandboxing
- memory systems
- deployment defaults

AGH is expected to be the best implementation at this layer, but none of it is required for protocol conformance.

---

## 6. Conformance Classes

### 6.1 Core Sender

A `Core Sender` MUST:

- produce valid core envelopes
- sign every emitted message
- generate unique message identifiers
- include anti-replay material
- conform to the schema of the selected `kind`

### 6.2 Core Receiver

A `Core Receiver` MUST:

- verify signatures
- validate sender identity binding
- reject expired or malformed envelopes
- deduplicate messages within a local replay window
- parse core kinds and reject unknown required extensions

### 6.3 Core Peer

A `Core Peer` MUST satisfy both `Core Sender` and `Core Receiver`.

### 6.4 NATS Profile Peer

A `NATS Profile Peer` MUST satisfy `Core Peer` and correctly map core semantics to the NATS profile specified in this RFC.

---

## 7. Core Identity Model

### 7.1 Handle

The canonical peer handle format is:

`nickname@fingerprint`

Where:

- `nickname` is a human-friendly label
- `fingerprint` is a deterministic identifier derived from the public key

### 7.2 Key Algorithm

The core algorithm is:

- **Ed25519**

This choice is deliberate:

- high-quality standard library support in Go
- easy implementation in multiple languages
- simple detached signatures
- no dependency on blockchain-oriented key formats

### 7.3 Fingerprint

The canonical fingerprint is:

- the first 16 bytes of `SHA-256(pubkey)`
- lower-case hexadecimal

This gives a 128-bit fingerprint surface for routing and display while keeping the full public key available for verification.

### 7.4 Identity Verification

Receivers MUST verify:

1. the sender handle includes a fingerprint
2. the claimed or resolved public key hashes to that fingerprint
3. the signature verifies against the canonicalized envelope

### 7.5 Trust Separation

The core explicitly separates:

- **identity** — who emitted this message
- **authentication** — whether a transport/session allowed it
- **trust** — whether the receiver should rely on the sender

The core standardizes identity integrity, not global trust policy.

---

## 8. Core Envelope

### 8.1 Canonical Structure

Every `AGH Network` message is a signed JSON object.

Canonical example:

```json
{
  "protocol": "agh-network/1",
  "id": "01JQKX3M4P5D4VQW9MKC9V8C2N",
  "kind": "say",
  "space": "main",
  "from": "alice@a1b2c3d4e5f67890deadbeefcafe1234",
  "to": null,
  "reply_to": null,
  "thread": null,
  "trace": {
    "trace_id": "01JQKX3YV9Q5JQCBGGV2V8Q3MS",
    "parent_id": null
  },
  "ts": 1775660400,
  "nonce": "7f3a9b2c4d5e6f70",
  "expires_at": null,
  "body": {
    "text": "who can translate Ancient Greek?"
  },
  "payment": null,
  "ext": {},
  "sig": "base64url(signature)"
}
```

### 8.2 Required Fields

| Field      | Type    | Description                           |
| ---------- | ------- | ------------------------------------- |
| `protocol` | string  | Core protocol version identifier      |
| `id`       | string  | Globally unique message ID            |
| `kind`     | string  | Semantic message kind                 |
| `space`    | string  | Logical namespace for the interaction |
| `from`     | string  | Canonical sender handle               |
| `ts`       | integer | Unix epoch seconds                    |
| `nonce`    | string  | Anti-replay random value              |
| `body`     | object  | Kind-specific payload                 |
| `sig`      | string  | Detached Ed25519 signature            |

### 8.3 Optional Fields

| Field        | Type            | Description                         |
| ------------ | --------------- | ----------------------------------- |
| `to`         | string or null  | Target peer for direct messages     |
| `reply_to`   | string or null  | Referenced message ID               |
| `thread`     | string or null  | Thread correlation ID               |
| `trace`      | object or null  | Cross-hop observability correlation |
| `expires_at` | integer or null | Expiration timestamp                |
| `payment`    | object or null  | Optional payment hook               |
| `ext`        | object          | Namespaced extensions               |

### 8.4 Canonicalization

The signed payload MUST use:

- **JCS / RFC 8785 canonical JSON**

The `sig` field itself is excluded from the signed content.

### 8.5 Anti-Replay

Receivers MUST use:

- `id`
- `nonce`
- `ts`

to reject duplicates and stale messages within an implementation-defined replay window.

### 8.6 Namespace Binding

The `space` field is part of the signed payload.

Profiles MAY add additional namespace validation rules, but they MUST NOT redefine the semantic meaning of `space` itself.

---

## 9. Core Message Kinds

### 9.1 Mandatory Core Kinds

#### `greet`

Announces peer identity and capabilities into a `space`.

Canonical body:

```json
{
  "pubkey": "base64url(pubkey)",
  "description": "Classical translator. Charges per word.",
  "capabilities": [
    { "id": "translation", "version": "1" },
    { "id": "ancient-greek", "version": "1" }
  ],
  "profiles": ["agh-network.nats/1"],
  "extensions": [],
  "endpoints": []
}
```

#### `say`

Broadcast message to every peer currently participating in the same `space`.

Canonical body:

```json
{
  "text": "need help translating 500 words from Ancient Greek to Portuguese"
}
```

#### `direct`

Direct peer-to-peer message to a specific `to` handle.

Canonical body:

```json
{
  "text": "I can do that for 0.10 USDC"
}
```

#### `recipe`

Publishes a teaching artifact. Recipes are **not** deterministic workflow execution plans in the core.

Canonical body:

```json
{
  "recipe_id": "sha256hex",
  "name": "parse-nfe-ptbr",
  "version": "1.2",
  "description": "Parse a Brazilian fiscal PDF and extract the CNPJ.",
  "inputs": [{ "name": "pdf_bytes", "type": "bytes", "required": true }],
  "outputs": [{ "name": "cnpj", "type": "string" }],
  "steps": [
    { "n": 1, "kind": "skill", "name": "ocr-pdf", "save_as": "text" },
    { "n": 2, "kind": "prompt", "text": "Extract the CNPJ from {{ text }}", "save_as": "cnpj" }
  ]
}
```

#### `whois`

Identity and peer card lookup.

Request body:

```json
{
  "handle": "alice@a1b2c3d4e5f67890deadbeefcafe1234"
}
```

Response SHOULD be carried as a `direct` reply containing the corresponding peer card.

### 9.2 Optional Core Kinds

#### `receipt`

Acknowledges successful, partial, or failed delivery of a prior interaction.

#### `echo`

Publishes a signed attestation about another peer's observed behavior.

#### `revoke`

Signals key rotation or identity revocation, signed by the prior key.

#### `trace`

Publishes a verifiable execution trace artifact describing how a result was produced.

---

## 10. Peer Card and Capability Advertisement

### 10.1 Peer Card

The core discovery surface is the **peer card**. It is intentionally small.

It SHOULD contain:

- canonical handle
- public key
- protocol version
- supported profiles
- supported extensions
- capability list
- optional endpoints
- optional authentication hints
- optional descriptive metadata

### 10.2 Why Minimal Discovery In Core

The core standardizes the shape of discoverability, not a global registry architecture.

This is intentional. The protocol must support:

- local broadcast-based discovery
- static peer caches
- well-known documents
- profile-specific registries
- future federation protocols

without forcing one governance model in v1.

---

## 11. Artifacts

### 11.1 Recipe Artifacts

Recipes are content-addressed artifacts with their own identity.

The `recipe_id` MUST be derived from the canonicalized recipe body excluding the `recipe_id` field itself.

### 11.2 Trace Artifacts

Traces are signed execution records describing:

- referenced recipe or interaction
- outcome
- steps executed
- steps improvised or skipped
- duration
- optional structured per-step log

### 11.3 Artifact Separation

Artifacts exist independently of the transport message that carries them.

This is a core design choice. Transport delivers artifacts; it does not define them.

---

## 12. Delivery Semantics

### 12.1 Core Guarantees

The core does **not** guarantee:

- exactly-once delivery
- total ordering across peers
- persistence
- replay history

These are profile-level or runtime-level concerns.

### 12.2 Core Requirements

The core does require:

- stable message identity
- deduplication behavior at the receiver
- expiry handling
- correlation fields
- signature verification before trust

### 12.3 Ordering

Unless a profile says otherwise:

- peers MUST assume messages may arrive duplicated
- peers MUST assume messages may arrive out of order
- peers MUST use `reply_to`, `thread`, and `trace` rather than transport ordering as the semantic correlation mechanism

---

## 13. Error Model

Receivers SHOULD use the following canonical rejection reasons:

| Code                    | Meaning                                                     |
| ----------------------- | ----------------------------------------------------------- |
| `invalid_signature`     | Signature verification failed                               |
| `unknown_sender`        | Sender identity could not be resolved                       |
| `expired`               | Message exceeded `expires_at` or replay window              |
| `duplicate`             | Message already processed                                   |
| `unsupported_kind`      | Kind is unknown or disallowed                               |
| `invalid_body`          | Body does not satisfy the kind schema                       |
| `unsupported_extension` | Required extension is not supported                         |
| `space_mismatch`        | Transport/profile namespace does not match signed namespace |
| `unauthorized`          | Local policy denied processing                              |

Profiles MAY define profile-specific transport errors, but these codes are the semantic baseline.

---

## 14. Correlation and Observability

### 14.1 Core Fields

The core tracing surface is intentionally small:

- `id`
- `reply_to`
- `thread`
- `trace.trace_id`
- `trace.parent_id`

### 14.2 Scope

The core enables:

- causal correlation between messages
- trace continuity across agent hops
- replay/debug tooling in runtimes

The core does **not** standardize:

- OpenTelemetry exporters
- sampling rates
- PII policies
- backend schemas

Those belong in operational profiles or runtimes.

---

## 15. Payment Hook

The `payment` field is an optional hook. It allows a sender to attach a transportable payment reference.

Canonical example:

```json
{
  "rail": "x402",
  "amount": "0.10",
  "currency": "USDC",
  "proof": "0xabc123"
}
```

The protocol carries this envelope. It does not validate or settle the payment.

---

## 16. Extension and Profile Model

### 16.1 Extensions

Extensions MUST:

- use namespaced keys
- declare version compatibility
- be ignorable unless explicitly required

### 16.2 Profiles

A profile is a named mapping of the core to a transport or stronger operational/security model.

Profiles SHOULD be advertised in peer cards.

### 16.3 Required Extension Handling

If a message depends on a required extension the receiver does not support, the receiver MUST reject it with `unsupported_extension`.

---

## 17. Security Considerations

### 17.1 In Scope

The core directly mitigates:

- sender spoofing
- envelope tampering
- basic replay attacks
- namespace rewriting when profile validation exists

### 17.2 Out Of Scope

The core does not, by itself, solve:

- prompt injection in natural-language payloads
- Sybil resistance
- global trust or reputation
- end-to-end message confidentiality
- regulatory compliance

### 17.3 Natural Language Risk

All text content in:

- `say`
- `direct`
- `recipe`
- `echo`
- `trace`

MUST be treated as untrusted external input by receivers.

---

## 18. AGH Network NATS Profile v1

### 18.1 Scope

This profile defines the normative mapping of `AGH Network Core` to NATS.

It exists to provide:

- immediate interoperable deployments
- a first-class Go and AGH implementation path
- a concrete profile without forcing NATS into the semantic core

### 18.2 Subject Prefix

All profile subjects MUST begin with:

`aghnet.v1`

### 18.3 Subject Mapping

The canonical mapping is:

| Core semantic   | NATS subject                                                   |
| --------------- | -------------------------------------------------------------- |
| `greet`         | `aghnet.v1.<space>.broadcast`                                  |
| `say`           | `aghnet.v1.<space>.broadcast`                                  |
| `direct`        | `aghnet.v1.<space>.peer.<to_fingerprint>`                      |
| `recipe`        | `aghnet.v1.<space>.artifact.recipe.<recipe_id>`                |
| `trace`         | `aghnet.v1.<space>.artifact.trace.<artifact_id>`               |
| `whois` request | `aghnet.v1.<space>.lookup.whois.request`                       |
| `whois` reply   | `aghnet.v1.<space>.lookup.whois.reply.<requester_fingerprint>` |
| `echo`          | `aghnet.v1.<space>.attest.echo.<about_fingerprint>`            |

### 18.4 Space Mapping

In this profile:

- the core `space` field maps to the subject namespace token `<space>`

Receivers MUST validate:

1. the subject-implied space
2. the signed `space` in the envelope

If they differ, the receiver MUST reject the message with `space_mismatch`.

### 18.5 Direct Delivery

For `direct` messages:

- the sender MUST set `to`
- the publisher MUST route to the subject containing the target fingerprint
- the receiver MUST still verify the signed `to` field

The NATS subject is routing metadata; the signed envelope remains authoritative.

### 18.6 Request/Reply

The profile MAY use NATS request/reply for `whois` and similar lookups, but the canonical semantic contract remains the core envelope and core correlation fields.

### 18.7 Delivery Class

This profile assumes:

- broker-mediated fan-out
- possible duplicate delivery
- no exactly-once guarantee
- no mandatory persistence

If a deployment adds JetStream or stronger persistence, that is an implementation choice unless a future profile standardizes it.

### 18.8 Wildcards

Wildcards are a NATS profile concern, not a core concern.

Typical subscriptions:

- `aghnet.v1.<space>.broadcast`
- `aghnet.v1.<space>.peer.<my_fingerprint>`
- `aghnet.v1.<space>.lookup.whois.request`

### 18.9 NATS Security

This profile does not standardize:

- NATS account layout
- ACL design
- TLS policy
- broker federation
- JetStream policy

Those are deployment concerns or future profiles.

---

## 19. Relationship To AGH

AGH is a reference runtime for `AGH Network`, not a prerequisite.

An implementation MAY conform to this RFC while:

- not embedding NATS
- not exposing AGH CLI semantics
- not using AGH memory models
- not adopting AGH runtime internals

AGH's competitive advantage should live in:

- better SDKs
- better runtime safety
- better memory and compaction
- better traces and replay tooling
- better operational defaults

That is a stronger and more defensible position than locking the protocol to one product.

---

## 20. Open Questions

The following remain intentionally open for iteration after this draft:

1. Whether a future HTTP profile becomes normative alongside NATS
2. Whether a richer signed peer card should be split into its own companion spec
3. Whether recipe body signatures should exist in addition to envelope signatures
4. Whether stronger trust and governance profiles should be published separately
5. Whether a federation profile should standardize registry synchronization

---

## 21. Recommended Next Steps

1. Approve this layer split as the official `AGH Network` architecture
2. Refine the exact envelope and peer card schema
3. Add canonical test vectors for signing and verification
4. Build the Go reference SDK against this RFC
5. Publish AGH's embedded runtime as a reference implementation profile, not as the protocol itself
