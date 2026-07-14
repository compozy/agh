---
id: NB-execution-participation-defaults
area: NB
title: Default execution owners to Local participation
persona: Ada
journey: J-23
expected: A plain session create, task run, Loop run, or task-backed automation fire persists one immutable `Local`/`built_in_local` snapshot, creates no Network channel or wake, exposes no Network prompt, environment, or coordination tools, and records zero Network usage. Spawn, review, and detached child sessions resolve independently and never inherit a parent conversation.
entry_points: Web task and automation forms; HTTP/UDS session, task, Loop, and automation create/start surfaces; Network channel catalog
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-006;TA-001;TA-004;TA-052
---

Planning flag for execution-owner participation. The next targeted QA cycle should compare the channel catalog and Network usage before and after each plain create/start path, inspect the persisted owner projection and provider prompt/environment/toolset, and prove that a child session remains Local even when its parent is Live.

The scenario does not cover authoring an explicit typed participation request or workspace-coordination controls; those public management surfaces are completed by the later Network contract tasks.
