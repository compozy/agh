---
id: ET-web-marketplace-installed-management
area: ET
title: Manage every installed Marketplace kind
persona: Bruno
journey: J-marketplace-acquisition
expected: Each kind's Installed scope exposes only daemon-backed controls: skill content, shadows, enable and update; extension enable, environment, diagnostics and provenance; MCP configuration, status and authorization; bundle scope, profile, update and deactivation.
entry_points: /marketplace/skills?tab=installed; /marketplace/mcps?tab=installed; /marketplace/extensions?tab=installed; /marketplace/bundles?tab=installed
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-web-extensions-manage; ET-web-extension-detail; ET-web-mcp-status-matrix
---

Added by the unified Marketplace hard cut. Use one installed item of each kind, including a skill
managed by a bundle, and confirm detail refresh preserves the same runtime truth.
