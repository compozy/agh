## TC-UI-002: `AgentCommandSelect` and `AgentCommandMultiSelect` Group by Category Across Three Call Sites

**Priority:** P0
**Type:** UI
**Module:** `web/src/systems/agent/components/agent-command-select.tsx` + `agent-command-multi-select.tsx` + `agent-command-list.tsx`
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that the three native `<select>` agent pickers were replaced by the new shared command components and that grouping, filtering, selection semantics, and existing test IDs are preserved across:

- Session-create dialog (`AgentCommandSelect`).
- Settings skills agent scope picker (`AgentCommandSelect`).
- Network create-channel dialog (`AgentCommandMultiSelect`).

---

### Preconditions

- [ ] Web dev server running against an isolated daemon with the same seeded categorized + root-level agents from TC-UI-001.
- [ ] User can reach the session-create dialog, settings skills page, and network create-channel dialog.

---

### Test Steps

1. **Open the session-create dialog.**
   - Input: Click "New session" from an agent route.
   - **Expected:** The trigger has `data-testid="session-create-agent-select"` (existing ID preserved). Clicking it opens a popover containing `Command`, `CommandInput`, grouped `CommandList`, `CommandGroup`, `CommandItem`.

2. **Verify grouping.**
   - Input: Inspect the popover.
   - **Expected:**
     - Folders render as `CommandGroup` headings whose `data-testid` is `agent-command-group-category:${joinedSegments}` (e.g., `agent-command-group-category:Marketing/Sales`).
     - Group heading text is the formatted label `Marketing / Sales` (single-space delimited).
     - Root-level agents appear under the `Agents` group.
     - Each item is `data-testid="agent-command-item-${agent.name}"`.

3. **Filter via search.**
   - Input: Focus the input and type the categorized agent's name.
   - **Expected:** Only matching items remain visible; group headings collapse appropriately. Empty search after type-then-clear restores all groups.

4. **Empty state.**
   - Input: Type a string that matches no agent.
   - **Expected:** A `CommandEmpty` state renders.

5. **Single-select close-on-pick.**
   - Input: Click the categorized agent.
   - **Expected:** Popover closes; trigger now displays the agent name + provider + formatted category label.

6. **Settings skills agent scope picker.**
   - Input: Open settings → Skills → agent scope.
   - **Expected:** Trigger has `data-testid="settings-agent-select"` (existing ID preserved). Same grouping behavior as step 2; selection updates the in-page state.

7. **Network create-channel multi-select.**
   - Input: Open network → create channel.
   - **Expected:**
     - Items have `data-testid="network-agent-option-${agent.name}"` (existing ID preserved).
     - Selected items have `data-checked="true"`.
     - Popover stays open after selection.
     - A selected count is visible.
     - Each item shows provider + category metadata.

8. **Keyboard navigation across all three pickers.**
   - Input: `ArrowDown`, `Enter`, `Escape`, `Tab`.
   - **Expected:** Focus moves through items, Enter selects, Escape closes the popover, Tab returns focus to the trigger.

---

### Behavioral Evidence

- Operator journey: choose a categorized agent in three different dialogs and observe identical grouping.
- Cross-surface: all three pickers consume the same `AgentPayload[]` and render the same group structure for a given agent.
- Disruption probe: the empty-search state and the multi-select keep-open behavior prove the new component handles edge cases the native `<select>` could not.

---

### Audit Coverage

- C4: operator + agent observers.
- C5: three Web dialog surfaces.
- C8: payload-to-DOM parity across all three call sites.
- C11: empty / keyboard / multi-select disruption.
- C14: `make web-test -- agent-command-select|agent-command-multi-select|session-create-dialog|network-create-channel-dialog` plus a manual browser pass with screenshots.

---

### Pass Criteria

- All eight steps pass with screenshots stored at `qa/screenshots/agent-command-*.png`.
- Every legacy `data-testid` survives.

---

### Failure Criteria

- Any picker regresses to a flat list with no grouping.
- Any pre-existing `data-testid` is removed.
- Multi-select closes on first pick.
