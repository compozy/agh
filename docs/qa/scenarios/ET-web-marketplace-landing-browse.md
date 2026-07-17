---
id: ET-web-marketplace-landing-browse
area: ET
title: Browse the grouped Web marketplace
persona: Bruno
journey: J-marketplace-acquisition
expected: The Marketplace landing page renders non-empty MCP, extension, and skill launch sections from one grouped request, exposes truthful installed/update actions, and remains keyboard operable at desktop and mobile widths; bundles remain derived from installed extensions.
entry_points: /marketplace; Marketplace sidebar item
qa_status: untested
bug_ids: BUG-20260714-keyboard-focus-invisible
fix_status: BUG-20260714-keyboard-focus-invisible fixed
retest_status: pending full-identity concurrent action state and extension Update routing
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-under-minute.json; /Users/pedronauck/Dev/compozy/agh/.tmp/bug-20260714-focus/focused.png
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-api-marketplace-namespace; ET-web-marketplace-search-fanout
---

Added by marketplace Task 06. The next Web QA cycle should compare the landing against VC01, VC02, VC03, and VC06, including installed and update states with an active workspace.

QA impact 2026-07-16: pending actions now use `(kind, entry_id)` with independent overlap counts,
and installed extension Update uses the lifecycle PUT instead of the install POST; reset for the next
browser QA cycle.

QA impact 2026-07-16: a fresh default catalog now exposes Context7, Repository Orientation, and
Documentation Writer without a search query; verify their cards and exact detail routes before any
install state exists.
