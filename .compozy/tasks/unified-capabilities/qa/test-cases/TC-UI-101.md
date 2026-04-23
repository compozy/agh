# TC-UI-101: Network Workspace Responsive Shell

**Priority:** P1
**Type:** UI/Visual
**Module:** Web Network Workspace
**Breakpoints:** Desktop, Tablet, Mobile

## Objective

Verify the network workspace shell remains usable and design-system compliant across responsive layouts.

## Preconditions

- Web dependencies are installed.
- Network route can load with mocked or local API data.
- Design tokens come from `DESIGN.md` and `packages/ui/src/tokens.css`.

## Test Steps

1. Open `/network` at 1280px width.
   **Expected:** Three-column layout appears when details are open; sidebar, timeline, and details panel are visible without overlap.
2. Toggle details closed.
   **Expected:** Layout changes to two columns on desktop and URL search includes `details=closed`.
3. Open at 768px width.
   **Expected:** Columns stack predictably and all primary controls remain reachable.
4. Open at 375px width.
   **Expected:** Text does not overlap, buttons remain tappable, and panels stack vertically.

## Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Long channel name | 80 chars | truncated, no layout break |
| Long message text | multi-line text | wraps inside timeline |
| Details panel open on mobile | mobile viewport | no horizontal overflow |

## Related

- TC-UI-002
- TC-UI-008
