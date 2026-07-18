---
id: ET-web-jobs-triggers-catalog
area: ET
title: Jobs and Triggers catalog plus deep-linkable detail
persona: Bruno
journey: J-24
expected: `/jobs` and `/triggers` render as ListingPage catalogs (PageHead + ListingToolbar search/filters/view + rows/cards) instead of SplitPane master-detail; row click opens `/jobs/$jobId` or `/triggers/$triggerId` with breadcrumb parent link; Create CTA stays in topbar actions; `?create=loop&loop=` from Loop detail still opens the create editor seeded at that Loop; dynamic Edit/Delete/Run now live in topbar actions on detail.
entry_points: web `/jobs`; web `/triggers`; Loop detail Add schedule/trigger CTAs
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: TA-052; TA-056; TA-automation-crud-loop-target; LP-033
---

Added by Route Chrome + catalog migration (2026-07-17). Selection moved from local state to detail child routes.
