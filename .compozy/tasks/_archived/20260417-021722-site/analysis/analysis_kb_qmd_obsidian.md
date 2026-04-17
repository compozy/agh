# AGH Local Knowledge Research: QMD, KB, and Obsidian

## Tool Availability and Limits

- `qmd status` works and shows the useful collections: `ai-harness` and `agent-networks` are populated; `agh-compozy` and `agh-docs` are empty.
- `obsidian help` fails because Obsidian is not running in this environment.
- `kb version` works, but `kb topic list` and `kb search` fail from this repository root because `kb` cannot discover a `.kb/vault/` here.
- `qmd search` and `qmd ls` work reliably enough to mine collection structure and article summaries.
- `qmd get` hit a transient SQLite lock during this session on a few document fetches, so the analysis below leans on search results plus local repo files.

## Key Findings from QMD Collections

### `ai-harness`

- The collection is structured like a real knowledge base, not a pile of notes. Its dashboard links a `Concept Index` and `Source Index`, and the concept index explicitly points readers to related vaults such as `agent-networks`, `claude-code`, `goclaw`, `hermes`, `openclaw`, `openfang`, and `pi-mono`.
- The `Karpathy Knowledge Base Pattern` article says the workflow is corpus-first: ingest sources, compile dense concept articles, maintain cross-links, then query and file answers back into the corpus. That implies AGH docs should be treated as a maintained knowledge system, not a one-time marketing page.
- The collection’s dashboard and index pattern suggests a clean documentation IA: a hub page, a concept/reference index, and a source index that preserves provenance. That is a good match for AGH because the site needs both marketing and durable technical reference.
- The `Memory Systems for Agents` article reinforces a useful structure: outputs, queries, briefings, dashboards, and index entry points. That points toward a docs site with distinct landing areas for “start here,” “reference,” and “research/notes,” rather than one long monolith.

### `agent-networks`

- `Agent Network Protocol` positions ANP as a DID-based, peer-to-peer protocol that wants to be “the HTTP of the agentic web era.” Its differentiator is trustless, internet-scale collaboration across agents that have no prior relationship.
- The article frames ANP as layered: identity and secure communication, meta-protocol negotiation, and application-layer discovery/description. That is a strong model for AGH protocol docs because it separates concept, negotiation, and implementation concerns.
- `The MCP-A2A Composition Pattern` and the broader landscape articles frame the 2026 agent stack as layered: MCP for tools, A2A/agent protocols for agent-to-agent communication, plus adjacent trust/payment/discovery protocols. AGH should position its network protocol inside that layered stack, not as a generic runtime feature.
- `Agent-to-Agent Protocol Landscape` makes the ecosystem story explicit: the market has converged on protocol layers, and the winning documentation angle is clarity about where each protocol sits. For AGH, that means the site should distinguish the runtime from the protocol and avoid blending execution, transport, and wire semantics into one narrative.
- `The Open Agentic Web` gives the higher-level narrative: the next web layer is agent-to-agent communication. AGH can borrow that framing, but its product story should stay narrower and concrete by emphasizing a real, implementable protocol plus a reference runtime.

## Key Findings from `.compozy/*` Artifacts

- `docs/plans/2026-04-08-agh-network-design.md` is the clearest product-positioning source. It says AGH Network is open, layered, transport-agnostic at the core, and intentionally not captive to AGH. It also says the moat is the runtime, SDK, observability, and DX, not protocol lock-in.
- The same plan says the protocol keeps a lightweight lifecycle instead of a workflow engine. That is a strong docs boundary: protocol docs should explain wire semantics and conformance; runtime docs should explain orchestration, session lifecycle, and operational behavior.
- `docs/rfcs/003_agh-network-v0.md` and `docs/rfcs/004_agh-network-v1.md` show the protocol arc: v0 defines the wire format and transport binding, v1 adds trust, conformance, and extension processing. That suggests the site should have a dedicated protocol section with separate conceptual, reference, and implementation pages.
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md` repeatedly states that the network protocol is a wire layer, not a workflow engine, and that multi-agent orchestration, state handoff, compensation, and observability belong in the daemon/runtime layer. That is exactly the runtime/protocol split the site needs to reflect.
- `.compozy/tasks/_archived/20260412-040024-network/_techspec.md` reinforces the same boundary in implementation terms: the network runtime is a transport/router/correlation layer, spaces are runtime-created, and workflow concerns stay out of the protocol. It also includes explicit docs/skill guidance for the network surface.
- `.compozy/tasks/_archived/20260409-183155-web-ui-redesign/_techspec.md` shows the product surface already being split into separate user-facing areas such as Sessions, Skills, and Knowledge. That supports a docs IA with similarly distinct top-level sections instead of a single “docs” bucket.

## Implications for AGH Docs and Site Planning

- The home page should split the product into two first-class pillars: `AGH Runtime` and `AGH Network Protocol`. That split is already justified by the repo’s own plans and RFCs.
- `AGH Runtime` should own the daemon, sessions, CLI, web UI, memory, skills, observability, and orchestration story. This is the operational product that users install and run.
- `AGH Network Protocol` should own identity, discovery, envelope semantics, message kinds, trust, transport profiles, conformance, and cross-harness interoperability. This is the thing other harnesses can implement independently.
- The marketing message should not describe AGH as “just another agent harness.” It should say AGH is a runtime plus a protocol, with the protocol available for third-party implementations and the runtime serving as the reference implementation.
- Because `agh-compozy` and `agh-docs` are empty, there is no existing AGH-specific docs corpus to reuse yet. That means the immediate workflow should be: define the taxonomy, create the docs structure, then seed the empty collections with the new site content so they become searchable and maintainable.
- The best local-docs pattern to borrow is the KB-style hub plus indexes. For AGH, that likely means a docs home, a runtime docs index, a protocol docs index, and a source/reference index for RFCs, plans, and design notes.

## Evidence

### Commands

- `qmd status`
- `qmd ls ai-harness`
- `qmd ls agent-networks`
- `qmd search "wiki index concept index source index dashboard" -c ai-harness`
- `qmd search "agent networks swarms messaging protocol positioning MCP A2A" -c agent-networks`
- `qmd search "agent network protocol MCP A2A positioning" -c agent-networks`
- `qmd search "MCP A2A composition pattern" -c agent-networks`
- `kb version`
- `kb topic list`
- `kb search "AGH Network protocol docs taxonomy" --topic ai-harness`
- `obsidian help`

### Local Paths

- `docs/plans/2026-04-08-agh-network-design.md`
- `docs/rfcs/003_agh-network-v0.md`
- `docs/rfcs/004_agh-network-v1.md`
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md`
- `.compozy/tasks/_archived/20260412-040024-network/_techspec.md`
- `.compozy/tasks/_archived/20260409-183155-web-ui-redesign/_techspec.md`
- `.codex/ledger/2026-04-15-MEMORY-site-docs.md`
- `.compozy/tasks/site/analysis/`
