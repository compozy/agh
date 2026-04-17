# AGH Site and Documentation Strategy Synthesis

## Summary

The research converges on one structural conclusion: AGH should not present itself as a single undifferentiated harness product. The site should introduce **two first-class surfaces**:

1. **AGH Runtime**: the local-first daemon and operator surface for running, governing, and observing AI agent work.
2. **AGH Network Protocol**: the interoperable agent-to-agent protocol that other harnesses can implement independently.

This split is not just information architecture. It is the correct product narrative. The runtime explains how people run AGH today. The protocol explains why AGH matters beyond its own runtime.

## Working assumptions

### Document type

This synthesis is an **explanation + strategy document**, not final site copy and not implementation planning.

### Target audiences

There are two primary audiences:

1. **Runtime operators**: developers and teams who want to install AGH, run agents locally, and control sessions, memory, skills, and automation.
2. **Harness builders / protocol adopters**: teams who may keep their own runtime but want an interoperable network layer for agent-to-agent messaging.

### User goals

- Runtime operators want to understand what AGH does, why it is different, and how to get to first value quickly.
- Protocol adopters want to understand what AGH Network is, what problem it solves, and how to implement or evaluate it without adopting AGH wholesale.

### Scope

This document defines:

- homepage positioning
- core selling points
- homepage information architecture
- documentation taxonomy
- the recommended planning artifact sequence

This document does not define:

- final long-form documentation page content
- framework/tooling decisions for the site
- page-by-page implementation tasks

## Product framing

### AGH Runtime

AGH Runtime is the local control plane for AI agents. It is the daemon, CLI, API, and web UI that manages sessions, streams events, persists history, attaches workspace-aware memory and skills, and exposes advanced operational surfaces such as automation, bridges, observability, and extensions.

The runtime should carry the practical product promise:

- single binary
- local-first
- no sidecars
- multi-agent session management
- replayable and observable runtime
- workspace-aware memory and skills
- one operator surface across CLI, HTTP/SSE, and web UI

### AGH Network Protocol

AGH Network is the interoperability layer. It defines envelope semantics, message kinds, interaction lifecycle, discovery, trust, transport profiles, and conformance without requiring other projects to adopt AGH Runtime.

The protocol should carry the ecosystem promise:

- cross-harness interoperability
- reference implementation already shipping inside AGH Runtime
- transport/profile split instead of one captive stack
- room for third-party implementations

## Homepage strategy

### Core positioning

The homepage should answer this question immediately:

> Why should someone care about AGH if many harnesses already manage agents?

The answer is:

- **AGH Runtime** gives operators a durable, local-first runtime for agent work.
- **AGH Network** gives the ecosystem an interoperable protocol that is bigger than one runtime.

### Copy direction

The homepage should not try to sell every subsystem. It should sell a clear two-part story:

1. Run agents with control, continuity, and observability.
2. Adopt an open network protocol that other harnesses can implement.

### Draft homepage headline options

#### Option A

**The local runtime for AI agents, with an open network protocol for the agentic web.**

Why it works: it gives equal weight to the installable product and the ecosystem differentiator.

#### Option B

**Run AI agents locally. Connect them through an open protocol.**

Why it works: very direct, short, and action-oriented.

#### Option C

**AGH Runtime for operators. AGH Network for interoperability.**

Why it works: explicit split, best when the audience is already technical.

### Draft subheadline

**AGH is a single-binary, local-first agent runtime with durable sessions, memory, skills, automation, and observability. It also ships the reference implementation of AGH Network, an agent-to-agent protocol other harnesses can adopt.**

### Supporting proof points

- Single binary. No sidecars. No external control plane.
- Durable sessions with event history, replay, and resume.
- Workspace-aware memory and skills.
- One runtime exposed through CLI, API, and web UI.
- AGH Network as a separate protocol surface, not a runtime-only feature.

### Primary CTAs

- **Run AGH Runtime**
- **Read the Protocol**

### Secondary CTAs

- **Get Started**
- **View Runtime Docs**
- **Read AGH Network RFCs**
- **See Conformance Model**

## Homepage information architecture

### Section 1: Hero

State the runtime + protocol split immediately. Avoid hiding the protocol below the fold.

### Section 2: Two-pillars split

Two side-by-side or stacked bands:

- **AGH Runtime**
- **AGH Network Protocol**

Each should have:

- one-sentence description
- three or four key benefits
- one CTA

### Section 3: Runtime capabilities

Focus on launch-worthy runtime differentiators:

- local daemon
- session lifecycle
- replayable event history
- memory
- skills
- automation and bridges

This section should stay operator-oriented, not architectural.

### Section 4: Protocol differentiator

Explain that AGH Network is not “runtime messaging.” It is the protocol boundary for cross-harness interoperability.

Key points:

- independent core semantics
- profiles and transport bindings
- trust / conformance path
- reference implementation in AGH Runtime

### Section 5: Two adoption paths

- **I want to run AGH**
- **I want to implement AGH Network**

This reduces ambiguity and routes both audiences to the correct docs surface.

### Section 6: Docs / learn paths

Surface the documentation entry points directly from the homepage:

- Runtime quickstart
- Runtime concepts
- Runtime reference
- Protocol overview
- Protocol reference
- RFCs and design notes

## Documentation taxonomy

The docs should follow a Diataxis-friendly split while preserving the product boundary between Runtime and Protocol.

### Top-level site structure

1. **Home**
2. **AGH Runtime**
3. **AGH Network Protocol**
4. **Docs**
5. **RFCs**

### Docs landing page

The docs landing page should be a router, not a long narrative page.

It should send readers into two doc trees:

- **Runtime Docs**
- **Protocol Docs**

### Runtime docs taxonomy

The runtime docs should not use only generic top-level buckets such as `Concepts`, `How-to`, and `Reference`. AGH already has enough explicit product areas that the navigation should be **domain-first**, with Diataxis applied inside each domain.

#### Recommended top-level runtime navigation

- `Getting Started`
- `Runtime Overview`
- `Sessions`
- `Agents`
- `Skills`
- `Memory`
- `Workspaces`
- `Automations`
- `Bridges`
- `Extensions`
- `ACP Drivers`
- `Observability`
- `CLI & API Reference`
- `Architecture`

#### What each runtime category should own

##### Getting Started

- Install and bootstrap
- Start the daemon
- Create the first session
- Send the first prompt
- Open the web UI
- Resume and inspect a session

##### Runtime Overview

- What AGH Runtime is
- How the daemon, CLI, API, and web UI relate
- Operator model
- Local-first and single-binary posture

##### Sessions

- Session lifecycle
- Prompting and streaming
- History, replay, and transcripts
- Stop, resume, wait, and inspect flows

##### Agents

- Agent definitions
- Agent selection and defaults
- Agent identity and metadata
- How agents map to sessions and drivers

##### Skills

- Bundled vs user vs workspace skills
- Skill loading and precedence
- Marketplace/install/update flows
- Skill-local MCP sidecars

##### Memory

- Global vs workspace memory
- Write/read/delete flows
- Consolidation
- Memory behavior in the runtime

##### Workspaces

- Workspace registration and resolution
- Config overlays
- Workspace-aware runtime behavior

##### Automations

- Jobs
- Triggers
- Runs
- Scheduled or event-driven execution

##### Bridges

- Bridge instances
- Routes
- Delivery behavior
- Target testing and operational guidance

##### Extensions

- Extension discovery
- Install/enable/disable/update flows
- Extension boundaries vs skills

##### ACP Drivers

- ACP compatibility model
- Supported drivers / harnesses
- Subprocess and JSON-RPC behavior
- Driver-specific operational notes

##### Observability

- Health
- Event streams
- Reconciliation
- Audits and runtime visibility

##### CLI & API Reference

- CLI command reference
- HTTP / SSE / UDS API reference
- Config reference
- Schemas and field reference

##### Architecture

- Runtime architecture
- Composition root
- Persistence model
- Observability model
- Package map
- Web UI to runtime mapping

#### Diataxis inside each domain

Each major runtime category should expose its own tutorial/how-to/reference/explanation split.

Example for `Skills`:

- Tutorial: install the first skill
- How-to: disable a workspace skill
- Reference: metadata and precedence rules
- Explanation: why AGH treats skills as runtime assets

Example for `Automations`:

- Tutorial: create the first automation
- How-to: debug a failing trigger
- Reference: job/trigger/run fields
- Explanation: how automation fits into the runtime model

### Protocol docs taxonomy

#### Overview

- What AGH Network is
- What it is not
- Why it exists separately from AGH Runtime

#### Concepts

- Envelope
- Message kinds
- Interaction lifecycle
- Discovery and capability signaling
- Trust model
- Profiles and bindings
- Conformance

#### How-to / Adoption

- Implement the core in another harness
- Map runtime events to envelopes
- Add a transport profile
- Add verified-mode support
- Test interoperability with AGH Runtime

#### Reference

- Field-by-field envelope reference
- Message kind reference
- Reason codes and semantics
- Trust profile reference
- Transport binding reference
- Conformance classes

#### Explanation

- Why the protocol is separate from runtime internals
- Why the lifecycle stays lightweight
- Why NATS is a profile rather than the whole protocol
- How AGH Runtime relates to the spec

### RFCs and source material

The protocol branch should expose RFCs directly, because implementers will want the primary design documents:

- `003_agh-network-v0`
- `004_agh-network-v1`
- related design notes

The docs system should also preserve provenance using a KB-like pattern:

- landing page
- index page(s)
- source / RFC index

## Naming and consistency notes

- Prefer **Memory** as the canonical user-facing term. The current `knowledge` route naming in the web app should be treated as an implementation detail or transitional label.
- Treat **AGH Runtime** and **AGH Network** as official product names in the site hierarchy.
- Avoid calling the protocol just “networking” or “channels,” because that collapses the spec into one runtime feature.

## Messaging guardrails

- Do not imply broad external adoption before it exists.
- Do not present the runtime network UI as if it were the protocol spec.
- Do not claim the protocol standardizes orchestration, workflows, or global federation.
- Do not make the NATS profile sound like the entire protocol.
- Do not bury the protocol under generic harness feature copy.

## Recommendation: PRD or TechSpec?

The right sequence is:

1. **Create a focused PRD first**
2. **Create a TechSpec second**

### Why not TechSpec first

Right now the main open work is not implementation detail. It is product definition:

- audience split
- site goals
- primary CTAs
- homepage narrative
- runtime vs protocol boundary
- documentation taxonomy
- non-goals

That is PRD territory. A TechSpec written first would force implementation decisions before the content model and product framing are approved.

### What the PRD should cover

- site purpose
- target audiences
- runtime vs protocol split
- homepage goals and conversion paths
- docs taxonomy
- required content surfaces
- explicit non-goals

### What the TechSpec should cover afterward

- site framework and content pipeline
- docs source format
- versioning and RFC surfacing
- search/index strategy
- how repo docs map into the new site
- how to seed QMD collections after taxonomy approval

## Recommended next artifact

Create `.compozy/tasks/site/_prd.md` first, with the site/docs strategy positioned as:

- a product-facing site refresh
- a dual-surface documentation system
- runtime and protocol as separate but connected product lines

After PRD approval, create `.compozy/tasks/site/_techspec.md` for implementation.

## Source analyses

- `analysis_resources_docs.md`
- `analysis_kb_qmd_obsidian.md`
- `analysis_runtime_capabilities.md`
- `analysis_network_protocol.md`
