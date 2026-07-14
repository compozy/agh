---
id: NB-participation-controls-serialize
area: NB
title: Session/task/automation participation controls serialize Local by default
persona: Bruno
journey: J-network-local-default
expected: Session create, task editor, and automation task drafts serialize network_participation with Local default and never include legacy participation channel/network_channel/coordination_channel_id fields on those create payloads.
entry_points: web session create, task editor, Loop run, and automation job/trigger drafts; HTTP/UDS/CLI/native owner create/edit/start verbs
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
