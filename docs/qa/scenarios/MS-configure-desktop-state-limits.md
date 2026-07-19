---
id: MS-configure-desktop-state-limits
area: MS
title: Configure durable desktop-state limits
persona: Bruno
journey: J-administer-desktop-state-limits
expected: Valid global desktop_state limits load and bound atomic workspace writes across restart; workspace-scoped writes are rejected; invalid ranges name the exact path and never replace durable state.
entry_points: global config.toml; agh config set desktop_state.max_value_kib --scope global; agh config set desktop_state.max_keys_per_workspace --scope global; rejected workspace config.toml/write attempts; daemon restart
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence:
last_report:
overlaps:
---

Added by the Task 01 QA impact flag. This scenario remains untested until the public desktop-state wiring lands and a targeted QA cycle walks the full configuration-to-runtime journey.
