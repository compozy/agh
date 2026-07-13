---
id: NB-web-bridge-setup
area: NB
title: Complete bridge setup in the Web
persona: Tessa
journey: J-complete-web-bridge-setup
expected: A browser-first operator can create a disabled Slack bridge, copy the generated manifest, follow daemon-derived setup state and remediation, register Telegram webhooks, and distinguish dry-run target checks from real test messages.
entry_points: Web bridges create dialog; Web bridge detail panel
qa_status: pass
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence: /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/notes/bridge-charter-results.json; /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/screenshots/ch055-create-handoff.png; /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/screenshots/ch055-failed-remediation.png; /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/screenshots/ch055-send-result.png
last_report: docs/qa/reports/2026-07-12-hermes-bridge.md
overlaps: NB-bridge-provider-setup; NB-026; NB-027; NB-036; NB-037; NB-038; NB-039
---

A first-time adopter completes provider setup through the Web without losing the distinction between
configuration checks and real delivery.

Added by the Hermes bridge Task 05 impact flag. Task 09 assigned it to `J-complete-web-bridge-setup` and `CH-web-bridge-setup`; Task 10 owns execution. Planning flag only; no QA session ran.

QA 2026-07-13: browser create, daemon manifest copy, bindings, inline failed-check remediation, refresh, dry-run, and real-send all completed; API/CLI/provider readbacks confirmed durable truth.
