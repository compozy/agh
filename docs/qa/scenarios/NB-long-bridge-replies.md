---
id: NB-long-bridge-replies
area: NB
title: Deliver long bridge replies safely
persona: Omar
journey: J-deliver-long-formatted-reply
expected: A terminal reply above the platform cap is delivered as ordered numbered messages within each provider limit, edit-capable cumulative previews remain one mutable message until terminal continuations materialize, platform markup stays readable, and the delivery acknowledgement points to the final remote message.
entry_points: Public bridge delivery through Slack; Telegram; Discord; Teams; Google Chat; WhatsApp adapters
qa_status: pass
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence: /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/notes/bridge-charter-results.json
last_report: docs/qa/reports/2026-07-12-hermes-bridge.md
overlaps: NB-039; NB-bridge-tool-progress
---

An operator or teammate receives a complete long agent reply without provider rejection, duplicate streaming continuations, or broken platform markup.

Added by the Hermes bridge Task 02 impact flag. Task 09 assigned it to `J-deliver-long-formatted-reply` and `CH-long-provider-replies`; Task 10 owns execution. Planning flag only; no QA session ran.

QA 2026-07-13: the adversarial shared corpus passed all six provider wire owners under race with provider-specific units, ordered continuations, fence repair, dialect handling, and terminal acknowledgements. This is a qualified deterministic-provider verdict.

Provider limits covered by this scenario are Slack 40,000 Unicode code points, Telegram 4,096 UTF-16 code units, Discord 2,000 Unicode code points, Teams 28,000 Unicode code points, Google Chat 32,000 UTF-8 bytes, and WhatsApp 4,096 Unicode code points.
