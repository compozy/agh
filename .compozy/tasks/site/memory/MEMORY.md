# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- task_01 complete: `packages/ui` (`@agh/ui`) created with design tokens + 12 base components.
- task_02 complete: `web/` migrated to consume `@agh/ui` — tokens imported, 12 components deleted, all imports updated.
- task_03 complete: `packages/site` (`@agh/site`) scaffolded with Fumadocs — two-collection docs site (runtime + protocol), DESIGN.md theming, Orama search with tags, static export producing `out/`.
- task_04 complete: CLI doc generation via Cobra `GenMarkdownTree` + Go post-processor (`internal/cli/docpost/`). Hidden `doc` subcommand, `make cli-docs` target, 108 MDX files generated.
- task_05 complete: Landing page with 8 section components in `packages/site/components/landing/`. Snapshot tests via vitest + @testing-library/react. 16 tests, build passes (122 static pages).
- task_06 complete: Three overview doc pages (what-is-agh, architecture, comparison) under `packages/site/content/runtime/overview/`. ASCII architecture diagram (no Mermaid support). Build passes (125 static pages).
- task_07 complete: Four Getting Started tutorial pages (installation, quick-start, first-agent, web-ui) under `packages/site/content/runtime/getting-started/`. All CLI commands verified against codebase. Build passes (129 static pages).

## Shared Decisions

- **@agh/ui exports source .ts files** — no dist build. Consumed via workspace protocol with bundler moduleResolution. `tsgo --noEmit` for type-checking only.
- **`shadcn/tailwind.css` stays in web/** — it's web-app-specific, not part of the shared token layer.
- **Fumadocs import from `@/.source/server`** — generated `.source` directory has no barrel index; use explicit `/server` subpath.
- **Fumadocs UI v16+ provider** — import from `fumadocs-ui/provider/next`, not `fumadocs-ui/provider`.
- **Static export with `trailingSlash: true`** — produces `/runtime/index.html` paths instead of `/runtime.html`.

## Shared Learnings

- `@base-ui/react` is the UI primitive library used by shadcn components (button, input, separator, badge, progress).
- Fumadocs MDX v14+ uses `toFumadocsSource()` (not `toRuntime()`) for source loader integration.
- For multi-source search in static export, combine indexes manually via `createSearchAPI('advanced', { indexes: [...] })`.
- `@tailwindcss/postcss` required as devDependency for Next.js + Tailwind CSS v4.
- MDX requires escaping `<` and `{` in non-code text (`\<`, `\{`). Tab-indented blocks must be converted to fenced code blocks for MDX compatibility.
- `@agh/site` has vitest configured (`packages/site/vitest.config.ts`) with jsdom environment and `@testing-library/react`. Added to root vitest workspace projects.
- Fumadocs does not render `mermaid` code blocks natively — use ASCII art in plain code blocks for architecture diagrams, or add a rehype-mermaid plugin.
- Fumadocs `root: true` changes navigation grouping but does not remove the directory segment from exported URLs. Content under `packages/site/content/runtime/core/*` still builds to `/runtime/core/*`.
- Agent-level permission overrides are part of `AGENT.md` frontmatter (`permissions` on `internal/config.AgentDef`), while global defaults live under `[permissions]` in `config.toml`.
- The docs package is named `@agh/site`; use `bunx turbo run build --filter=@agh/site` for filtered site builds. Task specs that say `--filter=packages/site` are using a stale selector.
- Current memory runtime docs must distinguish implementation from RFC 001: implemented memory is only global/workspace scoped, while RFC 001 agent-scoped memory fields and `.agents/<name>/memory/` are draft/future behavior.
- Current skills runtime maps only `name`, `description`, `version`, and `metadata` from top-level `SKILL.md` frontmatter; AgentSkills/Claude-style fields such as `allowed-tools`, `user-invocable`, and `argument-hint` currently warn and are ignored by AGH.
- Current skills marketplace CLI installs latest only: provenance stores `version`, but there is no user-facing `agh skill install --version` flag and version suffixes are not accepted in slugs.
- Current skills implementation treats `metadata.agh.memory_tags` as RFC-only; `metadata.agh.mcp_servers` and `metadata.agh.hooks` are implemented.
- Current bridge instances are runtime records managed through bridge API/CLI surfaces, not static `config.toml` blocks. Provider config and delivery defaults are JSON fields on the instance.
- Current stock bridge secret resolver supports only `env:NAME` refs with non-empty environment variables in the daemon process.
- Current bridge routing requires every enabled routing dimension to be present on the inbound event. Direct-message traffic usually uses `peer_id`, while shared channel/group traffic usually uses `group_id`; separate bridge instances may be needed for different conversation shapes.
- Current CLI reference docs live under `packages/site/content/runtime/cli-reference/` and route to `/runtime/cli-reference`; older task specs that mention `runtime/reference/cli` are stale.
- `make cli-docs` preserves the hand-authored CLI overview `cli-reference/index.mdx`; the generated root command reference is `cli-reference/agh.mdx`.
- Protocol docs use flat files under `packages/site/content/protocol/` and route to `/protocol/<slug>/`; nested `protocol/overview/index.mdx` conflicts with required flat `protocol/overview.mdx`.
- Current AGH Network v0 envelope fields are `protocol`, `id`, `kind`, `channel`, `from`, `to`, `interaction_id`, `reply_to`, `trace_id`, `causation_id`, `ts`, `expires_at`, `body`, `proof`, and `ext`; task/RFC shorthand such as `version`, `source`, or `target` is not the implemented wire shape.
- Current AGH Network `whois.query` is a string matched against peer ID, display name, capabilities, profiles, artifact support, or trust modes; do not document it as a structured query object.
- Current AGH Network implementation is still v0-only for trust/transport: it preserves `proof` opaquely, has no Ed25519/JCS verifier or conformance runner, uses `agh.network.v0` NATS subjects, and routes direct messages by SHA-256(peer ID) route tokens rather than v1 `nickname@fingerprint` identities.

## Open Risks

- Pre-existing `@agh/extension-sdk` build error (SessionState → SessionStatus rename) causes full monorepo `turbo run build` to fail — unrelated to site work.
- Current branch/worktree also has a design-token mismatch: `packages/ui/src/tokens.css` uses `#141312/#1e1c1b/#2e2c2b`, while `web/src/styles.test.ts` still asserts the older DESIGN.md palette `#121212/#1C1C1E/#2C2C2E`, so `make verify` fails before docs tasks can claim a clean full-repo gate.

## Handoffs
