# BUG-002: ACP mock exact prompt matching broke with situation context augmentation

## Status
- Fixed

## Severity
- P0 QA blocker

## Affected Lane
- `make test-e2e-web`
- `web/e2e/automation.spec.ts`
- `web/e2e/session-onboarding.spec.ts`

## Evidence
- Failing log: `.compozy/tasks/autonomous/qa/logs/final-test-e2e-web.log`
- Narrow rerun after fix: `.compozy/tasks/autonomous/qa/logs/final-test-e2e-web-rerun-failing.log`
- Crash bundles in the failed daemon homes showed `acpmock: no turn matched` after `<agh-situation-context>` was prepended to the prompt.

## Root Cause
Task 04 intentionally adds live situation context to prompt dispatch. The deterministic ACP mock driver exact-matched fixture `user_text` against the raw dispatched prompt, so user turns such as `Review payload deploy for main` and `run browser lifecycle flow` no longer matched once the daemon prepended the bounded situation context block.

## Fix
- Updated `internal/testutil/acpmock` turn matching to compare fixture `user_text` against the canonical user message after stripping the daemon-owned `<agh-situation-context>...</agh-situation-context>` prefix.
- Added regression coverage proving exact user-text fixtures still match situation-augmented prompts.

## Verification
- `go test ./internal/testutil/acpmock -count=1`
- `bun run --cwd web typecheck`
- `AGH_E2E_QA_OUTPUT_DIR=../.compozy/tasks/autonomous bunx playwright test e2e/automation.spec.ts e2e/session-onboarding.spec.ts e2e/tasks.spec.ts --reporter=list`
