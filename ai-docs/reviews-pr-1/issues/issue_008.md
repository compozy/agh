# Issue 8

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - MINOR
**File:** - `internal/skills/verify.go:44`
**Thread ID:** `PRRT_kwDOR5y4QM55CB-A`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610638

## Summary

Tighten the `you are now` verification regex to reduce false positives.

## Reviewer Comment

CodeRabbit reported that the current regex can catch benign instructions such as “you are now ready”.

## Triage Notes

`VALID`: The current `(?i)\\byou\\s+are\\s+now\\b` pattern matches benign phrases such as “you are now ready to proceed,” which can incorrectly flag normal skill instructions as critical prompt hijacks.

## Resolution

Tightened the verification regex to require role-like continuations and added `TestVerifyContentDoesNotFlagBenignYouAreNowPhrases` in `internal/skills/verify_test.go`.
