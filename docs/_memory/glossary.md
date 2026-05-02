# AGH Glossary

Canonical vocabulary for AGH and AGH Network. When the corpus is ambiguous (older RFC drafts, older `.codex/ledger` entries, internal Slack/notes), this document is authoritative.

---

## Core Concepts

### Capability

The single canonical name for **reusable agent artifacts** that describe transferable delegation offers, network discovery shapes, and AGH artifacts shipped between peers.

A capability is **interpretive**, not deterministic — it tells an agent what is available, not how to execute a deterministic program.

**Forbidden synonyms:** `recipe` (used in pre-rename RFC 003-old), `workflow`, `procedure`, `playbook`. If you find these in code, docs, or task artifacts targeting current behavior, rename them.

**Source:** RFC 003-v0 (`.../agh-rfcs-local/003-agh-network-v0.md`) renamed `recipe` → `capability`. RFC 004 enforces.

**Operational identity:** `(peer_id, capability_id)`.

---

### Skill

A **bundled procedural instruction** that an AGH session can activate before doing work. Skills are local to AGH (loaded via `internal/skills`), governed by `metadata.agh.*` frontmatter, scanned via `VerifyContent`, may declare MCP servers and lifecycle hooks.

**Skills vs. Capabilities:** Skills live inside an AGH instance and govern an agent's behavior locally. Capabilities cross AGH instances over the network and describe what an agent offers to peers. A skill could be exposed as a capability, but they are not the same artifact.

---

### Sandbox

The AGH execution boundary selected for a workspace or session. A sandbox profile is configured under `[sandboxes.<name>]`, selected by `sandbox_ref` or runtime flags, and carried through the session lifecycle as sandbox metadata.

Implemented providers are `local` and `daytona`. Provider lifecycle surfaces use `sandbox.prepare`, `sandbox.ready`, `sandbox.sync.before`, `sandbox.sync.after`, and `sandbox.stop` hooks, plus the extension Host API methods `sandbox/list`, `sandbox/info`, and `sandbox/exec`.

Do not call this product feature an `environment`. Reserve `environment`, `env`, and `environment variable` for process-level variables and operating-system context.

---

### AGENT.md (frontmatter format)

Self-contained agent definition: YAML frontmatter (provider/model/tools/permissions) + Markdown prompt. The agent directory `.agents/<name>/` is the unit of portability — it includes agent-scoped `skills/` and `memory/`.

**Status:** Proposed in RFC 001 (`.../agh-rfcs-local/001-agent-md-with-skills-memory.md`). Not yet fully implemented.

**vs AGENTS.md (project file)**:

- `AGENTS.md` = project-level instructions (industry convention, plain Markdown).
- `AGENT.md` = single-agent definition (AGH proposal, structured frontmatter).
- Different filenames, different purposes. Do not conflate.
- The standardization path (extension to AGENTS.md under AAIF, vs. standalone) is open per RFC 001 §6.6.

---

### Peer Card

The AGH Network discovery artifact: a peer's identity, addressable transport hints, and `peer_card.capabilities` index, optionally with `peer_card.ext["agh.capabilities_brief"]` for AGH-specific projection.

**vs A2A Agent Card:** A2A Agent Cards are an external industry standard. Peer Cards are AGH-Network specific but could be generated FROM an AGENT.md definition (RFC 001 §3.3 is open on the mapping). Today they are not unified.

---

## Identity

### `peer_id`

A network-scoped identifier matching `[a-z0-9][a-z0-9._-]{0,127}`. Deterministic — used to derive route tokens.

### `nickname@fingerprint`

The verified-format identity in AGH Network v1. `nickname` matches `[a-z0-9_-]{1,32}`; `fingerprint` is the first 32 lowercase hex of `SHA-256(pubkey)`.

**Critical:** A `nickname@fingerprint`-shaped identity arriving WITHOUT a valid `proof` MUST classify as `rejected`, NOT `unverified`. This is the proof-stripping defense from RFC 004 §3.3.

### `route_token`

NATS subject suffix derived from peer identity:

- Unverified peers: first 32 hex of `SHA-256(peer_id)`.
- Verified peers: fingerprint suffix.

### Caller Identity (operational)

Inside AGH, agent-facing CLI commands resolve identity from `AGH_SESSION_ID` / `AGH_AGENT` through `internal/agentidentity`. **Operator endpoints MUST NOT infer agent identity from environment variables.** Agent → identity-implicit. Operator → identity-explicit.

---

## Network Wire (RFCs 003-v0, 004)

### Message Kinds (MVP allowlist)

The seven canonical kinds: `greet`, `whois`, `say`, `direct`, `capability`, `receipt`, `trace`.

For coordination channels (autonomy MVP): `status`, `request`, `reply`, `blocker`, `handoff`, `result`, `review_request`.

Future-RFC kinds explicitly NOT in MVP: `contract-net`, `multi-home`, `vote`, `react`, `escalate`, `offer`, `accept`, `decline`, complex mention routing.

### Lifecycle States

`submitted → working → needs_input → completed | failed | canceled`. Post-terminal regression is forbidden.

### Cancellation Duality

- `receipt(canceled)` = initiator-side withdrawal.
- `trace(canceled)` = worker-side abort.

### Trust States

`verified` / `unverified` / `rejected`. Default classification for non-conformant proofs is `rejected` (not `unverified`).

### NATS Subject Mapping (v1)

- `agh.network.v1.<channel>.broadcast`
- `agh.network.v1.<channel>.peer.<route_token>`

### Replay Defense

Bounded replay window via `id`. Recommended 300-second clock-skew rejection when `expires_at` is null.

### Trust Profile

**Baseline Trust Profile** (RFC 004): Ed25519 + RFC 8785 JCS + SHA-256 fingerprints. Profile id `agh-network.trust.ed25519-jcs/v1`. Self-certified handles only — no DIDs, no revocation, no organization-level authorization, no federation policy in this profile.

---

## Memory

### Memory Types (taxonomy)

Per RFC 002 / Claude Code AutoDream / AGH `internal/memory/consolidation/`:

- `user` — persona, role, preferences, knowledge.
- `feedback` — rules and corrections from past interactions.
- `project` — context about ongoing work, who/why/by-when.
- `reference` — pointers to where info lives in external systems.

### Memory Scopes

- `agent` — local to a specific agent definition.
- `workspace` — shared across agents within a workspace.
- `global` — shared across workspaces in the AGH installation.

Default write scope is declared per agent in `memory.scope`.

### Consolidation Gates (cascade by cost)

**Time Gate** (default 24h since last consolidation) → **Session Gate** (default 5 sessions touched) → **Lock Gate** (`tryAcquireConsolidationLock` to prevent multi-instance races). All must pass. Never replace with naive heuristics.

---

## Autonomy

### `task_run`

The single durable work-queue row. Carries `claim_token`, `lease_until`, `heartbeat_at`, `coordination_channel_id`, plus owning `session_id`. **Never duplicated by a parallel queue.**

### `claim_token` / `claim_token_hash`

Opaque, fenced ownership token. Raw `claim_token` (`agh_claim_*`) NEVER appears over the wire, in logs, in SSE, in web UI, in channel messages, or in memory. Public form is `claim_token_hash`.

### `ClaimNextRun(criteria)`

The single authoritative claim primitive. Lives in `internal/task`. The mechanical scheduler does NOT call it.

### Coordinator

A managed AGH session whose semantic role is to orchestrate coordinated runs in a workspace. Auto-spawn is conservative (only when no healthy active coordinator AND a coordinated run is enqueued AND `coordination_channel_id` is stable AND auto-start is enabled AND spawn caps allow).

### Mechanical Scheduler

Daemon-owned operational-safety component (`internal/scheduler`). Idle registry, capability-aware wakeups, lease sweep, recovery, backpressure. **Does not claim runs.** Wake/observe/sweep are advisory.

### Coordination Channel

The single durable network channel bound to every workspace-scoped coordinated run via `coordination_channel_id`. "Bind always, speak when useful." Messages carry typed correlation but channels are NOT an ownership/status authority.

### Safe Spawn

Daemon-managed child-session creation. Defaults: `max_depth = 1`, `max_children = 5`, mandatory TTL. Permission narrowing on **concrete atoms only**: tools, skills, MCP server IDs, workspace path grants, network channels, env profile grants. Subset-only; unknown child atoms count as widening and reject.

---

## Verification & Testing

### `make verify`

Blocking commit gate: `fmt → lint → test → boundaries → build`. Zero warnings, zero errors.

### `make codegen` / `make codegen-check`

Regenerate / verify drift on `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`. Mandatory after any `internal/api/contract` change.

### Test Layers

- **Unit** (`make test`) — fast, race-enabled, per package.
- **Integration** (`make test-integration`) — `+integration` build tag, co-located.
- **E2E Runtime** (`make test-e2e-runtime`) — daemon-side Go harness against `acpmock`.
- **E2E Web** (`make test-e2e-web`) — Playwright against the daemon-served SPA.
- **E2E Nightly** (`make test-e2e-nightly`) — heavy E2E, runs in release-PR `dry-run` job only.

### Real-Scenario QA

Outer scenario orchestrator (`real-scenario-qa` skill) that builds a realistic multi-agent workspace and exercises CLI + Web + API surfaces end-to-end. Delegates to `qa-execution` and `qa-report` for inner mechanics.

---

## "What AGH Is Not"

For positioning consistency on the marketing site and in docs:

- AGH is **not a workflow engine**. Capabilities are interpretive, not deterministic programs.
- AGH is **not a federation protocol**. AGH Network v1 is a self-certified pairwise envelope, not a federated trust system.
- AGH is **not an MCP replacement**. MCP integrates _into_ AGH skills via `metadata.agh.mcp_servers`.
- AGH is **not an A2A replacement**. AGH Network is a peer-to-peer envelope; A2A is an industry standard. They can coexist.
- AGH **competes on runtime, SDK, observability, DX, and integration depth — NOT the open agent network protocol.** AGH Network must remain implementable outside AGH.

---

## Style

- File names: kebab-case for code/config, snake_case for memory files.
- Identifiers in code: Go conventions (`PascalCase` exported, `camelCase` unexported).
- Capability/skill IDs: kebab-case.
- Network subjects: dot-segmented, lowercase.
- Commit prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `build:` only.
