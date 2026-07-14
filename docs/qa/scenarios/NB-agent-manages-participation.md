---
id: NB-agent-manages-participation
area: NB
title: Manage participation through structured agent surfaces
persona: Ada
journey: J-run-bounded-live-collaboration
expected: CLI, HTTP, UDS, and native tools expose the same immutable mode, source, channel, finite bounds, consumption, and actual-or-unavailable usage; invalid or unauthorized requests return stable named diagnostics without partial execution state.
entry_points: agh session/task/loop/network commands -o json; HTTP/UDS execution create/start and Network status/usage routes; agh__network_* and owner native tools; GET /api/agent/context
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: NB-execution-participation-defaults;NB-participation-controls-serialize;RT-031;TA-049
---

Derived from US-019 and the Live flow. The structured surfaces must agree on `network_participation_unavailable`, `not_participating`, `loop_requires_live`, unknown channel, unsupported Live, authority denial, and invalid-combination behavior.

A Local agent must receive an explicit `not_participating` explanation rather than fictional Network context or silently missing controls; a child cannot widen beyond delegated authority.
