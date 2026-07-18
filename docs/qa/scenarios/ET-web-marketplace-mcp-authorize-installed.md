---
id: ET-web-marketplace-mcp-authorize-installed
area: ET
title: Authorize an MCP server from Installed scope
persona: Bruno
journey: J-mcp-authorize-repair
expected: An installed OAuth MCP server that needs login exposes Authorize in the MCP Installed scope and detail, reuses the scoped daemon authorization flow, and reports success only after authenticated status and token presence are both confirmed.
entry_points: /marketplace/mcps?tab=installed; /marketplace/mcp/<entry-id>
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-mcp-authorize; ET-web-mcp-authorize-manual; ET-web-mcp-status-matrix
---

Added by the unified Marketplace hard cut. Cover global and active-workspace definitions without
exposing OAuth codes, tokens, PKCE verifiers, or bound secret references.
