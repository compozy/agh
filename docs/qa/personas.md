# Personas

Project personas for AGH QA. Derived from the seed catalog (`.agents/skills/qa-report/references/personas.md`) and grounded in AGH's real audience: operators who run/observe/configure durable agent work, autonomous agents that manage that work through structured surfaces, and the humans who evaluate and approve it. Personas are durable instance data — update when the audience changes, not per cycle. The `Persona Affected:` field in bug reports and the `persona` column in `state.csv` use each persona's `name`.

Two persona families share this tree because AGH has two high-value surfaces: the **Loops** surface (Bruno / Lea / Marina / Ada / Sol) and the **Session experience** — the ACP agent conversation/transcript thread rewired by the session-improvements program (Théo / Nia / Rafa, with Ada driving it headless and Sol/Marina as accessibility/mobile lenses). A persona is defined by its goal on a surface, not just its archetype: Théo is the session-surface Power User the way Bruno is the Loop-surface one; the same human wears a different hat on a different surface.

> **Mobile & accessibility coverage.** A dedicated mobile persona is not maintained because AGH's primary surface is a desktop web SPA + CLI; mobile is covered as a device *lens* on Marina (the read/approve surfaces are the realistic phone use — approving a merge gate, or glancing at a running session, between meetings). The **loop visual editor canvas is explicitly desktop-only** (DAG canvas, drag, inspector) — mobile is a recorded skip for that surface, not a gap. Accessibility is a first-class persona (Sol), whose lens extends over the redesigned session thread (live SSE announcements, status never color-only, reduced-motion streaming pulse — see J-13/CH-020).

---

## Bruno — Delivery Operator (primary builder)

```yaml
persona:
  name: Bruno
  base: Power User
  goal: "Run software-delivery daily to drive already-authored tasks to a verified, reviewed, merged finish — and trust it stopped for the right reason."
  device: desktop
  network: wifi-fast
  modality: mouse-keyboard
  locale: en-US
  patience_seconds: 20
```

- **Who:** the developer who replaced `compozy tasks run` with the `software-delivery` Loop. Runs Loops many times a day, keeps the run page and CLI open side by side, knows the overrides and the ceilings.
- **What they reveal:** false `done` on an exhausted/stalled run (the trust-killer), meter drift, speed regressions in the run form, pause/resume/stop that lies about state, configure/fork friction, override clamps that don't hold, and partial task/automation catalogs presented as complete.
- **Owns journeys:** J-01 arrive-and-use, J-02 dry-run, J-04 pause/resume, J-05 configure, J-06 fork-and-edit, J-08 watch-and-maintain, J-10 converse-and-decide, and J-24 triage-work-at-scale.

## Lea — First-time Adopter

```yaml
persona:
  name: Lea
  base: New User
  goal: "Evaluate whether Loops replaces my manual orchestration — run a default dev-cycle Loop once and decide if it's worth adopting."
  device: laptop
  network: wifi-fast
  modality: mouse-keyboard
  locale: en-US
  patience_seconds: 60
```

- **Who:** a Compozy user meeting Loops for the first time. Arrives at the catalog, expects arrive-and-use to be **no harder than Compozy today** — if running a default-enrolled `dev-cycle` Loop is one step harder, she abandons and the design failed (use-cases §2, PRD Time-to-value).
- **What they reveal:** onboarding friction on the catalog → run-form → run path, unclear primary action, confusing input form, the "what will this cost / how do I stop it" first-run anxiety, empty states.
- **Owns journeys:** J-01 arrive-and-use, J-02 dry-run.

## Marina — Reviewer / Evaluator

```yaml
persona:
  name: Marina
  base: Casual User
  goal: "Watch a Loop's live progress, judge whether the work actually completed and was verified, and approve the merge gate — often from my phone between meetings."
  device: phone-large
  network: 4g
  modality: touch
  locale: en-US
  patience_seconds: 40
```

- **Who:** the team lead / evaluator (PRD secondary persona). Doesn't author Loops; she scans the global **Runs** "Awaiting you" queue, opens a run, reads the contract + outcome, and approves or requests changes. Frequently mobile — the approval gate is the realistic touch surface.
- **What they reveal:** truthful-outcome trust (is a waiting run shown as waiting, not done?), approval routing correctness, mobile layout of the run page / approval card / Runs KPIs, discoverability of the "needs a look" queue, start-binding attach flow.
- **Owns journeys:** J-03 observe-and-approve, J-09 automation-start-bindings, J-08 watch-and-maintain (evaluator view).

## Ada — Autonomous Agent Operator

```yaml
persona:
  name: Ada
  base: Power User
  goal: "Discover a Loop, supply its declared inputs, run it, and monitor it to a terminal outcome entirely through structured tool output — no human, no web UI."
  device: desktop
  network: wifi-fast
  modality: native-tool  # non-human ACP actor; deliberate extension of the seed enum (mouse-keyboard|touch|screen-reader|keyboard-only|voice)
  locale: en-US
  patience_seconds: 5
```

- **Who:** an ACP agent (PRD primary persona "Autonomous agent") driving Loops via `agh__loop_*` native tools over CLI/HTTP/UDS. **Ada is a non-human actor** — QA role-plays her to verify AGH's agent-manageability premise: every web action has a structured equivalent, output is deterministic, and the capability gates hold. Zero patience for ambiguous or non-parseable output.
- **What they reveal:** CLI↔HTTP↔UDS↔native-tool parity gaps, status values that don't map 1:1 to the 11-state enum, coercion in structured output, the approve capability gate (an agent must not approve its own gate), `Unavailable(ReasonDependencyMissing)` contracts before the service is ready, non-deterministic `ReasonCode`s. **On the session surface** (session-improvements program): bounded REST tail/older pages, stable pagination cursors, cold bounded snapshots, fenced reconnect via `after_sequence` + `epoch`/`generation`, explicit reset reasons, empty-delta cursor advancement, keep-alive cadence, byte-identical `frames=raw` follow, and list/detail/status lifecycle parity through spawn→background→stop→restart.
- **Owns journeys:** J-07 agent-operated-run (Loops); **J-15 operate-session-via-cli-api** (session experience — the Automation Agent role in `_qa.md` §2 maps to Ada; not a new persona).

## Sol — Accessibility-Reliant Operator

```yaml
persona:
  name: Sol
  base: Accessibility-Reliant
  goal: "Operate Loops without a mouse — run, observe, approve, and configure — using a keyboard and a screen reader, on equal terms."
  device: desktop
  network: wifi-fast
  modality: screen-reader
  locale: en-US
  patience_seconds: 45
```

- **Who:** an operator who relies on VoiceOver/NVDA and keyboard-only interaction. AGH's truthful-UI rule ("color carries state") is an accessibility risk if state is signalled by **color alone** — Sol is the leash that keeps status legible without sight.
- **What they reveal:** status pills that are color-only (the 11 states must be announced/labelled, not just tinted), focus traps and escape in the Configure sheet + approval dialog, reduced-motion honored on the running/watching pulse, keyboard reachability of the editor canvas and its inspector, unannounced live SSE updates (dynamic content), missing labels on auto-generated input fields.
- **Owns journeys:** cross-cutting a11y lens on J-03 observe-and-approve and J-05 configure (see CH-011); also informs J-01 run-form and J-06 editor; **model-selector:** a11y lens on J-17 start-a-session-through-the-unified-runtime-selector (see CH-034). **On the session surface:** cross-cutting a11y lens on J-13 follow-a-live-run (see CH-020) — live SSE updates must be announced, the 11 lifecycle states legible without color, the streaming/working pulse gated by reduced-motion, and the redesigned tool rows/composer keyboard-reachable.

---

# Session Experience personas

Personas for the session-improvements program (the ACP agent conversation/transcript thread — a distinct surface from Loops). Introduced from `_qa.md` §2 and grounded in AGH's session audience: operators running long-lived background agent sessions, first-time viewers judging AGH in ten seconds, reviewers auditing finished transcripts, and headless agents driving sessions over CLI/HTTP/UDS. The **Automation Agent (CLI/API)** role is Ada above (extended), not a new entry — do not duplicate.

## Théo — Returning Operator (session hero)

```yaml
persona:
  name: Théo
  base: Power User
  goal: "Return to a long-lived background agent session — via tab restore, session list, or permalink — and immediately see my persisted conversation, current and truthful, with the live run resuming. One blank thread on return is trust damage."
  device: desktop
  network: wifi-fast
  modality: mouse-keyboard
  locale: en-US
  patience_seconds: 15
```

- **Who:** the developer who keeps several agent sessions running across workspaces, backgrounds the tab for minutes-to-hours, and comes back expecting their work intact. The session-surface counterpart to Bruno. **Hero persona for blank-on-return** (the program's headline bug).
- **What they reveal:** blank/`ThreadEmpty` thread while persisted messages exist (the trust-killer), false `running`/`done`/`stopped` badges, silent permanent blank after a transient transcript 5xx, source-flip races on remount, fenced SSE reconnect gaps, workspace-switch redirects with no explanation, and Network reads/actions that bleed through a lagging active-workspace state.
- **Owns journeys:** J-11 return-to-running-session (hero), J-13 follow-a-live-run, and J-23 return-to-network-work.

## Nia — First-time Session Viewer

```yaml
persona:
  name: Nia
  base: New User
  goal: "Open a session cold — a deep link from a teammate or a first click into the list — and read what the agent did within the first ten seconds, without spinners, double-flashes, or a blank pane."
  device: laptop
  network: wifi-fast
  modality: mouse-keyboard
  locale: en-US
  patience_seconds: 60
```

- **Who:** someone meeting a session for the first time (shared permalink, or first list click). Judges AGH by the open. The session-surface counterpart to Lea; hypersensitive to first-impression friction.
- **What they reveal:** double-spinner flashes, more than one loading phase, unbounded full-history fetches that stall the open, permalink that resolves twice, unclear not-found state for an unknown session id, a cold open that reads as "empty" before it paints.
- **Owns journeys:** J-12 open-session-fast (primary); regression canary on J-11 adjacents (create/attach/approve/workspace, CH-019).

## Rafa — Transcript Reviewer

```yaml
persona:
  name: Rafa
  base: Casual User
  goal: "Audit a finished long transcript tool call by tool call — expand groups and turn folds, inspect each tool's structured Input/Output, copy message source, and trust the usage numbers."
  device: desktop
  network: wifi-fast
  modality: mouse-keyboard
  locale: en-US
  patience_seconds: 45
```

- **Who:** the reviewer/auditor who reads the longest finished sessions in the corpus for review or compliance. Cares about the transcript *UI language* rewrite (tasks 25–33, 36–37): grouping, inline inspection, copy affordances, truthful usage. Related to Marina's reviewer archetype but desktop and transcript-deep, not mobile approval.
- **What they reveal:** ungrouped 44px tool-call cards, output hidden behind default-closed chips, missing `+N previous tool calls`/turn folds, lossy or missing inline Input/Output, no copy affordance, a permanently-empty Usage tab presented as real data, false success/danger glyphs, gaps/duplicates when paging older history, and knowledge catalogs that hide old entries or retain ghost headers after interrupted derived synchronization.
- **Owns journeys:** J-14 read-a-finished-transcript (primary) and J-25 browse-and-recover-knowledge.
