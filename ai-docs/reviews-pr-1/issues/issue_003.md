# Issue 3

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - CRITICAL
**File:** - `internal/cli/skill.go:524`
**Thread ID:** `PRRT_kwDOR5y4QM55CB9o`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610600

## Summary

Ensure `skill view --file` resolves symlinks before enforcing the skill-directory boundary.

## Reviewer Comment

CodeRabbit reported that the current lexical boundary check can be bypassed by a symlink inside the skill directory that points outside the workspace.

## Triage Notes

`VALID`: The current filesystem check compares lexical paths only. `os.ReadFile(absTarget)` will follow a symlink placed inside the skill directory, so a symlinked resource can escape the intended boundary despite the `filepath.Rel` check.

## Resolution

Updated `internal/cli/skill.go` to resolve both the skill root and target through `filepath.EvalSymlinks` before the boundary check and before reading the file. Added `TestSkillViewCommandRejectsSymlinkEscape` in `internal/cli/skill_test.go`.
