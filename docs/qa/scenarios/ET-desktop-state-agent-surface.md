---
id: ET-desktop-state-agent-surface
area: ET
title: Manage workspace desktop state through every public transport
persona: Ada
journey:
expected: CLI, HTTP, UDS, and WebSocket expose one ordered workspace-scoped desktop-state contract; deletion purges state and restart preserves live entries and revisions.
entry_points: agh desktop-state; /api/workspaces/{workspace_id}/desktop-state; /api/workspaces/{workspace_id}/desktop-state/stream
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: MS-configure-desktop-state-limits
---

story: As an agent, I can inspect, arrange, delete, and watch a workspace desktop without depending on the web UI.

scope: Exercise structured CLI output, HTTP/UDS body parity, WebSocket snapshot-plus-delta ordering, workspace isolation, purge-on-delete, and restart durability through a real daemon.

qa-impact: OS Shell Task 02 introduced the complete public desktop-state surface. Flag only; the next QA cycle owns live retesting.
