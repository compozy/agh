# Issue 7

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - MAJOR
**File:** - `internal/skills/registry.go:252`
**Thread ID:** `PRRT_kwDOR5y4QM55CB9_`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610635

## Summary

Run bundled skills through the same verification gate as directory-backed skills.

## Reviewer Comment

CodeRabbit reported that bundled skills are overlaid without the critical-content verification used for other sources.

## Triage Notes

`VALID`: `loadBundledSkills` currently bypasses the verification gate that `loadSkillPaths` applies to user, agent, and workspace skills. That means a bundled skill with critical content would still load, making verification behavior depend on source instead of content.

## Resolution

Applied the same verification gate to bundled skills before overlaying them into the registry. Added `TestRegistryVerifyContentBlocksCriticalBundledSkills` in `internal/skills/registry_test.go`.
