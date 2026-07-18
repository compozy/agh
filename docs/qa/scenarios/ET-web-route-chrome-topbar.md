---
id: ET-web-route-chrome-topbar
area: ET
title: Route chrome topbar breadcrumb and PageHead focus
persona: Bruno
journey: J-marketplace-acquisition
expected: Every `_app` route shows a 3-zone topbar (breadcrumb · optional RouteNav · actions) with a fixed Home icon as the first breadcrumb item linking to Dashboard (`/`); on `/` the Home icon is the current page with no duplicate "Home" label; no H1/count/status in the topbar; page identity lives in content PageHead; after path navigation focus moves to the PageHead H1; parent breadcrumb links replace body back-chevrons on detail routes.
entry_points: web app shell TopbarShell; any catalog or detail route
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-catalog-navigation; ET-web-tasks-mode-url; ET-web-jobs-triggers-catalog
---

Added by Route Chrome alignment (2026-07-17). Flag only — retest in the next QA cycle.

Verify against `docs/design/opendesign/systems/design-system.html` route chrome contract (§03–§08).
