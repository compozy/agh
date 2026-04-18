# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Tasks 01–11 complete and verified.
- `packages/ui` Storybook ships 12 primitive story modules with `tags: ["autodocs"]`.
- `web` Storybook now includes the pre-existing `components/design-system/*` stories, 27 `components/ui/*` story modules, system-scoped stories for agent/automation/bridges/daemon/knowledge/network/session/skill/workspace, and per-system MSW mocks composed in `web/.storybook/preview.ts`.
- Final sampled a11y pass ran against one primitive, one web/ui overlay, one session-core story, and one system-panel story. It surfaced a real `button-name` critical violation in `systems/session/MessageComposer`; the production component was fixed by giving the icon-only send button an accessible name and the sample reran clean.

## Shared Decisions

- `web` Storybook now owns the shared integration decorator stack: MSW loader, story-scoped TanStack Query, memory-history router, and the existing class-based theme decorator.
- Primitive stories import components via local relative paths (`../button`, etc.) — not `@agh/ui` — so typings bind to the authored surface. Downstream `web/` stories import from `@agh/ui`.
- Task spec tests vs techspec "no new Vitest tests" conflict was resolved in favor of the techspec: Storybook build + grep + authoring discipline satisfy the acceptance criteria; no Vitest setup added in `packages/ui`.

## Shared Learnings

- `web/.storybook/main.ts` must serve `../public` so Storybook can load `mockServiceWorker.js`.
- `packages/ui` Storybook needs `@tailwindcss/vite` in `.storybook/main.ts` for Tailwind v4 preview CSS to compile without warnings.
- `bun run lint` (web) runs `oxfmt` first, which will reformat any story that drifts from its preferred style — author close to final layout to keep diffs tight.
- Base UI `Combobox` items of shape `{ value, label }` auto-filter without extra helpers; mix `ComboboxEmpty` with `ComboboxCollection` inside `ComboboxList` because list children are `ReactNode | RenderFn`, never both.
- Sonner stories MUST mount `<Toaster />` inside each `render()` rather than via the global preview to keep toasts confined to a single story iframe.
- Use offline-safe image sources for stories (data URIs for valid states, `*.invalid` hostnames for intentional 404s) so preview builds stay deterministic regardless of network state.
- Storybook addon a11y results are readable from the manager UI by opening `?path=/story/<id>&addonPanel=storybook/a11y/panel`; this was sufficient to sample critical violations without adding another a11y runner.

## Open Risks

- Downstream web stories should not add duplicate global QueryClient or router providers unless they intentionally override the shared preview behavior.

## Handoffs

- Use the existing `design-system/*` stories as the smoke surface when validating future changes to the web Storybook preview.
