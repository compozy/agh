# CLAUDE.md (packages/site)

Fumadocs documentation site at `agh.network`. Built with Next.js 16, Fumadocs 16, Remotion (for protocol illustrations), and Velite for the `/blog` and `/changelog` content layer. Bun-managed.

## Critical Rules

- **Pull tokens from `DESIGN.md` (repo root).** No invented colors, type, radii, spacing, or motion. Site obeys the same warm-dark palette as runtime + web.
- **Pull product language from `COPY.md` (repo root).** Landing copy, blog/changelog, runtime/protocol narrative docs, site config, OpenGraph metadata, SEO descriptions, and public CTAs MUST follow the copy system before inventing new wording.
- **Hero positioning is locked**: headline "An open workplace for AI agents." with subhead "AGH runs the agent CLIs you already use as durable sessions — with memory, autonomy, tools, and automation — connected on agh-network/v0 channels where they find each other, share capabilities, and close work with receipts." Open-workplace-first. Do not propose alternative hero copy without explicit user approval.
- **`packages/site` ships in same PR as backend contract changes** that affect documented APIs/CLI verbs (per `internal/api/contract` co-ship rule in root CLAUDE.md).

## Build Commands

```bash
# Turbo-backed validation commands run from the repo root.
make bun-typecheck                                      # full Bun workspace typecheck through turbo
make bun-test                                           # full Bun workspace test suite through turbo
bunx turbo run typecheck --filter=./packages/site       # focused @agh/site typecheck
bunx turbo run test --filter=./packages/site            # focused @agh/site tests
bunx turbo run build --filter=./packages/site           # focused @agh/site build

# Site generators and local dev shortcuts.
cd packages/site && bun run source:generate             # Fumadocs MDX -> .source/
cd packages/site && bun run content:generate            # Velite MDX/YAML -> .velite/
cd packages/site && bun run dev                         # next dev (predev runs both generators)
make site-dev                                           # equivalent dev shortcut
make cli-docs                                           # regenerate CLI reference from cobra JSON export
```

`predev`, `prebuild`, `pretypecheck`, `pretest` all run `source:generate` then `content:generate` in series. Both `.source/` and `.velite/` are generated artifacts — never commit them.

## Skill Dispatch

| Domain                          | Required Skills                                       | Conditional Skills                          |
| ------------------------------- | ----------------------------------------------------- | ------------------------------------------- |
| Fumadocs page authoring         | `documentation-writer` + `crafting-effective-readmes` | `find-docs` + `context7`                    |
| Marketing / landing copy        | `copywriting` + `documentation-writer`                | `seo-audit`                                 |
| Site UI / components            | `agh-design` + `impeccable`                           |                                             |
| Remotion / video / protocol viz | `remotion-best-practices`                             | `architecture-diagram` + `mermaid-diagrams` |
| Diagrams (architecture, flow)   | `mermaid-diagrams` + `architecture-diagram`           |                                             |
| Next.js / SSR / app router      | `next-best-practices`                                 | `vercel-react-best-practices`               |
| Tailwind v4 styling             | `tailwindcss`                                         |                                             |
| TanStack (when used in site)    | `tanstack` + `tanstack-router-best-practices`         |                                             |

## Coding Style

- TypeScript strict; no `any` when concrete type is known.
- Functional React components only. No `React.FC`. Named exports.
- File names kebab-case. Imports use `@/*` alias.
- MDX content lives under `content/runtime/` and `content/protocol/` (Fumadocs) and `content/blog/` (Velite). CLI docs are auto-generated under `content/runtime/cli/` — do not hand-edit those files; edit the cobra command source instead.
- Blog content layout: `content/blog/posts/<slug>.mdx`, `content/blog/changelog/<version>.mdx`, `content/blog/authors/<handle>.yml`. Frontmatter is zod-validated by `velite.config.ts`; broken frontmatter fails the build with line-numbered errors.
- Truthful UI applies to releases: every `content/blog/changelog/*.mdx` entry must reflect real merged work — source `added`/`changed`/`fixed`/`breaking` lists from `git log` and PR descriptions, not aspirational copy.
- Pages must have appropriate `<title>` and meta tags via Fumadocs metadata helpers.
- Code blocks use the project's syntax-highlighting theme; do not introduce new theme variants.

## Truthful Docs > Plausible Docs

- Document only behavior the runtime actually supports today. When the AGH Network RFC differs from the implemented daemon, the docs follow the daemon and link the RFC for "future profile" context.
- API/CLI references are generated from `openapi/agh.json` and the cobra JSON export — do not paraphrase. If the generated reference is wrong, fix the source.
- Vocabulary follows `docs/_memory/glossary.md`. The canonical artifact name is `capability`, never `recipe`.

## Testing

- The package `test` script is `vitest run`, but validation MUST invoke it through Turbo: `bunx turbo run test --filter=./packages/site` or `make bun-test` from the repo root.
- Do not use `cd packages/site && bun run test` or package-local equivalents as validation evidence; they bypass Turbo's cache/task graph.
- Snapshot tests cover MDX rendering; UI tests cover marketing components.
- After any change to source generation (`source.config.ts`), regenerate via `cd packages/site && bun run source:generate` and re-run `bunx turbo run typecheck --filter=./packages/site`.
- Do not commit `out/`, `.source/`, `tsconfig.tsbuildinfo`, or `.next/`.

## Cross-References

- Root rules and architecture: `/CLAUDE.md`, `/AGENTS.md`.
- Web runtime UI rules: `/web/CLAUDE.md`.
- Design tokens: `/DESIGN.md`.
- Copy system: `/COPY.md`.
- Lessons / glossary / standing directives: `/docs/_memory/`.
