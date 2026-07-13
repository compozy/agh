---
id: NB-bridge-provider-setup
area: NB
title: Set up, verify, and test bridge providers
persona: Tessa
journey: J-connect-bridge-provider
expected: Public AGH surfaces generate a schema-valid Slack manifest, accept interactive or strict JSON setup without exposing secrets, reject credential-bearing upstream endpoints from provider configuration, return actionable verification records without changing lifecycle state, register Telegram webhooks, send one real provider test message, and keep test-delivery dry-run only.
entry_points: agh bridge manifest; agh bridge setup; agh bridge verify; agh bridge send-test; agh doctor --only bridge; HTTP and UDS bridge manifest, verify, webhook-register, and send-test routes
qa_status: blocked-decision
bug_ids: BUG-20260713-telegram-route-shapes
fix_status: deferred
retest_status:
fix_commits:
evidence: /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/notes/bridge-charter-results.json; /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/issues/BUG-0039.md
last_report: docs/qa/reports/2026-07-12-hermes-bridge.md
overlaps: NB-025; NB-bridge-tool-progress; NB-long-bridge-replies; NB-provider-progress-rendering
---

An operator or agent configures a bridge through public AGH surfaces, catches credential and webhook mistakes before enablement, and proves real delivery.

Added by the Hermes bridge Task 04 impact flag. Task 09 assigned it to `J-connect-bridge-provider` and the provider setup charters; Task 10 owns execution. Planning flag only; no QA session ran.

QA 2026-07-13: WhatsApp/Discord validation and Slack/Telegram setup/control surfaces worked, but guided Telegram routing requires group plus thread and rejects documented direct-message and ordinary-group shapes. `BUG-20260713-telegram-route-shapes` requires a structural alternative-route decision.
