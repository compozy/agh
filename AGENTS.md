## Project Overview

AGH is an agent operating system — a Go single-binary daemon that manages AI agent sessions via ACP (Agent Client Protocol). It spawns ACP-compatible agents (Claude Code, OpenClaw, Hermes, etc.) as subprocesses, communicates via JSON-RPC over stdio, persists events in SQLite, and exposes interfaces via HTTP/SSE (web UI) and UDS (CLI). A Fumadocs site at `agh.network` documents the runtime and the AGH Network protocol.

**Goals**: daemon single-binary in background, strong observability, agent-first system (agents manipulate via CLI + REST), highly extensible, highly configurable.

**Core product premise**: every capability must be both extensible by the runtime and manageable by agents. Features are incomplete if they only work through internal Go calls or the web UI.

## Greenfield Alpha — Zero Legacy Tolerance

- **No production users exist.**
- Never sacrifice code quality for backward compatibility.
- Never write migration, compatibility, or defensive code for old state — delete obsolete code instead of working around it.
- **Hard cuts, not bridges:**
  - Renames must update code, storage, APIs, CLI, extensions, specs, RFCs, and `.compozy/tasks/*` artifacts all in a single change.
  - Do not create aliases, dual fields, or schema fallback paths.
- Every breaking-change techspec **MUST** explicitly list its delete targets.

## Critical Rules

- **`make verify` MUST pass** before completing ANY task (runs `codegen-check → bun-lint → bun-typecheck → bun-test → web-build → fmt → lint → test → build → boundaries` across the entire monorepo, not just `web/`). Zero warnings, zero errors. No exceptions.
- **`make lint` (Go golangci-lint) and `make bun-lint` (oxfmt + oxlint over every workspace) both have zero tolerance** — any warning or lint issue is a blocking failure.
- **Check dependent package APIs** before writing integration code or tests.
- **Never add dependencies by hand in `go.mod`** — always use `go get`.
- **Never use web search tools for local project code** — use Grep/Glob instead. Web search is only for external docs.
- **Never run destructive git commands** (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) **without explicit user permission**. If the worktree contains unexpected edits, read and work around them.
- <critical>NEVER ignore errors with `_` in production code or in tests — every error must be handled or have a written justification.</critical>
- <critical>NEVER COMMITS `ai-docs/` or `.tmp/` TO THE REPO. They are local tracking artifacts.</critical>
- **Subagents are read-only.** Use them for analysis, exploration, and parallel research. The author of every code change is the agent paired with the user. Subagent output is treated as evidence, not as committed work.
- **ALWAYS CHECK** the `internal/CLAUDE.md` when doing Go-related stuff
- **ALWAYS CHECK** the `web/CLAUDE.md` when doing things related to the web package

## Workflow Rules

These govern how features move from idea to ship. Internalize them before opening a TechSpec or running a task.

- **Multi-LLM pipeline is the default dev model.** Codex (`gpt-5.4` with `reasoning_effort=xhigh`) authors specs; Claude Opus pressure-tests them; `gpt-5.4-mini` with `reasoning_effort=high` does parallel breadth exploration when explicitly delegated. Do not substitute models without explicit user approval.
- **TechSpec peer review is opt-in and happens after draft approval.** `cy-create-techspec` must first present the complete draft, get the user's approval on that draft, and save `_techspec.md`. Only then should the agent ask whether to run `cy-spec-peer-review`. If the user opts in, run `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file <prompt>`, summarize blockers/nits/readiness, ask which findings to incorporate, apply only the selected findings, and ask whether to run another round or stop.
- **Every `_tasks.md` ends with a QA pair.** `cy-create-tasks` MUST append `$qa-report` and `$qa-execution` (with e2e for UI-bearing features) following the `.compozy/tasks/hermes` template.
- **Every backend task carries a `Web/Docs Impact` subitem.** List affected `web/` routes/components/hooks AND `packages/site` doc pages. Backend-only tasks may declare "no impact" but only after analysis.
- **Every spec/feature carries an extensibility + agent-manageability + config lifecycle analysis.** Creating, updating, or removing a feature MUST state how it integrates with AGH extensibility surfaces (extensions, hooks, skills/capabilities, tools/resources, bundles, registries, bridge SDKs), which CLI/HTTP/UDS surfaces let agents manage it, and whether `config.toml` keys/defaults/docs are added, changed, or removed. "No impact" is acceptable only with explicit evidence.
- **Reference competitors by file path in tasks.** When a TechSpec relies on `.resources/<repo>/` references, generated tasks must include explicit competitor file paths so implementing agents read them too. Reference-bearing analysis files belong under `.compozy/tasks/<slug>/analysis/`.
- **Worktree isolation is mandatory for parallel QA.** Concurrent runs use unique `AGH_HOME`, unique daemon ports, and unique `tmux-bridge` socket paths. Default home/port use is forbidden when concurrency is signaled.
- **Deterministic QA bootstrap is mandatory for local release/scenario QA.** Start with `agh-qa-bootstrap`, create a fresh lab for each new QA pass by default, and reuse a `bootstrap-manifest.json` only when continuing the same active QA session/loop.
- **Provider-home policy must match the provider contract during local QA.** Bound-secret and brokered QA credentials use `PROVIDER_HOME` and `PROVIDER_CODEX_HOME` from the bootstrap manifest. Exception: `native_cli` providers with `home_policy = operator` (for example direct Claude Code on the operator machine) must preserve the operator `HOME` / native login state unless a scenario explicitly tests isolated provider-home behavior.
- **Isolated Web QA must export `AGH_WEB_API_PROXY_TARGET`.** When the daemon is not on the default `:2123`, derive the proxy target from the bootstrap manifest/env instead of hardcoding localhost defaults.
- **Never parallelize config writes against one isolated QA home.** `agh config set` and similar config mutations must run sequentially when they target the same provider or runtime home.
- **Skill helpers must use explicit repo-root paths.** Do not write or execute ambiguous `scripts/...` helper paths when the helper actually lives under `.agents/skills/<skill>/scripts/`.
- **Two-touch rule.** If the same package or behavior has been patched twice in the same workstream, the third change MUST be a structural redesign, not a third patch. Open a new TechSpec.
- **Conversation in Brazilian Portuguese; artifacts in English.** Spoken/typed exchanges may use BR-PT. TechSpecs, ADRs, code, commit messages, docs are always English.
- **Pushback markers are escalation signals.** When the user uses "fraco", "leviano", "ruim", "está totalmente errado", "meia boca", "esquecendo coisas", slow down and re-clarify before acting.

## Design System

**`DESIGN.md` (repo root) is the authoritative design-system specification for every AGH surface** — runtime UI, marketing site, and docs. Any UI or asset work MUST:

- Pull tokens from `DESIGN.md` (colors, type, radii, spacing, motion) — never invent values.
- Follow the flat depth model (no shadows), warm-dark palette, Inter + JetBrains Mono + Playfair Display (site-home only) + NuixyberNext (wordmark only).
- Respect the signal palette: accent `#E8572A` = action, `#30D158` = success, `#FF453A` = danger, `#FFD60A` = warning, `#BF5AF2` = info.
- When a task belongs to `.compozy/tasks/redesign/`, run it through the `designer` agent (`.claude/agents/designer.md`) in **execution mode only** and activate the mandatory design skills listed below.
- **Truthful UI > plausible UI.** Don't render controls or metrics the runtime doesn't actually support. When Paper artboards conflict with daemon truth, daemon wins. Paper governs _composition_, `DESIGN.md` governs _grammar_.

## Copy System

**`COPY.md` (repo root) is the authoritative product-language specification for every AGH surface** - marketing copy, docs prose, release copy, package metadata, UI microcopy, CLI help, and public social/SEO/OpenGraph text. Any product-facing copy work MUST:

- Read `COPY.md` before writing or changing public copy, narrative docs, UI labels, metadata, changelog/release copy, or CLI/help text.
- Treat runtime truth as stronger than copy preference: generated API/CLI references, implemented code, tests, and release artifacts beat aspirational wording.
- Follow `docs/_memory/glossary.md` for canonical terms. The canonical artifact name is `capability`, never `recipe`, `workflow`, `procedure`, or `playbook` for current AGH behavior.
- Keep `DESIGN.md` as the visual authority and `COPY.md` as the verbal/product-language authority.
- Use the `COPY.md` claim standards before saying "today", "shipping", "supported", "live", "complete", or using product counts.

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain                                | Required Skills                                                                          | Conditional Skills                                |
| ------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------- |
| Go / Runtime                          | `agh-code-guidelines` + `golang-pro`                                                     | `context7`                                        |
| Config / Logging                      | `agh-code-guidelines` + `golang-pro`                                                     |                                                   |
| TUI / CLI Bubbletea                   | `bubbletea` + `agh-code-guidelines` + `golang-pro`                                       |                                                   |
| Bug fix                               | `systematic-debugging` + `no-workarounds`                                                | `testing-anti-patterns`                           |
| Writing Go tests                      | `agh-test-conventions` + `testing-anti-patterns` + `golang-pro`                          | `vitest` (only for test tooling docs)             |
| Cleanup / failure paths               | `agh-cleanup-failure-paths` + `agh-code-guidelines` + `golang-pro`                       | `deadlock-finder-and-fixer`                       |
| Schema / migration changes            | `agh-schema-migration` + `golang-pro`                                                    |                                                   |
| Contract / OpenAPI changes            | `agh-contract-codegen-coship`                                                            |                                                   |
| Task completion                       | `cy-final-verify`                                                                        |                                                   |
| Lessons learned                       | `lesson-learned`                                                                         |                                                   |
| Architecture audit                    | `architectural-analysis`                                                                 | `refactoring-analysis` + `ubs`                    |
| Concurrency / races                   | `deadlock-finder-and-fixer` + `golang-pro`                                               | `systematic-debugging`                            |
| AGH Network (`internal/network` only) | `nats` + `agh-code-guidelines` + `golang-pro`                                            | `deadlock-finder-and-fixer`                       |
| Performance / hot paths               | `extreme-software-optimization` + `golang-pro`                                           |                                                   |
| Security review                       | `security-review`                                                                        | `ubs`                                             |
| Creative / new features               | `brainstorming`                                                                          | `cy-idea-factory`                                 |
| Council debate (high-impact)          | `council`                                                                                | `brainstorming`                                   |
| PRD creation                          | `cy-spec-preflight` + `cy-create-prd`                                                    | `cy-idea-factory`                                 |
| TechSpec creation                     | `cy-spec-preflight` + `cy-create-techspec`                                               | `cy-spec-peer-review` + `cy-research-competitors` |
| Task generation                       | `cy-spec-preflight` + `cy-create-tasks` + `cy-tasks-tail-qa-pair` + `cy-web-docs-impact` |                                                   |
| Competitor research                   | `cy-research-competitors`                                                                | `context7` + `find-docs`                          |
| Execute a PRD task                    | `cy-execute-task`                                                                        | `cy-workflow-memory`                              |
| Review round / fixes                  | `cy-review-round` + `cy-fix-reviews`                                                     |                                                   |
| Release / scenario QA                 | `agh-qa-bootstrap` + `real-scenario-qa` + `qa-report` + `qa-execution`                   | `agh-worktree-isolation`                          |
| Git rebase / conflicts                | `git-rebase`                                                                             |                                                   |
| External docs lookup                  | `context7` + `find-docs`                                                                 | `exa-web-search-free`                             |
| Diagrams (spec / ADR)                 | `architecture-diagram`                                                                   | `mermaid-diagrams`                                |
| Documentation (internal)              | `documentation-writer`                                                                   | `crafting-effective-readmes`                      |
| Copy / public product language        | `copywriting` + `documentation-writer`                                                   | `seo-audit`                                       |
| Skill / agent-md authoring            | `skill-best-practices` + `agent-md-refactor`                                             |                                                   |
| UI / Design (any surface)             | `agh-design` + `design-taste-frontend` + `minimalist-ui`                                 | `frontend-design` + `interface-design`            |

Web-specific skill dispatch is in `web/CLAUDE.md` and `web/AGENTS.md`. Site-specific dispatch is in `packages/site/CLAUDE.md`.

Every domain change requires its skill — no skipping "because it's a small change". Activate multiple skills when code touches multiple domains.

## Build Commands

### Monorepo gate

```bash
make verify              # BLOCKING GATE — full monorepo: codegen-check → bun-lint → bun-typecheck → bun-test → web-build → fmt → lint → test → build → boundaries
```

`make verify` is the only gate that exercises the entire monorepo (Go + every Bun workspace). The targets below let you run individual stages in isolation.

### Bun workspaces (monorepo-wide)

```bash
make bun-lint            # bun run lint at repo root → oxfmt + oxlint over every workspace (zero tolerance)
make bun-typecheck       # bun run typecheck at repo root → turbo run typecheck across @agh/create-extension, @agh/extension-sdk, @agh/site, @agh/ui, agh-web
make bun-test            # bun run tests at repo root → bunx vitest run over the projects in vitest.config.ts (web, packages/ui, packages/site, sdk/typescript, sdk/create-extension)
```

These three are the bun-side commands the `Verify` gate runs. Never substitute the per-package `make web-*` / `cd packages/site && bun run …` commands when you need a guardrail-quality check — they only cover their own workspace and miss every other Bun package.

### Go (backend)

```bash
make fmt                 # Format with gofmt
make lint                # Strict golangci-lint (zero issues)
make test                # Run unit tests with -race flag
make test-integration    # Add -tags integration tests
make test-e2e-runtime    # Daemon-side E2E (Go harness)
make test-e2e-web        # Browser-side E2E (Playwright)
make test-e2e            # Both lanes
make test-e2e-nightly    # Heavy E2E (release PR dry-run only)
make build               # Compile binary
make codegen             # Regenerate openapi/agh.json + web/src/generated/agh-openapi.d.ts
make codegen-check       # Verify no codegen drift (mandatory after contract changes)
make deps                # Tidy and verify modules
```

### Site (Fumadocs at packages/site)

```bash
cd packages/site && bun run source:generate
cd packages/site && bun run typecheck   # workspace-only; for the gate use make bun-typecheck
cd packages/site && bun run test         # workspace-only; for the gate use make bun-test
cd packages/site && bun run build
make site-dev            # Dev server
make site-build          # Production build
make cli-docs            # Regenerate CLI reference from cobra JSON export
```

Web (`web/`) workspace-only commands (`make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`, `make web-dev`, `make web-fmt`) are documented in `web/CLAUDE.md`. They are scoped to `web/` only — for the full guardrail use the `make bun-*` targets above.

## Commit style

- ALWAYS USE: `<type>: <description>`
- Allowed commit prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `build:`
- **Do NOT use**: `chore:`, `style:`, or `ci:`.
- Use `build:` for tooling and CI changes.
- For PR-merged commits, append a `(#NN)` suffix.
- **Create exactly one commit per remediation batch.**
- Each `cy-fix-reviews` round must produce one local commit.
- Always run `make verify` **before and after** committing.
- If a pre-commit hook fails, do **not** use `git commit --amend`. Instead, fix the issue and create a new commit.

## Code Search Hierarchy

1. **Grep / Glob** — for local project code.
2. **`context7` / `find-docs` skills** — for external library documentation.
3. **`exa-web-search-free`** — for web research, news, external code examples when the local docs tools are insufficient.

## Surface Map

Repo layout. Each surface owns its instructions:

| Path            | Stack                                                                   | Instructions              |
| --------------- | ----------------------------------------------------------------------- | ------------------------- |
| `cmd/agh`       | Go binary entry point                                                   | `internal/CLAUDE.md`      |
| `internal/`     | Go runtime daemon (ACP, SQLite, autonomy kernel, HTTP/UDS, network)     | `internal/CLAUDE.md`      |
| `web/`          | React 19 SPA (Vite, TanStack, Tailwind, shadcn)                         | `web/CLAUDE.md`           |
| `packages/site` | Fumadocs documentation site (Bun)                                       | `packages/site/CLAUDE.md` |
| `packages/ui`   | Shared UI primitives (`@agh/ui`) consumed by `web/` and `packages/site` | `web/CLAUDE.md`           |

Backend architecture, autonomy contracts, security invariants, package layout, and `internal/`-specific debugging now live in **`internal/CLAUDE.md`**. Open it before touching any Go code under `cmd/` or `internal/`.

## Coding Style

- **Skill**: `agh-code-guidelines` (`.agents/skills/agh-code-guidelines/`).
- **When**: before writing or editing any production `*.go` file under `cmd/` or `internal/`.
- **Covers**: error wrapping (`%w`), `errors.Is`/`As` only, `slog` logging, `context.Context` discipline, compile-time interface assertions, no hardcoded config, CLI flag presence detection, comments policy, generic concurrency patterns.
- **Top-level invariants restated in Critical Rules**: no `_`-discarded errors, `make verify` must pass, `make lint` zero tolerance.

## Testing

- **Skill**: `agh-test-conventions` (`.agents/skills/agh-test-conventions/`).
- **When**: before writing or editing any `*_test.go` file.
- **Covers**:
  - `t.Run("Should ...")` subtests, `t.Parallel` default (with `t.Setenv` opt-out), table-driven layout.
  - Status-code + body assertions (status-code-only is insufficient).
  - `-race` / `CGO_ENABLED=1` discipline; Linux-Race CI parity for race-sensitive packages.
  - Integration / E2E build tags (`//go:build integration`, `make test-integration`, `make test-e2e-runtime`, `make test-e2e-web`).
  - Runtime-contract co-ship (E2E mock + matchers ship with contract changes).
  - 80% coverage floor per package.
  - Commit-gate semantics (`make verify` blocks; test failures are production bugs).

### Schema Migrations

- **Skill**: `agh-schema-migration`.
- **When**: any SQLite column, index, or constraint change.
- **Mandatory**: numbered migration in the registry — `EnsureSchema`-style boot reconciliation is forbidden for column changes.
- **Covers**: numbered registry, transactional wrap (`BEGIN IMMEDIATE`), `-wal` / `-shm` companion handling on recovery, `ORDER BY 0` pitfall, fresh-DB + reopen-after-restart tests.

## Vocabulary & Product Strategy

Repo-wide rules backed by RFC 001 / RFC 002. Runtime implementation details (precedence layers, memory taxonomy, consolidation gates, lifecycle hooks) live in `internal/CLAUDE.md`.

- **Capability vs Recipe**: reusable agent artifacts are called `capability`, NOT `recipe`/`workflow`/`procedure`/`playbook`. Capabilities are interpretive, not deterministic; they are not workflow programs in disguise.
- **Format extension default**: when integrating with an external spec (AgentSkills, AGENTS.md, MCP, A2A), extend via a namespaced metadata field (`metadata.agh.*` or `agh.*`) — never fork the format.
- **Runtime moat statement**: AGH competes on runtime, SDK, observability, DX, and integration depth — NOT the wire protocol. The AGH Network protocol must remain implementable outside AGH. Any feature requiring AGH to interoperate is a design smell.

## Memory & Lessons Learned

`docs/_memory/` is the project's institutional memory — durable engineering knowledge distilled from real incidents, ADR forensics, and standing engineering posture. Treat it as authoritative when CLAUDE.md is silent or ambiguous.

- **Standing directives** — `docs/_memory/standing_directives.md`. Perpetually-active engineering posture (SD-001..SD-011): long-running session supervision, greenfield-delete, BR-PT/EN, multi-LLM pipeline, real-scenario QA, forensic-first bug fixes, truthful UI, composition-root discipline, detached lifetime, extensible-and-agent-manageable design. Read before opening a TechSpec, defending an architecture pivot, or whenever someone proposes a compat shim.
- **Spec authoring playbook** — `docs/_memory/spec-authoring-playbook.md`. Mandatory preflight for `cy-create-prd` / `cy-create-techspec` / `cy-create-tasks`, with phase-by-phase MUST / MUST-NOT and evidence references. The `cy-spec-preflight` skill enforces this — always read before producing any `_idea.md` / `_prd.md` / `_techspec.md` / `_tasks.md`.
- **Lessons learned** — `docs/_memory/lessons/` (`L-001..L-015`, plus `README.md` index). One file per durable lesson with confirmed root cause + fix + evidence (ADR, commit, review issue, or QA bug). Scan the index whenever you hit a class of issue: concurrency / API, testing discipline, autonomy architecture, persistence, spec authoring.
- **Glossary** — `docs/_memory/glossary.md`. Canonical vocabulary (`capability` vs `recipe`, `AGENT.md` vs `AGENTS.md`, Peer Card vs Agent Card, autonomy primitives). Authoritative when older RFCs / ledgers conflict. Read when naming anything new, reviewing a rename PR, or when a term feels overloaded.
- **Cross-source synthesis** — `docs/_memory/_synthesis.md`. Cross-referenced findings from 8 forensic analyses, ranked by source count — the evidence corpus behind every rule in CLAUDE.md and the standing directives. Read when challenging or evolving a rule.
- **Forensic analyses** — `docs/_memory/analysis/analysis_*.md`. Per-source raw analyses (codex sessions / plans / ledger, compozy tasks, qmd collections, local / global runs, existing surfaces) feeding `_synthesis.md`. Read when synthesis cites a finding and you need the underlying evidence.

**Authoring rules:**

- New lesson → numbered file `L-NNN-kebab-title.md` + update `lessons/README.md`. One lesson per file. Cite specific evidence (file path, commit, review issue, ledger entry). Activate the `lesson-learned` skill.
- Don't duplicate CLAUDE.md or `standing_directives.md` rules in lessons — lessons explain **why** a rule exists; rules go in their respective files.
- Don't add speculative warnings — only confirmed incidents with evidence.
- New standing directive → next `SD-NNN` block in `standing_directives.md` with Posture / Required behavior / Source / Triggers re-evaluation when.

## CI / Release

- **No cron / schedule workflows.** Heavy/credentialed tests (`make test-e2e-nightly`, `make test-integration`) live in the `dry-run` job of the auto-created release PR. Rationale: release PR is the natural human-gated batching point.
- **Looper repo (`~/dev/compozy/looper`) is the canonical source** for compozy-org Go-repo CI: composite actions (`setup-go`, `setup-bun`, `setup-git-cliff`, `setup-release`), `ci.yml`, `release.yml`, `.goreleaser.yml`, `cliff.toml`. Verbatim copies into AGH.
- **Replace third-party CI actions with shell logic** when their setup fails on runners (lesson: `dorny/paths-filter@v3` runner instability replaced by inline git-based change detection).

## Cross-References

- **Backend rules**: `internal/CLAUDE.md` (Go architecture, autonomy contracts, security invariants, package layout, forensic bug-fix patterns).
- **Web rules**: `web/CLAUDE.md`.
- **Site rules**: `packages/site/CLAUDE.md`.
- **Institutional memory**: `docs/_memory/` — see the **Memory & Lessons Learned** section above for the per-surface map.
- **Authoritative design tokens**: `DESIGN.md` (repo root).
- **Authoritative copy system**: `COPY.md` (repo root).
