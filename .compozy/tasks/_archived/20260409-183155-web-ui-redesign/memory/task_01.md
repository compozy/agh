# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Replace entire CSS design token system: OKLCH → hex, Geist/Bricolage → Inter/JetBrains Mono, remove all gradients/textures/shadows for flat depth model.

## Important Decisions

- DESIGN.md tokens defined in `:root` as `--color-*` custom properties (e.g., `--color-canvas`, `--color-accent`)
- shadcn theme variables use direct hex values (no var() references to DESIGN.md tokens) to avoid circular dependencies
- `@theme inline` keeps shadcn → Tailwind utility mappings; does NOT duplicate DESIGN.md tokens (avoids `--color-accent` collision with shadcn's accent)
- Badge tint tokens added as `--color-*-tint` (e.g., `#E8572A26` = 15% opacity hex)
- Radius scale changed from `--radius: 1rem` to `--radius: 0.5rem` (8px base) per DESIGN.md
- Shimmer animation kept (functional loading state, not decorative gradient)

## Learnings

- `@theme inline` in Tailwind v4 does NOT create CSS custom properties — values are inlined into utility classes. This means `:root { --color-accent: #E8572A }` and `@theme inline { --color-accent: var(--accent) }` don't conflict.
- oxfmt lowercases all hex values in CSS (e.g., `#E8572A` → `#e8572a`). Tests must use case-insensitive matching.

## Files / Surfaces

**Modified:**
- `web/src/styles.css` — complete rewrite
- `web/package.json` — font dependency swap (bun add/remove)
- `web/bun.lock` — updated by bun
- `web/src/routes/_app.tsx` — removed `ds-texture-canvas-subtle` class
- `web/src/components/app-sidebar.tsx` — `--ds-*` → `--color-*`, `font-display` → `font-sans`
- `web/src/components/app-header.tsx` — `--ds-*` → `--color-*`
- `web/src/routes/_app/index.tsx` — `--ds-*` → `--color-*`
- `web/src/routes/_app/session.$id.tsx` — `--ds-*` → `--color-*`
- `web/src/systems/workspace/components/workspace-selector.tsx`
- `web/src/systems/session/components/session-sidebar-item.tsx`
- `web/src/systems/session/components/chat-header.tsx`
- `web/src/systems/agent/components/agent-sidebar-group.tsx`
- `web/src/systems/session/components/message-bubble.tsx`
- `web/src/systems/session/components/message-markdown.tsx`
- `web/src/systems/session/components/tool-call-card.tsx`
- `web/src/systems/session/components/thinking-block.tsx`
- `web/src/systems/session/components/processing-indicator.tsx`
- `web/src/systems/session/components/permission-prompt.tsx`
- `web/src/systems/session/components/message-composer.tsx`
- `web/src/systems/session/components/chat-view.tsx`
- `web/src/systems/session/components/tool-renderers/*.tsx` (all 6)
- `web/src/components/design-system/*.tsx` (all components + stories)

**Created:**
- `web/src/styles.test.ts` — design token verification tests

## Errors / Corrections

None.

## Ready for Next Run

Task complete. All subtasks done. `make web-lint && make web-typecheck && make web-test` pass.
