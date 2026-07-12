---
id: NB-bridge-edit-reply
area: NB
title: Preserve bridge edit and reply intent
persona: Bruno
journey:
expected: A supported Slack or Telegram message edit reaches the routed agent as a distinct edit with the affected message identity and replacement or deletion operation; Slack, Telegram, and Google Chat replies include bounded already-observed parent text and author when available, while a cache miss remains empty without a provider fetch or workspace, instance, or conversation bleed.
entry_points: Public Slack, Telegram, and Google Chat inbound bridge webhooks; routed session prompt
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-024; NB-bridge-tool-progress
---

An operator or teammate can correct a message or reply in context without the agent confusing
historical quoted text with the current instruction.

Added by the Hermes bridge Task 06 impact flag. Task 09 owns the durable persona, journey, and
charter mapping; Task 10 owns execution. Planning flag only; no QA session ran.
