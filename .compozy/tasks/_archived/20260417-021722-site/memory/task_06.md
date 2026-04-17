# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Write three Diataxis Explanation overview pages: what-is-agh, architecture, comparison.

## Important Decisions

- Replaced Mermaid diagram with ASCII art code block — Fumadocs has no built-in Mermaid rendering support in the current site configuration.
- Used "Why AGH" as the comparison page's sidebar title (from frontmatter `title: Why AGH`) for better reader framing vs. the generic "Comparison".
- Kept ASCII diagram style consistent with the README's existing architecture diagram.

## Learnings

- Fumadocs does not render `\`\`\`mermaid` code blocks as diagrams — requires a plugin (e.g. `rehype-mermaid`) or client-side rendering. ASCII diagrams in plain code blocks work reliably.
- The `...directoryname` spread syntax in `meta.json` automatically picks up subdirectory pages without listing them individually.
- Site build filter uses package name `@agh/site`, not path `packages/site`.

## Files / Surfaces

- `packages/site/content/runtime/overview/what-is-agh.mdx` (created)
- `packages/site/content/runtime/overview/architecture.mdx` (created)
- `packages/site/content/runtime/overview/comparison.mdx` (created)
- `packages/site/content/runtime/overview/meta.json` (created)
- `packages/site/content/runtime/meta.json` (modified — added `...overview`)

## Errors / Corrections

- First draft used Mermaid diagram syntax which rendered as raw code; replaced with ASCII art.

## Ready for Next Run

Task complete. All three pages build and render.
