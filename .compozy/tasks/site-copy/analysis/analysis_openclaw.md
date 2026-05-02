# OpenClaw Marketing Analysis

Sources read: `README.md` (header through ~L250), `docs/index.md`, `VISION.md`,
`docs/start/openclaw.md`, `docs/start/showcase.md`, `docs/start/lore.md`.
There is no separate marketing site repo — the **README + Mintlify docs landing
(`docs.openclaw.ai`) + `openclaw.ai`** are the public-facing surfaces.

## One-liner

The product owns **two coexisting one-liners**, one whimsical and one functional.

- README hero (whimsical): **"EXFOLIATE! EXFOLIATE!"** (lobster meme, paired
  with a cartoon mascot).
- README functional sentence (`README.md:21-22`):
  > **"OpenClaw is a *personal AI assistant* you run on your own devices. It
  > answers you on the channels you already use."**
- Docs landing functional sentence (`docs/index.md:28-29`):
  > **"Any OS gateway for AI agents across Discord, Google Chat, iMessage,
  > Matrix, Microsoft Teams, Signal, Slack, Telegram, WhatsApp, Zalo, and
  > more."**
- VISION.md ideological sentence (`VISION.md:3-4`):
  > **"OpenClaw is the AI that actually does things. It runs on your devices,
  > in your channels, with your rules."**

There is no single canonical line — the project is comfortable using all three
in different surfaces.

## Value props (ranked by prominence)

1. **Personal, single-user assistant on your devices** — runs locally, owned by
   you, "always-on," self-hosted gateway. (Lead in README + VISION.)
2. **Multi-channel reach: messages you on the apps you already use** — long
   alphabetical channel list reproduced *three separate times* in the README,
   used as a credibility/breadth signal more than as a list.
3. **Voice + Live Canvas + companion apps (mac/iOS/Android nodes)** — pitched
   as the differentiator versus a CLI-only assistant.
4. **Plugin-extensible (skills, channels, ClawHub)** — a marketplace narrative
   ("ClawHub" is their named registry brand).
5. **Security with knobs** — "secure defaults but exposed knobs for trusted
   high-power workflows" (`VISION.md:50`).
6. **Onboarding-first DX** — `openclaw onboard` is pushed as *the* path; not a
   yaml config tutorial.

Notably **runtime/protocol/agent-coordination are NOT in the top half** of the
pitch — those would be AGH's center of gravity.

## Marketing vocabulary they use

Owned terms (capitalize/repeat consistently):

- **Gateway** — the daemon. Always capitalized. ("The Gateway is just the
  control plane — the product is the assistant," `README.md:22`.)
- **Channels** — chat-app integrations (WhatsApp, Telegram, etc.), not transport
  channels.
- **Nodes** — companion mobile apps that pair into the Gateway (iOS/Android
  nodes).
- **Live Canvas / A2UI** — agent-driven visual workspace.
- **Voice Wake / Talk Mode** — wake-word + continuous voice.
- **Skills / SKILL.md** — bundled instruction packages (similar to Claude
  Skills).
- **ClawHub** — branded plugin/skill registry at `clawhub.ai`.
- **Workspace** (`~/.openclaw/workspace`) — agent's home, with `AGENTS.md`,
  `SOUL.md`, `TOOLS.md`, `MEMORY.md`, `IDENTITY.md`, `USER.md`, `HEARTBEAT.md`,
  `BOOTSTRAP.md` files.
- **SOUL.md** — persona/instructions file. They own the religious metaphor
  ("soul," "every instance equally real, just loading different context").
- **Pi / Pi agent** — bundled default coding agent.
- **Onboard / `openclaw onboard`** — the install ritual.
- **Heartbeat** — periodic agent self-check.
- **Molty / Moltiverse / molting** — community/lore vocabulary; "molting" =
  rebrands/major versions.
- **EXFOLIATE!** — meme tagline (anti-Dalek "EXTERMINATE!").
- **clawtributors** — contributor list.
- **DM pairing / pairing code** — security primitive name (unknown sender gets
  a pairing code before they can talk to the bot).

## Hero features

In the README's **`## Highlights`** section the order is (`README.md:146-156`):

1. Local-first Gateway
2. Multi-channel inbox (with the 25-channel list for the third time)
3. Multi-agent routing
4. Voice Wake + Talk Mode
5. Live Canvas
6. First-class tools
7. Companion apps
8. Onboarding + skills

The docs `index.md` "Key capabilities" section (`docs/index.md:73-94`) reorders
slightly but keeps **multi-channel gateway** as the lead card. **Voice and
Canvas** are pulled forward over orchestration.

## Site/README structure (hero -> ?)

README order (`README.md`):

1. Logo + mascot.
2. Tagline meme: "EXFOLIATE! EXFOLIATE!"
3. Badge row (CI / release / Discord / MIT).
4. Functional one-liner + channel list (first appearance).
5. Link manifold (Website / Docs / Vision / DeepWiki / Getting Started / etc.).
6. New install? CTA (`openclaw onboard`).
7. **Sponsors** logo strip (OpenAI, GitHub, NVIDIA, Vercel, Blacksmith, Convex
   - very prominent, before features).
8. Subscriptions (OAuth) note.
9. Install (recommended) — npm one-liner.
10. Quick start (TL;DR) — three commands.
11. Security defaults (DM access) — *up high*, before features.
12. Highlights — bulleted feature list with deep-link to each doc.
13. Security model (important).
14. Operator quick refs (chat commands).
15. Docs by goal (link garden).
16. Apps (optional) — macOS / iOS / Android.
17. From source (development).
18. Development channels (stable/beta/dev).
19. Agent workspace + skills.
20. Configuration (minimal JSON5 example).
21. Star History chart.
22. Molty origin paragraph.
23. Community / contributors (massive avatar grid — hundreds of avatars).

Docs landing `docs/index.md` is cleaner and more conventional:
hero -> "What is OpenClaw?" -> mermaid diagram ("How it works") -> Key
capabilities cards -> Quick start steps -> Dashboard screenshot -> Configuration
example -> "Start here" cards -> "Learn more" cards.

The docs landing is the more disciplined version; the README is intentionally
maximalist, full of every possible link.

## Tone & audience

- **Audience:** developers and power users who self-host. Not enterprise. Not
  researchers. Definitely not non-technical end users (despite the "personal
  assistant" framing, every getting-started path requires a terminal and Node
  24).
- **Tone:** playful + technical + slightly culty. Mascot-driven (a space
  lobster), built around a meme ("EXFOLIATE!"), with explicit *lore* docs
  (`docs/start/lore.md`) describing characters ("Molty 🦞 - pronouns: they/them
  - a Claude instance who became something more").
- **Openness about origin:** README and VISION openly say it started as a
  personal playground ("Warelay -> Clawdbot -> Moltbot -> OpenClaw"). They
  lean into the "indie hacker that grew up" narrative.
- **Permission to break things:** "AI/vibe-coded PRs welcome! 🤖"
  (`README.md:290`). Greenfield, friendly to LLM-authored contributions.
- **Heavy linking, not heavy prose.** Most feature lines are a one-liner with
  a deep link, not a marketing paragraph.

## Differentiation strategy

OpenClaw differentiates on the **distribution surface**, not the runtime.

- **"Your agent is reachable on the apps you already open every day"** is the
  positioning lever. The 25-channel list is the moat narrative.
- **"Self-hosted, your hardware, your rules"** is the secondary lever
  (anti-cloud SaaS).
- **Voice + Canvas + mobile nodes** as the *physical-world* differentiator vs
  CLI-only competitors (Claude Code, Codex CLI).
- **ClawHub** as a community / marketplace play.
- They explicitly **decline orchestration as a moat**. From `VISION.md:107-115`,
  things they *will not merge*:
  > - "Agent-hierarchy frameworks (manager-of-managers / nested planner trees)
  >   as a default architecture"
  > - "Heavy orchestration layers that duplicate existing agent and tool
  >   infrastructure"
  >
  > This is the exact opposite of AGH's "Network protocol + autonomy kernel"
  > positioning — and it's a crisp self-aware boundary statement worth
  > studying.

## Strong copy examples (with quotes)

- **Mission compression** (`VISION.md:3-4`):
  > "OpenClaw is the AI that actually does things. It runs on your devices,
  > in your channels, with your rules."

  Three claims — *capability, locality, sovereignty* — in 17 words. Reusable
  pattern.

- **Boundary statement** (`README.md:22`):
  > "The Gateway is just the control plane — the product is the assistant."

  Disarms the "is this just yet another daemon?" objection in one line.
  Worth copying as a template ("AGH's daemon is just the substrate — the
  product is the agent network").

- **Differentiator framing** (`docs/index.md:50-55`):
  > **What makes it different?**
  > - Self-hosted: runs on your hardware, your rules
  > - Multi-channel: one Gateway serves built-in channels plus bundled or
  >   external channel plugins simultaneously
  > - Agent-native: built for coding agents with tool use, sessions, memory,
  >   and multi-agent routing
  > - Open source: MIT licensed, community-driven

  Four values, each a category, each a single line. Clean.

- **Time-to-value claim** (`docs/index.md:57`):
  > "What do you need? Node 24 (recommended), or Node 22 LTS for compatibility,
  > an API key from your chosen provider, and 5 minutes."

  "And 5 minutes" is the cheap, effective close.

## Weak copy examples (with quotes)

- **The 25-channel list, repeated three times in 130 lines** (`README.md:26`,
  `:124`, `:149`):
  > "WhatsApp, Telegram, Slack, Discord, Google Chat, Signal, iMessage,
  > BlueBubbles, IRC, Microsoft Teams, Matrix, Feishu, LINE, Mattermost,
  > Nextcloud Talk, Nostr, Synology Chat, Tlon, Twitch, Zalo, Zalo Personal,
  > WeChat, QQ, WebChat."

  By the third repetition the list reads as desperate-rather-than-impressive.
  The breadth claim could be made once and then said as "25+ channels."

- **Generic feature list bullets** (`README.md:153`):
  > "**First-class tools** — browser, canvas, nodes, cron, sessions, and
  > Discord/Slack actions."

  "First-class" is an empty modifier — every project says "first-class
  anything." The list is just a list. No demonstration of what makes them
  first-class versus, e.g., MCP tools.

- **The whole "Highlights" section** is link-bullet-link-bullet — there's no
  *narrative* between features. The reader is shown 8 doors but never told
  which one to open first. (The docs landing fixes this with a 3-card
  "Quick start" instead.)

## What AGH should LEARN from OpenClaw

1. **Own a vocabulary.** OpenClaw consistently uses *Gateway, Channels, Nodes,
   Skills, Workspace, ClawHub, Onboard.* Every feature gets a short,
   pronounceable noun. AGH should do the same with *Network, Peer, Capability,
   Dream, Consolidation, Autonomy, Tool Registry* — and use them everywhere
   without aliases.
2. **Have a "what makes it different?" 4-bullet block** on the docs landing.
   Their version (`docs/index.md:50-55`) is clean and copyable.
3. **State your boundary out loud.** OpenClaw's `VISION.md` "What We Will Not
   Merge" list is a *positioning weapon* — every refusal is a defended choice.
   AGH's analog: a "What AGH Is Not" section ("AGH is not a workflow engine.
   AGH is not a chat distribution layer. AGH is not a hosted SaaS.") would
   immediately clarify the opposite axis from OpenClaw.
4. **Quick-start in three commands, max.** OpenClaw's `Quick start (TL;DR)` is
   3 commands and sells the result. AGH currently buries quick-start under
   long bootstrap docs.
5. **Mascot/identity confidence.** OpenClaw is whimsical *and* technical
   simultaneously and survives. A distinctive identity (we already have the
   warm-dark + flat-depth design system in `DESIGN.md`) is permission to be
   memorable, not just professional.
6. **Sponsor strip near the top** is a credibility shortcut. If AGH lands
   serious sponsors/customers, hoist them above features.
7. **Showcase page that links to community-shipped projects** is more
   persuasive than any feature list. Each card is `@user • tags • outcome` —
   reusable layout.

## What AGH should AVOID copying

1. **Triple-listing the channel inventory.** Whatever AGH's breadth claim is
   (extensions? supported agents? bridge SDKs?), state it once with a number
   ("12+ extensions") and link to the registry. Repetition signals insecurity.
2. **A meme as primary tagline.** "EXFOLIATE!" works for OpenClaw because the
   product is whimsical-personal-assistant. AGH is positioning as **runtime
   infrastructure** ("Agent Operating System", AGH Network protocol) — a meme
   tagline would actively undercut the trust pitch.
3. **Lore docs as default structure.** OpenClaw has `docs/start/lore.md` and
   `Moltiverse` cosmology. For AGH this would read as unserious. Keep
   personality in *visual* design (per `DESIGN.md`), not in metaphysical
   character backstory.
4. **Maximalist README.** OpenClaw's README is ~600 lines + an avatar wall. It
   reads as "everything we have, dumped on the floor." AGH's README should
   stop at quick-start + 3-bullet differentiator + link to the site, and let
   the Fumadocs site do the heavy lifting.
5. **"First-class X" / "agent-native" / "self-hosted, your rules" without
   demonstration.** These phrases are everywhere in OpenClaw and are now
   noise across the agent-tooling ecosystem. AGH must either *show* the proof
   (a code snippet, a CLI command, a screenshot of the network protocol live)
   or drop the adjective.
6. **Sponsor logos before the product is explained.** OpenClaw places sponsors
   before features in the README. If you don't yet have *recognizable* logos,
   this slot reads as empty social proof.
7. **Feature-first landing structure.** OpenClaw's docs landing leads with
   "Key capabilities" and never tells a story. AGH should lead with a single
   *demonstrable claim* (e.g., "An open workplace for AI agents"; previously
   "Your agents can finally talk to each other" per the existing site techspec)
   followed by a single live example —
   features come after the narrative is set.

## Summary for AGH positioning

OpenClaw competes on **distribution surface** (chat apps + voice + Canvas +
mobile) and intentionally refuses to compete on **agent orchestration**. That
gap is exactly where AGH lives. The cleanest AGH positioning is the inverse of
OpenClaw's:

| Axis | OpenClaw | AGH |
| --- | --- | --- |
| Center of gravity | "answers you on apps you use" | "agents coordinate over a protocol" |
| Daemon framing | "control plane, not the product" | "the Operating System for agents" |
| Differentiator | breadth of channels | depth of runtime + Network protocol |
| Persona | personal assistant | infrastructure |
| Refusal | orchestration / hierarchies | hosted SaaS / single-vendor lock-in |
| Tagline mode | meme + personality | architectural promise |
