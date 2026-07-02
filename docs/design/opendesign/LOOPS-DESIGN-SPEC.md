# Loops - Design Specification

> **What this document is.** A complete, implementation-facing specification of the
> AGH "Loops" feature *as designed* in the high-fidelity HTML mockups. It records the
> screens, their components, the data each surface reads/writes, the interactions, and
> the domain model the UI assumes.
>
> **What it is for.**
> 1. **PRD / TechSpec review** - section 9 lists every place where the design surfaces
>    a control, metric, default, or flow that the PRD/TechSpec must confirm, define, or
>    change. Treat section 9 as the actionable diff against the spec.
> 2. **Implementation context** - sections 4 to 8 are the build brief: component
>    inventory, per-screen data contracts (what the daemon must expose), states, and
>    interaction behavior.
>
> **Sources of truth (do not contradict).**
> - Spec: `/Users/pedronauck/dev/compozy/agh/.compozy/tasks/workflows/` (`_prd.md`,
>   `_techspec.md`, `product-ux.md`, `use-cases.md`, `requirements.md`, `adrs/`, `analysis/`).
> - Design system: `agh/DESIGN.md` + `PRODUCT.md` (tokens from `packages/ui/src/tokens.css`).
> - Design intent + history: `LOOPS-HANDOFF.md` (this folder).
> - The mockups: `loops-index.html`, `loops-catalog.html`, `loop-detail.html`,
>   `loop-run-form.html`, `run-detail.html`, `runs.html`, `loop-editor.html`,
>   `loop-configure.html`.
>
> **Status legend used below:** `[shipped-in-spec]` traced to spec/ADR · `[design-assert]`
> a deliberate design choice consistent with spec · `[VERIFY]` design surfaces something
> the spec must confirm or define · `[UNCONFIRMED]` not found in spec, flagged.

---

## 1. Scope of the design

Eight self-contained screens, each a content + in-page-header reference (the 56px
workspace rail and 244px nav sidebar live in the real app shell and are intentionally
absent from the mockups). All eight share one token block, one `.shell`/`.main` layout,
one `.pill--*` status vocabulary, and one type scale.

| # | File | Surface | State |
|---|------|---------|-------|
| 1 | `loops-index.html` | Overview / launcher (design-doc only; not a product route) | Built |
| 2 | `loops-catalog.html` | Loops catalog (the home of the feature) | Built |
| 3 | `loop-detail.html` | A single Loop's definition page | Built |
| 4 | `loop-run-form.html` | Run a Loop (the hero arrive-and-use path) | Built |
| 5 | `run-detail.html` | Live run monitor (the heaviest surface) | Built |
| 6 | `runs.html` | Runs history (global) | Built |
| 7 | `loop-editor.html` | Visual DAG builder (fork & edit) | Built |
| 8 | `loop-configure.html` | Configure sheet (no-fork tweaks) | Built |

---

## 2. Product model (the design's mental model)

- **Two nouns only:** **Loops** (the catalog of definitions) and **Runs** (their
  executions). No third noun. `[shipped-in-spec]`
- A **Loop** = a **contract** (goal, definition-of-done, verification, stop conditions)
  + a **body** (a static DAG of typed nodes) + typed **declared inputs**. It executes on
  AGH's autonomy kernel, not a second executor. `[shipped-in-spec]`
- **Arrive-and-use is the hero path:** pick a Loop, fill an auto-generated input form,
  Run, watch it iterate live. Built-ins run with zero authoring. `[shipped-in-spec]`
- Runs iterate across **generations**. Default re-attempt strategy `failed-only` (re-runs
  failed/pending nodes plus their transitive downstream dependents, carrying succeeded
  outputs forward as read-only); override `full-body`. `[shipped-in-spec]`
- **Terminal outcomes are explicit and honest:** `done` · `failed` · `exhausted` ·
  `stalled`. **`needs-approval` is a LIVE pause, not terminal.** Other live states:
  `queued / ready / running / paused / watching`. `[shipped-in-spec]`
- **Authoring is fork-and-edit only** (ADR-008): the builder opens an existing Loop or
  built-in. There is no blank-canvas from-scratch builder in v1. `[shipped-in-spec]`

---

## 3. Design system (as applied)

### 3.1 Atmosphere and hard rules
Warm near-black operator canvas; quiet, dense, intentional; flat depth; a single scarce
accent; signal colors as desaturated tint + text (never solid banners). The golden rule:
**color carries STATE, not category.** Node *class* (action/control/source) is a neutral
mono label; only run/node *state* gets color.

Hard rules honored across all screens (and required of any new screen): one accent used
sparingly; no side-stripe accent borders; no gradient backgrounds; flat depth (shadows
only on overlays/modals via `--shadow-overlay`); no em dashes in copy; real product UI
only (no demo/viewport/theme toggles, no metadata/process cards).

### 3.2 Tokens (copy verbatim into every screen `:root`)
```
surfaces  --rail #0c0b0b · --canvas #131211 · --canvas-soft #1a1918 · --canvas-tint #1c1b1a · --elevated #232220
text      --fg #ececef · --fg-strong #f6f6f8 · --muted #9a9a9f · --subtle #76767c · --faint #545458
lines     --line rgba(255,255,255,.055) · --line-soft .03 · --line-strong .09
accent    --accent #e8572a · --accent-hover #d14e25 · --accent-strong #f6874f · --accent-ink #17110f · --accent-tint .10
status    success #5fbf85/.08 · warning #d6a647/.08 · danger #e0635a/.09 · info #8e8eb5/.12 · neutral #7a7a80/.06
glaze     --row-hover .022 · --row-selected .03 · --input-fill .025 · --badge-fill .05 · --btn-fill .04 · --bar-fill .18
type      Inter Variable (cv01,ss03) + JetBrains Mono · body 13.5px · detail-h1 22px/-.028em · section label 10.5-11px upper · mono ids ~11px
radii     4/5/6/8/10/14 · pill 9999 · motion 140ms cubic-bezier(.2,0,0,1)
```

### 3.3 Shared status vocabulary (`.pill--*`, tint bg + saturated text)
| State | Token | Pulse | Kind |
|-------|-------|-------|------|
| running | accent | yes | live |
| watching | info | yes | live |
| needs-approval | info | no | live pause |
| queued / ready / paused | neutral / accent | - | live |
| done | success | no | terminal |
| failed | danger | no | terminal |
| exhausted | warning | no | terminal |
| stalled | neutral | no | terminal |

In dense tables a 6px state dot inside the pill is enough; the pulse is reserved for live
states and wrapped in `prefers-reduced-motion`.

### 3.4 Shared component vocabulary (build these once, reuse everywhere)
Topbar + breadcrumb; `.btn` / `.btn--primary` / `.btn--ghost` / `.btn--danger` /
`.btn--icon`; `.pill--*`; `count-chip`; section label (uppercase + hairline);
`.panelbox`; key-value rows (`.kv`); segmented filter (`.segment`/`.seg`); category pills;
status legend; form controls (`.input`, `select.input`, `.textarea`, `.switch`,
agent-picker, pill-group/segmented, `<details>` collapsible); meters/progress bars; node
spine (run timeline) and node canvas (editor); gate card (flat tint); embedded channel;
approval gate.

---

## 4. Screen specifications

> Each screen: **Purpose · Route/IA · Layout · Key components · Data shown ·
> Interactions · Spec terms surfaced · Data contract (what the daemon must expose).**

### 4.1 `loops-catalog.html` - Loops catalog
- **Purpose.** The home of the feature: browse built-in and forked Loops, filter, see
  last outcome + success rate, and launch a run. Arrive-and-use entry point.
- **Route/IA.** `home › Loops`. Topbar: search (`/`), `Runs` link, `New from template`
  (primary). `[VERIFY]` "New from template" = fork-and-edit entry (ADR-008), not a blank
  builder.
- **Layout.** Page head (title + `count-chip` + meta line) · filter toolbar · grouped
  **list** (Built-in, Custom), not a card grid.
- **Key components.** Filter `segment` by kind (All / Built-in / Custom) + category pills
  (Engineering / Operations / Evaluation / Content / Design); list rows with neutral icon
  well, name + kind tag + `agh.loop/v1` or source slug, one-line goal, meta
  (`N nodes` / `N inputs` / iteration cap or `human gate`), category, last-outcome pill,
  30d success-rate stat, inline `Run` button.
- **Data shown (8 sample Loops).** 6 built-in: `software-delivery` (Engineering, 8 nodes,
  4 inputs, cap 50), `reviews-watch` (Operations, watch-source, cap ∞), `eval-harness`
  (Evaluation, 5 nodes), `docs-sync` (Content, 4 nodes), `incident-triage` (Operations,
  watch-source, human gate), `design-audit` (Design, 3 nodes). 2 custom forks:
  `acme-release-train` (← software-delivery), `weekly-changelog` (← docs-sync).
- **Interactions.** Kind + category filters (JS, hides empty groups); row click → detail;
  `Run` → run form (stops propagation).
- **Spec terms surfaced.** kind (built-in/custom/fork), category, node count, input count,
  iteration_cap (incl. unbounded `∞`), last terminal/live state, success rate.
- **Data contract.** `GET /loops` returning, per Loop: name, kind, source-of-fork,
  category, goal, node count, declared-input count, iteration_cap, last-run state, and a
  30d success-rate aggregate. `[VERIFY]` success-rate and run-count aggregates exposed by
  daemon vs computed view (section 9.11).

### 4.2 `loop-detail.html` - Loop definition page
- **Purpose.** Everything about one Loop: contract, body graph (read-only), recent runs,
  declared inputs, limits vs ceilings, versions, 30d stats.
- **Route/IA.** `Loops › software-delivery`. Topbar: `View DSL`, overflow.
- **Layout.** DetailHeader (icon, name, `Built-in` + `v4 · published` tags, meta line;
  actions `Configure`, `Fork & edit`, `Run loop` = the one accent CTA) · two-column:
  main (Contract, Body·DAG, Recent runs) + 320px right rail (Declared inputs, Limits &
  budget, Versions, Last 30 days).
- **Key components.** `.kv` contract rows; verification rows (`command` / `agent-judge` /
  `human` with method line); terminal-outcome chips; read-only DAG (`.graph` of neutral
  `.gnode`, gate nodes tint verdict text, fan-out cluster of `.gbranch`); recent-runs rows
  with state pill; right-rail input rows (name, required `*`, type badge, desc, default),
  limits rows (`default / ceiling`), versions list (current badge), 4-up stat grid.
- **Data shown.** software-delivery: goal, DoD (`go test ./...`, `golangci-lint run`),
  4 verification checks, 4 terminal chips, 8-node body, 4 declared inputs
  (slug*, implementer, auto_merge, target_branch), 7 limit rows, v1-v4, 30d stats
  (92% / 48 runs / 1.9 avg gens / 8m median).
- **Interactions.** Links to editor, run form, runs. (Mostly static.)
- **Spec terms surfaced.** contract{goal, definition_of_done, verification[],
  terminal_states}; graph node classes/kinds; declared input types + defaults; the full
  limits/ceilings table; version model; run aggregates.
- **Data contract.** `GET /loops/:name` (full definition + computed stats). The right-rail
  limits show both per-loop default and daemon ceiling - the daemon must expose both.

### 4.3 `loop-run-form.html` - Run a Loop (hero path)
- **Purpose.** The arrive-and-use moment: an auto-generated, validated input form + a live
  contract preview, with optional per-run limit overrides.
- **Route/IA.** `Loops › software-delivery › Run`. Sticky bottom action bar.
- **Layout.** Page head (`Run software-delivery`) · two columns: form (left) + sticky
  contract preview (right) · sticky action bar (Cancel · Dry run · Run loop).
- **Key components.**
  - **Inputs (auto-generated from declared `inputs`).** `slug` string (required, mono),
    `implementer` agent (avatar-prefixed picker, default `code-implementer`),
    `target_branch` string (default `main`), `auto_merge` bool (switch). Each field shows
    a type badge; required marker `*`; inline required error.
  - **Advanced · per-run limit overrides** (`<details>` collapsible). 6 number fields, each
    showing the per-loop default value and the daemon ceiling, clamped at the ceiling:
    iteration cap 50 /100, token budget off /20M, wall clock off /7d, no-progress window
    3 /10, fan-out ceiling ≤ tasks /64, gate max revisions 3 /10 (cost is display-only, no
    input — §9.5.2). Badge flips "using Loop defaults" → "overrides set".
  - **Preview.** Live "what will run" summary (updates from inputs); read-only contract
    (goal, DoD, 4 verification rows, 4 terminal chips); lifecycle line
    (queued → running → verify → done).
- **Interactions.** Live summary recompute on every field; `auto_merge` toggles the merge
  line + the merge-approval verification note; override clamp at ceiling; required
  validation (Run disabled until `slug`); `Dry run` validates inputs + renders the plan
  without starting a run (toast).
- **Spec terms surfaced.** declared input types (string/agent/bool shown; number/file/ref
  defined in the type system); per-run overrides; daemon ceilings; dry-run; lifecycle
  states; terminal outcomes.
- **Data contract.** `GET /loops/:name` (inputs schema + contract + defaults + ceilings);
  `POST /loops/:name/runs` with `{inputs, overrides}`; a dry-run mode (section 9.1).
  The form is generated entirely from the declared-input schema, so the schema must carry
  per-input: name, type, required, default, description. `[VERIFY]` ref/file/agent picker
  data sources (section 9.10).

### 4.4 `run-detail.html` - Live run monitor (heaviest)
- **Purpose.** Truthful, real-time view of a running Loop: contract, live meters,
  generation-by-generation timeline, fan-out, gate verdicts, multi-agent channel, and the
  human approval gate.
- **Route/IA.** `Loops › software-delivery › r-8f3a2b`. Topbar: `Graph view`, `DSL`.
- **Layout.** Two columns: main (sticky contract header + meters, then timeline) + 332px
  right rail (Live events, Run facts, Terminal-outcome legend).
- **Key components.**
  - **Sticky contract header.** Live `Running` pill (pulse), `Generation N · attempt N of
    cap`, started/trigger/actor meta, the goal, `Pause` + `Stop run` actions.
  - **5 meters.** Attempts (2/50), Tokens (412K/2M), Wall clock (14m/30m), Cost ($1.84),
    Breadth (4/64). Bars warn-tint near ceiling only.
  - **Generation timeline.** Collapsible `gen` cards on a flat **node spine**. G1
    revised (slug → load-tasks file-import → fan-out over the authored tasks in dependency
    order, 1 failed branch → collect → review agent-judge → request_changes → revise). G2
    running (carry-forward of 3 task outputs, `execute-task` re-running only the failed task
    via run-agent, pending review + verify, human approval gate).
  - **Gate card.** Flat tint (`pass` success / `fail` danger), verdict + reason + route
    (`revise` / `next_generation`). No side-stripe, no gradient.
  - **Channel.** Embedded `#delivery-r8f3a` with implementer/reviewer/decision messages.
  - **Approval gate.** `needs-approval` tag, "Approve merge to `main`?", facts (branch,
    diff, tests, verifier), actions: `Approve & resume`, `Request changes`, `Reject & halt`.
  - **Right rail.** Streaming live events (`node_running`, `channel_msg`,
    `generation_started`, `gate_verdict`, `node_failed`, `node_succeeded`); run facts
    (loop, revision pinned, re-attempt, trigger, workspace, run id); terminal legend.
- **Interactions.** Collapse generations + node logs; gentle live simulation nudges meters
  + prepends events (guarded by reduced-motion).
- **Spec terms surfaced.** live + terminal states; generations; attempts vs iteration_cap;
  token/wall/cost/breadth meters; no-progress window count; node classes/kinds incl.
  `channel-post`/converse-and-decide and `harvest`; gate verdict + routing
  (`revise`/`next_generation`); carry-forward (failed-only); `needs-approval` live pause;
  pinned revision; trigger source.
- **Data contract.** `GET /runs/:id` + an SSE/event stream. Events the UI binds:
  `node_running|node_succeeded|node_failed`, `gate_verdict`, `generation_started`,
  `channel_msg`, `token_tick`/budget updates, `needs_approval`. Run object must expose:
  state, generation index, attempt/cap, token/wall/cost/breadth usage + caps, pinned
  revision, re-attempt mode, trigger, per-node status + output, gate verdicts + route,
  channel transcript + harvested decision, approval-gate payload. Controls: pause, resume,
  stop, and approval decision (approve / request-changes / reject). (section 9.3, 9.8)

### 4.5 `runs.html` - Runs history (global)
- **Purpose.** Every execution across every Loop, the full outcome spectrum, filterable.
- **Route/IA.** `Loops › Runs`. `[VERIFY]` product-ux currently says no separate global
  Runs route (section 9.4).
- **Layout.** Page head (live "3 active") · KPI strip (4) · outcome filter `segment` +
  Loop/date selects · Active table · Past table.
- **Key components.** KPIs (Active now, Awaiting you, Done today, Needs a look); outcome
  segment (All / Running / Watching / Needs-approval / Done / Failed / Exhausted /
  Stalled with counts); table columns (Outcome pill, Loop + run-id/trigger, Goal,
  Gens, Started/Ended, Budget mini-bar with warn/danger near limit, chevron).
- **Data shown.** 24 runs (3 active, 21 past) across all states, multiple Loops/triggers
  (web, cli, agent, schedule, webhook).
- **Interactions.** Outcome filter (JS, hides empty sections); row → run detail.
- **Spec terms surfaced.** full state spectrum; gens vs cap; trigger sources; budget
  usage + cap with overflow→exhausted; "awaiting you" = needs-approval queue.
- **Data contract.** `GET /runs?status=&loop=&since=` + the same aggregates the KPIs need.
  If global Runs stays, the daemon needs a workspace-wide run index (section 9.4).

### 4.6 `loop-editor.html` - Visual DAG builder (fork & edit)
- **Purpose.** Fork an existing Loop or built-in and edit its body on a canvas: nodes,
  edges, per-node inspector, inline linter, versions, Graph/DSL views. ADR-008: no
  blank-canvas builder.
- **Route/IA.** `Loops › software-delivery › Fork & edit`. Topbar: `Unsaved changes`
  chip, version selector (`v5 · draft`), `Validate`, `Save draft`, `Publish` (disabled
  while issues exist). Sub-toolbar: auto-layout / zoom / fit; Graph|DSL segmented; fork
  context; the 4 linter invariant chips.
- **Layout.** Three regions: 190px node palette · canvas (scrollable, dot-grid) with a
  bottom linter dock · 344px node inspector.
- **Key components.**
  - **Palette ("Add node").** Grouped by class: Action (run-agent, call-tool,
    channel-post), Control (fan-out, collect, branch, gate, sub-loop), Source
    (watch-source, file-import, input). Drag affordance. Fork-and-edit note.
  - **Canvas.** Positioned nodes + SVG edges (ReactFlow-style, not a graph lib). Neutral
    node cards (class·kind label, name, kind). Selected node = accent ring; error node =
    danger ring + badge. Fan-out renders 4 branch chips.
  - **Linter dock.** 4 invariants (acyclicity, reachability, termination, fan-out bounds)
    as pass/fail chips + an issue list. Demo issue: `implement.max_fan_out (80) exceeds
    the daemon ceiling of 64`, code `fan_out_ceiling_exceeded`, "publish returns 422 until
    resolved", with `Reveal node`.
  - **Inspector (per class/kind).** Rendered from a node model:
    - source/input: id, input_ref, type (read-only).
    - action/run-agent: id, kind, action_ref, session handle, isolated toggle, timeout,
      produces (harvest).
    - control/gate: id, check kind (command|agent-judge|human|extension), rubric, verdict
      source, on_result routing (in-body: continue|revise|branch|halt|escalate), max
      revisions, hint.
    - control/fan-out: id, collection_ref, max_fan_out (number, ceiling 64), paired
      collect, hint.
    - control/collect: id, joins, hint.
    - control/gate (DoD): id, check command, expect, on_result (definition-of-done:
      continue→done|next_generation|halt|escalate), the loop verification list.
  - **Graph|DSL toggle.** DSL view shows the `agh.loop/v1` YAML on disk (the bijective
    codec / FS-as-truth from ADR-015), with the offending `max_fan_out: 80` highlighted.
- **Interactions (real).** Click node → inspector swaps; lowering `max_fan_out` ≤ 64 in
  the inspector clears the issue, flips the invariant to pass, hides the node badge, and
  enables Publish (a faithful render of the linter→422→publish gate); dock collapse;
  Graph/DSL switch; zoom.
- **Spec terms surfaced.** ADR-003 node classes (action open; control/source closed);
  per-kind config field names; edges (`blocks` deps, acyclic); start[]; the 4 linter
  invariants and the 422-per-node publish contract; fork-and-edit (ADR-008); draft vs
  published + versions; bijective definition↔graph codec (ADR-015).
- **Data contract.** `GET /loops/:name` (definition → graph), `PATCH/POST /loops/:name`
  (graph → definition; returns 422 with per-node `{node_id, code, message, severity}`),
  validate (lint without publish), versions list/diff/revert. **The editor needs a store
  for node layout positions** keyed `(workspace_id, loop_name, node_id)` per ADR-015
  (section 9.7). Registry lookups for action kinds / agents / tools to populate the kind
  dropdowns (section 9.10).

### 4.7 `loop-configure.html` - Configure (light, no-fork)
- **Purpose.** Adjust how a Loop runs without changing its structure. Power ceiling, never
  the hero.
- **Route/IA.** Right-side **sheet/drawer** over a dimmed loop-detail backdrop; opened
  from `Configure` on detail/catalog. Close → reopen pill (mockup affordance).
- **Layout.** Sheet header (icon, `Configure`, `software-delivery · no-fork tweaks`,
  close) · scroll body (4 groups) · footer (Reset to defaults · Cancel · Save
  configuration; saved toast).
- **Key components / groups.**
  1. **Verification checks** ("declared in the Loop"): per-check enable switch + project
     command field where the check is a generic `command` gate (Test suite `go test ./...`,
     Linter `golangci-lint run`, Acceptance review agent-judge - cannot be removed without
     a fork). Disabling a command check disables its command field.
  2. **Human approval gate**: a single switch (Merge approval) - the human-gate toggle.
  3. **Re-attempt strategy**: two selectable cards `failed-only` (default) | `full-body`,
     with descriptions (ADR-009/011).
  4. **Stop limits** (per-loop defaults): 7 number fields with ceilings, clamped at
     ceiling (same set/values as the run-form Advanced panel).
- **Interactions.** Switches; command-field enable/disable; strategy cards; limit clamps;
  Reset restores defaults + failed-only; Save toast; close/reopen.
- **What CANNOT change here (needs a fork):** node order / DAG structure, node kinds,
  input declarations, terminal states / contract shape, goal / definition-of-done.
- **Spec terms surfaced.** no-fork config layer (ADR-009); verification selection vs
  structural edit boundary; human-gate toggle; re-attempt granularity; per-loop limit
  overrides bounded by ceiling; save/reset semantics.
- **Data contract.** `GET /loops/:name/config` + `PUT /loops/:name/config` writing per-loop
  defaults (limits, enabled checks, human-gate flag, re-attempt mode, project command
  overrides). Distinct from a fork (which writes a new definition). (section 9.6)

### 4.8 `loops-index.html` - Overview / launcher
- **Purpose.** A design-doc launcher that explains the two nouns and links to all screens
  with Built/Planned tags. **Not a product route** - it is scaffolding for review. All
  cards are now `Built`.

---

## 5. Cross-cutting domain model (UI-facing contract)

### 5.1 Run state machine (as rendered)
Live: `queued` → `ready` → `running` ⇄ `paused`, plus `watching` (watch-driven) and
`needs-approval` (live pause awaiting a human). Terminal: `done` | `failed` | `exhausted`
| `stalled`. The UI renders every state with the shared `.pill--*` vocabulary and never
invents a state outside this set. `[shipped-in-spec]`

### 5.2 Node taxonomy (ADR-003)
- **action** (open/pluggable): `run-agent`, `call-tool`, `channel-post`
  (converse-and-decide). Config: kind, execute/action_ref, session{handle, isolated},
  timeout, harvest, produces.
- **control** (closed enum): `fan-out` (collection_ref, max_fan_out), `collect` (join
  barrier), `branch` (condition - AST shape `[VERIFY]`), `gate` (check + on_result +
  max_revisions), `sub-loop` (loop_ref, own contract).
- **source** (closed enum): `watch-source` (watch_kind, spec, no_progress_window),
  `file-import` (pattern), `input` (input_ref).
Node *class* is a neutral mono label; never color-coded.

### 5.3 Verification kinds
`command` (exit code / stdout), `agent-judge` (rubric → revise/approve), `human`
(approval gate), extension via tool registry. A gate's `on_result` routing differs by
placement: **in-body** `continue|revise|branch|halt|escalate`; **definition-of-done**
`continue (→done)|next_generation|halt|escalate`.

### 5.4 Limits and ceilings (the numbers the UI renders)
> These are the values shown consistently across `loop-detail`, `loop-run-form`,
> `loop-configure`, and `run-detail`. Left = per-loop default, right = hard daemon ceiling.

| Limit | Default (delivery) | Ceiling | Notes |
|-------|--------------------|---------|-------|
| iteration_cap | 50 | 100 | watch loops default 0 (unbounded, shown `∞`) |
| budget.tokens | off (0) | 20M | 0 = unlimited; opt-in |
| budget.wall_clock | off (0) | 7d | opt-in |
| budget.usd (cost) | display-only | — | derived tokens × price; no enforced cap (§9.5.2) |
| no_progress.window | 3 | 10 | window count, advances on node completion |
| fan_out_ceiling | ≤ tasks | 64 | bounded by the loaded task count; hard cap 64; overflow → exhausted |
| gate.max_revisions | 3 | 10 | per gate; after limit, gate fails terminal |

Ceilings are hard backstops, never editable in the UI. A run making progress is never
killed by a budget wall (progress-first, ADR-012).

> **Resolved (TechSpec §9.5.1 / ADR-017):** the canonical defaults are `iteration_cap=50`
> and budgets `0`/unlimited (off). The mockups now render these canonical values across all
> screens; the earlier 10 / 2.0M design figures are retired.

### 5.5 Generations and re-attempt
A run iterates across generations. `failed-only` (default) re-runs failed/pending nodes +
transitive dependents and carries succeeded outputs forward read-only; `full-body` re-runs
everything. The strategy is an author/configure-time choice (ADR-009/011), shown as a
read-only **run fact** on `run-detail`, never chosen on the run form.

---

## 6. Component build inventory (for implementation)

Reusable, build-once components implied by the eight screens:

- **Shell:** `Topbar` (breadcrumb + trailing actions/search), content `Scroll`, sticky
  `ActionBar`, right `Rail`, `Sheet`/drawer (over scrim, `--shadow-overlay`).
- **Primitives:** `Button` (primary/ghost/danger/icon), `StatePill` (the `.pill--*` set
  + pulse), `CountChip`, `Tag`, `SectionLabel`, `Panelbox`, `KVRow`, `MetaLine`.
- **Forms:** `TextInput` (+ mono), `NumberInput` (+ ceiling clamp + error), `Select`,
  `Textarea`, `Switch`, `SegmentedControl`/`PillGroup`, `AgentPicker`, `RefPicker`,
  `FilePicker`, `FieldRow` (label + required + type badge + hint + error),
  `Collapsible` (`<details>`), `LimitOverridesGrid`.
- **Data display:** `FilterSegment` + `CategoryPills`, `LoopListRow`, `RunTableRow` +
  `BudgetMiniBar`, `KPICard`, `Meter` (with warn state), `StatGrid`, `StatusLegend`.
- **Loops-specific:** `ReadOnlyDAG` (graph view), `DAGCanvas` + `Node` + `Edge` +
  `NodeInspector` (per-class field sets) + `LinterDock` + `Palette` (editor),
  `GenerationTimeline` + `NodeSpine` + `NodeRow`, `GateCard` (pass/fail/route),
  `EmbeddedChannel` + `ChannelMessage`, `ApprovalGate`, `ContractPreview`, `DSLView`
  (YAML render of `agh.loop/v1`).

These map cleanly to `web/src/systems/loops/*` + `packages/ui` primitives. The editor's
node-inspector field sets are the single most reused data structure (defined per
class/kind in `loop-editor.html`'s `NODES` model) and should drive both the editor and the
read-only viewer reused on detail/run pages.

---

## 7. Per-screen data contract summary (daemon surfaces the UI assumes)

| Screen | Reads | Writes / actions | Realtime |
|--------|-------|------------------|----------|
| catalog | `GET /loops` (+ 30d aggregates) | run (→ run form) | - |
| detail | `GET /loops/:name` (definition + stats) | - | - |
| run form | `GET /loops/:name` (input schema, contract, defaults, ceilings) | `POST /loops/:name/runs {inputs, overrides}`; dry-run | - |
| run detail | `GET /runs/:id` | pause / resume / stop; approval decision | SSE event stream |
| runs | `GET /runs?status=&loop=&since=` (+ KPI aggregates) | - | active rows live |
| editor | `GET /loops/:name` (→ graph); registry/agent lists; node positions | `PATCH/POST /loops/:name` (→ 422 per-node); validate; versions diff/revert; save positions | - |
| configure | `GET /loops/:name/config` | `PUT /loops/:name/config` | - |

Every UI action must have a CLI/HTTP/UDS equivalent over the same daemon state
(agent-manageability, PRODUCT.md principle 3): `agh loop list|inspect|create|edit`,
`agh run start|list|inspect|pause|resume|stop|approve`.

---

## 8. Interaction fidelity built into the mockups
Not screenshots - the following behave: catalog kind/category filters; run-form live
preview + required validation + override clamps + dry-run; run-detail collapsible
generations/logs + live meter/event simulation; editor node selection → inspector swap +
the linter (fan-out width > ceiling → issue → publish blocked) + Graph/DSL toggle + zoom;
configure switches + command enable/disable + strategy cards + limit clamps + save/reset +
close/reopen; runs outcome filter.

---

## 9. Spec alignment - what to confirm or change in PRD / TechSpec

> The actionable output. Each item: what the design surfaces, the question, and a proposed
> resolution. Priority: **P0** blocks a truthful implementation; **P1** should be decided
> before build; **P2** polish/IA.

### 9.1 Dry run `[P1]`
- **Design.** Run form has a `Dry run` secondary action ("validates inputs and renders the
  first-generation plan without starting a run").
- **Question.** PRD/TechSpec do not formally define a dry-run verb or endpoint.
- **Proposal.** Either (a) add dry-run to the spec with explicit semantics (validate inputs
  against the input schema + contract, render the gen-1 plan/preview, spend no budget,
  create no Run, return a plan artifact), exposed as `POST /loops/:name/runs?dry=true` and
  `agh run start --dry-run`; or (b) drop the control from the UI. Recommend (a): it is a
  cheap, honest operator affordance.

### 9.2 Branch condition AST `[P0 for the editor]`
- **Design.** The editor's `branch` node needs a condition editor; the inspector currently
  treats it as an honest expression field (no invented AST UI).
- **Question.** The condition grammar/AST shape is underspecified in the DSL.
- **Proposal.** Pin the `branch.condition` grammar in the TechSpec DSL section (operators,
  operand references to prior node outputs, type rules) before the editor's branch
  inspector is implemented. Until then, branch editing ships read-only / raw-expression.

### 9.3 Run controls: pause / resume / stop `[P1]`
- **Design.** `run-detail` exposes `Pause` and `Stop run`; the state machine includes
  `paused`.
- **Question.** Confirm the daemon supports pause/resume/stop of a live run and the
  `paused` state transitions.
- **Proposal.** Confirm in TechSpec + expose `agh run pause|resume|stop` and the HTTP
  equivalents; define what "pause" means mid-generation (boundary vs immediate).

### 9.4 Global Runs route `[P2 / IA]`
- **Design.** `runs.html` is a global, workspace-wide run history with KPIs.
- **Question.** `product-ux.md` says run history lives under each Loop with no separate
  global Runs nav route.
- **Proposal.** Decide: (a) adopt global Runs as an operator convenience (update
  product-ux + add a workspace run index + nav entry), or (b) fold it into
  catalog/loop-detail and keep `runs.html` only as the "All runs" target. Recommend (a):
  the "Awaiting you" / active-across-loops view is operationally valuable.

### 9.5 Limit defaults + cost budget `[P0]`
- **Design.** Renders iteration_cap default **10** (/100) and budget.tokens default
  **2.0M** (/20M) consistently across four screens, plus a `Cost` meter ($) on run-detail
  and a `Cost cap (USD)` field on run-form/configure (opt-in).
- **Questions.**
  1. A spec re-read suggested different config defaults (delivery iteration_cap **50**;
     budgets default **0**/unlimited). Which is canonical?
  2. `budget.usd` is not in TechSpec/ADR-012. Is cost a first-class budget dimension with
     a ceiling, or display-only (derived from tokens × price)?
- **Proposal.** (1) Reconcile the canonical default numbers in TechSpec + `config.toml`
  defaults so the UI copy matches one source; if 50/0 win, update all four screens. (2) If
  cost is display-only, keep the run-detail Cost meter but **remove the `Cost cap (USD)`
  input** from run-form/configure (truthful-UI). If cost caps are real, add `budget.usd`
  to ADR-012 with a ceiling.

### 9.6 Configure write target vs fork `[P1]`
- **Design.** Configure writes per-loop defaults (limits, enabled checks, human-gate flag,
  re-attempt mode, project command overrides) without forking; structural edits require a
  fork.
- **Question.** Confirm the daemon has a per-loop **config** store distinct from the loop
  **definition**, and the exact boundary of what config can change.
- **Proposal.** Define `loop config` as a separate persisted object (`GET/PUT
  /loops/:name/config`, `agh loop config`); document the no-fork boundary verbatim
  (the "what cannot change" list in 4.7).

### 9.7 Node layout persistence `[P1 for editor]`
- **Design.** The editor positions nodes on a canvas.
- **Question.** ADR-015 mentions UI annotations keyed `(workspace_id, loop_name,
  node_id)`. Confirm the daemon persists node x/y layout.
- **Proposal.** Confirm/define a node-annotation store + read/write path so layouts
  survive reload and never collide across same-named forked loops in different workspaces.

### 9.8 Approval-gate decision verbs `[P1]`
- **Design.** `Approve & resume`, `Request changes`, `Reject & halt`.
- **Question.** Confirm the human-gate decision set and that `Request changes` routes to
  `revise` (next generation) while `Reject & halt` → terminal `failed`/halt.
- **Proposal.** Document the decision verbs + their routing in the gate/ADR-005 section and
  expose `agh run approve|request-changes|reject`.

### 9.9 channel-post / converse-and-decide as runtime primitive `[P0]`
- **Design.** `run-detail` renders an embedded multi-agent channel as a first-class
  `channel-post` action node with a harvested decision; the editor palette offers
  `channel-post`.
- **Question.** The handoff notes converse-and-decide ships as a documented *example*, not
  a runtime-installed built-in. But the UI treats `channel-post` + result-harvest as a
  runtime node primitive (ADR-014).
- **Proposal.** Confirm in TechSpec that `channel-post` (action) + `result` harvest is a
  shipped runtime primitive the UI can bind to, independent of any example Loop that uses
  it. If not shipped in v1, mark the channel surface as forward-looking and gate it.

### 9.10 Picker data sources: agent / ref / file kinds `[P1]`
- **Design.** Run form renders an `agent` picker; the type system includes `ref` and
  `file`. The editor kind dropdowns list registry actions/tools/agents.
- **Question.** What entity kinds can a `ref` target, and what feeds the agent/tool/action
  pickers?
- **Proposal.** Enumerate `ref` target kinds (task/run/channel/...) in the DSL input-type
  section, and define the registry-list endpoints the pickers read.

### 9.11 Computed aggregates: success rate, run counts, 30d stats `[P2]`
- **Design.** Catalog + detail show success rate, total runs, avg generations, median
  duration; runs KPIs show active/awaiting/done/needs-a-look.
- **Question.** Are these daemon-provided aggregates or a computed view?
- **Proposal.** Confirm an aggregate endpoint or document them as a derived read model so
  the UI is not inventing metrics.

### 9.12 Watch loops: unbounded cap + quiet/stall rendering `[P2]`
- **Design.** Catalog shows `reviews-watch` cap `∞`; runs shows watching/quiet/stall.
- **Question.** Confirm watch loops default iteration_cap 0 (unbounded), the quiet-window
  → stalled transition, and how the UI should render "unbounded".
- **Proposal.** Confirm in ADR-012 + the watch-source spec; standardize the `∞` rendering.

### 9.13 Versions + version diff `[P2]`
- **Design.** Editor shows `v5 · draft` + a version selector; detail shows a versions list;
  diff/revert are referenced.
- **Question.** Confirm the version model (draft vs published, revert creates a new
  version) and whether version-diff is in v1 scope.
- **Proposal.** Document the version lifecycle; schedule version-diff as a follow-up
  surface if out of v1.

---

## 10. Open decisions (carried from the handoff, now consolidated)
- **Palette:** AGH operator (warm-dark + orange), matching the real app. A Linear-indigo
  reskin is a pure token swap.
- **App shell:** mockups are content + header only; the real app supplies rail/sidebar.
- **Editor canvas texture:** a very subtle dot-grid (functional "this is a canvas" signal)
  is the one place a `radial-gradient` is used; remove if strict no-gradient is preferred.
- **Remaining optional surfaces (not yet designed):** version diff, node-run drilldown
  modal, watch-source quiet/stall detail state, empty states.

---

## 11. How to use this with the spec
1. Walk section 9 top to bottom; for each, open the cited spec file and either confirm the
   design or file a spec change. P0 items (9.2, 9.5, 9.9) gate a truthful build.
2. During implementation, use sections 4 (per-screen contract), 6 (component inventory),
   and 7 (data surfaces) as the build brief. The node-inspector field model in
   `loop-editor.html` is the canonical per-kind config schema.
3. Keep this file in sync: when a screen changes, update its section 4 entry and re-check
   section 9.
