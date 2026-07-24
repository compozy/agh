# Visual Validation Contract

No visual pass is implied by static checks. A PASS requires a durable Visual Contract Mode bundle and manual inspection.

## Capture matrix

- All 16 modal/sheet surfaces: 360×800, 768×900, and 1440×900.
- Agent create, bridge create, provider create/edit sheets, and other dense surfaces: add 1920×1080.
- RuntimeSelector, AgentCommandSelect, AgentCommandMultiSelect, ScopeSelector, WorkspaceCommandSelect, and catalog CommandSelect: matched production Storybook/static pairs at 1100×700.
- RuntimeSelector additionally captures open, stale, no-model, disabled, empty, and error routes, including the separate favorite toggle and Default + seven-effort footer.
- Use deterministic review routes from `STATE-MATRIX.md` on living surface HTMLs; never patch the canonical HTML for a capture state.

## Inspection order

1. Host boundary, breakpoint, and scroll ownership.
2. Region order, alignment, sizing, and spacing rhythm.
3. Component anatomy, typography, density, and selected/focus treatment.
4. Controls, copy, catalog states, errors, secret state, and responsive behavior.
5. Surface, hairline, radius, shadow, contrast, long strings, 200% zoom, keyboard path, and reduced motion.

## Completion rule

Implementation-only screenshots are not parity evidence. Every reference/implementation pair must have zero unresolved structural mismatch, a validated review file, and clean owned-process teardown.
