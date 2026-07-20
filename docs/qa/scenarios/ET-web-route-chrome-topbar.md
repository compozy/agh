---
id: ET-web-route-chrome-topbar
area: ET
title: Window chrome breadcrumb and PageHead focus
persona: Bruno
journey: J-marketplace-acquisition
expected: Every open desktop window owns one 3-zone 48px topbar with workspace and route breadcrumbs, the current route identity, optional RouteNav, and app actions; PageHead remains subordinate body metadata; focusing a window makes its topbar and URL authoritative without creating a second shell-level title.
entry_points: web desktop windows; any windowed catalog or detail route
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

QA impact 2026-07-18: route identity and navigation focus moved to the shell Topbar to match
`DESIGN.md`; PageHead remains visual summary metadata only.

QA impact 2026-07-18: PageContent width and count-chip geometry now compile from the canonical
custom-property tokens. Verify the wide-screen content gutter and both standard and compact count
chips retain their intended dimensions.

QA impact 2026-07-18: at the 320px collapsed breakpoint the route H1 and trailing action remain
visible, the ancestor breadcrumb hides, and the center RouteNav scrolls horizontally without
clipping the action rail. Desktop retains the centered symmetric layout.

QA impact 2026-07-18: isolated Task run, Task detail, and Loop detail component stories now mount
the same Topbar slot host as route tests, so their action and overflow rails remain visible in visual
contract captures.

QA impact 2026-07-18: Home keeps its connection indicator beside the subordinate body PageHead,
outside the persistent Topbar H1/action rails.

QA impact 2026-07-18: routes without a center RouteNav no longer reserve its grid column. Verify
long detail-route titles use the released width at collapsed and desktop viewports while trailing
actions remain aligned.

QA impact 2026-07-20: OS Shell Task 04 deleted the global `TopbarShell`. Route identity and
actions now live in each window's `TopbarSlotProvider`; reset to `untested` for the next QA cycle.
