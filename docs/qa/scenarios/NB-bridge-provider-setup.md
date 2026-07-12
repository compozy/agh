---
id: NB-bridge-provider-setup
area: NB
title: Set up, verify, and test bridge providers
persona: Operator
journey:
expected: Public AGH surfaces generate a schema-valid Slack manifest, accept interactive or strict JSON setup without exposing secrets, reject credential-bearing upstream endpoints from provider configuration, return actionable verification records without changing lifecycle state, register Telegram webhooks, send one real provider test message, and keep test-delivery dry-run only.
entry_points: agh bridge manifest; agh bridge setup; agh bridge verify; agh bridge send-test; agh doctor --only bridge; HTTP and UDS bridge manifest, verify, webhook-register, and send-test routes
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-025; NB-bridge-tool-progress; NB-long-bridge-replies; NB-provider-progress-rendering
---

An operator or agent configures a bridge through public AGH surfaces, catches credential and webhook mistakes before enablement, and proves real delivery.

Added by the Hermes bridge Task 04 impact flag. Provider-specific persona, journey, and charter expansion belongs to Task 09; Task 10 owns execution. Planning flag only; no QA session ran.
