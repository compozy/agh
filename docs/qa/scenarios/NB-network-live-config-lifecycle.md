---
id: NB-network-live-config-lifecycle
area: NB
title: Manage Live participation configuration lifecycle
persona: Bruno
journey: J-administer-network-live
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

Taxonomy note: this scenario owns the config-file and structured lifecycle branches. Web Settings interaction is owned by `NB-002`; disable/re-enable semantics are owned by `NB-network-availability-toggle`.
