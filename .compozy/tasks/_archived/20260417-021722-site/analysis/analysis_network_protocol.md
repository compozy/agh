# AGH Network as a Standalone Product Surface

## Protocol Summary

AGH Network should be positioned as a protocol product, not as a runtime feature. Its job is to define how independent agent harnesses discover each other, describe capabilities, exchange messages, negotiate trust, and preserve interoperability across implementations. The runtime is the reference implementation that makes the protocol real inside AGH, but it is not the protocol itself.

This matters because the protocol is the unique differentiator. Most harnesses already have a runtime, a UI, a session model, and some form of tool or agent orchestration. What they do not have is a separate, open agent-network surface with a wire contract, conformance language, and adoption path that other harnesses can implement without adopting AGH wholesale.

## Why This Is Strategically Different

AGH Network solves a narrower and more durable problem than a harness runtime:

- It gives third-party harnesses a way to interoperate at the agent-to-agent layer without binding them to AGH's internal session manager, CLI, or web UI.
- It creates a product surface that is portable across runtimes. A harness can keep its own lifecycle model and still speak AGH Network on the wire.
- It turns AGH into a reference implementation plus opinionated runtime, rather than a closed ecosystem.
- It creates a clearer moat than "we also have sessions, channels, peers, and a dashboard," because those features exist in many forms across the harness landscape.

The strategic claim should be careful and specific: AGH Network is not "better runtime messaging." It is a separate interoperability layer. The runtime is the best place to ship the first implementation, but the protocol is the thing other projects can adopt.

This separation is visible in the local research corpus. Reference projects such as acpx, OpenFang, OpenCode, and GoClaw treat runtime, docs, and protocol as distinct surfaces. Their docs are strongest when they keep the product story separate from the transport and lifecycle details. AGH should do the same, but with a sharper split because the protocol is itself a product.

## Runtime vs Protocol Split

The clean boundary is this:

| Surface              | Owns                                                                                              | Does not own                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| AGH Runtime          | daemon behavior, sessions, CLI, web UI, local storage, audit, message delivery inside the daemon  | wire format, transport binding, external conformance language   |
| AGH Network Protocol | envelope, message kinds, interaction lifecycle, discovery, trust, transport profiles, conformance | session internals, UI flows, storage layout, daemon composition |

### Runtime messaging

Runtime messaging is AGH talking to itself and to local agents. It includes:

- session lifecycle and turn management
- CLI and web interactions
- queued prompts and delivery into live sessions
- local audit and persistence
- internal coordination between daemon components

The `web/src/routes/_app/network.tsx` route is a runtime surface. It exposes channels, peers, and local channel creation in the AGH app. That is useful, but it should be described as the runtime's observability and control plane, not as the protocol itself.

### Protocol messaging

Protocol messaging is agent-to-agent communication across harness boundaries. It includes:

- canonical envelopes
- message kinds such as `greet`, `whois`, `say`, `direct`, `receipt`, `trace`, and `recipe`
- interaction identifiers and lightweight lifecycle semantics
- trust states and proof handling
- transport binding and conformance claims

The protocol should stay transport-agnostic at the core and let profiles define the binding. The runtime can implement the first profile, but it should not absorb protocol semantics into daemon-specific concepts like sessions or prompts.

### The translation boundary

AGH Runtime should translate between local runtime events and protocol envelopes.

- Local agent activity becomes protocol messages when it crosses the harness boundary.
- Incoming protocol messages become local prompts, queue entries, or runtime events after validation.
- Local identifiers such as session IDs should never leak into the public protocol as if they were the protocol's own identity model.

That boundary is the product line. Everything on the runtime side is AGH-specific. Everything on the protocol side should remain implementable by other harnesses.

## Recommended Protocol Doc Taxonomy

The site should make the split obvious in both navigation and content hierarchy.

### 1. AGH Runtime

This section should answer: "How do I run AGH?"

Recommended contents:

- overview of the daemon and single-binary runtime
- installation and startup
- configuration
- CLI usage
- session and workspace behavior
- web UI and operational views
- persistence, audit, and observability
- local network controls as runtime features

### 2. AGH Network Protocol

This section should answer: "How do independent harnesses speak AGH Network?"

Recommended contents:

- protocol overview and problem statement
- core envelope and lifecycle
- message kinds and field requirements
- discovery and capability signaling
- trust and verification model
- transport profiles
- conformance classes
- implementation guidance for third-party harnesses

### 3. Protocol Subpages

The protocol section should be broken down by concept, not by daemon feature:

- `Overview`
- `Envelope`
- `Message Kinds`
- `Interaction Lifecycle`
- `Discovery and Capability Signaling`
- `Trust and Verification`
- `Transport Profiles`
- `Conformance`
- `Adoption Guide`
- `Messaging Guardrails`

### 4. Runtime Subpages

The runtime section should stay operational:

- `Overview`
- `Daemon`
- `Sessions`
- `CLI`
- `Web UI`
- `Storage`
- `Observability`
- `Network Views`

This split keeps protocol readers from being dragged through local implementation detail and keeps runtime users from mistaking the app UI for the spec.

## Adoption Story

The adoption story for third-party harnesses should be simple:

1. Keep your runtime.
2. Map your internal agent model to AGH Network envelopes.
3. Implement the smallest viable core first.
4. Add the transport profile you need.
5. Add the trust profile if you want verified interoperability.

That story is attractive for three reasons:

- It does not force a rewrite of the harness's own session or orchestration model.
- It offers a concrete interoperability target with AGH, rather than a vague "compatible" claim.
- It lets a third-party harness adopt the protocol incrementally, starting with the core semantics and stopping before trust or transport profiles if needed.

This is the same basic pattern that makes other protocol ecosystems adoptable: a small core, then optional profiles, then a clear reference implementation. The difference here is that AGH can combine the protocol with a strong runtime and a polished local UI, which gives implementers a testable, visible target without making the protocol dependent on AGH.

The right product claim is: AGH Network is the interoperable agent-network layer. AGH Runtime is the reference implementation and local operator surface.

## Messaging Guardrails

The following claims should remain careful until the protocol and site are fully stabilized:

- Call the RFCs draft or approved-design material, not finished standardization.
- Avoid saying AGH Network is already broadly adopted. It is a product surface and reference path, not a market fact.
- Do not collapse runtime messaging into protocol messaging. The web UI and daemon network views are runtime surfaces.
- Do not claim the protocol defines orchestration, workflow execution, or global federation. Those are explicitly outside the current scope.
- Do not imply that third-party harness support exists today unless a concrete implementation ships.
- Do not present the NATS binding as the whole protocol. It is a profile or binding, not the semantic core.
- Do not describe `v0` as the final design. The local corpus shows an evolving path from v0 to v1 with added trust and conformance structure.

The safest wording is "AGH Network defines the protocol boundary" and "AGH Runtime is the reference implementation." Avoid language that makes the runtime sound like the product moat and the protocol sound like a feature flag.

## Evidence

- `docs/rfcs/003_agh-network-v0.md`
- `docs/rfcs/004_agh-network-v1.md`
- `docs/plans/2026-04-08-agh-network-design.md`
- `docs/_refacs/20260414-bundle-runtime-reconcile.md`
- `web/src/routes/_app/network.tsx`
- `.resources/acpx/README.md`
- `.resources/acpx/VISION.md`
- `.resources/openfang/README.md`
- `.resources/openfang/MIGRATION.md`
- `.resources/opencode/README.md`
- `.resources/goclaw/README.md`
- `.resources/goclaw/websocket-protocol.md`
- `qmd://agent-networks/wiki/concepts/agent-network-protocol.md`
- `qmd://agent-networks/wiki/concepts/agent-discovery-and-registries.md`
- `qmd://agent-networks/wiki/concepts/agent-authentication-and-zero-trust.md`
- `qmd://agent-networks/wiki/concepts/agent-observability-and-distributed-tracing.md`
- `qmd://ai-harness/wiki/concepts/agent-communication-protocols.md`
