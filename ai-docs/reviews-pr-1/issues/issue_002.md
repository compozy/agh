# Issue 2

**Status:** - [x] RESOLVED
**Disposition:** - INVALID
**Severity:** - TRIVIAL
**File:** - `go.mod:13`
**Thread ID:** `PRRT_kwDOR5y4QM55CB9h`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610591

## Summary

Evaluate whether the project should consolidate its two YAML libraries into one.

## Reviewer Comment

CodeRabbit noted that `gopkg.in/yaml.v3` and `github.com/goccy/go-yaml` overlap and suggested consolidating them if worthwhile.

## Triage Notes

`INVALID`: The two YAML libraries are serving different concrete needs today. `gopkg.in/yaml.v3` is used for `yaml.Node` decoding plus unknown-field inspection in the skills loader, while `github.com/goccy/go-yaml` is already used for strict unmarshaling in memory metadata parsing. Consolidating them would require a broader refactor without fixing a demonstrated bug in this PR.

## Resolution

No code change. The thread was resolved with rationale because the current split reflects two distinct APIs already in use.
