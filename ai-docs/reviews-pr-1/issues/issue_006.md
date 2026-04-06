# Issue 6

**Status:** - [x] RESOLVED
**Disposition:** - INVALID
**Severity:** - TRIVIAL
**File:** - `internal/skills/loader.go:200`
**Thread ID:** `PRRT_kwDOR5y4QM55CB91`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610621

## Summary

Avoid unmarshaling skill frontmatter twice in `decodeSkillMeta`.

## Reviewer Comment

CodeRabbit suggested decoding from the existing `yaml.Node` instead of parsing the frontmatter twice.

## Triage Notes

`INVALID`: This is a micro-optimization, not a correctness defect. `decodeSkillMeta` is executed only while loading skill definitions, and the current double unmarshal is clear and already covered by existing parsing tests. There is no demonstrated bug or regression tied to this code path.

## Resolution

No code change. The thread was resolved with rationale because the comment identified an optional optimization, not a behavioral problem.
