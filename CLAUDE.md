# CLAUDE.md

## Project Overview

AGH is an Agent Operating System — a Go single-binary daemon that manages AI agent sessions via ACP (Agent Client Protocol). It spawns ACP-compatible agents (Claude Code, Codex, Gemini CLI, etc.) as subprocesses, communicates via JSON-RPC over stdio, persists events in SQLite, and exposes interfaces via HTTP/SSE (web UI) and UDS (CLI). A Fumadocs site at `agh.network` documents the runtime and the AGH Network protocol.

**Goals**: daemon single-binary in background, strong observability, agent-first system (agents manipulate via CLI + REST), highly extensible, highly configurable.

**Core product premise**: every capability must be both extensible by the runtime and manageable by agents. Features are incomplete if they only work through internal Go calls or the web UI.

## Greenfield Alpha — Zero Legacy Tolerance

No production users exist. Never sacrifice code quality for backward compatibility. Never write migration, compat, or defensive code for old state — delete the old thing instead of working around it.

**Hard cuts, not bridges.** Renames sweep code, storage, APIs, CLI, extensions, specs, RFCs, AND `.compozy/tasks/*` artifacts in the same change. No aliases, no dual fields, no schema fallback paths. Every breaking-change techspec MUST explicitly name its delete targets.

## Critical Rules

- **`make verify` MUST pass** before completing ANY task (runs `fmt → lint → test → build`). Zero warnings, zero errors. No exceptions.
- **`make lint` has zero tolerance** — any golangci-lint issue is a blocking failure.
- **Check dependent package APIs** before writing integration code or tests.
- **Never add dependencies by hand in `go.mod`** — always use `go get`.
- **Never use web search tools for local project code** — use Grep/Glob instead. Web search is only for external docs.
- **Never run destructive git commands** (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) **without explicit user permission**. If the worktree contains unexpected edits, read and work around them.
- <critical>NEVER ignore errors with `_` in production code or in tests — every error must be handled or have a written justification.</critical>
- <critical>NEVER COMMITS `ai-docs/`, `.tmp/`, or `.compozy/tasks/*/memory/` TO THE REPO. They are local tracking artifacts.</critical>
- **Subagents are read-only.** Use them for analysis, exploration, and parallel research. The author of every code change is the agent paired with the user. Subagent output is treated as evidence, not as committed work.

## Workflow Rules

These govern how features move from idea to ship. Internalize them before opening a TechSpec or running a task.

- **Multi-LLM pipeline is the default dev model.** Codex (`gpt-5.4` with `reasoning_effort=xhigh`) authors specs; Claude Opus pressure-tests them; `gpt-5.4-mini` with `reasoning_effort=high` does parallel breadth exploration when explicitly delegated. Do not substitute models without explicit user approval.
- **Every TechSpec is peer-reviewed before approval.** Run `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file <prompt>`; resolve every blocker before approving.
- **Every `_tasks.md` ends with a QA pair.** `cy-create-tasks` MUST append `$qa-report` and `$qa-execution` (with e2e for UI-bearing features) following the `.compozy/tasks/hermes` template.
- **Every backend task carries a `Web/Docs Impact` subitem.** List affected `web/` routes/components/hooks AND `packages/site` doc pages. Backend-only tasks may declare "no impact" but only after analysis.
- **Every spec/feature carries an extensibility + agent-manageability + config lifecycle analysis.** Creating, updating, or removing a feature MUST state how it integrates with AGH extensibility surfaces (extensions, hooks, skills/capabilities, tools/resources, bundles, registries, bridge SDKs), which CLI/HTTP/UDS surfaces let agents manage it, and whether `config.toml` keys/defaults/docs are added, changed, or removed. "No impact" is acceptable only with explicit evidence.
- **Reference competitors by file path in tasks.** When a TechSpec relies on `.resources/<repo>/` references, generated tasks must include explicit competitor file paths so implementing agents read them too. Reference-bearing analysis files belong under `.compozy/tasks/<slug>/analysis/`.
- **Worktree isolation is mandatory for parallel QA.** Concurrent runs use unique `AGH_HOME`, unique daemon ports, and unique `tmux-bridge` socket paths. Default home/port use is forbidden when concurrency is signaled.
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

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain                       | Required Skills                                                                          | Conditional Skills                     |
| ---------------------------- | ---------------------------------------------------------------------------------------- | -------------------------------------- |
| Go / Runtime                 | `agh-code-guidelines` + `golang-pro`                                                     | `context7`                             |
| Config / Logging             | `agh-code-guidelines` + `golang-pro`                                                     |                                        |
| TUI / CLI Bubbletea          | `bubbletea` + `agh-code-guidelines` + `golang-pro`                                       |                                        |
| Bug fix                      | `systematic-debugging` + `no-workarounds`                                                | `testing-anti-patterns`                |
| Writing Go tests             | `agh-test-conventions` + `testing-anti-patterns` + `golang-pro`                          | `vitest` (only for test tooling docs)  |
| Cleanup / failure paths      | `agh-cleanup-failure-paths` + `agh-code-guidelines` + `golang-pro`                       | `deadlock-finder-and-fixer`            |
| Schema / migration changes   | `agh-schema-migration` + `golang-pro`                                                    |                                        |
| Contract / OpenAPI changes   | `agh-contract-codegen-coship`                                                            |                                        |
| Task completion              | `cy-final-verify`                                                                        |                                        |
| Lessons learned              | `lesson-learned`                                                                         |                                        |
| Architecture audit           | `architectural-analysis`                                                                 | `refactoring-analysis` + `ubs`         |
| Concurrency / races          | `deadlock-finder-and-fixer` + `golang-pro`                                               | `systematic-debugging`                 |
| Performance / hot paths      | `extreme-software-optimization` + `golang-pro`                                           |                                        |
| Security review              | `security-review`                                                                        | `ubs`                                  |
| Creative / new features      | `brainstorming`                                                                          | `cy-idea-factory`                      |
| Council debate (high-impact) | `council`                                                                                | `brainstorming`                        |
| PRD creation                 | `cy-spec-preflight` + `cy-create-prd`                                                    | `cy-idea-factory`                      |
| TechSpec creation            | `cy-spec-preflight` + `cy-create-techspec` + `cy-spec-peer-review`                       | `cy-research-competitors`              |
| Task generation              | `cy-spec-preflight` + `cy-create-tasks` + `cy-tasks-tail-qa-pair` + `cy-web-docs-impact` |                                        |
| Competitor research          | `cy-research-competitors`                                                                | `context7` + `find-docs`               |
| Execute a PRD task           | `cy-execute-task`                                                                        | `cy-workflow-memory`                   |
| Review round / fixes         | `cy-review-round` + `cy-fix-reviews`                                                     | `fix-coderabbit-review`                |
| Release / scenario QA        | `real-scenario-qa` (delegates to `qa-execution` + `qa-report`)                           | `agh-worktree-isolation`               |
| Git rebase / conflicts       | `git-rebase`                                                                             |                                        |
| External docs lookup         | `context7` + `find-docs`                                                                 | `exa-web-search-free`                  |
| Diagrams (spec / ADR)        | `mermaid-diagrams`                                                                       | `architecture-diagram`                 |
| Documentation (internal)     | `documentation-writer`                                                                   | `crafting-effective-readmes`           |
| Skill / agent-md authoring   | `skill-best-practices` + `agent-md-refactor`                                             |                                        |
| UI / Design (any surface)    | `agh-design` + `design-taste-frontend` + `minimalist-ui`                                 | `frontend-design` + `interface-design` |

Web-specific skill dispatch is in `web/CLAUDE.md` and `web/AGENTS.md`. Site-specific dispatch is in `packages/site/CLAUDE.md`.

Every domain change requires its skill — no skipping "because it's a small change". Activate multiple skills when code touches multiple domains.

`nats` skill is installed but architecturally forbidden in AGH (see Architecture Principles). Do not activate it.

## Build Commands

### Go (backend)

```bash
make verify              # BLOCKING GATE: fmt → lint → test → boundaries → build
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
cd packages/site && bun run typecheck
cd packages/site && bun run test
cd packages/site && bun run build
make site-dev            # Dev server
make site-build          # Production build
make cli-docs            # Regenerate CLI reference from cobra JSON export
```

Web (`web/`) commands are documented in `web/CLAUDE.md`.

## Commit style: <type>: <description>

Allowed prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `build:`. **NO `chore:`, `style:`, or `ci:`.** Tooling and CI changes use `build:`. PR-merged commits include `(#NN)` suffix.

**One commit per remediation batch.** `cy-fix-reviews` rounds produce exactly one local commit per round. Run `make verify` BEFORE and AFTER the commit. Never `git commit --amend` after pre-commit hook failures — fix and create a new commit.

## Code Search Hierarchy

1. **Grep / Glob** — for local project code.
2. **`context7` / `find-docs` skills** — for external library documentation.
3. **`exa-web-search-free`** — for web research, news, external code examples when the local docs tools are insufficient.

## Old Project Reference

The `.old_project/` directory contains the previous AGH implementation (78K+ LOC). **Reference only** — do not modify, do not import, do not include in builds. Exclude from code search results.

## Surface Map

Repo layout. Each surface owns its instructions:

| Path             | Stack                                                                   | Instructions              |
| ---------------- | ----------------------------------------------------------------------- | ------------------------- |
| `cmd/agh`        | Go binary entry point                                                   | `internal/CLAUDE.md`      |
| `internal/`      | Go runtime daemon (ACP, SQLite, autonomy kernel, HTTP/UDS, network)     | `internal/CLAUDE.md`      |
| `web/`           | React 19 SPA (Vite, TanStack, Tailwind, shadcn)                         | `web/CLAUDE.md`           |
| `packages/site`  | Fumadocs documentation site (Bun)                                       | `packages/site/CLAUDE.md` |
| `packages/ui`    | Shared UI primitives (`@agh/ui`) consumed by `web/` and `packages/site` | `web/CLAUDE.md`           |

Backend architecture, autonomy contracts, security invariants, package layout, and `internal/`-specific debugging now live in **`internal/CLAUDE.md`**. Open it before touching any Go code under `cmd/` or `internal/`.

## Coding Style

Detailed Go style and concurrency rules live in the `agh-code-guidelines` skill (`.agents/skills/agh-code-guidelines/`). Activate it before writing or editing any production `*.go` file under `cmd/` or `internal/`. Top-level invariants restated in Critical Rules: no `_`-discarded errors, `make verify` must pass, `make lint` zero tolerance.

## Testing

Detailed Go test conventions (`Should ...` subtests, `t.Parallel`, table-driven layout, status-code+body assertions, `-race`/`CGO_ENABLED`, integration/E2E build tags, runtime-contract co-ship, 80% coverage floor, commit gate semantics) live in the `agh-test-conventions` skill (`.agents/skills/agh-test-conventions/`). Activate it before writing or editing any `*_test.go` file.

### Schema Migrations

Schema migrations are mandatory for any SQLite column, index, or constraint change — `EnsureSchema`-style boot reconciliation is forbidden for column changes. Detailed authoring patterns (numbered registry, transactional wrap, `-wal`/`-shm` companion handling, `ORDER BY 0` pitfall, fresh-DB + reopen-after-restart tests) live in the `agh-schema-migration` skill.

## Memory & Skills (RFC-backed)

These rules come from RFC 001 (`.../agh-rfcs-local/001-agent-md-with-skills-memory.md`) and RFC 002 (`.../agh-rfcs-local/002-skills-system-final.md`):

- **Five-layer skill/memory/agent precedence**: Bundled → Marketplace → User → Additional → Workspace, with agent-local overriding all. Higher precedence wins on collision; an audit trail logs every shadow.
- **Memory taxonomy**: `user | feedback | project | reference` types; scopes `agent | workspace | global`. Default write scope declared per agent in `memory.scope`.
- **Memory consolidation gates**: Time → Sessions → Lock cascade ordered by computational cost. Default gates: 24h, 5 touched sessions, file-lock. Never replace gates with naive heuristics.
- **Lifecycle hooks** (`on_session_created`, `on_session_stopped`) execute in hierarchy precedence then alphabetical order; configurable timeout (default 5s); fail-open semantics (errors logged, never block); JSON over stdin.
- **Format extension default**: when integrating with an external spec (AgentSkills, AGENTS.md, MCP, A2A), extend via a namespaced metadata field (`metadata.agh.*` or `agh.*`) — never fork the format.
- **Capability vs Recipe**: reusable agent artifacts are called `capability`, NOT `recipe`/`workflow`/`procedure`/`playbook`. Capabilities are interpretive, not deterministic; they are not workflow programs in disguise.
- **Runtime moat statement**: AGH competes on runtime, SDK, observability, DX, and integration depth — NOT the wire protocol. The AGH Network protocol must remain implementable outside AGH. Any feature requiring AGH to interoperate is a design smell.

## CI / Release

- **No cron / schedule workflows.** Heavy/credentialed tests (`make test-e2e-nightly`, `make test-integration`) live in the `dry-run` job of the auto-created release PR. Rationale: release PR is the natural human-gated batching point.
- **Looper repo (`~/dev/compozy/looper`) is the canonical source** for compozy-org Go-repo CI: composite actions (`setup-go`, `setup-bun`, `setup-git-cliff`, `setup-release`), `ci.yml`, `release.yml`, `.goreleaser.yml`, `cliff.toml`. Verbatim copies into AGH.
- **Replace third-party CI actions with shell logic** when their setup fails on runners (lesson: `dorny/paths-filter@v3` runner instability replaced by inline git-based change detection).

## Cross-References

- **Backend rules**: `internal/CLAUDE.md` (Go architecture, autonomy contracts, security invariants, package layout, forensic bug-fix patterns).
- **Web rules**: `web/CLAUDE.md`. **Site rules**: `packages/site/CLAUDE.md`.
- **Spec authoring playbook** (mandatory preflight for `cy-create-prd`/`cy-create-techspec`/`cy-create-tasks`): `docs/_memory/spec-authoring-playbook.md`.
- **Standing directives** (perpetual posture): `docs/_memory/standing_directives.md`.
- **Lessons learned** (durable engineering insights with evidence): `docs/_memory/lessons/` — see `README.md` for the index.
- **Glossary** (canonical vocabulary — `capability` vs `recipe`, AGENT.md vs AGENTS.md, Peer Card vs Agent Card, autonomy primitives): `docs/_memory/glossary.md`.
- **Cross-source synthesis** (evidence trail behind every rule above): `docs/_memory/_synthesis.md` and `docs/_memory/analysis/analysis_*.md`.
- **Active TechSpec**: `.compozy/tasks/autonomous/_techspec.md`. **ADRs**: `.compozy/tasks/autonomous/adrs/`.
- **Authoritative design tokens**: `DESIGN.md` (repo root).
