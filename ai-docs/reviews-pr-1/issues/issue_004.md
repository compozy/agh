# Issue 4

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - MAJOR
**File:** - `internal/cli/skill.go:612`
**Thread ID:** `PRRT_kwDOR5y4QM55CB9t`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610607

## Summary

Harden skill names before using them as directory names and YAML values.

## Reviewer Comment

CodeRabbit reported that `normalizeSkillName` still permits control characters and YAML-sensitive input that can make the scaffolded directory name diverge from the parsed `SKILL.md` name.

## Triage Notes

`VALID`: `normalizeSkillName` currently rejects path traversal but still accepts names containing spaces, colons, and control characters. The resulting value is reused as both a directory segment and an unquoted YAML scalar in `SKILL.md`, so malformed input can create a scaffold whose on-disk directory and parsed metadata do not match safely.

## Resolution

Restricted skill names to `[A-Za-z0-9._-]+` and changed the default template to quote the YAML `name` field. Expanded `TestSkillCreateCommandSupportsDefaultNameAndRejectsUnsafeNames` with YAML-sensitive and control-character cases.
