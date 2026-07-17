---
id: ET-cli-marketplace-search
area: ET
title: Search the marketplace through structured CLI output
persona: Ada
journey: J-agent-marketplace-parity
expected: `agh marketplace search` returns fixed-order grouped JSON for all kinds, supports `--kind` browse and `jsonl`, and preserves truthful installed and update fields from the daemon.
entry_points: agh marketplace search [query] -o json; agh marketplace search [query] --kind <kind> -o json; agh marketplace search [query] -o jsonl; agh.network/runtime/core/marketplace (guide)
qa_status: pass
bug_ids: BUG-20260715-native-marketplace-extension-parity
fix_status: fixed
retest_status: pass
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-marketplace-task11-final-20260715-20260716-011529-818379-lab/qa-artifacts/qa/notes/marketplace-agent-parity-final.json
last_report: docs/qa/reports/2026-07-15-marketplace.md
overlaps: ET-007; ET-016
---

Added by marketplace Task 02 after the hard cut to one discovery namespace. The next agent-surface QA cycle should compare CLI JSON byte semantics with HTTP and UDS for the same daemon state, including one isolated kind failure.
