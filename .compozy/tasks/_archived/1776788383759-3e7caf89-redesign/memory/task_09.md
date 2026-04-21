# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Ship `@agh/ui` `CodeBlock` primitive per DESIGN.md §4: canvas-deep container, JetBrains Mono 14px/1.6, accent `$ ` prompt, optional language eyebrow, ghost copy button with 1.5s check-swap on success. Export, unit tests, story with four required variants + interaction play.

## Important Decisions

- **Prompt rendering per-line.** `showPrompt=true` iterates lines and adds `$ ` only where `shouldRenderPrompt(line)` passes (skips empty / whitespace-indented continuations / `#` comments). Keeps the mock's `agh network` shell example clean without asking callers to strip `$ ` manually.
- **Absolute positioning for chrome.** Language eyebrow top-left, copy button top-right (ghost, `icon-xs`). `<pre>` padding compensates (`pt-9` when `language`, `pr-12` when `copyable`) so chrome never overlaps code.
- **Copy button reuses `@agh/ui` Button (ghost/icon-xs).** Override ghost's default `hover:text-foreground` with `hover:text-[color:var(--color-accent)]` (DESIGN.md §4) and flag `data-copied="true"` for success state, coloring the check with `--color-success`.
- **Clipboard call is best-effort.** `handleCopy` guards `typeof navigator === "undefined"` and swallows rejection — avoids crashing in insecure contexts or on permission denial. Tests install a clipboard descriptor then restore afterwards.

## Learnings

- **Lucide icon SVGs carry `lucide lucide-<name>` classes** (see `createLucideIcon`). `button.querySelector("svg.lucide-copy")` vs `"svg.lucide-check"` is a reliable icon-swap assertion in jsdom without relying on `aria-label`, which already changes on copy state.
- **Vitest fake timers + React effect-cleanup timer:** the Promise returned by `navigator.clipboard.writeText` must flush first (`await Promise.resolve()` × 2 before `vi.advanceTimersByTime(1500)`). Without that flush, `setCopied(true)` hasn't run when the timer advances.

## Files / Surfaces

- `packages/ui/src/components/code-block.tsx` (new)
- `packages/ui/src/components/code-block.test.tsx` (new — 8 tests, 96.77% stmts / 100% lines)
- `packages/ui/src/components/stories/code-block.stories.tsx` (new — ShellCommand, MultilineWithoutPrompt, LanguageLabel, CopyDisabled, CopyInteraction play)
- `packages/ui/src/index.ts` (added `CodeBlock` + `CodeBlockProps` export adjacent to `KindChip`)

## Errors / Corrections

## Ready for Next Run

- `CodeBlock` is consumable by future tasks (tasks detail view, knowledge detail, session tool-call inspector) via `import { CodeBlock } from "@agh/ui"`.
