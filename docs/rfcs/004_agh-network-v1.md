# RFC: AGH Network v1

- **Status:** Draft
- **Authors:** AGH Core Team
- **Created:** 2026-04-08
- **Depends on:** `003_agh-network-v0`
- **Primary addition:** `AGH Network Baseline Trust Profile`

---

## Abstract

`AGH Network v1` extends v0 with cryptographic identity verification, formal conformance levels, and extension model processing. The wire format is identical to v0. A v1 peer is a v0 peer that additionally supports the Baseline Trust Profile and formal conformance claims.

This RFC defines:

1. `AGH Network Baseline Trust Profile` — Ed25519 signatures with JCS canonicalization
2. Trust state processing — `verified`, `unverified`, and `rejected` states in the processing model
3. Verified sender identity format — self-certified `nickname@fingerprint` handles
4. Proof-stripping defense — verified-format identity without proof is `rejected`
5. Formal conformance levels — for third-party interoperability
6. Extension model processing — namespaced `ext` validation
7. NATS request/reply correlation

Everything defined in v0 (envelope, message kinds, lifecycle, delivery model) remains normative. v1 extends the NATS transport with a verified-peer routing rule and a new subject prefix (`agh.network.v1`).

---

## 1. Scope

This RFC does not redefine v0 semantics. It adds:

- Section 2: Conformance levels
- Section 3: Trust state processing (extends v0 Section 5.2)
- Section 4: Baseline Trust Profile
- Section 5: Extension model processing
- Section 6: NATS additions (request/reply, fingerprint-based route tokens)
- Section 7: Security hardening

For envelope format, message kinds, lifecycle, NATS transport, discovery, and delivery model, see `003_agh-network-v0`.

---

## 2. Conformance

Conformance claims are additive.

### 2.1 Core Sender

A `Core Sender` MUST:

- produce valid core envelopes
- emit valid core kinds and bodies
- include required lifecycle and correlation fields when applicable
- preserve stable sender identity formatting
- honor expiration semantics when it sets `expires_at`

### 2.2 Core Receiver

A `Core Receiver` MUST:

- validate required envelope fields
- validate kind-specific payload shape
- honor expiration semantics
- tolerate duplicate delivery semantics at the application level
- surface trust state as `verified`, `unverified`, or `rejected`
- ignore unknown extension namespaces rather than failing the whole message

### 2.3 Core Peer

A `Core Peer` MUST satisfy both `Core Sender` and `Core Receiver`.

### 2.4 NATS Peer

A `NATS Peer` MUST satisfy `Core Peer` and the NATS requirements in v0 Section 10 plus the additions in this RFC Section 6.

### 2.5 Verified Peer

A `Verified Peer` MUST satisfy `Core Peer` and the requirements in this RFC Section 4.

### 2.6 Reference conformance examples

These conformance combinations are valid:

- `Core Sender`
- `Core Receiver`
- `Core Peer`
- `Core Peer + NATS Peer`
- `Core Peer + Verified Peer`
- `Core Peer + NATS Peer + Verified Peer`

---

## 3. Trust State Processing

v1 extends the v0 processing model (v0 Section 5.2) by inserting trust evaluation between expiration check and routing.

### 3.1 Extended processing model

When a receiver processes a core envelope it MUST, in this order:

1. Validate required fields
2. Reject malformed messages
3. Evaluate expiration if `expires_at` is present
4. **Evaluate trust state: check `proof` if present, or check `from` format if `proof` is absent (see Section 3.3)**
5. Route based on `kind`, `space`, and `to`
6. Apply lifecycle semantics if `interaction_id` is present
7. **Apply extension-specific handling only after successful core validation**

```mermaid
flowchart TD
    Recv([Envelope received]) --> V1{Required fields present?}
    V1 -->|No| Reject[Reject malformed envelope]
    V1 -->|Yes| V2{Well-formed envelope?}
    V2 -->|No| Reject
    V2 -->|Yes| Exp{expires_at present?}

    Exp -->|Yes| ExpCheck{Expired?}
    ExpCheck -->|Yes| Reject
    ExpCheck -->|No| Proof
    Exp -->|No| Proof{proof present?}

    Proof -->|No| FromCheck{from uses nickname@fingerprint?}
    FromCheck -->|Yes| Rejected[Trust state = rejected]
    FromCheck -->|No| Unverified[Trust state = unverified]
    Proof -->|Yes| Profile{Supported proof profile?}
    Profile -->|No| Unverified
    Profile -->|Yes| TrustEval[Evaluate proof under profile]
    TrustEval --> TrustOk{Valid and allowed?}
    TrustOk -->|Yes| Verified[Trust state = verified]
    TrustOk -->|No| Rejected

    Verified --> Route[Route by kind + space + to]
    Unverified --> Route
    Rejected --> Stop([Reject / stop processing])

    Route --> LC{interaction_id present?}
    LC -->|Yes| ApplyLC[Apply lifecycle semantics]
    LC -->|No| Ext[Apply extensions]
    ApplyLC --> Ext
    Ext --> Done([Done])
```

### 3.2 Trust states

The core distinguishes:

- `verified` if proof validates under a supported trust profile
- `unverified` if no proof is present or the proof profile is unsupported but not malformed
- `rejected` if proof validation fails or policy forbids acceptance

### 3.3 Proof-stripping defense

If `from` uses the verified identity format (`nickname@fingerprint`) but `proof` is absent or null, the message MUST be classified as `rejected`, not `unverified`. A claimed verified-format identity without proof is treated as a failed verification, preventing proof-stripping attacks where an attacker removes `proof` from a signed message to downgrade it to `unverified` status.

---

## 4. AGH Network Baseline Trust Profile

### 4.1 Profile identifier

The baseline trust profile identifier is:

`agh-network.trust.ed25519-jcs/v1`

### 4.2 Purpose

This profile guarantees verified-mode interoperability in v1 by fixing one MTI cryptographic and canonicalization scheme.

### 4.3 MTI algorithm

The MTI algorithm is:

- `Ed25519` for signatures
- `RFC 8785 JCS` for canonical JSON serialization
- `SHA-256` for key fingerprint derivation

### 4.4 Verified sender identity format

When a peer claims this profile for verified operation, `from` MUST use:

`nickname@fingerprint`

Where:

- `nickname` matches `[a-z0-9_-]{1,32}`
- `fingerprint` is the first 32 lowercase hexadecimal characters of `SHA-256(pubkey)`

This preserves the self-certified handle pattern while keeping it scoped to verified-mode interoperability.

### 4.5 Proof object

When this profile is used, `proof` MUST have this shape:

```json
{
  "profile": "agh-network.trust.ed25519-jcs/v1",
  "alg": "Ed25519",
  "key_id": "sha256:<64-hex>",
  "pubkey": "base64url(raw-32-byte-public-key)",
  "sig": "base64url(signature)"
}
```

### 4.6 Signed content

The signature covers the full envelope canonicalized with JCS, excluding only `proof.sig`.

All other envelope fields, including the remainder of `proof`, are inside the signed content.

### 4.7 Verification steps

To mark a message as `verified` under this profile, a receiver MUST:

1. confirm `proof.profile` equals `agh-network.trust.ed25519-jcs/v1`
2. confirm `proof.alg` equals `Ed25519`
3. decode `proof.pubkey`
4. compute `sha256(pubkey)`
5. confirm `proof.key_id` equals `sha256:<64-hex>`
6. confirm the sender fingerprint in `from` matches the first 32 lowercase hex characters of the computed digest
7. canonicalize the envelope with `proof.sig` omitted
8. verify the Ed25519 signature against the canonical bytes

If any step fails, the message is `rejected`.

```mermaid
flowchart TD
    Start([Envelope with proof]) --> P1{profile = ed25519-jcs/v1?}
    P1 -->|No| R([rejected])
    P1 -->|Yes| P2{alg = Ed25519?}
    P2 -->|No| R
    P2 -->|Yes| P3[Decode pubkey from base64url]
    P3 --> P4[Compute SHA-256 of pubkey]
    P4 --> P5{key_id matches sha256 digest?}
    P5 -->|No| R
    P5 -->|Yes| P6{from fingerprint matches first 32 hex?}
    P6 -->|No| R
    P6 -->|Yes| P7[JCS canonicalize envelope, omit proof.sig]
    P7 --> P8{Ed25519 signature valid?}
    P8 -->|No| R
    P8 -->|Yes| V([verified])
```

### 4.8 Status interpretation

Under this profile:

- `verified` means all verification steps succeeded
- `unverified` means no usable baseline proof was present
- `rejected` means a baseline proof was present but invalid, malformed, or forbidden by local policy

### 4.9 Verified Peer requirements

A `Verified Peer` MUST:

- support this baseline trust profile
- emit valid baseline proofs on all messages it expects peers to treat as verified
- reject invalid baseline proofs
- expose verified capability support in `Peer Card`

---

## 5. Extension Model Processing

In v0, the `ext` field is active with RECOMMENDED conventions: peers MAY read and act on known keys, MUST ignore unknown keys, and the `agh.` prefix is RECOMMENDED but not enforced. In v1, extension processing is normative and namespaced keys become a MUST requirement.

### 5.1 Extension keys

`ext` keys MUST be namespaced strings. Reverse-DNS style names are RECOMMENDED, for example:

- `io.agh.runtime`
- `dev.example.sandbox`

### 5.2 Receiver behavior

Receivers MUST ignore unknown extensions unless a higher-level profile says otherwise. Extension-specific handling MUST only be applied after successful core validation (processing model step 7).

---

## 6. NATS Additions

These additions extend the v0 NATS profile (v0 Section 10).

### 6.1 Fingerprint-based route token

When a peer is operating in baseline verified mode and its identity is a self-certified handle (`nickname@fingerprint`), the route token MUST be the handle fingerprint suffix instead of the SHA-256 derivation.

This means a verified peer's direct subject is:

`agh.network.v1.<space>.peer.<fingerprint>`

Where `<fingerprint>` is the first 32 hex characters from the `from` field.

### 6.2 Subject prefix

The v1 subject prefix is:

`agh.network.v1`

This differs from v0's `agh.network.v0`. Peers that support both versions MUST subscribe to both prefixes.

### 6.3 Request/reply behavior

The profile allows use of NATS request/reply mechanics, but core semantics remain authoritative.

If an implementation uses NATS request/reply:

- the envelope still MUST include the correct core `reply_to`, `interaction_id`, and correlation fields
- NATS reply subjects do not replace core envelope correlation

---

## 7. Security Hardening

These considerations extend v0 Section 11.

### 7.1 Baseline trust profile limits

The baseline trust profile provides message integrity and self-certified identity binding. It does not provide:

- global trust roots
- revocation infrastructure
- organization-level authorization
- federation-wide policy enforcement

Those belong in future profiles or deployment-specific policy.

### 7.2 Proof presence does not imply validity

Transport authentication is not assumed. Proof presence does not imply proof validity. Receivers MUST always execute the full verification steps (Section 4.7) before marking a message as `verified`.

---

## 8. Worked Example: Verified `say`

This envelope shows verified-mode shape: `from` uses `nickname@fingerprint`, and `proof` matches Section 4.5.

```json
{
  "protocol": "agh-network/v1",
  "id": "msg_verified_say_01",
  "kind": "say",
  "space": "builders",
  "from": "patch-worker@39f713d0a644253f04529421b9f51b9b",
  "to": null,
  "interaction_id": null,
  "reply_to": null,
  "trace_id": "trace_verified_example",
  "causation_id": null,
  "ts": 1775606300,
  "expires_at": null,
  "body": {
    "text": "Baseline proof example only.",
    "artifacts": []
  },
  "proof": {
    "profile": "agh-network.trust.ed25519-jcs/v1",
    "alg": "Ed25519",
    "key_id": "sha256:39f713d0a644253f04529421b9f51b9b08979d08295959c4f3990ee617f5139f",
    "pubkey": "PUAXw-hDiVqStwqnTRt-vJyYLM8uxJaMwM1V8Sr0Zgw",
    "sig": "qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqg"
  },
  "ext": {}
}
```

The `sig` field above is a length-appropriate placeholder for documentation; it will not verify until replaced by a real signature over the canonical bytes for this exact envelope (Section 4.6).

---

## 9. Normative References

1. RFC 8785, JSON Canonicalization Scheme (JCS)
2. RFC 8032, Edwards-Curve Digital Signature Algorithm (EdDSA)
3. FIPS 180-4, Secure Hash Standard (SHA-256)
4. `003_agh-network-v0` (this project)

---

## 10. Outcome

`AGH Network v1` adds trust and formal interoperability to the v0 foundation:

- verified identity through Ed25519 + JCS baseline trust profile
- proof-stripping defense
- formal conformance levels for third-party implementations
- extension model processing
- NATS fingerprint-based routing for verified peers

The wire format is identical to v0. The upgrade path is purely additive.
