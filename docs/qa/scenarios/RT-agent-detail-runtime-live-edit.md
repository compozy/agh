---
id: RT-agent-detail-runtime-live-edit
area: RT
title: Agent detail live runtime selector mutation
persona: Bruno
journey: J-31
expected: The body DetailHeader Provider · Model · Reasoning selector submits a version-aware agent update immediately; pending disables without collapsing; conflict and failure keep server truth on the closed trigger and surface recoverable errors; workspace scope is preserved.
entry_points: web /agents/$name; PUT /api/agents/:name
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: RT-076;RT-078
---

Added by agent-details remediation 2026-07-12 for the new live runtime control on the detail header.
