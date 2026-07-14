---
id: GL-judge-session-contract
area: GL
title: Constrain and clean up every Goal judge session
persona: Lea
journey: J-26
expected: Each agent command-judge attempt has a verdict-only capability boundary, captures one schema-valid verdict or one bounded typed failure, and releases its temporary session and process on success malformed output failure cancellation and replay.
entry_points: web Goal Run and agent session list; HTTP Goal turns; daemon provider lifecycle
qa_status: pass
bug_ids: BUG-20260713-goal-judge-unconstrained-leaks-session
fix_status: fixed
retest_status: pass
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-automation-features-20260713-20260713-044543-173594-lab/qa-artifacts/qa/screenshots/goal-lifecycle-blocked-after-three-turns.dom.txt;/Users/pedronauck/dev/qa-labs/agh-automation-features-20260713-20260713-044543-173594-lab/qa-artifacts/qa/screenshots/goal-judge-sessions-remain-active.dom.txt;/Users/pedronauck/dev/qa-labs/agh-automation-features-post-onboarding-fix-20260713-20260713-203513-816377-lab/qa-artifacts/qa/screenshots/catalog-global-goal-approved.png;/Users/pedronauck/dev/qa-labs/agh-automation-features-post-onboarding-fix-20260713-20260713-203513-816377-lab/qa-artifacts/qa/network/catalog-global-goal-acceptance.json
last_report: docs/qa/reports/2026-07-13-automation-features.md
overlaps: GL-004;GL-037
---

The live Cursor/Grok run proves the product boundary: Goal work sessions may use tools, but verdict-only judge sessions must not inherit that runtime authority or survive their one criterion.

Final retest: real Cursor/Grok judge `sess-284fdef67433e103` returned one strict JSON verdict with zero tool events and stopped after approving `looprun-a6a4368bf1fc8c49`. Active-judge Clear then canceled and joined `sess-3e07f85d0d2ac987` with no surviving system session or successor generation.
