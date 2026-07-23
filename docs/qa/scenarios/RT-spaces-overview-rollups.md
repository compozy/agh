---
id: RT-spaces-overview-rollups
area: RT
title: Spaces overview shows real agent/session rollups and creates spaces
persona: Théo
journey:
expected: ⇧⌘S opens the Spaces overview with a header ("Spaces" + "N spaces · M agents distributed" once workspace details load + "New space" button) over one card per workspace showing monogram, name, Current pill on the active space, "N agents · M sessions · K windows" from runtime data, a Members monogram stack from workspace agents, the workspace root path, and an Enter affordance. Card click switches workspace and closes; New space closes the overlay and opens the workspace setup dialog.
entry_points: web desktop shell ⇧⌘S, command palette "Spaces overview", menubar View menu
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence: web/src/systems/os/components/os-spaces-overview.tsx; web/src/systems/os/hooks/use-space-details.ts
last_report:
---

## Steps

1. Press ⇧⌘S on the desktop shell.
2. Verify the header subtitle counts spaces and total agents (after detail queries resolve).
3. Verify each card's agents/sessions/windows counts match `GET /api/workspaces/{id}` and the live window manager.
4. Click "New space" — overlay closes and the Add workspace dialog opens.
5. Click a non-active card — workspace switches and the overlay closes.
