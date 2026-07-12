---
id: NB-provider-progress-rendering
area: NB
title: Render live tool progress across bridge providers
persona: Operator
journey:
expected: Slack, Telegram, and Discord render default-on tool progress as one editable channel bubble with typing and reaction affordances; opted-in Teams and Google Chat update one status; WhatsApp emits sparse append-only statuses; the final answer stays separate; disabled progress makes no platform call; and GitHub and Linear acknowledge progress without writing to issues.
entry_points: Public bridge turns through Slack; Telegram; Discord; Teams; Google Chat; WhatsApp; GitHub; Linear adapters; per-instance delivery_defaults.progress
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-bridge-tool-progress; NB-long-bridge-replies
---

An operator or teammate sees current tool activity in the channel without confusing it with the final answer or issue content.

Added by the Hermes bridge Task 03 impact flag. Provider-specific persona, journey, and charter expansion belongs to Task 09; Task 10 owns execution. Planning flag only; no QA session ran.
