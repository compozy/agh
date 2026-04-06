# Issue 5

**Status:** - [x] RESOLVED
**Disposition:** - VALID
**Severity:** - MAJOR
**File:** - `internal/skills/catalog.go:21`
**Thread ID:** `PRRT_kwDOR5y4QM55CB9v`
**Comment URL:** - https://github.com/compozy/agh/pull/1#discussion_r3039610612

## Summary

Escape double quotes when writing skill names into XML attributes.

## Reviewer Comment

CodeRabbit reported that `entry.name` is emitted into `name="..."` without quote escaping, which can corrupt the catalog output.

## Triage Notes

`VALID`: `BuildCatalog` writes `entry.name` into `name="..."` but only escapes `&`, `<`, and `>`. A skill name containing `"` can break the generated attribute and corrupt the prompt catalog structure.

## Resolution

Split catalog escaping into text and attribute-specific replacers so quoted skill names are encoded safely in `name="..."`. Updated `TestBuildCatalogFormatsCatalogSortedEscapedAndWithUsageInstructions` to cover embedded quotes.
