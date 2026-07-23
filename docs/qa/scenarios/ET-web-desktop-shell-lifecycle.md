---
id: ET-web-desktop-shell-lifecycle
area: ET
title: Operate the desktop shell across workspaces and connection states
persona: Bruno
journey:
expected: A fresh workspace renders an empty desktop with menubar, dock, wallpaper, and command hint; workspace switching isolates arrangements; loss of the desktop-state stream shows a non-blocking degraded indicator while local actions continue, and reconnection preserves touched keys while adopting daemon truth for untouched keys.
entry_points: web desktop root; workspace trigger; desktop-state WebSocket stream
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-desktop-state-agent-surface; ET-web-window-routing-lifecycle
---

story: As a builder, I can keep arranging work when persistence is temporarily unavailable and recover without mixing workspace state.

qa-impact: OS Shell Task 04 introduced the runnable desktop, workspace-scoped hydration, degraded posture, and recovery policy. Menubar mark hard-cut from invented Logo variants to the official AGH `symbol`. 2026-07-22 — Sessions catalog moved from floating rail to global modal; desktop doc no longer persists `railOpen`. Flag only; the next QA cycle owns live retesting.
