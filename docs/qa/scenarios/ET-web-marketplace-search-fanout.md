---
id: ET-web-marketplace-search-fanout
area: ET
title: Search marketplace kinds with partial failure isolation
persona: Bruno
journey: J-marketplace-acquisition
expected: One search updates all kind sections; a failed kind owns its error strip while siblings remain usable, zero sections collapse, and an all-zero query offers one clear-search recovery state.
entry_points: /marketplace?q=<query>; Marketplace search field
qa_status: untested
bug_ids: BUG-20260714-keyboard-focus-invisible
fix_status: BUG-20260714-keyboard-focus-invisible fixed
retest_status: Retained sibling results and truthful stale/error isolation passed; keyboard focus passed the shared two-pixel contract
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/web/marketplace-skill-stale-served.png; /Users/pedronauck/Dev/compozy/agh/.tmp/bug-20260714-focus/focused.png
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-api-marketplace-namespace; ET-web-marketplace-landing-browse
---

Added by marketplace Task 06. Exercise VC04 and VC05 with a deterministic per-kind source failure and prove the browser issues one grouped request rather than four independent searches.

QA impact 2026-07-17: default listing view is now rows with optional cards; reset to confirm
partial-failure isolation still holds in both views.
