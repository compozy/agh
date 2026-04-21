# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Author 12 Storybook stories (one per primitive exported by `@agh/ui`) under `packages/ui/src/components/stories/`, each with `title: "ui/<Name>"`, `tags: ["autodocs"]`, explicit `Meta<typeof Component>`, every story typed `StoryObj<typeof meta>` with an `args` field, and tokens-only styling.

## Important Decisions

- Imports use local relative paths (`../button`, etc.) per task requirement — do NOT import via `@agh/ui`.
- Test-scope conflict reconciliation: task bullets say "Unit tests required" but `_techspec.md` section "Testing Approach" says "No new Vitest tests are introduced for story files themselves" and treats `build-storybook` as the test. Followed techspec: no Vitest tests added in `packages/ui`; acceptance criteria verified by storybook build + grep review + author discipline. The "`data-variant="default"`" wording in the Button test bullet is interpreted as the `variant: "default"` args value, since the primitive does not emit that DOM attribute.
- Button default story uses `children: "Action"` + `variant: "default"` so the accessible name is "Action".
- Compound components (Card, Alert, Progress, Table, Kbd with KbdGroup) use `render` with `args: {}`; leaf primitives (Button, Badge, Input, Skeleton, Spinner, Separator, Label) use `args`.

## Learnings

- `packages/ui/.storybook/preview.ts` already wires `withThemeByClassName` (defaultTheme: "dark") so stories do not need their own theme decorator.
- `Progress` auto-composes its own `ProgressTrack`/`ProgressIndicator` children; when passing a custom body, pass `ProgressLabel` + `ProgressValue` as children and let Progress inject Track/Indicator itself.
- `Table` primitive wraps itself in an overflow container via `data-slot="table-container"`; no external scroller needed.

## Files / Surfaces

- Authoring: `packages/ui/src/components/stories/{alert,badge,button,card,input,kbd,label,progress,separator,skeleton,spinner,table}.stories.tsx`
- Verification: `bun run --cwd packages/ui build-storybook`

## Errors / Corrections

- oxfmt re-wrapped long `docs.description.component` strings during commit hook; accepted as intentional.

## Ready for Next Run

- Downstream story tasks (task_03, task_04, task_06+) should reuse the title prefix convention (`components/ui/...`, `systems/<name>/...`) and skip `autodocs` per ADR-003.
- Commit landed at b1f6931 on `storybook` branch; not yet pushed.
