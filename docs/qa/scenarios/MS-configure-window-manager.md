---
id: MS-configure-window-manager
area: MS
title: Configure window behavior and declarative layouts safely
persona: Bruno
journey: J-administer-window-manager
expected: General, Snapping, Layouts, Shortcuts, and Advanced settings expose every supported `[window_manager]` value; invalid ranges, duplicate ratios, conflicting shortcuts, and unsupported bindings identify the exact path and preserve the active known-good configuration; a valid save hot-applies to the next command without restarting; workspace layout overrides remain isolated.
entry_points: Settings window management; global config.toml; agh config get|set|apply; layout editor
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps: ET-window-manager-layout-recovery; ET-window-manager-layout-gestures
---

story: As an operator, I can tune window behavior and layouts without accepting a partial or internally conflicting runtime configuration.

qa-impact: 2026-07-22 replaced storage-limit settings with validated behavior defaults, shortcuts, bindings, gaps, snap thresholds, and declarative layout editing. Flag only; the next QA cycle owns live retesting.
