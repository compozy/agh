# Analysis: QMD Collections (agh-\* and related)

## Scope

This analysis covers the QMD knowledge collections that are most relevant to the AGH Agent Operating System and the AGH Network protocol drafts. The QMD `query` (hybrid) command was unavailable in this environment because the `sqlite-vec` virtual-table module failed to load (`SQLiteError: no such module: vec0`), so all retrieval was performed with `qmd ls`, `qmd search` (BM25), and `qmd get` for full-document fetches.

Sources read in full:

- `qmd://agh-rfcs-local/001-agent-md-with-skills-memory.md` (21.8 KB)
- `qmd://agh-rfcs-local/002-skills-system-final.md` (19.9 KB)
- `qmd://agh-rfcs-local/003-agh-network-old.md` (49.4 KB) — the original two-section "v1" draft that has since been split
- `qmd://agh-rfcs-local/003-agh-network-v0.md` (40.9 KB) — current v0 (transport + lifecycle, no trust)
- `qmd://agh-rfcs-local/004-agh-network-v1.md` (12.3 KB) — current v1 delta over v0 (Baseline Trust Profile)
- `qmd://claude-code/raw/articles/deepwiki-memory-extraction-and-autodream.md` (15.7 KB)

Sampled via targeted BM25 searches (representative hits inspected):

- `qmd://agent-networks/` (106 files) — A2A, ANP, NLIP, AP2, AGNTCY, ACP, ACNBP, agent-card, identity/VCs, capability negotiation, observability
- `qmd://hermes/` (93 files) — ACP adapter, subagent delegation, skills system, progress hooks, tool registry
- `qmd://ai-memory/` (173 files) — CoALA, episodic/semantic/procedural memory, complementary learning systems, memory consolidation/forgetting, agent memory benchmarks
- `qmd://claude-code/` (74 files) — AutoDream, memdir taxonomy, hook system, permission engine, plugin/skills system, token budget cascade, JSONL transcripts
- `qmd://openfang/`, `qmd://openclaw/`, `qmd://goclaw/`, `qmd://pi-mono/`, `qmd://ai-harness/` — adjacent reference implementations of skill registries, daemon lifecycles, JSONL session stores, and context engineering

The five `agh-*` site/docs/compozy collections (`agh-compozy`, `agh-docs`, `agh-site-archived`, `agh-site-ledger`, `agh-site-plans`) all index `0` files at the time of analysis — only the RFC collection has content.

Key BM25 queries that produced load-bearing hits:

- `agent harness daemon`, `ACP protocol agent client`
- `memory consolidation`, `AutoDream consolidation gates`, `memdir taxonomy user feedback project reference`
- `skills progressive disclosure SKILL.md`, `skill catalog bundled embedded marketplace`, `skillify meta-skill`, `VerifyContent skill scanning`
- `agent network protocol envelope NATS`, `verifiable credentials DID identity agent fingerprint`, `agent-to-agent A2A capability discovery card`
- `ETH Zurich context bloat`, `context engineering token budget compaction`
- `Recipe artifact agent procedure reusable`, `lifecycle hooks session created stopped event`

Cross-referenced against the live AGH repo at `/Users/pedronauck/Dev/compozy/agh/` to confirm which RFC ideas are already implemented.

---

## RFC Findings (agh-rfcs-local)

### RFC 001 — Self-Contained Agent Definitions with Scoped Skills and Memory

**Thesis.** The agent ecosystem has standards for project instructions (AGENTS.md), reusable skill bundles (SKILL.md / agentskills.io), and tool integration (MCP), but no standard for _the agent itself_. The proposal: `AGENT.md` (YAML frontmatter + Markdown prompt) as a self-contained definition format, with the agent directory (`.agents/<name>/`) as the unit of portability — including agent-scoped `skills/` and `memory/` subdirectories.

**Key decisions.**

- **Five-layer skill resolution**: bundled → global (`~/.agh/skills/` + `~/.agents/skills/`) → workspace (`.agh/skills/` + `.agents/skills/`) → `extra_sources` → agent-local (`.agents/<name>/skills/`); higher precedence wins on collision; `skills.disabled` removes named skills before merge; an override audit trail logs all shadows.
- **Three-layer memory resolution**: global → workspace → agent; same shadow rules; `memory.scope` declares the default write scope (`agent` | `workspace` | `global`).
- **Auto-consolidation** of agent memories at session end (`memory.auto_consolidate: true`) — the explicit justification quotes the ETH Zurich Feb 2026 finding that irrelevant context "increased inference costs by 20%+ while reducing task success."
- **Implementation pattern**: `Registry.ForAgent()` and `Assembler.ForAgent()` extend the existing `ForWorkspace()` shape — no new packages, all flat-architecture-friendly extensions.
- **Format philosophy**: structured frontmatter is explicitly chosen over pure prose because `provider/model/tools/permissions` are inherently structured. This deliberately diverges from the AGENTS.md "plain Markdown" convention.

**Open questions** (explicitly listed in §6):

1. Format convergence with AGENTS.md (frontmatter vs pure prose).
2. Cross-platform handling of provider-specific frontmatter fields (`claude.permissions` namespacing vs ignore-unknown).
3. Memory namespacing when a single agent directory is copied across projects.
4. Heuristic vs LLM-based memory consolidation (cost vs accuracy).
5. Skill precedence between agent-local and `extra_sources`.
6. Standardization path — extension to AGENTS.md under AAIF, or a standalone spec.

### RFC 002 — Daemon-Managed Skills with Lifecycle, MCP Bridge, and Security

**Thesis.** AgentSkills (Dec 2025, 26+ platforms) defines a _format_, not a _runtime_. Four runtime concerns are unaddressed: load-time security scanning, declarative MCP server provisioning, lifecycle hooks, and bidirectional memory integration. The proposal extends standard SKILL.md frontmatter under a `metadata.agh.*` namespace so that AGH-specific features degrade to "ignored fields" on other platforms.

**Key decisions.**

- **Security at the boundary**: `VerifyContent` runs on every non-bundled skill, every load (not just install). Three severity tiers — `Critical` (block), `Warning` (log + allow), `Info` (log only). Direct response to the **ClawHavoc** incident (Feb 2026, 1,184+ malicious skills on ClawHub; Snyk's 36.82% security-flaw figure) and to the time-of-check / time-of-use gap in registry-only scanning.
- **Bundled skills are trusted** because they ship through `go:embed` and are immutable for the process lifetime. This is the single legitimate "skip scanning" exception.
- **Declarative MCP lazy-load**: skills declare `metadata.agh.mcp_servers`. On session creation, the daemon collects, requests user consent for marketplace skills, resolves `${ENV}` substitutions only after consent, and injects servers into `StartOpts.MCPServers`. User/workspace/additional skills are auto-approved (deliberate local placement); marketplace skills require persisted consent in `skills.allowed_marketplace_mcp`.
- **Lifecycle hooks** (in current TechSpec scope): `on_session_created` and `on_session_stopped` only. `on_prompt_assembly` is **explicitly deferred**. Hooks execute in hierarchy precedence then alphabetical order; configurable timeout (default 5 s); fail-open semantics (errors logged, never block session); JSON over stdin for context.
- **Memory integration is forward-looking**: the RFC explicitly notes that `tag-filtered injection`, a `memory query API`, and `skill-guided writes` are deferred to a follow-up spec. Today, memory and skills coexist in the prompt without coupling.
- **Skill auto-proposal** ("skillify meta-skill"): detect 3+ recurring patterns across sessions, propose at session end, and use a bundled `skillify` to formalize the workflow into SKILL.md. This is positioned as the "key differentiator — the system improves itself through use."
- **Provenance**: SHA-256 captured on install and rechecked on every load; override audit trail; quarantine retains block-on-load semantics until re-approval UX exists.
- **Five-layer precedence** mirrors RFC 001: Bundled → Marketplace → User → Additional → Workspace.

**Increment status (§5).**

| Increment                                                                                          | Status       |
| -------------------------------------------------------------------------------------------------- | ------------ |
| 1: Loader, dual-scope registry, prompt injection, `VerifyContent`, CLI, bundled skills, hot-reload | **Complete** |
| 2: MCP lazy-load, lifecycle hooks, skill auto-proposal                                             | Planned      |
| 3: Marketplace, hash-based provenance, override audit trail                                        | Planned      |

Cross-checked: `/Users/pedronauck/Dev/compozy/agh/internal/skills/` confirms `verify_test.go`, `registry.go` calls `VerifyContent(content)`, plus `provenance.go`, `mcp.go`, `mcp_sidecar.go`, `hook_decl.go`. RFC 002 is partly the _driver_ and partly the _retrospective_ spec for code already in tree.

**Open questions** (§6):

1. Hook execution ordering across skills (precedence + alphabetical vs explicit dependencies).
2. MCP consent persistence (revocability, expiry, scope).
3. Memory tag taxonomy (controlled vocabulary vs freeform).
4. Auto-proposal accuracy threshold (false positives erode trust).
5. Marketplace governance balance (post-ClawHavoc).

### RFC 003-old — AGH Network v1 (combined draft, superseded)

**Thesis.** A lightweight, chat-first, artifact-aware agent-to-agent protocol that is _implementable outside AGH_ yet aligned with AGH's runtime. Three normative layers in a single RFC: `AGH Network Core` (transport-agnostic semantics), `AGH Network over NATS` (v1 transport binding), `AGH Network Baseline Trust Profile` (Ed25519 + JCS).

**Decisions.**

- **Layer separation** as the central architectural choice: "the protocol must be reusable outside AGH" _and_ "AGH should still be the best implementation."
- **Seven core message kinds**: `greet`, `whois`, `say`, `direct`, `recipe`, `receipt`, `trace`. (Note: this old draft uses `recipe`; the v0/v1 split renamed it to `capability`.)
- **Six lifecycle states**: `submitted → working → needs_input → completed | failed | canceled`. Post-terminal regression is forbidden — terminal state is authoritative.
- **Cancellation duality**: `receipt(canceled)` = initiator-side withdrawal; `trace(canceled)` = worker-side abort.
- **Trust states**: `verified` / `unverified` / `rejected`. **Proof-stripping defense**: a verified-format identity (`nickname@fingerprint`) without `proof` is `rejected`, not `unverified`.
- **NATS subject mapping**: `agh.network.v1.<channel>.broadcast` and `agh.network.v1.<channel>.peer.<route_token>`. Route token = first 32 hex of `SHA-256(peer_id)` for unverified peers, fingerprint suffix for verified.
- **Replay defense**: bounded replay window via `id`; recommended 300-second clock-skew rejection when `expires_at` is null.
- **Conformance combinations**: Core Sender / Core Receiver / Core Peer / + NATS Peer / + Verified Peer (additive).

### RFC 003-v0 — AGH Network v0 (current, implementable)

**Thesis.** Ship a wire-compatible, implementable protocol _now_ by deferring crypto. Identical envelope schema as v1; v1 layers on top of v0 by adding the Baseline Trust Profile and proof-stripping detection.

**Differences from the old combined draft.**

- The artifact kind was renamed from `recipe` to **`capability`**. A capability is now positioned as "the single AGH delegation artifact" used in three roles: discovery index in `peer_card.capabilities`, optional rich discovery via `whois`, and transferable artifact via `kind:"capability"`. Operational identity becomes `(peer_id, capability_id)`.
- `proof` is reserved on the wire but unprocessed in v0; all messages classify as `unverified`.
- `peer_id` MUST match `[a-z0-9][a-z0-9._-]{0,127}` (deterministic route-token derivation across implementations).
- AGH-specific extension keys are RECOMMENDED to use the `agh.` prefix (e.g., `agh.session_id`, `agh.workspace`, `agh.capabilities_brief`); v1 makes namespacing a MUST.
- An AGH-specific "capability brief" projection (`peer_card.ext["agh.capabilities_brief"]`) is defined as an optional, ordered companion to `peer_card.capabilities`.

### RFC 004 — AGH Network v1 (current delta)

**Thesis.** v1 is a v0 peer that additionally supports the Baseline Trust Profile, formal conformance levels, and namespaced extensions. The wire format is identical.

**Decisions.**

- **MTI cryptography**: Ed25519, RFC 8785 JCS, SHA-256 fingerprints. Profile id `agh-network.trust.ed25519-jcs/v1`.
- **Verified sender identity**: `nickname@fingerprint`, where `nickname` matches `[a-z0-9_-]{1,32}` and `fingerprint` is the first 32 lowercase hex of `SHA-256(pubkey)`.
- **Signed content** is the JCS-canonicalized envelope with `proof.sig` omitted; everything else (including `proof.profile`, `proof.alg`, `proof.key_id`, `proof.pubkey`) is inside the signature.
- **Eight verification steps** (§4.7) including matching `from`'s fingerprint to the first 32 hex of `SHA-256(pubkey)` — a self-certified handle pattern.
- **Proof-stripping defense** is the explicit motivation for "verified-format identity without proof = rejected" classification.
- **Baseline trust profile limits** (§7.1): no global trust roots, no revocation, no organization-level authorization, no federation policy. Those are explicitly future work.
- **Subject prefix bumps** to `agh.network.v1`; verified peers use fingerprint suffix as the route token.

**Open questions across the three network drafts (preserved from §16/§19 of old draft and implicit in v0/v1).**

- Future profiles for JetStream durability, federation, multi-org routing, replay/retention conventions.
- Marketplace governance for capabilities and recipes.
- A "rich" registry vs the current minimal `greet`/`whois` discovery.

---

## Cross-Collection Themes

Patterns that recur across `agh-rfcs-local`, `agent-networks`, `hermes`, `ai-memory`, `claude-code`, `openfang`, and `openclaw`:

### 1. The "harness daemon" pattern is now mainstream

Multiple reference systems converge on a long-running daemon as the right abstraction for serious agent work:

- `qmd://openfang/wiki/concepts/openfang-architecture.md` calls itself "an Agent Operating System... a 14-crate, ~137K LoC workspace... a persistent daemon with its own kernel struct holding every subsystem."
- `qmd://openfang/wiki/concepts/cli-and-daemon-lifecycle.md` contrasts daemon-based systems with Claude Code's "single-process session where every instance loads its own state, no daemon exists, and multiple instances coordinate via file-based IPC."
- `qmd://hermes/wiki/concepts/process-management.md` documents a `ProcessRegistry` for spawning, polling, and reading paginated output of long-running subprocesses — the same pattern AGH uses for ACP agents.

AGH's "single-binary daemon that is the sole composition root" is _not novel positioning_ but it _is_ a deliberate alignment with the strongest reference implementations.

### 2. Memory: CoALA + complementary learning systems is the shared canon

`qmd://ai-memory/` is dominated by two foundational frameworks Pedro has clearly internalized:

- **CoALA** (Sumers et al. 2023) — four memory types (working, episodic, semantic, procedural), four-phase decision loop (Retrieve → Reason → Act → Observe). Source: `coala-cognitive-architectures.md`, `cognitive-architectures-for-language-agents.md`, `procedural-memory-and-skill-learning.md`, `episodic-and-semantic-memory-in-ai-agents.md`.
- **Complementary Learning Systems** — hippocampus (fast, episodic) vs neocortex (slow, semantic); the basis for "dream consolidation." Sources: `complementary-learning-systems.md` (Kumaran/Hassabis/McClelland 2016; CLS revisited; arXiv:2504.14727; arXiv:2507.11393; arXiv:2508.16651).

Both frameworks are reflected in the AGH `memory/consolidation` package name and in the RFC 001 `memory.scope: agent|workspace|global` design (which approximates working/episodic/semantic separation by _scope_ rather than by _type_).

### 3. AutoDream is the model for AGH's `memory/consolidation`

`qmd://claude-code/raw/articles/deepwiki-memory-extraction-and-autodream.md` describes Claude Code's `AutoDream` service. The vocabulary alignment with AGH is exact:

- "AutoDream is a background consolidation service that runs when specific 'gates' are passed" — three gates evaluated in order of computational cost: **Time Gate** (default 24h since last consolidation), **Session Gate** (default 5 sessions touched), **Lock Gate** (`tryAcquireConsolidationLock` to prevent multi-instance races).
- Memory taxonomy: `user`, `feedback`, `project`, `reference`.
- Path security: `sanitizePathKey` rejects null bytes, URL-encoded traversals, Unicode normalization attacks; `realpathDeepestExisting` resolves symlinks for the deepest existing ancestor to defeat symlink-escape.
- Consolidation runs as a "forked agent" via `runForkedAgent` driving a `DreamTask`.

This is not "inspired by" — it is the _direct ancestor_ of `internal/memory/consolidation/` and the "dream triggers" reference in `CLAUDE.md`'s package layout.

### 4. Skills as a four-pillar problem

The skills articles across `hermes`, `claude-code`, `openfang`, and `openclaw` consistently identify four concerns that the AgentSkills spec explicitly does _not_ address:

| Concern                          | Hermes                        | Claude Code                             | OpenFang                   | RFC 002                                                                   |
| -------------------------------- | ----------------------------- | --------------------------------------- | -------------------------- | ------------------------------------------------------------------------- |
| Progressive disclosure (3 tiers) | yes (`prompt_builder.py`)     | yes (plugins/skills)                    | yes                        | yes (inherited)                                                           |
| Static security scanning         | `tools/skills_guard.py` regex | n/a                                     | signature verification     | `VerifyContent` (Critical/Warning/Info)                                   |
| Skill ↔ MCP coupling             | n/a                           | "complementary layers" but disconnected | `openfang-skills` registry | declarative `metadata.agh.mcp_servers`                                    |
| Lifecycle participation          | n/a                           | hook system (25+ events)                | n/a                        | `on_session_created`/`on_session_stopped` (deferred `on_prompt_assembly`) |

RFC 002 is the only spec that addresses all four under a single, portable frontmatter namespace.

### 5. The 2026 agent-network protocol stack converged on layering

`qmd://agent-networks/wiki/concepts/agent-to-agent-protocol-landscape.md` (April 2026) describes consolidation under Linux Foundation governance: A2A, ANP, MCP, ACP, NLIP, LOKA, ACNBP, AGNTCY, AP2, x402. The _layered envelope + transport binding + trust profile_ shape RFC 003 adopts mirrors NLIP's "envelope layer that other protocols assume without standardizing" (`nlip-and-agent-communication-standards.md`), AGNTCY's "Internet of Agents" infrastructure split, and A2A's Agent Cards.

The RFCs are explicitly aware of this: §16 and §17 of the old 003 draft trace 18 directly-consulted concept notes.

### 6. Self-certified identity is the pragmatic minimum

`qmd://agent-networks/wiki/concepts/ap2-mandates-and-cryptographic-authorization.md` documents the typical pattern: `did:key`/`did:web`/`did:ion` issuer + Ed25519/ECDSA-P256 proof. RFC 004 deliberately _avoids_ DIDs and revocation infrastructure, picking the smaller "self-certified handle" (`nickname@fingerprint`) pattern. The Baseline Trust Profile is positioned as a _floor_ for verified interoperability, not an aspirational federation story.

### 7. The "context bloat tax" is a recurring decision driver

`qmd://ai-harness/wiki/concepts/context-engineering.md`, `qmd://claude-code/wiki/concepts/token-budget-and-context-compaction.md`, and the ETH Zurich citation in RFC 001 all reach the same conclusion: irrelevant context is expensive _and_ harmful. This justifies (a) agent-scoped skills/memory in RFC 001, (b) tag-filtered memory injection (deferred) in RFC 002, (c) lightweight peer cards instead of fat capability catalogs in RFC 003, (d) auto-consolidation, (e) compaction cascades.

---

## How Reference Material Informed AGH

Concrete, evidence-backed lineage:

1. **`internal/memory/consolidation` ← Claude Code AutoDream.** The package name, "dream triggers" reference in `CLAUDE.md`, the gates (time → sessions → lock), and the consolidation-by-forked-agent pattern are direct lifts from `deepwiki-memory-extraction-and-autodream.md`. AGH's `memory/MEMORY.md` index file mirrors `memdir`'s layout.
2. **`internal/skills/VerifyContent` ← RFC 002 ← ClawHavoc.** `verify_test.go` exists in tree; `registry.go` calls `VerifyContent(content)`. The Critical/Warning/Info severity tiers are 1:1 with RFC 002 §2.2. The RFC explicitly names the ClawHavoc Feb 2026 incident as the trigger.
3. **`internal/skills/mcp.go` + `mcp_sidecar.go` ← RFC 002 declarative MCP lazy-load.** The `metadata.agh.mcp_servers` frontmatter pattern with consent gating for marketplace skills is in the spec and the implementation.
4. **`internal/skills/hook_decl.go` ← RFC 002 lifecycle hooks.** The two-event scope (`on_session_created`, `on_session_stopped`) and JSON-over-stdin protocol are the implementation contract.
5. **`internal/skills/provenance.go` ← RFC 002 §2.7 hash-based provenance.** Marketplace SHA-256 capture + recheck on load is in tree.
6. **Five-layer skill precedence (Bundled → Marketplace → User → Additional → Workspace) ← RFC 002 §2.8.** `registry.go`, `registry_workspace_cache.go`, `loader.go` together implement this hierarchy.
7. **Daemon-as-composition-root in `internal/daemon/` ← OpenFang's "kernel struct holding every subsystem" pattern.** OpenFang's tools-and-hands article and 14-crate workspace article are the closest reference architecture.
8. **`internal/api/contract/` + `udsapi/` + `httpapi/` split ← Hermes ACP adapter pattern.** Hermes exposes "two orthogonal forms of agent composition: the ACP adapter ... and the `delegate_task` tool." AGH's UDS-for-CLI / HTTP-SSE-for-web split matches the same orthogonality (transport layer is independent of session model).
9. **`internal/store/sessiondb/` JSONL-style event store ← OpenClaw + pi-mono session models.** Both reference systems use per-session JSONL persistence; AGH chose SQLite per-session for indexability but the eventing model (record everything, replay deterministically) is the same.
10. **Memory taxonomy `user/feedback/project/reference` ← Claude Code memdir.** The MEMORY.md auto-memory file in user state already uses this exact taxonomy (visible in the Pedro-MEMORY.md context block).
11. **AGH Network protocol shape ← NLIP envelope + A2A capabilities + AGNTCY layering.** The "envelope + transport profile + trust profile" three-layer structure of RFCs 003/004 is the consensus shape from the agent-networks corpus, distilled to the smallest implementable subset.
12. **Self-certified `nickname@fingerprint` ← agent-networks DID/VC literature, simplified.** The RFCs deliberately _do not_ adopt DIDs; they pick the smallest pattern that gives proof-stripping protection.

---

## Skill Candidates

Skills that would help apply RFC/reference knowledge to current AGH work:

1. **`agh-network-rfc-author`** — encapsulates the layered RFC structure (Core / Transport / Trust), the v0/v1 wire-compat constraint, the seven canonical message kinds, lifecycle state machine, NATS subject grammar, and JCS+Ed25519 verification steps. Use when authoring new transport profiles (JetStream, WebSocket) or trust profiles. Pulls from RFC 003-v0 + RFC 004 + `agent-networks/agent-to-agent-protocol-landscape.md`.
2. **`agh-skills-runtime-extension`** — captures RFC 002's `metadata.agh.*` namespace pattern, the five-layer precedence, `VerifyContent` severity tiers, hook contract (stdin JSON + fail-open + alphabetical+precedence ordering), and consent gates for marketplace MCP. Use when adding new lifecycle events or trust tiers.
3. **`agh-memory-consolidation-design`** — encodes the AutoDream gate cascade (Time → Sessions → Lock), forked-agent execution, four-type memory taxonomy (`user/feedback/project/reference`), three-scope writes (`agent/workspace/global`), and the path-security pattern (`sanitizePathKey`, `realpathDeepestExisting`). Use when extending `internal/memory/consolidation/`.
4. **`agh-agent-md-author`** — RFC 001's frontmatter schema, skills/memory inheritance and disable patterns, `extra_sources` precedence, portability invariants ("directory is the unit of portability"). Use when scaffolding new agent definitions or designing CLI commands like `agh agent create`.
5. **`agh-context-budget-discipline`** — distills the "20%+ inference cost penalty for irrelevant context" finding from ETH Zurich (Feb 2026) plus the five-layer compaction cascade from Claude Code into a checklist for any prompt-assembly or memory-injection change.
6. **`agh-ecosystem-positioning`** — single-source-of-truth on what AGH is _not_ (workflow engine, federation protocol, MCP replacement, A2A replacement) and what it competes on (runtime/SDK/observability/DX _outside_ the wire protocol). Critical for marketing-site and docs work because the agh-site-\* collections are currently empty.

---

## Lesson-Learned Candidates

Insights from the RFCs that should be lessons for future work:

1. **"Format, not runtime" is the recurring trap.** AgentSkills, AGENTS.md, and A2A Agent Cards are all _file formats_ without runtime governance. AGH's repeated decision is to _extend_ (not fork) the format and _add the runtime_ (security scanning, hooks, MCP provisioning, lifecycle, observability). Future AGH work should ask "is the upstream a format or a runtime?" before deciding to compose vs replace.
2. **Defer crypto to ship.** RFC 003 originally tried to ship Core + NATS + Trust together. The current v0/v1 split is the lesson: define the wire envelope so v0 → v1 is purely additive, then ship v0 immediately. Wire-compatibility-first beats crypto-first.
3. **Proof-stripping is a real attack class.** The "verified-format identity without proof = rejected" rule (RFC 004 §3.3) is non-obvious; the natural default is "treat absence as unverified." That default is exploitable. Always classify identity-shape-without-proof as a verification _failure_.
4. **Bundled is the only legitimate skip-scanning tier.** RFC 002's "bundled skills are trusted because `go:embed` makes them immutable" is the _only_ defensible exception to load-time scanning. Any future "trusted" tier must be defended with the same level of immutability proof.
5. **Auto-consolidation needs gates, not heuristics.** Claude Code's three-gate cascade (time → sessions → lock) prevents both over-consolidation (cost) and races (corruption). A naive "consolidate at session end" would be wrong. Gates are cheap to evaluate and orderable by cost — this is the correct shape.
6. **Five-layer precedence is the right number.** Both RFC 001 (skills) and RFC 002 (skills) and the implicit memory hierarchy converge on 5 layers (Bundled → Marketplace → User → Additional → Workspace, or Bundled → Global → Workspace → ExtraSources → Agent). Six is too many to reason about; three loses important distinctions (especially Marketplace vs User trust tiers). Use 5.
7. **"Capability" beats "Recipe" as the artifact name.** The 003-old → 003-v0 rename is instructive: an artifact named after _what it does_ (capability) generalizes; an artifact named after _what it looks like_ (recipe) collapses into "a workflow program in disguise." The old draft's repeated "interpretive, not deterministic" disclaimers were a code smell that the rename eliminated.
8. **Self-certified handles are a pragmatic floor.** Instead of trying to bootstrap DIDs, VCs, and revocation, AGH picks `nickname@fingerprint` and explicitly disclaims revocation/federation as future-profile work. This is a defensible MVP shape.
9. **Greenfield disambiguates "extend" from "fork."** Pedro's CLAUDE.md "Greenfield Alpha — Zero Legacy Tolerance" rule is the right lens for these RFCs: when the format is upstream, _extend_ with namespaced metadata; when the implementation is internal, _replace_ the old thing rather than work around it.
10. **State the runtime moat explicitly.** RFC 003-old §4.5 ("Product boundary") is unusually candid: "AGH competes on runtime, SDK, observability, and DX rather than on making the wire protocol private." Explicitly stating the moat helps every future decision (what to standardize, what to keep proprietary, what to open-source).

---

## System Prompt Candidates for CLAUDE.md

Rules implied by the RFCs that aren't yet in the project `CLAUDE.md`:

1. **RFC discipline.** _"All RFCs in `.compozy/tasks/` and `agh-rfcs-local` are layered: every RFC must declare which layer it belongs to (envelope, transport, trust, runtime, lifecycle) and what it depends on. Wire-format changes require wire-compatibility justification."_
2. **Format extension default.** _"When integrating with an external spec (AgentSkills, AGENTS.md, MCP, A2A), extend via a namespaced metadata field (`metadata.agh._`or`agh._`) — never fork the format. AGH-specific features must degrade gracefully on platforms that ignore unknown fields."_
3. **Five-layer precedence rule.** _"Skill, memory, and agent resolution use a 5-layer precedence (Bundled → Marketplace → User → Additional → Workspace, with agent-local overriding all). Higher precedence wins on collision; an audit trail must log every shadow."_
4. **Load-time security scan.** _"Every non-bundled skill must be scanned via `VerifyContent` on every load (not just install). Critical findings block loading; warning findings log; info findings log silently. Bundled skills are exempt because `go:embed` provides immutability."_
5. **Memory consolidation gates.** _"Background consolidation runs only when all gates pass in order of computational cost: Time Gate → Session Gate → Lock Gate. Default gates: 24h, 5 touched sessions, file-lock. Never replace gates with naive heuristics."_
6. **Memory taxonomy.** _"User-facing memories use the four-type taxonomy `user | feedback | project | reference`, written to scopes `agent | workspace | global`. The default write scope is declared per agent in `memory.scope`."_
7. **Identity proof-stripping defense.** _"In any signed-message processing path, an identity in verified format (`nickname@fingerprint`) without a valid proof must classify as `rejected`, not `unverified`. Test every code path for this case."_
8. **Capability vs Recipe naming.** _"Reusable agent artifacts are called `capability`, not `recipe`, `workflow`, `procedure`, or `playbook`. Capabilities are interpretive, not deterministic; they are not workflow programs."_
9. **Runtime moat statement.** _"AGH's competitive surface is runtime, SDK, observability, DX, and integration depth — not the wire protocol. The wire protocol must remain implementable outside AGH. Any feature that requires AGH to interoperate is a design smell."_
10. **Context-budget hygiene.** _"Any change that adds content to a prompt must justify the inclusion against the ETH Zurich Feb 2026 finding (irrelevant context = +20% inference cost, lower task success). Default to scoped, tag-filtered injection rather than full-file dumps."_
11. **Path security helpers.** _"All filesystem helpers that resolve user-controlled or agent-controlled paths must use the `sanitizePathKey` + `realpathDeepestExisting` pattern (defenses against null-byte, URL-encoded traversal, Unicode normalization, and symlink-escape attacks)."_
12. **Hook semantics are fail-open.** _"Lifecycle hooks (`on_session_created`, `on_session_stopped`) execute in hierarchy precedence then alphabetical order. Hook errors log as warnings and never block the session. Hooks receive context as JSON over stdin."_

Several of these are partially reflected in the project's "Greenfield Alpha" stance and "Zero Legacy Tolerance" rule but are not stated explicitly enough to act as guardrails for new contributors or new agents.

---

## Open Questions / Tensions

Things the RFCs leave unresolved that are now visible across the corpus:

1. **The `recipe` vs `capability` rename is partial in the corpus.** RFC 003-old still references `recipe` as a first-class kind; RFC 003-v0 / 004 use `capability`. Several references in the network drafts (research corpus, knowledge-base concept notes) still cite "agora-recipe-design.md." A canonical glossary entry resolving this is missing.
2. **AGENT.md vs AGENTS.md namespace conflict.** RFC 001 deliberately diverges from AGENTS.md by using YAML frontmatter; the standardization-path question (§6.6) is unresolved. The risk is real because the agh-site-\* collections are empty, meaning the public-facing positioning isn't yet pinned.
3. **Memory integration is doubly-deferred.** RFC 002's deep memory integration (tag-filtered injection, memory query API, skill-guided writes) is deferred, _and_ `on_prompt_assembly` is deferred — but memory tag injection requires `on_prompt_assembly` to work. Either the deferral chain is consistent (both ship together later) or it's a hidden coupling that will create rework.
4. **Marketplace governance is unresolved.** Post-ClawHavoc, RFC 002 §6 leaves "manual review vs automated scanning vs combination" open. The current code has the load-time scanner but no marketplace UX, no consent revocation, no expiry.
5. **Agent identity portability vs project memory namespacing.** RFC 001 §6.3 acknowledges that copying an agent directory to multiple projects diverges memories naturally — but this is not the same as agent identity _across_ AGH instances on the network (RFC 003 `nickname@fingerprint`). The two identity models are not unified.
6. **No JetStream profile yet.** RFC 003 v0 §11.10 explicitly defers JetStream durability, dead-letter, ACL, account/tenancy. These are real operational needs for any production AGH-Network deployment. The RFC-as-MVP shape is intentional but the next-profile sequencing is undefined.
7. **The `agh-compozy/`, `agh-docs/`, `agh-site-*` collections are empty.** Public docs/site/ledger/plans collections all index 0 files. The Fumadocs site project (per Pedro's auto-memory: "Approved techspec: Fumadocs site at agh.network") has no captured artifacts yet — there's no public-facing record corresponding to the depth of internal RFC work.
8. **No A2A Agent Card mapping defined.** RFC 001 §3.3 ("vs A2A Agent Cards") notes that "Agent Cards could be _generated from_ an AGENT.md definition" but there's no defined mapping. If AGH is going to publish AGENT.md as a portable format and also speak AGH-Network (which exposes Peer Cards), the relationship between AGENT.md / Peer Card / A2A Agent Card needs to be pinned.
9. **`recipe` artifact still appears in the AGH-Network non-normative section of the old draft as "first-class," but it has no SKILL.md/AGENT.md analogue in the v0/v1 RFCs.** Is a capability the same thing as a skill exposed over the network? This is hinted at but not stated.
10. **Skill auto-proposal accuracy.** RFC 002 §2.6 says "3+ occurrences across different sessions" is the threshold but acknowledges in §6.4 that false positives erode trust. There is no calibration data, no test plan, no opt-out UX. This is the most ambitious item in increment 2 and the least specified.

---

## Notes for Synthesis

- The five-RFC bundle is internally coherent and reflects roughly the same architectural philosophy that drives `CLAUDE.md`: pragmatic flat layout, runtime-as-composition-root, format-extension over format-fork, security-as-default, no fire-and-forget concurrency, no workflow-engine creep.
- Cross-references to the larger reference collections (`agent-networks`, `claude-code`, `hermes`, `ai-memory`) are not decorative — they are load-bearing. RFC 002's "ClawHavoc" justification, RFC 001's "ETH Zurich Feb 2026" justification, and the entire 003/004 layered-protocol shape would be hard to defend without that corpus.
- The RFCs are _substantially_ implemented in tree (RFC 002 increment 1; RFC 001 partially via `internal/skills/registry_workspace_cache.go` + `internal/memory/`; the `memory/consolidation` package; `verify_test.go`; `provenance.go`; `mcp.go`; `hook_decl.go`). The RFCs are doing both _driver_ and _retrospective documentation_ duty, which is fine for greenfield but suggests the docs site (currently empty) should publish them as-is once stabilized.
- The biggest _unwritten_ document the corpus suggests AGH needs is a **glossary / canonical positioning document** that pins down: AGENT.md vs AGENTS.md, `capability` vs `recipe` vs `skill`, AGH-Network Peer Card vs A2A Agent Card vs AGENT.md frontmatter, and the "what AGH is not" list. This belongs on the marketing/docs site once those collections are populated.
- Skills and lessons proposed above can be authored mechanically from the RFC content; the system-prompt candidates are higher-leverage because they convert latent design rules into enforceable guardrails for future contributors and agents.
