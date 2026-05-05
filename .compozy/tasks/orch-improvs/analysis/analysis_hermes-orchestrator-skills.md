# Analysis: hermes-orchestrator-skills

Read-only exploration of `.resources/hermes/` (kanban-as-orchestrator skills) for the AGH task `orch-improvs`. Cross-referenced with AGH `internal/skills/` and RFCs 001/002.

## Scope

This analysis covers how Hermes wraps its kanban primitive into a multi-role orchestration system through skills, focused on `kanban-video-orchestrator` (an opinionated meta-pipeline that bootstraps a working board from a brief) and the bundled `kanban-orchestrator` / `kanban-worker` devops skills (which provide reusable orchestration playbooks). Cross-referenced against AGH's skills system per RFC 001/002 and `internal/skills/`.

In scope: SKILL.md surface and references, the `bootstrap_pipeline.py` plan-to-setup compiler, `monitor.py` polling loop, the three asset templates (`soul.md.tmpl`, `brief.md.tmpl`, `setup.sh.tmpl`), and the bundled devops skill docs.

Out of scope: the kanban runtime/dispatcher itself (which lives in the hermes-agent core, not these skills), and the per-medium creative skills (`ascii-video`, `manim-video`, etc.) referenced by the orchestrator.

- Path explored: `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/` (full), `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-{worker,orchestrator}.md` (full).
- Topic: kanban as orchestration primitive — multi-role pipeline, role archetypes, plan compiler, intake funnel, monitor loop, anti-temptation rules.
- Files read in full vs. sampled: 14 files read in full (SKILL.md + all references + scripts + assets + bundled devops docs). Cross-reference RFCs sampled.
- Total available files: ~30 in this skill bundle plus 2 bundled devops docs.

## Overview

Hermes turns a kanban into a **multi-role agent pipeline** by composing four pieces:

1. **A baseline runtime contract** — every kanban worker spawned by the dispatcher gets `KANBAN_GUIDANCE` auto-injected into its system prompt (lifecycle: orient → work → heartbeat → block/complete; the `kanban_create` fan-out pattern; the "decompose, don't execute" rule). This is implemented in `agent/prompt_builder.py` (referenced from `devops-kanban-worker.md:11`).

2. **Two reusable, content-only skills** — `kanban-orchestrator` and `kanban-worker` are deeper playbooks loaded via `always_load` when a profile needs more than the baseline. They are **convention bearers**, not code: they ship decomposition recipes, summary/metadata shapes, retry diagnostics, anti-temptation rules.

3. **An opinionated wrapper skill** — `kanban-video-orchestrator` is a *meta-pipeline*: it does not render anything; it scopes a brief, designs a team from a role library, and **emits a `setup.sh` that creates the profiles and fires the initial kanban task**. The skill ships scripts + templates + per-style references plus SKILL.md.

4. **Profiles + a shared `dir:` workspace as the integration substrate** — every role becomes a long-lived Hermes profile (cloned config, custom `SOUL.md`, patched `toolsets`/`always_load`). All profiles for one project share `~/projects/video-pipeline/<slug>/` as their `--workspace dir:` so artifacts pass via filesystem + structured `kanban_complete` handoffs.

The mental model in one line: **the kanban is the queue, the role library is the fleet template, the brief is the contract, `dir:`-workspace is shared memory, and `kanban_complete(summary, metadata)` is the typed message between roles.**

The orchestrator/worker split mirrors a planner/executor split:
- `kanban-orchestrator` is loaded only on `director`-style profiles; its rules **forbid execution** (`devops-kanban-orchestrator.md:48-53`) — "Do not execute the work yourself … decompose, route, summarize."
- `kanban-worker` is loaded on every spawned worker; it adds workspace handling, tenant prefixing, good summary shapes, retry diagnostics, and pitfalls (`devops-kanban-worker.md:32-150`).

Worker-vs-orchestrator division of labor:

| Concern | Orchestrator (director) | Worker (specialist) |
|---|---|---|
| Tools | kanban + (sometimes) terminal/file but rules forbid use | kanban + terminal + file + role-specific (vision/video) |
| Output | New kanban tasks; their own task `kanban_complete` with task-graph metadata | Concrete artifacts in shared workspace; `kanban_complete(summary, metadata)` |
| Failure mode | Re-decompose, comment with feedback, spawn re-run task as child | `kanban_block(reason)` for human input; `kanban_comment` for context |
| Lifetime | Long, one per kanban "job" | Short, one per task |

## Mechanisms / Patterns

### 1. Adaptive intake → brief contract

Discovery is a 3-tier funnel (`references/intake.md:8-141`):

- **Tier 0 baseline** (always asked, max 3 questions): what / how long / aspect+platform.
- **Style classification** maps to one of nine archetypes (narrative / product / music video / explainer / tutorial / ASCII / abstract / documentary / installation), each with its own per-style follow-up question bank.
- **Tier 2 closing** (brand assets, codec, deadline, quality bar) plus a "Reasonable assumption defaults" table (`intake.md:144-159`) that lets the agent state assumptions instead of asking.

Anti-pattern explicitly called out: "Asking 10 questions at once. Maximum 4 per turn" (`intake.md:163`).

The output is `brief.md` (`assets/brief.md.tmpl`) — an 8-section structured contract: Concept, Scope, Scenes (table), Audio, Deliverables, Constraints. Closing line is the load-bearing rule: *"This brief is the contract. The director and every downstream profile read it. If the brief changes, the kanban must be re-fired — don't edit live"* (`brief.md.tmpl:78-79`). This gives the pipeline a versioning story: the brief is **read-only after setup**, mutations require re-firing.

### 2. Role archetypes as a composable library

`references/role-archetypes.md` is the heart of the orchestration mental model. It defines roles as *archetypes* — composable templates, not fixed slots:

- **Always-present**: `director` (the only mandatory role; toolsets `kanban+terminal+file`, skill `kanban-orchestrator`; SOUL.md forbids execution; `role-archetypes.md:14-29`).
- **Pre-production**: `writer`, `copywriter`, `concept-artist`, `storyboarder`, `cinematographer/dp` (`role-archetypes.md:32-93`).
- **Production**: `renderer` plus 11 **specialized renderer variants** keyed by skill (`renderer-ascii` → `ascii-video`, `renderer-manim` → `manim-video`, `renderer-3d` → `blender-mcp`, …). The variant naming convention is `kebab-case` with a descriptive suffix when multiple instances of the same role need different focus (`role-archetypes.md:7-9`, `108-126`).
- **Post-production**: `editor`, `colorist`, `audio-mixer`, `captioner`, `masterer`.
- **QA**: `reviewer`, `brand-cop`.

The library is paired with a **composition heuristics block** (`role-archetypes.md:266-285`) that tells the orchestrator which roles to add ("Add `writer` if scripted dialogue exceeds a tagline", "Add `cinematographer` if multiple renderer instances need consistent visual language", "Add `audio-mixer` when there are 2+ audio sources") and an **anti-pattern block** (`role-archetypes.md:288-298`) — "One renderer doing everything", "A separate profile per scene", "A 'general' profile that does everything".

Key insight: **profiles are per-role, not per-scene** (`role-archetypes.md:293`). Eight scenes use one or two renderer profiles, not eight. The kanban tasks fan out per scene over a small profile fleet.

### 3. Tool matrix — scope restriction per archetype

`references/tool-matrix.md` is the role × (toolset, always_load skill, env_required) table. Scope restriction is per-role:

- `writer / copywriter` — `kanban + file` only, no terminal (`tool-matrix.md:113-124`). "Writers don't need it."
- `cinematographer` — `kanban + terminal + file + video + vision` (`tool-matrix.md:163-180`); `video` toolset is opt-in (`hermes tools enable video`) and gives `video_analyze` (sends a clip to a multimodal LLM via OpenRouter, 50 MB cap).
- `reviewer / brand-cop` — `kanban + terminal + file + video + vision` so it can natively review without frame-extraction code (`tool-matrix.md:271-282`).
- `director` — same toolset shape as a worker but the SOUL.md *plus* `kanban-orchestrator` skill restrict it from executing. Restriction is policy + persona, not tool removal — `role-archetypes.md:27-29`: "The director has the same toolset as everyone else, but its `SOUL.md` rules **forbid** execution."

This is significant: scope is enforced through **content-level prompt rules + audit logs**, not capability removal. The kanban dispatcher's audit trail catches violations after the fact rather than preventing them at the tool layer.

### 4. Skill packaging — `bootstrap_pipeline.py` as a content compiler

`scripts/bootstrap_pipeline.py` is a 502-line Python compiler that turns a structured `plan.json` into an idempotent `setup.sh`:

- Validates a typed plan (`validate_plan`, `bootstrap_pipeline.py:78-129`) — required keys, profile-name regex `[a-z0-9][a-z0-9_-]{0,63}`, deduplicated profile names, `team` must include a `director` role, `slug` regex.
- Renders three artifacts via per-asset templates: `brief.md` (`render_brief`, `:132-201`), `TEAM.md` (`render_team_md`, `:204-321`) — including a derived **task graph** with conventional T-IDs and parent links (`:225-309`), and `setup.sh` (`render_setup_sh`, `:324-400`) with API-key checks, scene-dir creation, profile-create commands, profile-config patching commands, and SOUL writes.
- Writes per-role `SOUL.md` with `render_soul_md` (`:403-461`) — common rules (read brief/team graph; pass `workspace_kind="dir"` and `workspace_path`; use the project tenant; emit heartbeats), plus director-only rules ("Do not execute the work yourself", "Read TEAM.md", "Load `kanban-orchestrator`").

The compiler **derives the task graph from team composition + scene list**, encoding conventional dependencies:
- `cinematographer` becomes the parent of all renderers when present.
- `music-supervisor` becomes a parent of every renderer (so beats are available).
- `audio-mixer` parents = `music-supervisor + voice-talent` outputs.
- `editor` parents = all renderer scenes + `audio-mixer/voice-talent/music-supervisor`.
- `captioner` chains off `editor`; `reviewer` is the terminal node.

This is a **conventional graph generator** baked into Python, not in markdown — meaning the skill ships actual executable orchestration logic, not just instructions.

### 5. The setup.sh — bootstrap as a single idempotent script

`assets/setup.sh.tmpl` (185 lines) implements a 7-step setup:
1. Verify required API keys via `check_key` (env file + macOS Keychain) — abort if missing (`setup.sh.tmpl:20-41`).
2. Create the workspace directory tree (`mkdir -p`) — `taste/`, `audio/{voiceover,sfx}/`, `assets/`, `scenes/`, `output/`, etc.
3. Create profiles: `hermes profile create <name> --clone 2>/dev/null || true` (idempotent).
4. Configure profiles via the **PyYAML-based `configure_profile` function** (`:63-123`): mutates `~/.hermes/profiles/<name>/config.yaml` to set only `toolsets` and `skills.always_load`. Explicitly **does not touch `approvals.mode`** (security setting) or `terminal.cwd` (kanban overrides per-task). Re-reads to validate.
5. Write per-profile `SOUL.md` to `~/.hermes/profiles/<name>/SOUL.md` via heredoc.
6. Copy brief, TEAM.md, taste templates.
7. **Fire the initial kanban**: `hermes kanban create --assignee director --workspace dir:"$WORKSPACE" --tenant "$TENANT" --priority 2 --max-runtime 4h` with a body that explicitly requires every child task to use `workspace_kind="dir"`, `workspace_path="$WORKSPACE"`, `tenant="$TENANT"` (`:156-175`).

Critical invariant from `kanban-setup.md:248-277`: **`workspace_kind="dir"` + `workspace_path="<absolute>"` on every `kanban_create` call**. Without this, profiles can't share artifacts.

### 6. Templates — what is system-supplied vs user-supplied

The three templates encode a strict separation:

- **`brief.md.tmpl`** (user-supplied at intake, frozen at setup): all fields are project-specific (`{{TITLE}}`, `{{ONE_LINE_PITCH}}`, `{{EMOTIONAL_NORTH_STAR}}`, scene rows, audio approach, deliverables). The template ships only the *shape* and the closing rule that "the brief is the contract."

- **`soul.md.tmpl`** (mostly system-supplied, role-keyed): role responsibilities, inputs read, outputs produced, toolsets, skills, role rules — all parameterized; the actual *common rules* and *common commands* are filled in by `render_soul_md` from system-supplied strings. Director-only rules are **conditionally appended** based on `role == "director"` (`bootstrap_pipeline.py:419-428`). User-customizable surface is small.

- **`setup.sh.tmpl`** (entirely system-supplied scaffolding): user-supplied content is injected only as `{{KEY_CHECKS}}`, `{{SCENE_DIRS}}`, `{{PROFILE_CREATE_COMMANDS}}`, `{{PROFILE_CONFIG_COMMANDS}}`, `{{SOUL_WRITES}}`, `{{BRIEF_CONTENTS}}`, `{{TEAM_CONTENTS}}`, `{{TASTE_WRITES}}`, `{{ASSET_COPIES}}`. Everything around it (`check_key`, `configure_profile`, the directory layout, the initial-kanban fire) is fixed scaffolding.

This template-layering means **the user describes intent (brief + team plan), the skill provides the entire structural runtime (workspace, profile config, kanban firing).**

### 7. Monitor loop — best-effort observability without auto-recovery

`scripts/monitor.py` (196 lines) is deliberately conservative: it polls `hermes kanban list --tenant <slug> --json` every 30s, falls back to text parsing if `--json` isn't supported (`:33-66`), and runs three issue detectors (`detect_issues`, `:82-131`):

- **STUCK** — RUNNING task with no heartbeat in 2+ minutes.
- **OVERTIME** — RUNNING longer than its `max_runtime_s` cap.
- **FLAPPING** — task with retries >= 2.

Issues print to **stderr**; the loop never auto-restarts. The docstring is explicit (`monitor.py:14-15`): *"This is best-effort observability. It does not auto-restart tasks; intervention decisions should remain human/AI-overseen."*

`references/monitoring.md` ships a richer **diagnostic table** (`monitoring.md:46-63`) mapping symptoms to actions, plus **intervention recipes**:
- Reject bad output → `kanban comment` + create re-render task with parent link.
- Add a new dependency mid-flight → `kanban create` then `kanban link <parent_id> <child_id>` (parent first; argument order is a documented pitfall).
- Stop a stuck worker → `kanban block` + `kanban archive` + diagnose with `show/tail/log`.
- Pivot the brief → cancel director + RUNNING children, edit brief, re-fire.

The dispatcher's automatic SIGTERM-on-`max-runtime` is the recovery primitive; everything else is operator/AI judgment (`monitoring.md:54`).

### 8. Worker contract — typed handoffs

`devops-kanban-worker.md` defines the **worker contract**:
- Workspace kinds: `scratch` (GC'd), `dir:<path>` (shared persistent — *"other runs will read what you write"*), `worktree` (git, commit work here) — `:34-42`.
- Tenant prefixing: when `$HERMES_TENANT` is set, prefix memory entries with the tenant to avoid leakage (`:44-50`).
- **Typed `kanban_complete(summary, metadata)` shapes** (`:54-94`) — coding/research/review variants showing how to encode `changed_files`, `tests_passed`, `findings[]`, `recommendation`, `benchmarks{}`. Explicit goal: "Shape `metadata` so downstream parsers (reviewers, aggregators, schedulers) can use it without re-reading your prose."
- Heartbeats *worth* sending: name progress (`"frame 240/720"`, `"epoch 12/50"`), every few minutes max, skip for tasks under ~2 min (`:113-117`).
- Retry diagnostics: open `kanban_show`, read `runs[]`, read `outcome` (`timed_out` / `crashed` / `spawn_failed` / `reclaimed` / `blocked`), don't repeat the failed path (`:119-128`).
- "Do NOT" list: don't substitute `delegate_task` for `kanban_create`, don't modify files outside the workspace, don't reassign follow-ups to yourself, don't complete a task you didn't finish (block instead) (`:129-133`).

### 9. Orchestrator playbook — five-step decomposition

`devops-kanban-orchestrator.md:34-44` defines the **"when to use the board" gate**: create kanban tasks if any of (multiple specialists, survives crash, human-in-the-loop, parallelism, review/iteration, audit trail) — otherwise use `delegate_task` or just answer.

Then the five-step playbook (`:71-150`):
1. Understand the goal (clarify if ambiguous — "cheap to ask; expensive to spawn the wrong fleet").
2. **Sketch the task graph in prose to the user before creating anything** (`:75-87`).
3. Create tasks and link, using `parents=[t1, t2]` for fan-in. The dispatcher auto-promotes children to `ready` when all parents reach `done`.
4. Complete your own task with `task_graph` metadata so downstream observers can reconstruct what was created.
5. Report back to user in plain prose.

Common patterns (`:152-160`): fan-out + fan-in, pipeline with gates, same-profile queue (50 tasks to one translator profile), human-in-the-loop. Pitfalls (`:162-170`): reassignment vs new task, `kanban_link(parent, child)` argument order, "Don't pre-create the whole graph if shape depends on intermediate findings — orchestrators can spawn orchestrators", tenant inheritance via `os.environ.get("HERMES_TENANT")`.

## Relevant Code Paths

- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/SKILL.md:1-207` — workflow + critical rules + file map.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/intake.md:1-167` — adaptive discovery banks, 9 style archetypes.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/role-archetypes.md:1-299` — role library + composition heuristics + anti-patterns.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/tool-matrix.md:1-318` — role × toolset/skill/env table.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/kanban-setup.md:1-277` — workspace layout, profile-config patching pattern, TEAM.md convention, API-key check.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/monitoring.md:1-181` — diagnostic table, intervention recipes, gotchas.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/examples.md:1-228` — six worked pipelines.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/scripts/bootstrap_pipeline.py:1-501` — plan-to-setup compiler with task-graph derivation.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/scripts/monitor.py:1-195` — polling loop + STUCK/OVERTIME/FLAPPING detectors.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/assets/soul.md.tmpl:1-39` — per-profile personality skeleton.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/assets/brief.md.tmpl:1-79` — frozen brief contract.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/assets/setup.sh.tmpl:1-185` — 7-step bootstrap script.
- `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-worker.md:1-153` — worker contract: workspace, tenancy, handoff shapes, retries.
- `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-orchestrator.md:1-171` — orchestrator playbook.

AGH cross-reference (read-only):
- `docs/rfcs/001_agent-md-with-skills-memory.md` — the AGENT.md+skills+memory triad.
- `docs/rfcs/002_skills-system-final.md` — daemon-managed skills, `metadata.agh.*` extension namespace, MCP lazy-load, lifecycle hooks (`on_session_created`, `on_session_stopped`).
- `internal/skills/` — skill registry, loader, MCP sidecar, hook decl, provenance, watcher (no kanban-orchestration primitive present today).
- `docs/_memory/glossary.md:13-27` — canonical `capability` term, capability-vs-skill distinction.

## Transferable Patterns

These are the patterns AGH could adapt for a multi-role orchestration primitive on top of `task_runs` + the autonomy kernel. Naming honors AGH's vocabulary: **capability** for cross-AGH artifacts, **skill** for in-instance behavior, **role/profile** for an agent identity.

1. **Two-tier orchestration content (planner playbook + worker playbook).** AGH's bundled skills could ship `agh-orchestrator-skill` (planner playbook with anti-temptation rules, decomposition gate, task-graph sketching) and `agh-worker-skill` (workspace handling, typed handoff shapes, retry diagnostics). These are *content-only* skills — markdown loaded by `always_load` — that turn `task_runs` into a multi-role pipeline without changing the runtime. Maps cleanly to AGH skills v2 (`metadata.agh.lifecycle` + `always_load`).

2. **Role archetype library as a composable resource.** Adapting `references/role-archetypes.md`: a domain-specific archetype catalog (researcher, analyst, writer, reviewer, backend-eng, frontend-eng, ops, pm — already convention-named in `kanban-orchestrator`) that lives as a skill resource. Composition heuristics + anti-patterns are part of the playbook, not just decoration.

3. **Plan compiler that generates a setup script.** `bootstrap_pipeline.py` is the strongest reusable mechanism: a typed plan JSON → a single idempotent shell script that creates profiles, patches configs, writes per-role personalities, fires the initial task. AGH already has `agh profile`, `agh task create`, and config primitives — a `bootstrap-capability.py`-equivalent could wrap them. The PyYAML config-patcher pattern (touch only `toolsets` and `always_load`, leave `approvals` and `cwd` alone) is directly transferable to AGH profile config.

4. **Frozen brief + TEAM.md as the orchestration contract.** The "brief is the contract; if it changes, re-fire the kanban — don't edit live" rule (`brief.md.tmpl:78-79`) is a useful invariant for AGH's autonomy kernel. Combined with the `TEAM.md` task-graph file, this gives the planner a deterministic source of truth that's separate from the live task state.

5. **Typed `complete(summary, metadata)` handoff shapes.** AGH already has `task_runs` records; codifying *typed metadata schemas per role* (research findings, code changes, review verdicts) as part of bundled skills would make downstream consumers parse without re-reading prose. Mirrors the `kanban_complete(metadata={...})` patterns in `devops-kanban-worker.md:55-94`.

6. **Adaptive 3-tier intake.** The Tier 0 / Tier 1 / Tier 2 funnel (3 baseline questions → style-keyed bank → closing assumptions) with a "max 4 questions per turn" rule and a "Reasonable assumption defaults" table is a clean pattern for any AGH onboarding skill (e.g., a `cy-spec-preflight` analog or a "scaffold a new feature" capability).

7. **Persona-as-policy for execution restriction.** Hermes restricts the director's *behavior* via `SOUL.md` rules + the `kanban-orchestrator` skill, while leaving the *toolset* identical to a worker. AGH's equivalent — restricting execution through an AGENT.md persona + a loaded skill — fits RFC 001's AGENT.md model and avoids inventing new runtime gates. Note the limitation: enforcement is post-hoc via audit logs, not at the tool layer.

8. **Best-effort polling monitor with named issue classes.** `monitor.py` shows a 100-line pattern: poll `task list --json` periodically, classify into STUCK / OVERTIME / FLAPPING, print to stderr, never auto-recover. AGH has SSE/UDS surfaces and `task_runs` schema; this is straightforward to mirror.

9. **`kanban watch / list / show / tail / stats / dispatch / link / unlink / heartbeat / log` CLI surface** (enumerated in `monitoring.md:31-34`). AGH already has `agh task` commands; the gap to close is the orchestration verbs (`link`, `unlink`, `block`, `archive`, `tail`, `dispatch`, `stats`, `heartbeat`) that turn a flat task list into a wired graph.

10. **Profile cloning + per-task config override.** `hermes profile create <name> --clone` plus per-task `--workspace dir:<path>` (overrides profile cwd) is a clean separation: profile = identity + skills + persona; task = workspace + scope. AGH currently mixes some of these in a single `agh-config.toml`.

11. **API-key preflight as part of bootstrap.** The `check_key` helper (`setup.sh.tmpl:22-41`) checks env file + macOS Keychain before firing tasks; aborts cleanly with a clear message if a key is missing. Avoids burning task slots on credential errors. Directly applicable to AGH's provider-secret model.

12. **Examples doc as mode-recognition table.** `examples.md:208-225` provides a "When the user describes X, look for these signals" table that maps brief shape → example pipeline. Useful pattern for any AGH skill that has multiple modes.

## Risks / Mismatches

1. **Vocabulary clash.** Hermes uses *kanban / board / lane / task graph* freely. AGH's CLAUDE.md is strict: artifacts are `capability`, never `recipe`/`workflow`/`procedure`/`playbook`. AGH should call the equivalent primitive a **task graph** or **task pipeline** or expose orchestration as a *capability* — not adopt "kanban" wholesale. The orchestration *playbook* itself can be a *skill* (in-instance behavior), but anything cross-instance (role definitions, archetype catalog) would need to be a *capability* per the glossary.

2. **AGH skills are static text + lifecycle hooks; Hermes ships executable Python in the skill.** RFC 002 §2.1 emphasizes "Extend, don't fork" with `metadata.agh.*` and progressive disclosure (metadata → instructions → resources). Hermes' `bootstrap_pipeline.py` is a 500-line Python compiler that *executes*. AGH would need an explicit decision: (a) ship orchestration helpers as bundled CLI tools (`agh capability scaffold`) rather than skill-internal scripts, or (b) extend the skills resource model to support executable assets with an explicit security review (RFC 002 §2.2 already mandates `VerifyContent` scanning for non-bundled skills — this would tighten the gate further).

3. **Persona-only execution restriction is policy, not enforcement.** Hermes director profiles can technically still execute (terminal toolset is enabled); the restriction relies on the prompt + audit logs. AGH's autonomy-kernel security posture leans toward stronger guarantees. Adopting this pattern means accepting the policy-only ceiling, or building real per-role tool gating (which the kanban-video-orchestrator does *not* do).

4. **Shared `dir:` workspace as the integration substrate** is convenient but creates coupling: every role's outputs depend on filename conventions and "predictable paths" (`soul.md.tmpl` common rules: *"Write outputs to predictable paths. Other profiles depend on your filename conventions"*). AGH's `task_runs` already has typed handoffs; building on the typed handoff side rather than a shared mutable directory is a stronger primitive.

5. **No skill versioning at the orchestration layer.** The video-orchestrator's role archetype catalog and tool matrix are markdown; if `ascii-video` ships a breaking change, the renderer profile silently breaks. AGH's RFC 002 includes registry/provenance signals that could be plumbed through, but the pattern as imported wouldn't carry that.

6. **Heavy reliance on `--clone` profile creation** assumes the cloned profile inherits sane defaults. A user with an unusual base profile (custom approval mode, unusual model, unusual toolsets the patcher doesn't touch) gets surprises. The patcher explicitly *avoids* touching `approvals` and `cwd` — but that means a base profile with `approvals.mode=auto` cascades through every spawned worker.

7. **Brief immutability + re-fire** is a strong invariant but it's incompatible with AGH's "agents manage features through CLI/REST" goal if the orchestration is exposed agent-managed: a peer agent that wants to refine the brief mid-flight would either need to abort+restart or break the invariant.

8. **The "director never executes" rule is enforced by content, not capability scoping.** Hermes acknowledges this (`role-archetypes.md:27-29`). AGH should decide whether it wants a stronger guarantee (per-role tool whitelisting at the runtime) given its emphasis on observability and security.

9. **The 9-style intake ontology is video-specific.** It does not generalize. Other domains (research projects, code refactors, infra migrations) need their own intake ontology — the *pattern* (Tier 0 baseline → style-keyed Tier 1 → Tier 2 closing + assumption defaults) generalizes, but every concrete intake bank is bespoke.

10. **Missing AGH primitives for some kanban verbs.** AGH's `task_runs` doesn't currently expose `block / unblock / link / unlink / archive / tail / heartbeat / stats` as first-class verbs. Adopting the pattern requires extending the task surface — not free.

## Open Questions

1. Should AGH expose a **dedicated orchestration primitive** (e.g., `agh orchestrate <plan.json>` or a typed task-graph capability) or rely on bundled `agh-orchestrator-skill` + `agh-worker-skill` + raw `agh task create` calls in skill bodies? The Hermes split (kanban runtime + content-only orchestrator/worker skills) suggests the second is sufficient, but AGH's "agents manipulate via CLI + REST" goal might favor first-class CLI verbs.

2. Where should role archetype catalogs live — bundled skills, capability catalogs (RFC 005), or a new resource type? The Hermes archetype list is markdown inside the skill; AGH's RFC 005 introduces capability catalogs which might be a better home for a domain-spanning role library.

3. How does AGH reconcile "the brief is the contract; re-fire to change" with agent-managed orchestration where a peer agent wants to revise scope mid-flight? Treating brief revisions as events on the orchestration graph (rather than file edits) is one path; another is enforcing brief immutability and modeling pivots as new top-level tasks.

4. Should orchestration scripts (the `bootstrap_pipeline.py` analog) ship as **bundled CLI subcommands** (`agh capability scaffold`), as **agent-callable tools** (so a planner agent itself can scaffold a sub-pipeline), or as **executable skill assets**? Each has different security and discoverability trade-offs; RFC 002 §2.2 (security scanning) leans against the third without further gates.

5. How tight is the coupling between role archetypes and per-domain skills (`renderer-ascii` ↔ `ascii-video`)? Hermes ships them in the same skill bundle; AGH's `metadata.agh.related_skills` field could express the dependency declaratively but discovery (which renderer for which scene type) lives in the orchestrator playbook.

6. Should the monitor loop be a **bundled CLI** (`agh task monitor --watch`) with built-in stuck/overtime/flapping detection, or a **skill** that an overseer agent loads? The first is more discoverable; the second composes with custom intervention policies.

7. What is the **shared-workspace policy** in AGH? Hermes' `dir:` workspace assumes filesystem coordination; AGH's autonomy kernel and `task_runs` give a typed alternative. Should multi-role pipelines be filesystem-coordinated, handoff-coordinated, or both?

8. How does AGH express the **"always-on baseline guidance"** that Hermes auto-injects via `KANBAN_GUIDANCE`? AGH's RFC 001 has AGENT.md + skill catalogs; a similar "every task gets the worker contract injected" mechanism could live as a daemon-side prompt-assembly contract or as a mandatory always-loaded `agh-task-contract` skill.

9. Should AGH adopt the **planner-as-task** pattern — the planner agent is itself a kanban task, completes itself with the task-graph metadata, and is replayable from history (`devops-kanban-orchestrator.md:122-138`)? This naturally extends `task_runs` for nested orchestration.

## Evidence

Inline-cited above by absolute path + line range. Primary sources:

- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/SKILL.md:1-207` — overall skill mental model, file map, critical rules.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/intake.md:1-167` — adaptive discovery, 9 style archetypes, assumption defaults.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/role-archetypes.md:1-299` — role library, composition heuristics, anti-patterns. Director's persona-only restriction at lines 27-29.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/tool-matrix.md:1-318` — role × toolset/skill/env mapping, API-key requirements.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/kanban-setup.md:1-277` — workspace layout, profile-config patching, TEAM.md convention, API-key checks, critical rules.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/references/monitoring.md:1-181` — diagnostic table at 46-63, intervention recipes at 66-130.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/scripts/bootstrap_pipeline.py:1-501` — plan-to-setup compiler. Validation at 78-129. Task-graph derivation at 225-309. SOUL render at 403-461.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/scripts/monitor.py:1-195` — polling + STUCK/OVERTIME/FLAPPING detection at 82-131.
- `.resources/hermes/optional-skills/creative/kanban-video-orchestrator/assets/soul.md.tmpl:1-39`, `assets/brief.md.tmpl:1-79`, `assets/setup.sh.tmpl:1-185`.
- `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-worker.md:1-153` — worker contract.
- `.resources/hermes/website/docs/user-guide/skills/bundled/devops/devops-kanban-orchestrator.md:1-171` — orchestrator playbook.

AGH cross-reference:
- `docs/_memory/glossary.md:13-27` confirms `capability` is canonical and skill ≠ capability.
- `docs/rfcs/002_skills-system-final.md` §2.1-2.4 establishes `metadata.agh.*` extension model, security scanning, MCP lazy-load, lifecycle hooks.
- `internal/skills/` listing confirms AGH has registry/loader/watcher/MCP-sidecar/provenance primitives but no kanban-orchestration primitive today — adoption is an additive design space, not a replacement.
