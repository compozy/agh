---
id: ET-web-extensions-manage
area: ET
title: Manage installed extensions and active bundles
persona: Bruno
journey: J-marketplace-acquisition
expected: The Extensions page lists daemon-owned installed extensions and bundle activations, applies enable changes immediately, derives extension Update from the catalog join and bundle Update from spec_drift, blocks removal while a bundle is active, then permits typed removal after deactivation.
entry_points: /extensions; /extensions?tab=bundles; Marketplace Manage actions
qa_status: untested
bug_ids:
fix_status:
retest_status: pending exact catalog-entry enrichment and cross-owner cache reconciliation
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-management-lifecycle.json; /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/web/marketplace-extension-web-management-final.png
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-019; ET-020; ET-021; ET-027; ET-028
---

Added by marketplace Task 07. Exercise the complete toggle, update, typed-remove conflict, deactivate, and successful removal sequence against one real daemon.

QA impact 2026-07-16: installed extension description and Update truth now come from the API's exact
`provenance.catalog_entry_id` join rather than a capped Marketplace browse page; reset for the next QA cycle.

QA impact 2026-07-16: update and removal now invalidate both installed inventory and Marketplace
discovery so action labels and update badges reconcile together.

QA impact 2026-07-16: removal now fails closed while bundle dependency activity is loading or
failed, exposes Retry, and proceeds only after a successful dependency response; malformed 2xx
extension envelopes surface as request errors instead of incomplete inventory data.
