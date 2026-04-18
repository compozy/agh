# Docs UI Kit

Recreation of the AGH documentation layout from `packages/site/app/docs/*` as composable React components. Dark-only. Content is illustrative (inspired by the runtime surface) rather than lifted verbatim.

## Components

**Shell**

- `DocsHeader` — global top nav with Docs chip + nav pills + search.
- `DocsSidebar` — left rail with grouped headings: Getting started, Runtime, AGH Network, Reference. Active item gets the accent-on-tint pill + left border.
- `DocsToc` — right-rail "On this page" anchor list.
- `DocsFooter` — wordmark + copyright + link trio.

**Content blocks (`DocBlocks.jsx`)**

- `Breadcrumb`, `DocH1`, `DocH2`, `DocP`, `InlineCode`, `Callout` (info / warn / success), `CommandTable`, `PageNav`.

**Pages (`Pages.jsx`)**

- `SessionsPage` — sample docs page composing all the blocks above.

Shares `CodeBlock` + `Eyebrow` from the marketing kit's `Primitives.jsx`.

## Known gaps vs production

- Real fumadocs MDX search isn't wired; search palette is decorative.
- No live syntax highlighter; `CodeBlock` uses a simple accent-colored `$` prompt instead.
- Only one sample page is implemented — the sidebar selection state exists but only `sessions` has content.
