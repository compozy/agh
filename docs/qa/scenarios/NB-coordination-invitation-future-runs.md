---
id: NB-coordination-invitation-future-runs
area: NB
title: Coordination invitation accepts for future runs only
persona: Ada
journey: J-23
expected: On an active multi-agent run with coordination off and Network available, the invitation is visible, states that acceptance does not change the active run, accept enables workspace coordination for future runs, and dismiss persists via daemon invitation GET across reload.
entry_points: Task run detail; PUT /network-coordination; PUT /network-coordination/invitation; Web invitation card
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-execution-participation-defaults
---

Planning flag for Task 05 invitation UX. Next QA cycle should prove the visibility matrix, double-accept idempotency, daemon-backed dismiss, and future-runs-only copy.
