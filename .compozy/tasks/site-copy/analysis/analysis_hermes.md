# Hermes Marketing Analysis

Source repo: `/Users/pedronauck/Dev/compozy/agh/.resources/hermes`
Primary external surfaces examined:

- `README.md` (GitHub landing — the de facto homepage)
- `website/docs/index.md` (docs landing at `hermes-agent.nousresearch.com/docs`)
- `website/docusaurus.config.ts` (site title + tagline)
- `website/docs/getting-started/quickstart.md`
- `website/docs/getting-started/learning-path.md`
- `website/docs/user-guide/features/overview.md`
- `website/docs/user-guide/features/{skills,memory,delegation,cron}.md`
- `pyproject.toml` (package description string)

Note: there is no standalone marketing landing page. The Docusaurus site lives at `/docs/` and the docs landing **is** the homepage. The README is doing double duty as both repo intro and primary marketing surface — that shapes everything below.

---

## One-liner

The canonical one-liner appears verbatim in three places (README L14, `website/docs/index.md` L12, `pyproject.toml` description):

> **"The self-improving AI agent built by Nous Research."**

Followed immediately by:

> "It's the only agent with a built-in learning loop — it creates skills from experience, improves them during use, nudges itself to persist knowledge, searches its own past conversations, and builds a deepening model of who you are across sessions."

The Docusaurus tagline (`docusaurus.config.ts` L7) is the trimmed version: **"The self-improving AI agent."**

The pyproject description compresses the same idea: **"The self-improving AI agent — creates skills from experience, improves them during use, and runs anywhere."**

So: one stable claim, repeated everywhere, owned via "the only agent with…". This is disciplined and high-confidence.

---

## Value props (ranked by prominence)

Order on the README + docs index. They lead with the loop, not the chat:

1. **Self-improving learning loop** — skills it creates, skills that improve during use, memory nudges, cross-session FTS5 search, Honcho user modeling. This is the headline differentiator and gets its own paragraph + a feature row called "A closed learning loop".
2. **Runs anywhere, not your laptop** — six terminal backends (local, Docker, SSH, Daytona, Singularity, Modal). Heavy emphasis on "$5 VPS" and serverless hibernation ("costs nearly nothing when idle"). They keep coming back to this.
3. **Lives where you do** — 15+ messaging platforms (Telegram, Discord, Slack, WhatsApp, Signal, Matrix, Email, SMS, etc.) from a single gateway. "Talk to it from Telegram while it works on a cloud VM" is repeated twice.
4. **Any model you want** — long enumerated list of providers (Nous Portal, OpenRouter, NIM, Xiaomi, GLM, Kimi, MiniMax, OpenAI). "No code changes, no lock-in."
5. **Real terminal interface** — TUI is sold as a product feature, not an afterthought. Multiline editing, slash-command autocomplete, streaming tool output.
6. **Scheduled automations** — cron with natural language and platform delivery.
7. **Delegation and parallelization** — subagents, RPC tool calls via `execute_code` collapsing pipelines into "zero-context-cost turns".
8. **Research-ready** — batch trajectories, Atropos RL, training-data generation. Last because narrow audience.

Note the missing slot: there is no "team" or "enterprise" value prop. They never market to orgs.

---

## Marketing vocabulary they use

Words that recur (these are the load-bearing nouns/phrases):

- **"self-improving"** — flagship adjective, repeated in every surface.
- **"learning loop"** / **"closed learning loop"** — proprietary-sounding phrase they've coined.
- **"agent"** — never "harness", never "framework". Always *agent*. Singular.
- **"skills"** / **"procedural memory"** — they own this with the agentskills.io standard.
- **"memory"** / **"persistent memory"** / **"agent-curated memory"** — bounded, curated, nudged.
- **"runs anywhere"** / **"lives where you do"** — locational verbs.
- **"hibernates when idle"** / **"costs nearly nothing"** — concrete cost framing.
- **"gateway"** — single word for the messaging integration layer.
- **"delegate"** / **"subagent"** / **"parallel workstreams"**.
- **"trajectory"** — research-coded term they keep.
- **"toolset"** — grouped tools as a first-class concept.
- **"context engine"** / **"context files"** / **"context references"** — they expose context as a product surface.
- **"open standard"** — used to position skills as portable.
- **"progressive disclosure"** — used technically, but it's also marketing for "we don't waste tokens".

Words they avoid (notably absent):

- "orchestration" — never used.
- "swarm" — never used.
- "harness" — never used (despite being a harness in everything but name).
- "platform" / "framework" — almost never; they say "agent".
- "enterprise" / "production" / "scale" / "team collaboration".
- "MLOps" / "AgentOps" / "observability" / "tracing".
- "multi-agent" — they say "delegate" / "subagent" instead.

---

## Hero features

The README feature table (L18-26) ranks them in this order:

1. **A real terminal interface** — TUI as headline. Surprising for an "agent" project; they treat the shell as the product.
2. **Lives where you do** — messaging gateway.
3. **A closed learning loop** — memory + skills + Honcho.
4. **Scheduled automations** — cron.
5. **Delegates and parallelizes** — subagents.
6. **Runs anywhere, not just your laptop** — terminal backends.
7. **Research-ready** — RL/Atropos last.

The docs index reorders to lead with the loop ("Key Features" L46-57), suggesting the README is tuned for a developer scanning a GitHub page (TUI first = "I can use this") while the docs site is tuned for someone who already clicked through (loop first = "why this is special").

---

## What they DON'T market

Features that exist in code/repo but are not in marketing copy:

- **ACP adapter** — `acp_adapter/` is a real subsystem and they ship an IDE integration page, but ACP gets one line in feature overview, no top-level mention. Notable for AGH because we are ACP-native.
- **Web dashboard / web UI** — `web/` exists with full Vite/React app and `tui_gateway/`, but the README does not mention a web dashboard. They lean into TUI as the product.
- **MCP** — has its own page but is buried in "Integrations". Treated as plumbing, not a hero.
- **API server (OpenAI-compatible HTTP)** — exists, mentioned in the docs feature index, but absent from the README hero. They don't market themselves as a backend.
- **Plugins system** — "Customization" section, not a hero. Three plugin types exist but extensibility is not pitched as a moat.
- **Hooks / event lifecycle** — exists, low marketing weight.
- **Honcho** — mentioned by name but not explained on the homepage. Insider-coded shorthand.
- **Voice mode** — exists with a guide but not in the README hero (only in the docs index).
- **Migration from OpenClaw** — full section in README but framed as utility ("if you're coming from OpenClaw"), not a competitive comparison.
- **Security / approval / container isolation** — has docs but never appears as a value prop in the headline copy.

---

## Tone & audience

**Audience**: solo developers and AI hobbyists who are comfortable with a terminal, run their own infra, want to use cloud LLMs but not be locked into a vendor, and are interested in the AI research conversation. Distinctly *not* aimed at engineering teams, enterprises, or non-technical users.

**Tone**:

- Confident, declarative ("the only agent with…", "no code changes, no lock-in").
- Concrete and operational ("$5 VPS", "60 seconds", "47 tools", "15+ platforms"). Numbers everywhere.
- Lightly opinionated ("Rule of thumb: if Hermes cannot complete a normal chat, do not add more features yet").
- Researcher-adjacent vocabulary (trajectory, RL, Atropos) without being academic.
- Friendly imperative voice ("start chatting!", "Pick the row that matches your goal").
- Almost no hype words. No "revolutionary", no "next-gen", no "AI-powered". They earned the right to skip those by being specific.

**Technical depth**: high. They quote provider names, license terms, file paths, character limits ("MEMORY.md 2,200 chars / ~800 tokens"), and protocol names without softening. The reader is presumed technical from line one.

---

## Differentiation strategy

Hermes positions on three axes simultaneously:

1. **Continuous learning vs. stateless chat.** "The only agent with a built-in learning loop." Their wedge against Claude Code, Codex CLI, Aider, and other ephemeral session tools.
2. **Anywhere-deployable vs. laptop-bound.** Telegram + cloud VM is the canonical scene. "Not tied to your laptop" is a direct shot at IDE-coupled tools.
3. **Built by model trainers.** Nous Research brand transfers credibility — "the lab behind Hermes, Nomos, and Psyche" appears twice. They use this as proof they understand the underlying tech.

What they consciously *don't* differentiate on:

- Multi-agent coordination (they have it as `delegate_task` but don't pitch it).
- Protocol or interop (no mention of being open-protocol, A2A, network).
- Enterprise / SSO / audit.
- Performance benchmarks.

The flagship moat is **memory + skills feedback loop**. Everything else is supporting cast.

---

## Strong copy examples (with quotes)

1. README L14, the value-prop sentence:

   > "It's the only agent with a built-in learning loop — it creates skills from experience, improves them during use, nudges itself to persist knowledge, searches its own past conversations, and builds a deepening model of who you are across sessions."

   Why it works: parallel verbs ("creates… improves… nudges… searches… builds"), concrete behaviors not abstract claims, "the only agent" stakes a category.

2. README L21, the "lives where you do" framing:

   > "Telegram, Discord, Slack, WhatsApp, Signal, and CLI — all from a single gateway process. Voice memo transcription, cross-platform conversation continuity."

   Why it works: lists name-brand platforms (familiarity transfer), then ends on operational detail. The user can map this to their own life.

3. README L24 / docs index L21:

   > "Talk to it from Telegram while it works on a cloud VM you never SSH into yourself."

   Why it works: a single image of a workflow that no IDE-coupled tool can offer. This one line is doing the heavy lifting against Claude Code/Cursor.

4. Docs index L21, the negative-space pitch:

   > "It's not a coding copilot tethered to an IDE or a chatbot wrapper around a single API. It's an autonomous agent that gets more capable the longer it runs."

   Why it works: explicit competitor framing in two crisp categories, then a single-clause positive claim. "Gets more capable the longer it runs" is the whole product in seven words.

5. README quickstart rule (`quickstart.md` L30):

   > "Rule of thumb: if Hermes cannot complete a normal chat, do not add more features yet. Get one clean conversation working first, then layer on gateway, cron, skills, voice, or routing."

   Why it works: an honest admission that the product is a stack, not a magic box, and it earns trust by giving operational guidance instead of promises.

---

## Weak copy examples (with quotes)

1. Docs feature overview (`features/overview.md` L9):

   > "Hermes Agent includes a rich set of capabilities that extend far beyond basic chat. From persistent memory and file-aware context to browser automation and voice conversations, these features work together to make Hermes a powerful autonomous assistant."

   Why it's weak: "rich set of capabilities", "extend far beyond basic chat", "powerful autonomous assistant" — three generic phrases in one sentence. Could describe any agent. The README is much sharper than the docs at this layer.

2. Docs index L51:

   > "Built by model trainers — Created by Nous Research, the lab behind Hermes, Nomos, and Psyche. Works with Nous Portal, OpenRouter, OpenAI, or any endpoint."

   Why it's weak: tries to do two things (credibility + provider neutrality) and succeeds at neither. "Built by model trainers" is a fact, not a benefit — the reader has to do the work to figure out *why* they should care. Compare to L14 of README which never has this problem.

---

## What AGH should LEARN from Hermes

1. **Pick one phrase and repeat it everywhere.** "The self-improving AI agent" is in the README, the docs index, the Docusaurus tagline, and the pyproject.toml description. Verbatim. AGH should pick its phrase ("An open workplace for AI agents" per the current techspec; previously "Your agents can finally talk to each other") and refuse to deviate.
2. **Stake a category with "the only".** "The only agent with a built-in learning loop" is a category claim, not a feature claim. AGH's analogue is "the only runtime that…" or "the only protocol that…". Pick the slot.
3. **Concrete numbers as decoration.** "47 tools", "15+ platforms", "$5 VPS", "60 seconds", "2,200 chars". Specificity beats adjectives. AGH should quote actual numbers (number of capabilities, agents per session, latency budgets).
4. **A single image-rich workflow sentence per audience.** Hermes has "Telegram while it works on a cloud VM you never SSH into yourself." AGH needs the equivalent: a single sentence picture of two agents coordinating that no competitor can copy.
5. **Negative-space framing.** "Not a coding copilot tethered to an IDE or a chatbot wrapper around a single API" defines the field by exclusion before claiming the positive ground. AGH can do this against Claude Code, Codex CLI, Cursor.
6. **Treat the feature table as a hero, not a footnote.** The README's seven-row table (L18-26) is the entire pitch in one screen. Each row leads with a strong noun phrase ("A closed learning loop", "Lives where you do") before the description. AGH's home page should have a table or grid built the same way.
7. **README is a marketing surface.** They put real work into the GitHub README — banner, badges, table, install one-liner, command index. Don't treat AGH's README as documentation; treat it as the second landing page.
8. **Don't bury the differentiator in features.** They lead the README with the value-prop sentence, then the table, then install. Features come after the *why*. AGH's site should mirror this order, not start with a "Get Started" CTA.
9. **List the integration receipts.** Long enumerated lists of providers/platforms/backends create the impression of a mature ecosystem. AGH can list ACP-compatible agents, supported MCP servers, transports.

---

## What AGH should AVOID copying

1. **Buried hero features.** Hermes treats ACP as plumbing in the docs, but for AGH this is core. Don't let your protocol moat hide in an "Integrations" subsection. Lead with the network.
2. **No web/UI marketing.** Hermes effectively hides their web dashboard. AGH has a real web UI that is a major differentiator vs. Claude Code's CLI-only model — market it.
3. **No team/enterprise lane.** Hermes targets solo dev only, which is fine for them but leaves money on the table. AGH should leave room (without selling out) for a team narrative — "agents that coordinate" naturally implies multiple users.
4. **Provider name-soup.** "OpenRouter, NVIDIA NIM, Xiaomi MiMo, z.ai/GLM, Kimi/Moonshot, MiniMax, Hugging Face, OpenAI" works for them because model-pluralism is part of their pitch. AGH does not need to enumerate every LLM — that crowds out the unique claim.
5. **Insider shorthand on the homepage.** "Honcho dialectic user modeling", "Atropos RL environments", "FTS5 session search" appear without explanation. Strong for the in-crowd, opaque for the rest. AGH should explain its terms (capability, network, peer card) the first time, every time.
6. **Generic feature-overview voice.** The docs `features/overview.md` is significantly weaker than the README. Make sure AGH's docs landing matches the energy of the marketing landing — don't let the writing degrade by surface.
7. **README-as-only-homepage.** Their Docusaurus site has no real custom landing page; `/docs/` is the homepage. AGH is investing in a Fumadocs marketing site precisely so the home page can do work the README can't (interactive demos, hero animation, social proof). Use it.
8. **"Built by model trainers" without the benefit.** AGH should not lead with "Built by Compozy" or any builder-credibility claim until it's connected to a benefit the reader cares about. Hermes can pull this off because Nous is famous to the audience; AGH cannot assume the same recognition yet.
