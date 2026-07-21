---
id: ET-web-window-routing-lifecycle
area: ET
title: Operate window lifecycle with URL and desktop-state parity
persona: Bruno
journey:
expected: Dock, palette, pointer, and keyboard activation open or focus one window instance; drag, resize, zoom, minimize, restore, and close preserve the documented geometry and successor focus; the focused window owns the URL with one history write per user cause; browser, CLI, and peer-browser changes converge live.
entry_points: web desktop dock and windows; browser history; agh desktop-state
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-desktop-shell-lifecycle; ET-desktop-state-agent-surface; ET-web-route-chrome-topbar
---

story: As an operator or agent, I can arrange one shared desktop and trust every surface to observe the same focused windows and geometry.

qa-impact: OS Shell Task 04 introduced the window manager, routing coordinator, live convergence, and agent-arranged geometry. Flag only; the next QA cycle owns live retesting.
