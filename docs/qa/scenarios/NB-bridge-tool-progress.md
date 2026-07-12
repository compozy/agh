---
id: NB-bridge-tool-progress
area: NB
title: Observe safe bridge tool progress
persona: Operator
journey:
expected: With progress enabled, a tool-heavy bridged turn shows an ordered and redacted started-to-completed-or-failed lifecycle while the corresponding session transcript and ACP history contain no progress chrome.
entry_points: Bridge channel turn; public bridge delivery path; web session transcript; ACP history
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps:
---

An operator or teammate sees trustworthy, non-secret bridge progress without polluting the agent transcript.

Added by the Hermes bridge Task 01 impact flag. It covers canonical projection, redaction, ordered queue coalescing, terminal preservation, and transcript purity through public bridge delivery. Task 09 owns provider-specific expansion plus persona, journey, and charter mapping from `.compozy/tasks/hermes-bridge/_qa.md`.
