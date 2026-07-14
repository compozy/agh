---
id: NB-coordination-invitation-future-runs
area: NB
title: Coordination invitation accepts for future runs only
persona: Bruno
journey: J-enable-coordinated-conversations
expected: On an active multi-agent run with coordination off and Network available, the invitation is visible, states that acceptance does not change the active run, accept enables workspace coordination for future runs, and dismiss persists via daemon invitation GET across reload.
entry_points: web task run detail and kanban invitation; GET/PUT /api/workspaces/:id/network-coordination over HTTP/UDS; PUT /api/workspaces/:id/network-coordination/invitation; agh network coordination and invitation commands
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
