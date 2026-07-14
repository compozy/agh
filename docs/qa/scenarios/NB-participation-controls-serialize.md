---
id: NB-participation-controls-serialize
area: NB
title: Session/task/automation participation controls serialize Local by default
persona: Ada
journey: J-23
expected: Session create, task editor, and automation task drafts serialize network_participation with Local default and never include legacy participation channel/network_channel/coordination_channel_id fields on those create payloads.
entry_points: Session create dialog; task editor; automation job drafts; HTTP/UDS create
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-execution-participation-defaults
---

Planning flag for Task 05 participation controls co-ship.
