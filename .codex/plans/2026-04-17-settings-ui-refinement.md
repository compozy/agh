# Settings UI Refinement Plan

## Summary

- Rework the settings experience as a stronger Paper-aligned refinement of the current AGH implementation, not a redesign from zero.
- Preserve the existing AGH `dark operator` language: dark surfaces, orange action accent, Inter + JetBrains Mono, flat depth, Lucide-based iconography already used by the app.
- Fix the root layout issues in shared settings primitives first, because the current problems come from the shell and row components more than from any one page.

## Implementation Changes

### Shared Shell And Footer

- Refactor `web/src/systems/settings/components/settings-page-shell.tsx` into a true three-band layout:
  - fixed page header
  - independently scrollable content body
  - dedicated footer slot for save actions
- Move save controls out of the page body flow. `SettingsSaveBar` should render through the shell footer slot so it sits flush against the content edges and bottom boundary, with no inner card margin.
- Keep footer placement global and stable across editable settings pages. Pages without inline edits, such as collection/list pages, keep their existing dialog-driven save flows and do not render the footer bar.
- Standardize a global top-right action cluster in the shell. `Restart daemon` stays universally available through the shared restart state. The secondary config affordance should reuse existing General-section data rather than requiring a new backend API.

### Shared Primitives And Visual Hierarchy

- Rebuild `SettingsFieldRow` around a predictable responsive grid instead of the current `justify-between` + `shrink-0` layout.
- On desktop, use a stable label/meta column and a stable control column so inputs line up vertically across the page.
- On mobile, collapse rows to a single-column stack with label and description above the control.
- Keep labels above helper text, and stop placing descriptive copy, hint text, and controls in competing horizontal lanes.
- Retune `SettingsSectionCard` to create clearer section boundaries with divider-led grouping, larger vertical spacing, and calmer section headers. Avoid turning every group into a boxed card.
- Add a small shared runtime/stat block primitive for read-only summaries used by General, Automation, Network, and Observability so those pages stop mixing ad hoc metric layouts.
- Reduce header/status noise:
  - header line keeps daemon state plus only the most important 2-3 facts
  - secondary counts move into section summaries or runtime blocks
  - non-semantic metadata shifts to muted mono text instead of colored chips

### Color And Density Discipline

- Restrict saturated color to real semantics and primary actions only:
  - orange for primary action and active navigation
  - green/yellow/red only for actual status or warnings
  - neutral text and dividers for everything else
- Remove decorative color competition in table rows, section notes, and badges. Replace “always colored” pills with neutral mono labels unless the value is truly success, warning, or error.
- Flatten high-noise tables and panels so the page reads more like a structured operator document and less like stacked widgets.

### Page-Level Retuning

- Summary/edit pages (`general`, `memory`, `skills`, `automation`, `network`, `observability`):
  - restore the Paper-style rhythm of `status -> divider -> section -> divider -> section`
  - align controls consistently
  - move operational links and low-priority explanatory copy out of the visual hot path
  - simplify compound numeric groups so they scan left-to-right without awkward gaps
- Collection pages (`providers`, `environments`, `mcp-servers`, `hooks-extensions`):
  - keep the table-first structure
  - tone down legends and state indicators
  - make table headers, row spacing, and actions feel calmer and more uniform
  - avoid loud per-row badge stacks unless the row truly carries warning/error state
- Navigation shell:
  - keep the current settings route tree and left section nav
  - refine active-state emphasis so it is clear without competing with page content

### Public Interfaces / Types

- No backend or public HTTP contract changes are required for this pass.
- Reuse the existing settings API and shared restart state.
- If the shell needs config-path metadata for the global header affordance, source it from the existing General settings query rather than expanding the API surface.

## Test Plan

- Update shared component tests around:
  - shell header/body/footer structure
  - save bar footer docking behavior
  - responsive field-row alignment and stacking
- Update route tests for representative pages such as General, Providers, Automation, and Network so they assert the new structure without hard-coding cosmetic churn.
- Extend the existing settings browser coverage in `web/e2e/settings.spec.ts` with layout-focused assertions for:
  - footer save bar placement on editable pages
  - visible section separation
  - calmer header/status presentation
  - aligned controls on representative forms
- Final verification for implementation:
  - `make web-lint`
  - `make web-typecheck`
  - `make web-test`
  - `make test-e2e-web`
  - `make verify`

## Assumptions

- Preserve AGH’s current visual system and tokens; do not import a new bright/minimal/light aesthetic from the skills.
- Dialog-based collection editing remains as-is; the docked footer applies only to pages with inline mutable drafts.
- The implementation should favor shared primitive cleanup over per-page one-off fixes.
- The accepted plan, once approved for execution, should be persisted under `.codex/plans/` per workspace policy.
