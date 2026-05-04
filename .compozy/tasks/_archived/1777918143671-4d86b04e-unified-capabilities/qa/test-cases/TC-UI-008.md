# TC-UI-008: Network UI Design-System Compliance

**Priority:** P2
**Type:** UI/Visual
**Module:** Web Network Workspace + Settings
**Requirement:** AGH UI must use the canonical warm-dark design system.

## Objective

Verify the network UI follows `DESIGN.md` tokens and avoids ad-hoc visual styling.

## Preconditions

- `DESIGN.md` and token CSS are available.
- Network workspace and settings pages render under mocked data.

## Test Steps

1. Inspect network shell, sidebar, timeline cards, dialog, and settings page classes.
   **Expected:** Colors reference CSS variables from the design system instead of new hex values.
2. Verify depth model.
   **Expected:** UI uses flat backgrounds and 1px dividers; no content shadows or gradients.
3. Verify typography.
   **Expected:** UI uses Inter for readable text and JetBrains Mono for metadata/chips.
4. Verify semantic chips.
   **Expected:** Success, danger, warning, info, neutral, and accent states use tint formula.
5. Verify controls.
   **Expected:** Buttons, icons, inputs, dialog, tabs, and filters use existing AGH/shadcn primitives where available.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Disabled controls | disabled send/save | tokenized disabled colors |
| Focus states | keyboard navigation | accent focus ring visible |
| Long text | labels/messages | no overlap or clipped actionable text |

## Related

- TC-UI-101
- TC-UI-006
