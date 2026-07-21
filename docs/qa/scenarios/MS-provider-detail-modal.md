---
id: MS-provider-detail-modal
area: MS
title: Provider detail opens as a centered modal with overlay dismissal
persona: Operator
journey:
expected: In Settings → Providers, opening a provider presents a centered modal (width 720px token) with a "Provider" eyebrow header, Overview | Configure lane tabs mapping to inspect/edit, and Cancel / Save provider footer. Clicking the overlay or pressing Esc dismisses it; Configure tab seeds the edit draft and Overview returns to inspect without saving.
entry_points: web Settings window → Providers → row/card click
qa_status: untested
bug_ids:
fix_status:
retest_status:
fix_commits:
evidence: web/src/systems/settings/components/provider-detail-dialog.tsx
last_report:
---

## Steps

1. Open the Settings window → Providers.
2. Click a provider row: a centered modal opens (not a side sheet).
3. Switch to Configure — the edit form appears; switch back to Overview — inspect view returns.
4. Click the scrim outside the modal — it closes without saving.
5. Reopen, press Esc — it closes.
