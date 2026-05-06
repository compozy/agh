# TC-INT-004: Extractor Failure Recovery Uses Audit Artifacts

**Priority:** P1
**Status:** Not Run

## Preconditions

- Extractor is enabled and can emit failures.
- Operator can inspect _system/extractor/failures and ledger.jsonl artifacts.

## Steps

1. Force one extractor failure into the dead-letter queue.
2. List pending failures.
3. Replay the failure and then drain the extractor.

**Expected:** Failures land under _system/extractor/failures, replay and drain report truthful counts, and the session ledger.jsonl captures the recovery path.

## Required Evidence

- Pending failure payload.
- Replay payload.
- Drain payload.
- File-system or API proof for _system/extractor/failures and ledger.jsonl.
