---
id: NB-run-bounded-live-collaboration
area: NB
title: Run bounded Live collaboration without duplicate wakes
persona: Ada
journey: J-run-bounded-live-collaboration
expected: An explicitly Live execution durably accepts eligible direct messages and mentions, coalesces one causal burst, prompts the target once with untrusted Network context, settles actual usage or a truthful canceled/deadline outcome, recovers queued wakes after restart, and accumulates without prompting at depth or total-budget ceilings.
entry_points: HTTP/UDS/CLI/native execution start; Network thread and direct send; network usage and conversation reads; daemon restart
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-execution-participation-defaults;NB-020;RT-073
---

Planning flag for the Task 04 Live executor. The next targeted QA cycle should compare a Local control run with one explicit Live run, send a ten-message same-root burst plus a depth-capped reply, interrupt one wake through disable/cancel, restart with one admitted-but-unclaimed wake, and reconcile conversation, task-run, ledger detail, and aggregate usage after each branch.

Taxonomy note: the journey and scenario cover the functional happy path, realistic cancellation/restart/error branches, zero-activation Local control, workspace isolation, and agent-manageable structured surfaces. Visual responsiveness and browser interaction are deliberate skips because Task 04 changes no Web UI; later public-surface tasks own the visible controls and invitation experience.
