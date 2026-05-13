# Analysis: Existing Instruction Surfaces & Gaps

Map of every active AGH instruction surface as of 2026-04-26, for synthesis to dedup against. Read-only analysis.

## CLAUDE.md Catalog

`CLAUDE.md` (182 lines) and `AGENTS.md` (183 lines) are byte-identical except AGENTS.md ends with `<critical>NEVER COMMITS ai-docs/ TO THE REPO</critical>`. Treat them as one surface. `web/CLAUDE.md` and `web/AGENTS.md` are also byte-identical (131 lines). The rules they currently enforce, sectioned for dedup checks:

**Project framing**: AGH = Go single-binary daemon, ACP subprocess agents, JSON-RPC over stdio, SQLite event store, HTTP/SSE web, UDS CLI. Phases listed as 1) Agent core (current) → 2) Memory/Skills/State → 3) Network protocol.

**Greenfield Alpha — Zero Legacy Tolerance**: no production users; no migration/compat/defensive code; delete the old thing.

**Critical rules**: `make verify` MUST pass; `make lint` zero tolerance; check dep APIs first; `go get` not hand-edit `go.mod`; no web search for local code; no destructive git (`git restore|checkout|reset|clean|rm`) without permission.

**Design system**: `DESIGN.md` (793 lines, repo root) is authoritative for runtime UI + site + docs. Pull every token from it; flat depth (no shadows); warm-dark palette; fonts Inter / JetBrains Mono / Playfair Display (site-home only) / NuixyberNext (wordmark only); signal palette `#E8572A` action / `#30D158` success / `#FF453A` danger / `#FFD60A` warning / `#BF5AF2` info; `.compozy/tasks/redesign/` flows through the `designer` agent in execution mode only.

**Skill dispatch table** (root CLAUDE.md): Go/Runtime → `golang-pro` (cond `context7`); Config/Logging → `golang-pro`; Bug fix → `systematic-debugging`+`no-workarounds` (cond `testing-anti-patterns`); Writing Go tests → `testing-anti-patterns`+`golang-pro` (cond `vitest`); Task completion → `cy-final-verify`; Architecture audit → `architectural-analysis` (cond `adversarial-review`+`refactoring-analysis`); Concurrency → `deadlock-finder-and-fixer`+`golang-pro` (cond `systematic-debugging`); Performance → `extreme-software-optimization`+`golang-pro`; Security review → `security-review` (cond `ubs`); Creative → `brainstorming` (cond `cy-idea-factory`); PRD/TechSpec → `cy-create-prd`+`cy-create-techspec`+`cy-create-tasks`; Execute task → `cy-execute-task` (cond `cy-workflow-memory`); Review round → `cy-review-round`+`cy-fix-reviews` (cond `fix-coderabbit-review`); Git rebase → `git-rebase`; External docs → `context7`+`find-docs` (cond `exa-web-search-free`); UI/Design → `agh-design`+`design-taste-frontend`+`minimalist-ui` (cond `frontend-design`+`interface-design`). Multi-domain → multi-skill mandatory.

**Build commands** listed: `make verify` (BLOCKING), `fmt`, `lint`, `test`, `build`, `deps`. Commit style: `<type>: <description>`. Code search hierarchy: Grep/Glob → `context7` → web search. `.old_project/` is reference-only.

**Architecture principles**: incremental extension, pragmatic-flat with discipline, `daemon/` sole composition root, no package imports `daemon`/`api`/`cli`, interfaces defined where consumed, direct function calls (no event bus, no NATS, no reflection routing), notifier pattern, no back-pointers, functional options, maps under 10 items (no registry interfaces), file-level organization, CI-enforceable boundaries.

**Concurrency**: explicit goroutine ownership via ctx; `WaitGroup` tracking; `select` with `ctx.Done()`; prefer channels over mutex; RWMutex read-heavy, Mutex write-heavy; no `time.Sleep()` in orchestration.

**Runtime**: single-binary local-first; sidecars need techspec; deterministic + observable.

**Package layout table** lists `cmd/agh`, `internal/{config,acp,session,store,store/globaldb,store/sessiondb,observe,memory,memory/consolidation,skills,skills/bundled,workspace,transcript,frontmatter,fileutil,filesnap,procutil,api/contract,api/core,api/httpapi,api/udsapi,api/testutil,testutil,cli,daemon,logger,version}`, plus `web/` and `web/src/systems/`.

**Coding style**: wrapped errors with `%w`; `errors.Is/As` not string compare; no `_` ignored errors; no `panic`/`log.Fatal` in production; `log/slog` only; `context.Context` first arg; `var _ Interface = (*Type)(nil)`; no `any` when concrete known; no reflection without justification; never hardcode config.

**Testing**: table-driven `t.Run` subtests; `t.Parallel()`; `t.TempDir()`; `t.Helper()`; `-race` mandatory; mock via interfaces; 80% coverage minimum. Integration tests: `//go:build integration` tag, co-located, `make test` = unit / `make test-integration` = all, `TestMain`, real deps, ~30s max per package.

**`web/CLAUDE.md`**: React 19 SPA + Vite 8 + TanStack Router/Query + Tailwind v4 + shadcn (base-nova) + Zustand + Zod + oxfmt + oxlint. `make web-lint` + `make web-typecheck` mandatory; oxlint zero tolerance; shadcn kebab-case; `bun add` only; tokens from DESIGN.md / `packages/ui/src/tokens.css`. Web-specific skill dispatch covers React/Tailwind/Vercel React/TanStack/Zustand/Zod/Vitest/Storybook/AI-SDK/Composition/Impeccable. `app-renderer-systems` pattern: `adapters → lib → hooks → components` unidirectional, cross-system only through public barrel. TanStack Query: hierarchical keys, typed errors, `AbortSignal`, `onSettled`, optimistic with rollback. UI pure presentational; orchestration in pages/routes; state hierarchy local > Zustand > Query > URL; server state Query-only; route-level fetching; components never import stores/adapters; kebab-case files; named exports; functional only (no `React.FC`); `useEffect` escape hatch only; loading/error/empty states; composition over booleans; `@/*` alias.

## Existing Skill Catalog (project-relevant)

User-global `/Users/pedronauck/.claude/skills/` is a mix of real directories and symlinks into `/Users/pedronauck/.agents/_skills/` (e.g. `agent-browser`, `gws-*`, `neon-postgres`, `smux`, `humanizer`, `find-docs`). Some skills come from harness plugins (the `vercel:*`, `figma:*`, `impeccable:*`, `codex:*`, `firecrawl:*`, `ralph-loop:*` families). Project-relevant groupings:

- **Compozy workflow chain (cy-\*)**: `cy-create-prd` → `cy-create-techspec` → `cy-create-tasks` → `cy-execute-task` (with `cy-workflow-memory`) → `cy-review-round` → `cy-fix-reviews` (with `fix-coderabbit-review`) → `cy-final-verify`. Plus `cy-idea-factory` upstream. The `compozy` skill explains the system.
- **Go**: `golang-pro` (canonical), `bubbletea` (TUI), `nats` (installed but architecturally forbidden — see Stale section).
- **Quality / debugging**: `systematic-debugging`, `no-workarounds`, `testing-anti-patterns`, `deadlock-finder-and-fixer`, `extreme-software-optimization`, `architectural-analysis`, `refactoring-analysis`, `security-review`, `ubs`, `lesson-learned`.
- **QA**: `qa-report` (plans, cases, bug reports), `qa-execution` (full-project end-to-end execution), and now `real-scenario-qa` (project-local, see dedicated section).
- **External docs / research**: `context7`, `find-docs`, `exa-web-search-free`, `kb`, plugin `firecrawl:firecrawl-cli`.
- **Design / UI**: `agh-design` (project-specific), `design-taste-frontend`, `minimalist-ui`, `frontend-design`, `interface-design`, `redesign-existing-projects`, `impeccable:*` plugin family (polish/layout/typeset/critique/audit/delight/harden/clarify/animate/adapt/colorize/overdrive/quieter/bolder/shape/distill).
- **Web stack**: `react`, `tailwindcss`, `shadcn`, `vercel-composition-patterns`, `vercel-react-best-practices`, `next-best-practices`, `tanstack`, `tanstack-router-best-practices`, `tanstack-query-best-practices`, `tanstack-start-best-practices`, `zustand`, `zod`, `typescript-advanced`, `vitest`, `storybook-stories`, `app-renderer-systems`, plus assistant-ui ecosystem (`ai-sdk`, `streaming`, `runtime`, `tools`, `primitives`, `assistant-ui`).
- **Other**: `git-rebase`, `crafting-effective-readmes`, `documentation-writer`, `mermaid-diagrams`, `architecture-diagram`, `viz`, `tech-logos`, `favicon-gen`, `brainstorming`, `council`, `humanizer`, `copywriting`, `skill-best-practices`, `agent-md-refactor`, `tmux`, `obsidian-cli`, `qmd`, `seo-audit`, `remotion-best-practices`.

**Project-level agents** (`.claude/agents/`): six council archetypes (`architect-advisor`, `pragmatic-engineer`, `product-mind`, `security-advocate`, `devils-advocate`, `the-thinker`) consumed by the `council` skill, plus `designer` (execution-mode-only) referenced by the redesign-tasks rule. None duplicates a skill.

## Existing User Memory Catalog

`/Users/pedronauck/.claude/projects/-Users-pedronauck-Dev-compozy-agh/memory/MEMORY.md` indexes three project notes:

1. **`project_site_docs.md`** (2026-04-15; hero relocked 2026-05-01) — Approved Site & Docs techspec. Hero "Your agents can finally talk to each other." (network-first) → relocked to "An open workplace for AI agents." (open-workplace-first) on 2026-05-01. Fumadocs at `agh.network` on Vercel, lives in `site/` in monorepo (NOTE: ADR-011 + tasks 15-16 reference `packages/site/` — memory may be slightly drifting). Two doc products (Runtime Docs + Network Protocol Spec); CLI ref via cobra JSON export, API via OpenAPI + fumadocs-openapi; ~63 pages in 3 waves. "AGH Network Protocol is the key differentiator." Research at `.compozy/tasks/site/analysis/`, techspec at `.compozy/tasks/site/_techspec.md`.

2. **`feedback_ci_no_cron.md`** (2026-04-17) — CI philosophy. NO GitHub Actions cron/schedule workflows. Heavy/credentialed tests (`make test-e2e-nightly`, `make test-integration`) live in the `dry-run` job of the auto-created release PR. Rationale: release PR is the natural human-gated batching point. PR workflow `ci.yml` = verify + e2e-combined only; release workflow `release.yml` has 3 jobs (release-pr / dry-run / release).

3. **`reference_looper_ci_patterns.md`** (2026-04-17) — `~/dev/compozy/looper` is the canonical CI/release source for compozy-org Go repos. Verbatim copies: `.github/versions.yml`, `setup-go/setup-bun/setup-git-cliff/setup-release` composite actions, `ci.yml`, `release.yml`, `.goreleaser.yml`, `cliff.toml`. `pr-release` module = `github.com/compozy/releasepr@v0.0.17`. Cross-repo secrets `RELEASE_TOKEN`, `GORELEASER_KEY`; AGH-specific `DAYTONA_API_KEY`, `DAYTONA_SNAPSHOT`/`DAYTONA_IMAGE`.

User memory does NOT yet cover the autonomous mode techspec, ADRs 002-012, the autonomy-MVP QA pattern, the manual-first execution boundary, coordination channels, claim/lease, safe spawn, or the new `real-scenario-qa` skill. Memory is CI- and site-positioning-heavy and silent on the runtime substrate that has been the dominant work since 2026-04-25.

## ADR Implied Rules Not Yet in CLAUDE.md

`.compozy/tasks/autonomous/adrs/` (001-012) encodes load-bearing rules that govern day-to-day implementation but appear nowhere in CLAUDE.md/AGENTS.md.

**ADR-002 — Agent-Facing CLI Before MCP Tools**: identity-aware CLI/UDS first (MCP later); identity inference from `AGH_SESSION_ID`/`AGH_AGENT` MUST flow through `internal/agentidentity`, never parsed env directly; stable `-o json` and `-o jsonl` are compatibility contracts; no command aliases (no `done`, no `pass`); `agh ch` = operational comms, `agh task` owns claim/lease/terminal state.

**ADR-003 — Extend Task Runs for Atomic Claim/Lease**: `task_runs` is the single durable source of truth; NEVER add a parallel queue. `ClaimNextRun(criteria)` is canonical. Lease invariants: exactly one active claim token per non-terminal run; heartbeat/complete/fail/release all compare run owner + claim token; stale/late after recovery fails explicitly; sweep + heartbeat serialize via SQLite tx; boot recovery before scheduler accepts wake/claim traffic; lease extension bounded by config; one active lease per session in MVP. Use `BEGIN IMMEDIATE`; CAS predicates for sweep. Capability matching = durable exact-match rows in `task_run_required_capabilities`/`task_run_preferred_capabilities`, NOT JSON metadata.

**ADR-004 — Split Semantic vs Mechanical Scheduling**: coordinator-agent owns decomposition/validation/synthesis; daemon scheduler owns idle registry/wakeups/sweep/recovery/backpressure but does NOT directly claim runs in MVP. Scheduler is never a second run claimant. Coordinator never bypasses `ClaimNextRun`, leases, spawn limits, or channel permissions.

**ADR-005 — Configurable Spawn-On-Run-Enqueue Coordinator**: coordinator auto-spawns ONLY when workspace has no healthy active coordinator AND a coordinated run is enqueued by publish/start/approval AND run has stable `coordination_channel_id` AND auto-start enabled AND spawn caps allow. Task creation alone NEVER triggers coordinator startup. Trigger idempotent per workspace; coordinator cannot spawn another coordinator. Global-scope runs don't auto-spawn in MVP. Config precedence: workspace override → global `[autonomy.coordinator]` → bundled default. Conservative defaults: auto-start disabled, identity `coordinator`, TTL 2h, max children 5, max active per workspace = 1.

**ADR-006 — Safe Spawn**: default max depth = 1, max children per parent = 5; mandatory TTL on every spawned session; children auto-stop with parent (within caps); permission narrowing compares concrete atoms only (tools, skills, MCP server IDs, workspace path grants, network channels, env profile grants); subset-only; unknown child atoms count as widening and reject; reaper releases leases (`parent_stopped`, `ttl_expired`) before stopping; daemon NEVER silently narrows.

**ADR-007 — Minimal Network Evolution**: single-channel sessions in first MVP; coordination message kinds limited to `status`/`request`/`reply`/`blocker`/`handoff`/`result`/`review_request`; defer contract-net/multi-home/vote/react/escalate/offer/accept/decline/complex mention routing; wire-format bumps tied to implemented kinds, not speculative.

**ADR-008 — Memory Provenance Before Rich Scopes**: start with provenance (`agent_name`, `session_id`) + recall metadata (file/scope/agent/score/freshness); append-only session-scope summaries; don't store every network message as memory; defer peer/channel memory and turn extraction.

**ADR-009 — Autonomy Hooks First-Class**: typed events + payloads + patches + dispatch + introspection through `internal/hooks`; NEVER ad-hoc callbacks or parallel plugin system; no generic event bus. Surface: `coordinator.{pre_spawn,spawned,decision,stopped,failed}`, `task.run.{enqueued,pre_claim,post_claim,lease_extended,lease_expired,lease_recovered,released}`, `spawn.{pre_create,created,parent_stopped,ttl_expired,reaped}`. Scheduler wake/no-match/recovery stay internal metrics. No `workflow.*` hooks in MVP. Pre-commit hooks dispatch at the call site BEFORE the tx; post hooks dispatch after the audit event; do NOT tail the events table. Hooks can deny/narrow/annotate but cannot bypass `ClaimNextRun`, tokens, leases, TTL, lineage, spawn caps, permission narrowing. `task.run.pre_claim` may only ADD required capabilities or raise `PriorityMin`. New resource kinds are post-MVP.

**ADR-010 — Manual Operator Control First-Class** (most-repeated rule across the techspec): autonomy is additive, never replacement; ONE shared task/run model and ONE shared session manager for user/automation/coordinator/agent-spawned cases; operator commands are identity-explicit, agent commands identity-implicit; manual assignment and autonomous claim converge on the same token/lease/heartbeat/complete/fail/release rules; task creation alone never enqueues claimable work or starts coordinator; publish/start/approval is the run-enqueue boundary that coordinator-bootstraps; no separate manual/autonomous/coordinator queues; operator UI must visually distinguish creation vs publish/approval vs run enqueue vs coordinator spawn; E2E MUST cover BOTH manual-first bookends (user create → publish → coordinated execution; AND user-start session → direct prompt without coordinator).

**ADR-011 — Generated Contracts and Docs Co-Ship**: any change to `internal/api/contract` co-ships in the same PR with regen of `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`, updates to `web/src/systems/*/types.ts` and Storybook/MSW fixtures, and passes `make web-typecheck` AND `make web-test`. HTTP read endpoints return `claim_token_hash`, never raw `claim_token`. Step 10 demo MUST co-ship: Tasks UI label/copy/disabled-state pass, one web e2e for ADR-010 bookends, minimum docs at `packages/site/content/runtime/core/autonomy/`, CLI reference for `agh me`/`ch`/`spawn`/new task verbs, hook/config/session docs updates. Explicit out-of-scope MVP: no `web/src/systems/coordinator`, no autonomy sidebar, no lease dashboard, no spawn lineage tree, no idle-agent registry view, no eval/replay UI, no coordinator config GUI, no marketing rewrite. Site commands: `cd packages/site && bun run source:generate`, then `bun run typecheck` and `bun run test`. `cy-create-tasks` attaches contract/docs obligations to the implementation tasks, not a late cleanup task.

**ADR-012 — Task-Run Coordination Channels**: every workspace-scoped coordinated run has ONE durable `coordination_channel_id` on `task_runs`. Bind always, speak when useful — heartbeats/lease transitions never mirrored as chat. Channel messages carry typed correlation: `task_id`, `run_id`, `workflow_id`, `coordination_channel_id`, `message_kind`, `correlation_id`. Channels are NEVER an ownership/status authority. Raw `claim_token` MUST NEVER appear in channel messages/logs/SSE/web/memory. Channel `status`/`result` messages cannot mutate ownership/terminal state. Network layer validates the message_kind enum and rejects raw claim_token in metadata.

## TechSpec Implied Rules

`_techspec.md` (706 lines) layers operational rules on top of the ADRs that aren't yet in CLAUDE.md.

**Architecture boundaries**

- `internal/session`, `internal/task`, `internal/network`, `internal/memory`, `internal/hooks`, `internal/resources` MUST NOT import `daemon`. Already implied by CLAUDE.md but the techspec spells out the new packages it covers.
- Scheduler logic lives behind narrow interfaces consumed from `internal/scheduler` (a NEW package not yet in CLAUDE.md's package layout table).
- The coordinator-agent is a managed session, NOT a privileged in-process scheduler.

**Runtime invariants**

- Boot recovery runs BEFORE the scheduler accepts wake/claim traffic.
- A session may hold at most ONE active task-run lease in MVP.
- The reaper releases leases before stopping a child session.

**Observability conventions** (NEW metrics + log fields not in CLAUDE.md)

- Metrics: `scheduler.wake.count`, `scheduler.wake.no_match`, `scheduler.lease_sweep.count`/`error`, `task.run.claim.success`/`error`, `task.run.lease.extended`/`expired`/`recovered`, `coordinator.spawned`/`failed`, `spawn.created`/`rejected`/`reaped`, `session.loop.detected`, `session.budget.exceeded`, `manual.task.created`, `manual.session.created`.
- Structured log fields: `workspace_id`, `session_id`, `parent_session_id`, `root_session_id`, `agent_name`, `task_id`, `run_id`, `claim_token_hash`, `lease_until`, `workflow_id`, `coordinator_session_id`, `scheduler_reason`, `hook_event`, `hook_name`, `spawn_depth`, `actor_kind`, `actor_id`, `release_reason`.

**Contract DTOs**

- `internal/api/contract` is the source of truth for transport-agnostic DTOs; UDS-first agent endpoints with optional HTTP parity.
- Operator endpoints must NOT infer agent identity from environment variables; that's an agent-only behavior.

**`/agent/context` payload ordering** — this is a contract:

- `self`, `workspace`, `session`, `task`, `coordination_channel`, `inbox_summary`, `peer_roster`, `capabilities`, `limits`, `provenance`. Each list section bounded with truncation metadata.

**Build sequencing rule** (techspec step 1-15)

- "Steps 1-10 are the local autonomy MVP." Tasks 11-15 are follow-on TechSpecs.
- Contract DTOs and feature flags BEFORE behavior. Hook taxonomy BEFORE behavior depends on it.
- Steps that touch `web/` MUST also pass `make web-typecheck` and `make web-test`. Steps that touch `packages/site` MUST pass site source generation, typecheck, and test.

## Recent Commit Patterns

133 total commits; the last ~25 form the autonomy MVP block on `autonomous` branch and land in dependency order matching `_tasks.md`. Sequence (compressed): config foundation → contract DTOs → hook taxonomy → situation providers → caller identity → self/channel verbs → claim/lease schema → claim/lease service → task lease verbs → execution boundary → scheduler → session lineage → safe spawn → coordinator bootstrap → tasks UI labeling → autonomy docs → QA artifacts → QA execution → review round 001 → review batch 001. Cadence: roughly one commit per PRD task; commits are bottom-up (schema/contract/hooks before behavior).

Recurring patterns across the last ~50 commits:

- Heavy QA + review remediation tail. "fix: qa testing and fixes (#73)" + 5 review-remediation commits in the last 10. Pedro routinely runs 2-4 review rounds per feature (`fix: resolve session-driver-override review round 4` is the high-water mark).
- Aggressive refactoring of names and module boundaries: `refactor: hermes adjustments`, `refactor: project structure`, `refactor: storybook for web and ui`, `refactor: kb/memory improvements`, `refactor: rename spaces to channels` (recent enough that legacy memory may still say "spaces").
- Large milestone redesign passes: `feat: redesign network workspace (#59)`, `feat: redesign ui (#48)`, `feat: production grade adjustments (#66)`.
- `docs: archive prd(s)` runs after PRDs land.

Conventional commit usage: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `build:` are the only prefixes. NO `chore:`, NO `style:`, NO `ci:` — tooling and CI changes are `build:`. PR-merged commits include `(#NN)` suffix; direct branch commits don't.

## Stale CLAUDE.md Items

A handful of CLAUDE.md statements no longer match the codebase or how Pedro currently works.

**1. Phase ordering**

- CLAUDE.md says phases are 1) Agent core (current) → 2) Memory/Skills/State → 3) Agent network protocol.
- Reality (per recent commits and the user-memory `project_site_docs.md`): network protocol exists today (`agh-network/v2`, seven-kind wire), channels are first-class, network workspace was redesigned in `feat: redesign network workspace (#59)`. The current major effort is the autonomy kernel (situation surface, claim/lease, coordinator), which doesn't cleanly map to any of the three named phases.
- Phase labels are stale; they predate the autonomous mode.

**2. Package layout table**

- The package layout table in CLAUDE.md does NOT include:
  - `internal/scheduler` (techspec mandates this; commit `feat: add mechanical scheduler` landed it).
  - `internal/agentidentity` (commit `feat: add agent caller identity layer` landed it; ADR-002 mandates flowing through it).
  - `internal/situation` (`feat: add situation surface providers`; techspec uses `internal/situation.Service`).
  - `internal/hooks` (autonomy hooks live here per ADR-009).
  - `internal/task` (only `internal/task` is implied indirectly).
  - `internal/network` (channels/peers, post `feat: redesign network workspace`).
  - `internal/resources` (`internal/resources/projector.go`, `codec.go`, `validate.go` are referenced by ADRs).
  - `packages/site` (Fumadocs site at `packages/site/content/runtime/...` per ADR-011 and recent commits).
- The table is materially out of date.

**3. NATS skill listed**

- The skill catalog includes `nats` (NATS messaging system).
- ADR-004 explicitly excludes NATS from AGH: "Direct function calls through interfaces — no event bus, no NATS, no reflection-based routing." The architecture principles in CLAUDE.md itself say "no NATS."
- A `nats` skill being available doesn't mean it should be activated for AGH; the skill catalog is what's installed, but the dispatch table correctly omits it. Still worth noting that if any future task accidentally pulls `nats`, it would conflict with the documented architecture.

**4. The `cy-final-verify` row says "Task completion"**

- That line is correct, but Pedro's actual completion flow runs `cy-final-verify` PLUS QA artifacts (`task_17` plan, `task_18` execution, plus the new `real-scenario-qa`). The catalog under-represents how big the verification surface has become.

**5. "Web search tools" guidance**

- The hierarchy says web search is for external code/news/research. In practice the project has `find-docs`, `context7`, `exa-web-search-free`, and `firecrawl:firecrawl-cli` (a plugin skill). The single "Web search tools" bucket understates the actual decision space.

**6. Git "destructive command" list**

- CLAUDE.md prohibits `git restore`, `git checkout`, `git reset`, `git clean`, `git rm` without permission. The rule still holds, but the CLAUDE.md framing ("If the worktree contains unexpected edits, read and work around them") doesn't cover the active worktree state right now: at session start there are 26 modified files plus 4 untracked dirs/files (autonomous task and ADR edits). This isn't stale per se, but the synthesis agent should know the rule is being actively exercised.

**7. Old Project Reference**

- Says `.old_project/` is reference-only. Verified the directory still exists in the file tree. Not stale.

## Skill Collision Map

Where new project-specific skills could overlap with existing ones, and where existing ones already overlap with each other.

**QA cluster — three skills now overlap**

- `qa-execution` (harness/plugin): "Executes full-project QA like a real user … run build, lint, test, and startup commands, exercising core workflows end-to-end through CLI, HTTP, … real persisted data."
- `qa-report`: "Generate comprehensive test plans, manual test cases, regression test suites, and bug reports."
- NEW `real-scenario-qa` (project-local): "Runs production-like release and feature QA by building a realistic startup workspace, creating agents/channels/tasks/automations/hooks/extensions/skills, … documenting bugs, fixing root causes, rerunning gates."
- The SKILL.md for `real-scenario-qa` explicitly resolves the collision by directing the agent to "Use `qa-report` when planning test cases or documenting issues" and "Use `qa-execution` when running gates, starting services, exercising CLI/API/Web flows." It positions itself as the _outer scenario orchestrator_ that delegates to `qa-execution` and `qa-report` as inner tools. That's clean — but CLAUDE.md doesn't yet mention this layering, so a reader looking only at the skill dispatch table would not know `real-scenario-qa` exists or what slots above the others.

**Design cluster — already well-mapped**

- `agh-design` (project-specific) + `design-taste-frontend` + `minimalist-ui` (mandatory) and `frontend-design` + `interface-design` (conditional) collide-by-design. CLAUDE.md handles ordering with the table. Note the `impeccable:*` family (polish/layout/typeset/critique/audit/etc.) is NOT in the dispatch table at all yet, despite being installed; web/CLAUDE.md mentions `impeccable:polish` + `impeccable:layout` + `impeccable:typeset` for "Design polish passes."

**Architecture-audit cluster**

- `architectural-analysis`, `refactoring-analysis`, `adversarial-review`, `ubs`, `security-review` — multiple deep-dive scanners that all examine code for issues. The dispatch table maps `architecture-audit` to `architectural-analysis` (required) plus `adversarial-review` and `refactoring-analysis` (conditional), and security to `security-review` + `ubs`. Nothing new collides here, but a synthesis agent shouldn't introduce a "code-review" skill that overlaps further.

**Council/advisory cluster**

- `council` skill orchestrates the six `.claude/agents/*-advisor.md` archetypes plus the `the-thinker` agent. No collision, but a synthesis agent should not propose a new "advisory" skill.

**Workflow memory**

- `cy-workflow-memory` already manages `.compozy/tasks/<name>/memory/` dirs — exactly where `.compozy/tasks/autonomous/memory/MEMORY.md` and per-task `task_NN.md` files live. Don't propose anything that writes to that directory under a different name.

**Documentation generation**

- `documentation-writer` (Diátaxis-style technical writing) and `crafting-effective-readmes` cover docs. A new "AGH-docs" skill would collide with both.

**Compozy chain (cy-\* skills)**

- `cy-create-prd` → `cy-create-techspec` → `cy-create-tasks` → `cy-execute-task` is the canonical flow. `cy-review-round` + `cy-fix-reviews` is the review loop. `cy-final-verify` is the gate. `cy-idea-factory` is the upstream brainstorming step. Don't propose anything that reorders or replaces this chain.

## "real-scenario-qa" Skill — What Gap It Fills

`/Users/pedronauck/Dev/compozy/agh/.agents/skills/real-scenario-qa/` is project-local (lives in the repo, not the user-global skills tree) and explicit-trigger-only. SKILL.md, two references, two assets, one bootstrap script.

**What it does**: takes a `scope-or-context` argument (`release-candidate`, `autonomy-feature`, `network-tasks`, etc.); runs `scripts/init-scenario-workspace.sh` to scaffold a startup workspace + `qa/` artifact root; activates `qa-report` + `qa-execution` + `systematic-debugging` + `no-workarounds` as collaborators; discovers the verification gate from the repo and runs it as baseline; builds a realistic startup scenario with multiple agents (CEO/CTO/backend/frontend/marketing/copy/finance/review/QA/operator), multiple channels (leadership/dev/marketing/finance/ops/review/launch), and realistic skills/hooks/extensions/automations/cron/webhook/knowledge/memory/tasks; drives ALL operations through public CLI/HTTP/Web/UDS surfaces (no internal shortcuts); reproduces every issue with the narrowest real command before editing code; writes `BUG-NNN` issues; fixes at root cause; adds regression coverage; re-runs scenario; writes a final report using a template.

`references/scenario-matrix.md` defines 9 tracks (release candidate, feature-focused, autonomy/orchestration, network collaboration, automations, knowledge/memory, hooks/extensions, web integration, performance/stability) with minimum composition (6+ agents, 4+ channels, 1+ automation, 1+ trigger, 1+ task tree, 1+ knowledge entry, 1+ web pass, 1+ failure probe). `references/evidence-checklist.md` enforces baseline-before-mutation, real-data-not-mocks, public-surface coverage, reproducible failure, release-readiness invariants.

**What gap it fills**: Pedro just ran QA in exactly this pattern for autonomy MVP (`task_18`). The verification report at `.compozy/tasks/autonomous/qa/verification-report.md` proves the pattern: baseline `make verify`, real daemon-served Playwright e2e, three root-caused bugs (BUG-001 workspace onboarding race, BUG-002 ACP mock context-matching after Task 04 augmentation, BUG-003 Tasks E2E expecting empty Agents state when manual-first publish renders an active run), full re-verify, site source-generate/typecheck/test/build. The skill encodes that pattern as a reusable surface for future major-release validation (network protocol, memory layer, skills layer) and delegates to `qa-execution`/`qa-report` for inner mechanics rather than duplicating them. Project-local because the bootstrapping (`init-scenario-workspace.sh`, the realistic-agent matrix, AGH-specific tracks) is too AGH-shaped for a generic skill.

**What its existence implies about project QA needs**: real integration/e2e is the only accepted final proof — mocks/stubs/fake agent replies/unit-only tests are explicitly NOT final proof; QA artifacts live under predictable per-task paths (`qa/issues/`, `qa/test-cases/`, `qa/test-plans/`, `qa/logs/`, `qa/screenshots/`, `qa/verification-report.md`); real-startup scaffolding (multiple agents + channels + tasks + automations) is mandatory for release validation; browser/credential blockers must be named explicitly; persistence inspection via supported APIs is allowed for debugging; "operator-facing history is understandable and not dominated by protocol noise" is a standing concern from coordination-channel work. The skill is NOT yet in the CLAUDE.md dispatch table — release/autonomy validation tasks would not auto-activate it from the table alone.

## Notes for Synthesis

Cross-references the synthesis must respect when proposing new rules or skills.

**1. Don't duplicate the manual-first contract**
ADR-010 + ADR-005 + ADR-012 already establish the manual-first contract in great detail. Any new rule about task creation, publish, start, approval, coordinator triggering, or coordination channels MUST reference these ADRs as the source. The `_techspec.md` data-flow + manual-control-contract sections are also load-bearing and probably belong as a CLAUDE.md summary, not as fresh rules.

**2. The skill dispatch table is the canonical activation map**
Don't propose new rule blobs that re-state the table. Either (a) add a row, (b) add a conditional column, or (c) add a "domain" the table doesn't yet cover (e.g. "Real-scenario QA" mapping to `real-scenario-qa` + delegations to `qa-execution`/`qa-report`/`systematic-debugging`/`no-workarounds`).

**3. The package layout table is the canonical package map**
If there's a gap (`internal/scheduler`, `internal/agentidentity`, `internal/situation`, `internal/hooks`, `internal/task`, `internal/network`, `internal/resources`, `packages/site`), update the table; don't write a new "packages" section.

**4. Do not propose any cron/schedule CI**
`feedback_ci_no_cron.md` is an explicit user-stated rejection. Heavy tests live in the release PR `dry-run` job, not in cron.

**5. Don't propose alternate workflow-memory storage**
`cy-workflow-memory` already owns `.compozy/tasks/<name>/memory/`. The autonomous task already has `MEMORY.md` and `task_NN.md` files there. Synthesis suggestions about "where to keep workflow memory" are duplicates.

**6. Don't propose alternate review-loop tooling**
`cy-review-round` + `cy-fix-reviews` + `fix-coderabbit-review` is the canonical review remediation chain. Recent commits prove Pedro uses it (`fix: resolve autonomous review round 001 issues`, `fix: resolve autonomous review batch 001`). The reviews-001 dir with `_meta.md` + 34 issue files is a fresh artifact of this chain.

**7. Don't propose new agent files in `.claude/agents/`**
The six council archetypes plus `designer` are intentional. Adding more would dilute the council pattern.

**8. Use `claim_token_hash` over the wire, raw `claim_token` only on issuing transport**
This is a security invariant from ADR-003 + ADR-011 + ADR-012. Any rule about API contracts, observability, logs, web UI, SSE, or memory MUST honor it.

**9. The autonomy MVP is `done`. Tasks 01-18 are completed.**
Any new rule should not assume the kernel is in flight. The kernel is now substrate. Future rules apply to: post-MVP work (steps 11-15 in the techspec), QA/release patterns built on top, future TechSpecs (network expansion, broader memory, eval/replay, web visibility), and the site/docs ramp.

**10. AGENTS.md vs CLAUDE.md**
The `<critical>NEVER COMMITS ai-docs/ TO THE REPO</critical>` line in AGENTS.md is the only divergence. If synthesis adds new rules, they should land in CLAUDE.md and AGENTS.md should mirror — or the divergence should be resolved by promoting the ai-docs rule into CLAUDE.md proper.

**11. The user-memory file is silent on autonomy**
Three memory files cover Site/Docs and CI; none covers the autonomy kernel, ADRs 001-012, the manual-first contract, the QA pattern from `real-scenario-qa`, or the autonomy hook taxonomy. If synthesis decides any of these belong as user-memory rather than CLAUDE.md, that's a clean addition (no overlap with existing memory).

**12. `make codegen` is load-bearing but not in CLAUDE.md**
The build commands list `make verify`, `fmt`, `lint`, `test`, `build`, `deps` — but `make codegen` (which regenerates `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`) is mandatory per ADR-011. Same for `make codegen-check` (used in QA verification). And `make test-e2e-web`, `make test-e2e-nightly`, `make test-integration`. The build commands section is a candidate for refresh.

**13. Site commands aren't in CLAUDE.md**
`cd packages/site && bun run source:generate`, `bun run typecheck`, `bun run test`, `bun run build` are referenced in ADR-011 and run during autonomy QA, but CLAUDE.md and web/CLAUDE.md don't list them. The Site subtree appears to be a third surface on par with Go backend and React web that lacks a CLAUDE.md or AGENTS.md.

**14. Test pyramid clarification**
CLAUDE.md says 80% coverage minimum and `make test` = unit only. Per `_techspec.md`, autonomy work requires a layered test pyramid: unit → integration (`-tags integration`) → daemon/web e2e (`make test-e2e-web`, `make test-e2e-nightly`). The 80% rule applies per package, but the e2e layer is what catches the manual-first/coordination-channel regressions. CLAUDE.md doesn't mention `make test-e2e-*`.

**15. The "spaces → channels" rename is recent**
`refactor: rename spaces to channels (#17)` happened relatively recently (~mid-history). If any user memory, skill, or doc still says "spaces," that's stale. The `agh-network/v2` wire kinds, `coordination_channel_id`, and `agh ch` verbs are the post-rename names.

**16. Two "hermes" references in commits/dirs**
`refactor: hermes adjustments (#69)` and the `_tasks.md` mentions "the same QA planning/execution handoff pattern used by `.compozy/tasks/hermes`" — there is or was a hermes feature whose QA handoff pattern is the prototype `task_17`/`task_18` + `real-scenario-qa` extends. Synthesis should look at hermes if it needs prior art for the QA pattern (out of scope for this analysis, just a pointer).

**17. `_techspec.md` is unusually authoritative**
Most TechSpecs in the repo are archived (`docs: archive prd`, `docs: archive prds`). The autonomous techspec is _current and operational_ — tasks 01-18 reference it directly with `<critical>ALWAYS READ _techspec.md</critical>` blocks. This is the live operating manual until the next major TechSpec lands.
