# AGH Linear-Like Redesign Exploration

## Summary

- Create 4 Paper artboards focused on the task list page only, with "free exploration" as the visual constraint: proposals may break further from the current AGH warm-operator style while keeping AGH recognizable through a restrained orange anchor.
- Use `docs/design/redesign-linear` as negative reference: keep useful Linear research from V1/V2, exclude V3 entirely, and avoid the current problems: too many borders, hard app shell segmentation, muddy gray combinations, loud colored badges, and "operator/dev/tech" over-signaling.
- Ground the work in local `DESIGN.md`, `docs/design/design-system/`, the current screenshots, and external Linear-style references from getdesign.md.
- Paper target: current `Scratchpad`, `Page 1`.

## Key Changes

- Palette moves away from AGH's current warm charcoal ramp toward cooler, calmer graphite and slate-gray systems while retaining muted AGH orange as the single recognizable action/accent color.
- Badges/statuses replace saturated semantic badge fills with low-chroma glyphs, text labels, or neutral chips. Semantic colors remain available only as tiny state signals.
- App shell reduces the rigid three-panel bordered feeling and favors calmer massing, background contrast, spacing, selected-row weight, and typographic hierarchy.
- Typography keeps a product UI sans stack. Mono is reserved for identifiers or compact metadata, not every section label.
- `v3-linear-editorial` is excluded as a foundation. If this exploration becomes repo code later, that variant should be removed rather than maintained.

## Paper Artboards

1. **Proposal A: Graphite Linear**
   - Mood: overcast graphite.
   - Dark, cool, near-neutral shell with very low border count.
   - Task list rows sit directly on the surface with section grouping by spacing and soft background shifts, not full-width bars.
   - Orange appears only on primary action, active workspace/task state, and focus.

2. **Proposal B: Focused Inbox**
   - Mood: quiet inbox.
   - Stronger left navigation reduction: fewer icons, lower contrast labels, more breathing around the active team/task scope.
   - Task list becomes the hero surface. Sidebar recedes instead of competing with the list.
   - Status becomes an icon lane plus muted copy, not colorful badges.

3. **Proposal C: Paper Slate**
   - Mood: slate document.
   - More radical: keeps dark mode but uses a document/editor rhythm, lighter typography, fewer chrome elements, and a softer reading surface.
   - Sections use headings and count labels rather than hard bars.
   - Useful to test how far AGH can move from "operator" without losing task clarity.

4. **Proposal D: Warm Accent, Cold System**
   - Mood: instrument gray with ember accent.
   - Keeps AGH orange most visibly, but cools every neutral around it so the orange feels more premium and less "terminal warning."
   - Task groups have denser information than Proposal C, but with softer hierarchy than V1/V2.
   - Best candidate if the final system must be implementable with minimal token disruption.

## Implementation Procedure

- Use Paper MCP in execution mode.
- Call Paper `get_basic_info` and `get_font_family_info` before writing typography.
- Create four 1440 x 900 desktop artboards.
- Build each artboard incrementally with `write_html`: shell, header, sidebar, filter row, task groups, then repeated task rows.
- Use realistic AGH task data from `docs/design/redesign-linear/shared/data.jsx`, simplified where comparison benefits from less noise.
- After each artboard, call `get_screenshot`, review spacing, typography, contrast, alignment, artboard fit, and repetition, then make targeted fixes.
- Finish with `finish_working_on_nodes`.

## Test Plan

- Compare all four Paper screenshots side by side against the provided Linear screenshots and current V1/V2/V3 attempts.
- Confirm each proposal visibly reduces border density, badge color saturation, operator/dev signaling, and app-shell hardness.
- Confirm task title, status, owner, and date remain scannable without loud badges.
- Identify which proposal maps back to `DESIGN.md` tokens with the fewest changes and which requires a broader styleguide pivot.

## Assumptions

- No production code is edited during this exploration.
- `PRODUCT.md` is absent, so `impeccable` product context remains blocked; the design uses `DESIGN.md`, `COPY.md` rules where relevant, `docs/design/design-system/`, screenshots, and the accepted brief.
- This is a design exploration, not a production implementation. No `make verify` is required for this Paper-only phase.
- Conversation remains in BR-PT; persistent artifacts, Paper layer names, and design notes use English.
