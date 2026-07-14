---
id: RT-session-message-reload
area: RT
title: Authored session messages survive reconciliation and reload
persona: Théo
journey: J-11
expected: Ordinary prompts and structured slash commands render exactly once in authoritative server chronology before the work they initiate; live SSE reconciliation and a cold permalink reload preserve the exact authored text without duplication, movement, or loss.
entry_points: web session thread; POST session prompt; transcript REST and SSE
qa_status: pass
bug_ids: BUG-20260713-session-user-message-reorders-or-disappears
fix_status: fixed
retest_status: pass
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-automation-features-post-onboarding-fix-20260713-20260713-203513-816377-lab/qa-artifacts/qa/screenshots/session-user-message-reload-fixed.json
last_report: docs/qa/reports/2026-07-13-automation-features.md
overlaps: RT-045, RT-058, TA-089
---

story: As an operator I can trust that every message I authored remains in its original place when live agent output arrives and when I reload the session later.

The 2026-07-13 live replay first reproduced duplicate/reordered ordinary prompts and a `/goal` command that disappeared after reload. The same-persona post-fix replay used a fresh Cursor/Grok 4.5 session, completed two ordinary turns plus a two-turn approved Goal, then reloaded the exact permalink. All three authored inputs remained present exactly once and in strict request/response chronology.

2026-07-14 explicit retest after the final daemon rebuild: the same permalink reloaded in 874 ms. Both ordinary prompts and the exact `/goal` input remained present exactly once and kept their original order (`orderPreserved=true`, `allExactlyOnce=true`).
