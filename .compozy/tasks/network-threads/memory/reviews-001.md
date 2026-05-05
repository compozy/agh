# Task Memory: reviews-001

## Objective Snapshot

- Open CodeRabbit review round 001 for `.compozy/tasks/network-threads` Phase D.
- Convert CodeRabbit agent-readable output into `.compozy/tasks/network-threads/reviews-001/` using the cy-codex-loop conversion helper.

## Important Decisions

- Full `coderabbit review --agent` failed because CodeRabbit counted 454 PR files, over its 150-file limit.
- `coderabbit review --agent --type uncommitted` at repo root failed because CodeRabbit counted 170 files, still over the 150-file limit.
- The round was split into CodeRabbit-supported changed-code scopes:
  - `internal`: completed, 0 findings.
  - `web`: completed, 1 finding.
- `.compozy/tasks/network-threads/qa` exceeded the limit because of persisted QA artifacts, and narrower QA/memory/test-plan/test-case/bug-report scopes were ignored by CodeRabbit. Raw outputs are persisted under `reviews-001/raw/`.

## Learnings

- CodeRabbit CLI returns NDJSON events for `--agent`; for this round, the NDJSON was normalized into a single JSON object with a `review.findings[]` array before running `coderabbit-to-rounds.py`.
- CodeRabbit’s file-count limit can require scope splitting when QA evidence and task artifacts are part of the worktree.

## Files / Surfaces

- `.compozy/tasks/network-threads/reviews-001/issue_001.md`
- `.compozy/tasks/network-threads/reviews-001/raw/coderabbit-combined.json`
- `.compozy/tasks/network-threads/reviews-001/raw/coderabbit-internal.ndjson`
- `.compozy/tasks/network-threads/reviews-001/raw/coderabbit-web.ndjson`
- `.compozy/tasks/network-threads/reviews-001/raw/coderabbit-all.ndjson`
- `.compozy/tasks/network-threads/reviews-001/raw/coderabbit-compozy-qa.ndjson`

## Errors / Corrections

- Full and repo-root uncommitted CodeRabbit attempts were blocked by CodeRabbit service limits, not by repository test failures.
- QA artifact directories were not converted into review issues because CodeRabbit reported either too many files or all files ignored.

## Ready for Next Run

- Round 001 is closed clean. Next loop invocation should open CodeRabbit round 002.

## Round Summary

- Round directory: `.compozy/tasks/network-threads/reviews-001/`.
- Issues created: 1.
- Critical/high issues: none identified in the converted CodeRabbit output.
- D.2 triage result: `issue_001.md` was valid against `COPY.md` because the missing-thread UI only needed to state the failed resource and the available operator action. The `ThreadOverlay` error description now says `Could not load thread ${threadId}. Choose an existing thread from #${channel}.`
- Verification: `make verify 2>&1 | tee .compozy/tasks/network-threads/reviews-001/verify-after-fix.log` exited 0. Bun lint reported `Found 0 warnings and 0 errors`; Vitest reported `355 passed` files and `2223 passed` tests; Web build completed; Go tests reported `DONE 8401 tests`; boundaries reported `OK: all package boundaries respected`.
- Clean check: `check-rounds-clean.py .compozy/tasks/network-threads/reviews-001` reported `clean=true critical=0 high=0 total=1`.
- Next phase should be D.1 / `coderabbit_round` for round 002.
