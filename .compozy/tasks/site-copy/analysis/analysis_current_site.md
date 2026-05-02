# AGH Current Site Copy Audit

Audit of marketing copy at `packages/site` against the founder's claim that the
site under-sells strong features (network, orchestration, dream, consolidation,
memory, tool registry, autonomy kernel) and over-sells weak ones (replayable).
Source page: `packages/site/app/(home)/page.tsx`.

## Site structure (homepage sections in order)

Composition root: `packages/site/app/(home)/page.tsx:16-33`. Render order:

1. `Hero` — `packages/site/components/landing/hero.tsx`
2. `BentoSection` — `packages/site/components/landing/bento-section.tsx`
3. `FeaturesSection` — `packages/site/components/landing/features-section.tsx`
4. `SupportedAgents` — `packages/site/components/landing/supported-agents.tsx`
5. `NetworkSection` — `packages/site/components/landing/network-section.tsx`
6. `RuntimeSection` — `packages/site/components/landing/runtime-section.tsx`
7. `SandboxSection` — `packages/site/components/landing/sandbox-section.tsx`
8. `BridgesSection` — `packages/site/components/landing/bridges-section.tsx`
9. `ExtensibilitySection` — `packages/site/components/landing/extensibility-section.tsx`
10. `InstallSection` — `packages/site/components/landing/install-section.tsx`
11. `Comparison` — `packages/site/components/landing/comparison.tsx`
12. `FinalCta` — `packages/site/components/landing/final-cta.tsx`

Header navigation labels — `packages/site/components/site/home-header.tsx:11-17`:
`Home`, `Runtime`, `AGH Network`, `Blog`, `Changelog`.

`packages/site/CLAUDE.md` reaffirms: "Hero positioning is locked: 'An open
workplace for AI agents.' (previously locked: 'Your agents can finally talk to
each other.') Open-workplace-first." The current hero (below) does NOT match
that lock.

## Current Hero

File: `packages/site/components/landing/hero.tsx:36-51`.

Eyebrow (line 36-40):
> "AGH — Agent Operating System"

Headline (line 42-44):
> "An agent runtime with a network built in."

Sub (line 46-50):
> "Sessions, memory, skills, workspaces, automation, bridges — the whole runtime
> in a single local binary. Then the part nobody else ships: an open protocol
> so your agents discover peers, delegate work, and collect receipts across
> machines."

CTAs (line 53-58): "Install the runtime" / "See the network".

Signal grid (line 4-21):
- "Complete agent runtime — Sessions, memory, skills, workspaces, automation, bridges — one binary."
- "Built-in agent network — Agents discover peers, delegate work, and collect receipts across machines."
- "Local-first, self-hosted — No Docker. No Postgres. Start with agh daemon start."
- "Open protocol, open source — agh-network/v0 is an open wire spec. Bring any agent you like."

Critique:
- The headline frames AGH as "runtime + bonus network." That structurally
  buries the differentiator. Network is the noun, not a parenthetical.
- It violates the locked positioning in `packages/site/CLAUDE.md` ("Your agents
  can finally talk to each other"; relocked 2026-05-01 to "An open workplace
  for AI agents"). Current hero has no emotional hook, no peer metaphor, no
  "agent-to-agent" claim. It reads like a feature manifest.
- "An agent runtime with a network built in" mirrors competitor framing
  ("X with Y built in") and dilutes uniqueness — the *runtime* is the table
  stakes part.
- Signal grid item 1 ("Complete agent runtime") and signal grid item 2 ("Built-in
  agent network") are co-equal in real estate, which the founder's brief says
  is wrong: network should dominate.
- Eyebrow "Agent Operating System" is OK, but no concrete version/peer count/
  message-kind claim that proves the network is real today.

## Section-by-section copy inventory

### Section 2 — BentoSection

File: `packages/site/components/landing/bento-section.tsx`.

Five bento tiles, each a giant verbless slogan:

- **Runtime tile** — line 61-64: "Your agents. *Under control.*" → promotes runtime/sessions. Strength **3/5**: tight, but generic; "under control" reads as control-plane vibe, not specific to AGH.
- **Network tile** — line 97-99: "Built-in network. *Delegate. Deliver.* Done." → promotes Network. Strength **4/5**: punchy, action verbs, but "built-in" again subordinates the network to the runtime; lacks the delivery proof ("receipts", "across machines").
- **Bridges tile** — line 132-134: "From anywhere. *Into a session.*" → promotes Bridges. Strength **4/5**: clean "external → internal" payoff.
- **Memory tile** — line 166-168: "Context that *remembers.*" → promotes Memory. Strength **2/5**: tautological ("memory remembers"). No mention of plain Markdown, four memory types, dream/consolidation, or per-workspace overlay — i.e. zero proof of the actual memory kernel.
- **Trace tile** — line 198-200: "Every step. *traceable.*" → promotes Observability/replay. Strength **2/5**: pure adjective slogan, indistinguishable from any monitoring tool.

Bento label-row (line 53-56, 89-91, 122-125, 157-160, 189-192) hard-codes
the five categories: Runtime, Network, Bridges, Memory, Trace. **Tool registry,
autonomy kernel, dream, consolidation, capabilities, hooks all missing from
this top-of-fold tile grid.**

### Section 3 — FeaturesSection ("Everything a modern agent runtime should have")

File: `packages/site/components/landing/features-section.tsx:55-98`.

Section header — line 60-62:
> Eyebrow: "What you get"
> Title: "Everything a modern agent runtime should have."
> Description: "You already know you need sessions, memory, and skills. AGH
> ships all of it, local-first, with an operator surface you can script."

Strength **2/5** for the header — frames AGH as a checklist of generic runtime
features. The whole section is positioned as "we have the basics" instead of
"we have the differentiators."

Six feature cards — `FEATURES` array, line 4-53:

1. **Sessions** (line 6-12): "Resume any agent run / Every agent run is a durable session. Stop, resume, inspect every step, fork from any point." → Sessions. Strength **3/5**, accurate but unremarkable.
2. **Memory** (line 13-20): "Context that survives restarts / Global and per-workspace memory in plain Markdown. Four types, one index per scope." → Memory. Strength **3/5** — finally mentions "four types" but no naming, no dream/consolidation, no autonomy hook.
3. **Skills** (line 21-28): "Reusable playbooks / Drop-in SKILL.md bundles with YAML frontmatter. Bundled library, workspace overrides, community catalog." → Skills/capabilities. Strength **3/5** — uses "playbooks" which conflicts with the glossary canonical "capability" (`docs/_memory/glossary.md`).
4. **Workspaces** (line 29-36): "Per-project everything / Agents, skills, memory, and config overlay per workspace. Switch projects, switch context." → Workspaces. Strength **3/5**.
5. **Automation** (line 37-44): "Cron + webhooks, durable / Schedule recurring work. Trigger sessions from external events. Every run tracked in SQLite." → Automation. Strength **3/5**, but doesn't connect to autonomy kernel/task_runs single queue.
6. **Observability** (line 45-52): **"Everything logged, everything replayable"** / "Token usage, permission audit, tool calls, errors — streamed over SSE, persisted to disk." → Observability/replay. Strength **2/5** — "replayable" is one of the founder's flagged weak claims and gets card real estate equal to Memory and Sessions.

### Section 4 — SupportedAgents ("Works with your agent CLIs")

File: `packages/site/components/landing/supported-agents.tsx:50-90`.

Eyebrow + headline (line 55-58):
> "Works with your agent CLIs"
> "Bring the CLI you already use. AGH spawns it, manages it, and persists every event."

Eight provider tiles (line 18-43): Claude Code, Codex, Gemini CLI, OpenCode,
Copilot CLI, Cursor, Kiro CLI, Pi.

Strength **4/5** — concrete, proof-of-breadth, links to "Configure providers."
This is one of the few high-trust sections.

### Section 5 — NetworkSection ("AGH Network — the differentiator")

File: `packages/site/components/landing/network-section.tsx:19-100`.

Header — line 22-47:
> Eyebrow: "AGH Network — the differentiator"
> Title: "agh-network/v0 — shipping today."
> Description: lists seven message kinds (greet, whois, say, direct,
> capability, receipt, trace) and explains "Your agent discovers a peer,
> selects a channel, and hands off work with an explicit target and message
> kind."

Three feature cards (line 53-79):
- "CLI today / Real commands, not docs-ware" — quotes `agh network status | peers | channels | send | inbox`.
- "Transport / NATS under the hood, JSON over the wire" — "Stand up a peer with a NATS URL, a shared key, and a channel name. No new infra to learn."
- "Auditable / Receipts are first-class" — "Every delegation returns a receipt with status and trace IDs. Every message is persisted to the audit log."

Closing paragraph (line 83-87):
> "Every other agent tool stops at the single-runtime boundary. AGH Network
> gives agents a shared wire protocol so a coder on your laptop can hand work
> to a deployer on CI, watch progress, and collect a signed receipt — without
> either side changing stacks."

Plus shell example (line 9-17): `agh network status | peers builders | send … | inbox`.

Strength **5/5**. This is the strongest section on the page. Specific, concrete,
ships-today framing, message kinds named, real CLI surface. The problem is it
is *section 5*, not the hero — it should drive everything above it.

### Section 6 — RuntimeSection ("A daemon built for sessions, not chats")

File: `packages/site/components/landing/runtime-section.tsx:49-97`.

Header (line 55-65):
> Eyebrow: "Runtime"
> Title: "A daemon built for sessions, not chats."
> Body: "Start `agh daemon start`. Every agent run becomes a session with a
> durable event log, an SSE stream, resumable state, and one operator surface
> shared by the CLI, API, and web UI."

Four feature cards (`FEATURES` array, line 6-42):
1. "Sessions / Durable sessions in SQLite" → Sessions. Strength 3/5.
2. **"Events / Replayable event stream"** (line 21-25) — "Every prompt, tool call, permission decision, and agent message is persisted with a monotonic sequence. SSE replay at /api/sessions/:id/stream." → Observability/replay. Strength **2/5**, founder-flagged "replayable" again.
3. "Surfaces / Three operator surfaces, one daemon" → CLI/HTTP/UDS. Strength 4/5 — concrete, names ports.
4. "Permissions / Permission modes with an audit trail" → Security. Strength 3/5.

Plus shell example (line 44-47): `agh daemon start`, `agh session new`, `agh session events --follow`, `agh session resume`.

Critique: Section name "A daemon built for sessions, not chats" is a decent
but small angle — it positions AGH against ChatGPT-style assistants instead of
against the *real* competition (multi-agent orchestrators, LangGraph, CrewAI,
Mastra, etc., which the user has compared AGH against). Replayability is
prominent. Autonomy kernel + task_runs single queue + ClaimNextRun are absent.

### Section 7 — SandboxSection ("Run agents away from the host filesystem")

File: `packages/site/components/landing/sandbox-section.tsx:102-141`.

Header — line 105-110:
> Eyebrow: "Sandbox"
> Title: "Run agents away from the host filesystem."
> Description: "Keep a session local when that is enough, or bind a workspace
> to a Daytona sandbox with explicit sync, lifecycle, and provider metadata."

Three cards (line 5-27): Local backend, Daytona profile, Sync mode.

Strength **3/5** — accurate but takes prime real estate (between Network and
Bridges) for a niche feature. Daytona alone, no Modal/E2B/Firecracker
breadth — this is a thin moat that AGH itself doesn't own.

### Section 8 — BridgesSection ("Your users live on these. Now so do your agents.")

File: `packages/site/components/landing/bridges-section.tsx:70-132`.

Header (line 73-78):
> Eyebrow: "Bridges"
> Title: "Your users live on these. Now so do your agents."
> Description: "Webhooks in, sessions out. Responses stream back to the original thread. No serverless glue, no second runtime — the bridge adapter runs inside the daemon."

Eight bridge tiles (line 24-68): Slack/Discord/Telegram (live), WhatsApp/Teams/
Google Chat/GitHub/Linear (next).

Flow strip (line 106-129): Platform → daemon (verify · route) → Agent → Thread reply.

Strength **4/5** — clear value prop, real platforms, honest "live/next" pills,
flow diagram pays the claim off.

### Section 9 — ExtensibilitySection ("Hooks, skills, automation, extensions")

File: `packages/site/components/landing/extensibility-section.tsx:41-89`.

Header (line 44-49):
> Eyebrow: "Extensibility"
> Title: "Hooks, skills, automation, extensions."
> Description: "The daemon is extensible at every seam you actually need. No plugins to write — contracts are plain files."

Four cards (line 6-39):
1. "Hooks / Observe and mutate lifecycle events" — "~40 lifecycle events — session start, tool call, permission request, network receipt." Strength 4/5 (concrete number, names events).
2. "Skills / Drop-in SKILL.md bundles" — Strength 3/5.
3. "Automation / Cron + webhook + event triggers" — Strength 3/5.
4. "Extensions / Install from local or marketplace" — "bundle skills, hooks, bridge adapters, and MCP servers. Ship them as zip files or via a GitHub registry." Strength 3/5.

Closer (line 65-77): "A skill is a Markdown file with frontmatter. A hook is
a TOML block in your config. … Contract on disk — not a plugin API." Strength
**4/5**.

Critique: This section is the only place "Hooks" lives. It's pleasant but
section 9 is too late for a load-bearing differentiator. Tool registry is
missing entirely. "Capability" appears only in the Network section as a
message kind — never as a first-class artifact.

### Section 10 — InstallSection ("Three commands. First session in under a minute.")

File: `packages/site/components/landing/install-section.tsx:99-181`.

Header (line 101-106):
> Eyebrow: "Getting started"
> Title: "Three commands. First session in under a minute."
> Description: "macOS and Linux today. Install with go, or build from a source checkout if you are already inside the repository."

Two install tabs (line 12-26): `go install`, `git clone … go build`.

Three steps (line 28-49): bootstrap → daemon start → session new.

Strength **4/5** — concrete, fast, real commands.

### Section 11 — Comparison ("Other tools stop at the runtime boundary")

File: `packages/site/components/landing/comparison.tsx:66-153`.

Header (line 69-74):
> Eyebrow: "Positioning"
> Title: "Other tools stop at the runtime boundary."
> Description: "AGH is the only approach with a shipped cross-runtime protocol. The rest centralize coordination or skip it entirely."

Four-row table (`APPROACHES`, line 18-56): Assistant gateway, All-in-one
agent OS, Multi-tenant gateway, AGH (highlighted, "agh-network/v0 — shipped",
"8 ACP CLIs", cross-runtime ✓).

Strength **4/5** — sharp positioning, network is the visible win column.
But the competitors are unnamed — "All-in-one agent OS" is vague. Could be
sharper with named comparisons (LangGraph, CrewAI, Mastra, OpenAI Assistants).

### Section 12 — FinalCta ("Install AGH. Run a session. Join the network.")

File: `packages/site/components/landing/final-cta.tsx:6-46`.

Eyebrow "Ship it" (line 11-13).
Headline (line 14-16): "Install AGH. Run a session. Join the network."
Sub (line 17-19): "One binary. No infrastructure. Shipped today."
CTAs (line 22-32): "Install AGH" / "Read agh-network/v0 spec" / Star on GitHub.

Strength **4/5** — tight, action-oriented. "Join the network" is the closing
beat — that beat should be amplified upstream.

## Coverage map: feature → copy

| Feature | Current copy presence | Real estate (high/med/low/none) | Verdict |
| --- | --- | --- | --- |
| **AGH Network** | Hero sub mention; bento Network tile (slogan); NetworkSection (full, 90 lines); FinalCta closing line; Comparison highlight row | **HIGH** below the fold, **MEDIUM** above | Strong section in section 5; needs to be the *hero* anchor, not buried below 4 sections |
| **Autonomy kernel** | NONE. Word "autonomy"/"kernel" never appears in landing copy. | **NONE** | Critical gap. Differentiator (task_runs single queue, ClaimNextRun, hooks dispatch) totally absent. |
| **Dream** | NONE. | **NONE** | Critical gap. Founder-flagged. |
| **Consolidation** | NONE. | **NONE** | Critical gap. Founder-flagged. |
| **Memory** | Bento Memory tile ("Context that remembers"); Features card ("Context that survives restarts / Global and per-workspace memory in plain Markdown. Four types, one index per scope"). | **MEDIUM** | Mentioned but shallow. No "four types" naming, no dream/consolidation, no agent-managed memory loop. Slogan tier only. |
| **Tool registry** | NONE. The recently-shipped tools registry (commit `f0c53baf feat: tools registry (#85)`) has zero copy. | **NONE** | Critical gap — this is brand-new flagship work. |
| **Capabilities** | Word appears once in Network as a *message kind* (`network-section.tsx:41`) and once inside `network-protocol-visual.tsx:278`. As a first-class agent artifact: NONE. Skills are called "playbooks" instead. | **LOW** | Glossary says "capability" is the canonical artifact name; site uses "playbooks" and "SKILL.md bundles." Inconsistent. |
| **Hooks** | Single card in ExtensibilitySection (`extensibility-section.tsx:8-13`). Mentions "~40 lifecycle events". | **LOW** | Buried in section 9. "Hooks dispatch at call site" is a load-bearing autonomy primitive — currently a one-card afterthought. |
| **Extensions** | Single card in ExtensibilitySection (line 31-37). | **LOW** | Mentions zip + GitHub registry. No marketplace screenshot, no extension count. |
| **ACP providers** | SupportedAgents grid (8 logos). Word "ACP" itself appears only in Comparison row "8 ACP CLIs". | **MEDIUM** | Honest, concrete — but "ACP" as a buzzword anchor is missing. |
| **Observability / Events / "replayable"** | Bento Trace tile; Features "Everything logged, everything replayable" card; Runtime "Replayable event stream" card; SSE stream mentions. | **HIGH** | Over-represented. "Replayable" appears in 3 distinct places, all top-half real estate. Founder-flagged as weak/generic. |
| **Sandbox / Daytona** | Full SandboxSection (section 7). | **MEDIUM-HIGH** | Disproportionate weight for a single sandbox provider integration. Daytona alone, between Network and Bridges. |
| **Bridges (Slack/Discord/Telegram)** | Bento Bridges tile; full BridgesSection. | **HIGH** | Justified — concrete, live integrations. |
| **Workspaces** | Features card "Per-project everything". | **LOW** | Adequate. |
| **Automation / cron** | Features card; Extensibility card. | **LOW-MEDIUM** | Two mentions but not tied to autonomy kernel. |

## Top problems

1. **Hero violates locked positioning.** `packages/site/CLAUDE.md` says
   "Hero positioning is locked: 'An open workplace for AI agents.' (previously
   locked: 'Your agents can finally talk to each other.') Open-workplace-first."
   Current hero (`hero.tsx:42-44`) says
   *"An agent runtime with a network built in."* That structure puts "agent
   runtime" first as the noun and "network" as a modifier — exactly inverted.
   The locked emotional hook is gone.

2. **"Replayable" gets premium real estate three times.** Founder-flagged:
   - `features-section.tsx:47` "Everything logged, everything replayable" (full feature card).
   - `runtime-section.tsx:21` "Replayable event stream" (feature card in section 6).
   - `bento-section.tsx:198-200` "Every step. *traceable.*" (top-of-fold bento tile).

   This is a generic monitoring claim, not an AGH differentiator.

3. **Tool registry has ZERO copy.** Most recent flagship feature
   (commit `f0c53baf feat: tools registry (#85)`). No mention in any landing
   component, bento tile, header, or comparison. Brand-new work invisible.

4. **Autonomy kernel has ZERO copy.** "Autonomy", "kernel", "task_runs",
   "ClaimNextRun" — never appear in landing copy. Per institutional memory
   (`docs/_memory` → `project_autonomy_kernel.md`) this is a load-bearing
   differentiator shipped 2026-04-26.

5. **Dream + consolidation have ZERO copy.** Founder-flagged as flagship
   memory features. Memory section (`features-section.tsx:13-20`) only says
   "Four types, one index per scope" without naming the types or the
   dream/consolidation cycle.

6. **"Capability" is misnamed.** Per `docs/_memory/glossary.md` (`capability`
   is canonical, NOT `recipe`/`workflow`/`procedure`/`playbook`),
   `features-section.tsx:23` calls skills "Reusable playbooks." Network
   section uses "capability" only as a message-kind chip. The first-class
   artifact name is missing from skills/extensibility prose.

7. **Sandbox occupies position 7 (between Network and Bridges) but is one
   provider integration.** SandboxSection is a full 100+ line component for
   Daytona alone. That real estate cost is high for a non-differentiating
   feature; sandbox is table-stakes for any modern agent runtime.

8. **Section ordering buries the differentiator.** Network (the differentiator
   per `network-section.tsx:24` "the differentiator") is section 5, after
   bento + features + supported-agents. Founder's brief says "o foco
   principal é fazer marketing encima da nossa network" — yet 4 sections of
   real estate run before it.

9. **Comparison table competitors are anonymous.** `comparison.tsx:18-56`
   names rows as "Assistant gateway," "All-in-one agent OS," "Multi-tenant
   gateway." No actual product names. Weakens the moat claim.

10. **Header navigation is fine but misses Tool Registry / Capabilities.**
    `home-header.tsx:11-17` lists Home / Runtime / AGH Network / Blog /
    Changelog — no first-class entry for capabilities, tools, extensions
    marketplace.

## Real-estate reallocation recommendation

### REMOVE

- **Bento "Trace" tile** (`bento-section.tsx:175-205`). Pure adjective
  "traceable" slogan. No proof. Replace this slot with **Network** or
  **Tool registry**.
- **Features card "Everything logged, everything replayable"**
  (`features-section.tsx:45-52`). Founder-flagged. The replay angle does
  not justify a card that sits next to Memory and Sessions.
- **Runtime card "Replayable event stream"** (`runtime-section.tsx:18-25`).
  Demote to a sub-bullet inside a broader "Observability" claim, OR delete.

### DEMOTE

- **SandboxSection** (`sandbox-section.tsx`). Move from section 7 to a thin
  card inside ExtensibilitySection or a feature row in the Runtime section.
  Remove the full diagram. Sandbox is not a top-five differentiator.
- **FeaturesSection** (`features-section.tsx`) overall. Today it's a "modern
  agent runtime checklist"; demote and rebrand as "Operator surface" with
  fewer cards (drop Observability/replay, drop Workspaces' generic framing).
  Move below Network so Network leads.
- **"Skills" framing as "playbooks"** (`features-section.tsx:23`). Rename to
  capabilities to match `docs/_memory/glossary.md`.

### PROMOTE

- **NetworkSection** (`network-section.tsx`). Move to section 2 (immediately
  after Hero). It is the strongest copy on the page and the locked
  differentiator.
- **Hooks + Autonomy kernel.** Pull "hooks dispatch at ~40 lifecycle events"
  out of section 9 into either the Hero signal grid or a new section
  "Autonomy kernel — agents own the loop." Source: `internal/CLAUDE.md` +
  `docs/_memory/project_autonomy_kernel.md`.
- **Capability as first-class artifact.** Rebuild the Skills card as
  "Capabilities — reusable agent behavior, on disk." Cite the glossary
  rule + the network "capability" message kind so the term is doing two
  jobs (artifact + wire transfer).

### ADD

- **Hero rewrite to lock-in line.** Per `packages/site/CLAUDE.md` lock,
  the headline must lead with the open-workplace claim. Current locked form:
  "An open workplace for AI agents." (previously proposed:
  "Your agents can finally talk to each other.") with sub: "AGH runs the
  agent CLIs you already use as durable sessions — with memory, autonomy,
  tools, and automation — connected on agh-network/v0 channels where they
  find each other, share capabilities, and close work with receipts."
  (earlier subhead drafts proposed: "Run the agent CLIs you already use as
  a team — they find each other, share capabilities, and close work with
  receipts on agh-network/v0." and "AGH ships an agent runtime *and* an
  open protocol so they delegate, deliver, and prove every handoff. One
  binary. Local-first.")
- **New "Memory kernel" section** that names the four memory types and
  describes dream + consolidation as the agent-managed memory loop.
  Currently those words don't exist on the homepage. Source:
  `internal/memory/*` and the autonomy-kernel memory.
- **New "Tool registry" tile or section.** Cite shipped commit
  `f0c53baf feat: tools registry (#85)`. Should mirror the bento/feature
  card pattern with a code sample (`agh tools list | publish | install`)
  and a 1-line value prop ("Tools are first-class artifacts agents
  discover, register, and call — not hard-coded plugins.").
- **Autonomy kernel section.** Frame: "task_runs is the single queue.
  ClaimNextRun is authoritative. Hooks dispatch at the call site." This
  is the engineering signal that the runtime is not a wrapper — it owns
  the agent loop.
- **Named competitors in Comparison.** Replace anonymous archetypes with
  product names (LangGraph / CrewAI / Mastra / OpenAI Assistants /
  ChatGPT Apps) so the cross-runtime ✓ does load-bearing work.
- **Network proof-of-life signal in Hero.** Concrete number: "7 message
  kinds. 8 ACP agents. Shipped today." instead of the abstract signal
  grid items.

### Suggested final section order

1. Hero — network-locked headline + protocol-shipping signal grid.
2. **NetworkSection** (promote from 5).
3. BentoSection — replace Trace tile with Tool Registry, replace Memory
   slogan with dream/consolidation hook.
4. **Memory kernel section (NEW)** — four types + dream + consolidation.
5. **Autonomy kernel section (NEW)** — task_runs / ClaimNextRun / hooks.
6. RuntimeSection (sessions/permissions/SSE) — drop replayable card.
7. **Tool registry section (NEW)** or merged into Bento tile.
8. ExtensibilitySection — promote hooks card to lead position.
9. BridgesSection — keep.
10. SupportedAgents — keep, possibly merge with Bridges row.
11. Sandbox — demote to a card inside Extensibility or Runtime.
12. InstallSection — keep.
13. Comparison — keep, name real competitors.
14. FinalCta — keep, "Join the network" closing beat is correct.
