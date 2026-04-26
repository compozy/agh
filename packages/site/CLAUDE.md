# CLAUDE.md (packages/site)

Fumadocs documentation site at `agh.network`. Built with Next.js 16, Fumadocs 16, Remotion (for protocol illustrations). Bun-managed.

## Critical Rules

- **Pull tokens from `DESIGN.md` (repo root).** No invented colors, type, radii, spacing, or motion. Site obeys the same warm-dark palette as runtime + web.
- **Hero positioning is locked**: "Your agents can finally talk to each other." Network-protocol-first. Do not propose alternative hero copy without explicit user approval.
- **`packages/site` ships in same PR as backend contract changes** that affect documented APIs/CLI verbs (per `internal/api/contract` co-ship rule in root CLAUDE.md).

## Build Commands

```bash
cd packages/site && bun run source:generate   # MUST precede typecheck/test/build
cd packages/site && bun run typecheck
cd packages/site && bun run test                # vitest run
cd packages/site && bun run build               # next build
cd packages/site && bun run dev                 # next dev (predev runs source:generate)
make site-dev                                   # equivalent dev shortcut
make site-build                                  # equivalent build shortcut
make cli-docs                                   # regenerate CLI reference from cobra JSON export
```

## Skill Dispatch

| Domain                          | Required Skills                                          | Conditional Skills                          |
| ------------------------------- | -------------------------------------------------------- | ------------------------------------------- |
| Fumadocs page authoring         | `documentation-writer` + `crafting-effective-readmes`    | `find-docs` + `context7`                    |
| Marketing / landing copy        | `copywriting`                                            | `seo-audit`                                 |
| Site UI / components            | `agh-design` + `design-taste-frontend` + `minimalist-ui` | `frontend-design` + `interface-design`      |
| Remotion / video / protocol viz | `remotion-best-practices`                                | `architecture-diagram` + `mermaid-diagrams` |
| Diagrams (architecture, flow)   | `mermaid-diagrams` + `architecture-diagram`              |                                             |
| Next.js / SSR / app router      | `next-best-practices`                                    | `vercel-react-best-practices`               |
| Tailwind v4 styling             | `tailwindcss`                                            |                                             |
| TanStack (when used in site)    | `tanstack` + `tanstack-router-best-practices`            |                                             |

## Coding Style

- TypeScript strict; no `any` when concrete type is known.
- Functional React components only. No `React.FC`. Named exports.
- File names kebab-case. Imports use `@/*` alias.
- MDX content lives under `content/runtime/` and `content/protocol/`. CLI docs are auto-generated under `content/runtime/cli/` — do not hand-edit those files; edit the cobra command source instead.
- Pages must have appropriate `<title>` and meta tags via Fumadocs metadata helpers.
- Code blocks use the project's syntax-highlighting theme; do not introduce new theme variants.

## Truthful Docs > Plausible Docs

- Document only behavior the runtime actually supports today. When the AGH Network RFC differs from the implemented daemon, the docs follow the daemon and link the RFC for "future profile" context.
- API/CLI references are generated from `openapi/agh.json` and the cobra JSON export — do not paraphrase. If the generated reference is wrong, fix the source.
- Vocabulary follows `docs/_memory/glossary.md`. The canonical artifact name is `capability`, never `recipe`.

## Testing

- `bun run test` is `vitest run`. Snapshot tests cover MDX rendering; UI tests cover marketing components.
- After any change to source generation (`source.config.ts`), regenerate via `bun run source:generate` and re-run typecheck.
- Do not commit `out/`, `.source/`, `tsconfig.tsbuildinfo`, or `.next/`.

## Cross-References

- Root rules and architecture: `/CLAUDE.md`, `/AGENTS.md`.
- Web runtime UI rules: `/web/CLAUDE.md`.
- Design tokens: `/DESIGN.md`.
- Lessons / glossary / standing directives: `/docs/_memory/`.
