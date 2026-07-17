---
id: ET-web-mcp-status-matrix
area: ET
title: MCP management status matrix renders daemon truth
persona: Bruno
journey: J-mcp-authorize-repair
expected: The `/mcp` matrix renders `configured × auth_status.status × runtime_status.{state,probe,tool_count}` as four independent signals with truthful tones (success/warning/danger/neutral). No hardcoded success dot and no hardcoded connected topbar. Tool count shows only on a succeeded probe; `probe=skipped` is never a failure. Unknown daemon values render neutral preserving the diagnostic. Authorize/Reauthorize is offered only to OAuth http/sse remotes not authenticated (or auth_refresh_failed); stdio and non-OAuth remotes never get it.
entry_points: web `/mcp?scope=workspace|global`; `GET /api/settings/mcp-servers`
qa_status: untested
bug_ids:
fix_status:
retest_status: pending workspace-guard action gating
fix_commits:
evidence: web/src/systems/settings/lib/mcp-status-view-model.ts; web/src/systems/settings/components/mcp-servers-table.tsx; .compozy/tasks/marketplace/evidence/visual/task-08/matrix-desktop; /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/mcp-oauth-name-segment.json; /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/web/mcp-oauth-confirmed.png
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: MS-029
---

story: As an operator I read every configured MCP server's real configuration, authorization, runtime, and probe status at a glance without a plausible green masking a broken server.

src: docs/design/opendesign/mcp-management.html

inventory: Needs QA

QA impact 2026-07-15: new behavior from Task 08 (ADR-006). The `/mcp` page replaced the hardcoded success dot with the composed status matrix. Flagged untested for the next QA cycle.

QA impact 2026-07-16: the Add server action is now absent while the workspace guard cannot
resolve a valid scope, eliminating the previous visible-but-inert control.
