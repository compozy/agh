---
id: ET-web-tasks-mode-url
area: ET
title: Tasks mode navigation via URL search param
persona: Bruno
journey: J-24
expected: Tasks List/Kanban/Dashboard/Inbox modes are RouteNav links driven by `?mode=` (not local-only pills); the active mode has `aria-current="page"`; refreshing preserves the mode; switching modes updates the URL without losing workspace scope.
entry_points: web `/tasks`; `/tasks?mode=kanban|dashboard|inbox`
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-route-chrome-topbar
---

Added by Route Chrome alignment (2026-07-17). Mode moved from local state into the URL as RouteNav.
