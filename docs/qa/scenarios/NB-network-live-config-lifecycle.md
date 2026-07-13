---
id: NB-network-live-config-lifecycle
area: NB
title: Manage Live participation configuration lifecycle
persona: Ada
journey: J-23
expected: Supported `[network.live.defaults]` and `[network.live.limits]` values survive reload and restart, while removed Network keys are rejected without changing active availability.
entry_points: config.toml; agh config set; agh config reload; agh status -o json
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps:
---

Planning flag for the Network participation hard cut. The next targeted QA cycle should walk valid duration and wake-budget updates, strict rejection of removed keys, and durable availability parity across CLI, HTTP, UDS, reload, and daemon restart.

Taxonomy note: this scenario owns the functional and error branches of the config lifecycle. UI responsiveness and visual consistency are deliberate skips because this change adds no Web control; adjacent Network UI behavior remains covered by `NB-003` and journey `J-23`.
