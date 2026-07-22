---
id: ET-web-command-palette-shortcuts
area: ET
title: Open desktop apps, sessions, and actions from the keyboard
persona: Bruno
journey:
expected: ⌘/Ctrl+K opens one global palette over any desktop or composer and filters real apps, sessions, workspaces, and actions; Enter performs the selected action; ⌘/Ctrl+J remains scoped to the session runtime picker; ⌘/Ctrl+N, ⇧⌘/Ctrl+S, ⌘/Ctrl+W, ⌘/Ctrl+M, and Escape perform the documented shell actions with one-layer overlay unwinding.
entry_points: web desktop keyboard; command palette; session composer; menubar Help
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-desktop-shell-lifecycle; ET-web-window-routing-lifecycle
---

story: As a keyboard operator, I can discover and execute every global desktop action without losing the session runtime shortcut or trapping focus in stale overlays.

qa-impact: OS Shell Task 04 moved ⌘/Ctrl+K ownership to the global palette, rebound the runtime selector, and added the shell shortcut set. 2026-07-22 — Toggle sessions opens the global Sessions modal overlay instead of a persisted rail. Flag only; the next QA cycle owns live retesting.
