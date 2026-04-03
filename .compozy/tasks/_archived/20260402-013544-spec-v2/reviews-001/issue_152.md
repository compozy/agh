---
status: resolved
file: web/src/app.css
line: 106
severity: medium
author: claude-reviewer
---

# Issue 152: Dashboard sidebar does not respect collapsed width from CSS variable on body-level grid



## Review Comment

The `.dashboard-body` grid uses a fixed `var(--sidebar-width)` for its column template:

```css
.dashboard-body {
    display: grid;
    grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
    min-height: 0;
}
```

However, when the sidebar is collapsed, the sidebar element changes its own width to `var(--sidebar-collapsed-width)` (40px), but the grid column still allocates `var(--sidebar-width)` (240px). This creates a 200px gap between the collapsed sidebar and the canvas.

The sidebar component uses a CSS class `.collapsed` which sets `width: var(--sidebar-collapsed-width)`, but this only affects the sidebar element itself, not the parent grid column. The parent grid continues to allocate 240px for the sidebar column.

The `@media (max-width: 1279px)` rule also only changes the sidebar width, not the grid template.

**Suggested fix**: The grid template columns should be dynamic based on the sidebar state. Either:
1. Use a CSS custom property toggled by JavaScript to control the grid column width
2. Use the sidebar width transition on both the sidebar element and the grid column
3. Use `grid-template-columns: auto minmax(0, 1fr)` so the grid follows the sidebar's intrinsic width

## Triage

- Decision: `valid`
- Notes:
  - The CSS currently collapses only the sidebar element width, while `.dashboard-body` continues reserving the full `--sidebar-width` column.
  - That creates a visible layout gap between the collapsed sidebar and the canvas, so this is a user-facing rendering defect.
  - The fix belongs in the scoped stylesheet and should update the parent grid column width when a collapsed sidebar is present.
  - Resolution: updated `web/src/app.css` so the body grid shrinks with collapsed sidebars, including the small-screen collapsed layout, and added a CSS regression test in `web/src/app-css.spec.ts`.
