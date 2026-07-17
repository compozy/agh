---
id: ET-web-catalog-navigation
area: ET
title: Navigate the Catalog information architecture
persona: Bruno
journey: J-marketplace-acquisition
expected: The sidebar Catalog group appears in the exact Marketplace, Extensions, Bridges, Skills, MCP, Knowledge order and System contains only Sandbox, Vault, Settings, with child routes preserving the correct active item.
entry_points: web app sidebar; Catalog and System destinations
qa_status: untested
bug_ids:
fix_status:
retest_status: pending Marketplace nested-route active indicator
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-under-minute.json
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-web-marketplace-landing-browse; ET-web-extensions-manage
---

Added by marketplace Task 07. Verify the exact D13 ordering without count badges.

QA impact 2026-07-16: Marketplace now uses fuzzy route matching so kind and entry detail routes
retain the sidebar active indicator.
