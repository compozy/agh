---
id: NB-run-conversation-bounds-usage
area: NB
title: Run detail shows conversation, bounds, and truthful usage
persona: Bruno
journey: J-enable-coordinated-conversations
expected: Coordinated run detail shows the participation chip, an empty conversation explaining silence when no messages exist, paginated history when messages exist, and workspace-scoped usage labeled actual or usage_unavailable with no fabricated totals.
entry_points: web task run detail conversation and usage panels; GET /api/network/usage over HTTP/UDS; agh network usage -o json; run conversation SSE and paginated history projections
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-run-bounded-live-collaboration
---

Planning flag for Task 05 run conversation/usage surfaces.
