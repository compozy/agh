---
id: NB-bridge-restart-recovery
area: NB
title: Recover unfinished bridge delivery after restart
persona: Omar
journey: J-recover-mid-turn-restart
expected: After a daemon restart, every durable active bridge delivery is reconciled within its exact scope and workspace before new prompt or delivery registration side effects; the channel receives a visible standard terminal error post without replaying persisted text, even for append-only providers or a stale remote anchor, and durable metrics survive the restart.
entry_points: Daemon restart during public bridge response delivery; bridge channel; delivery health metrics
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-bridge-tool-progress; NB-long-bridge-replies; NB-provider-progress-rendering
---

An operator or teammate sees an unfinished streamed response terminate visibly after a daemon
restart instead of being silently abandoned.

Added by the Hermes bridge Task 06 impact flag. Task 09 assigned it to `J-recover-mid-turn-restart` and `CH-mid-turn-bridge-restart`; Task 10 owns execution. Planning flag only; no QA session ran.
