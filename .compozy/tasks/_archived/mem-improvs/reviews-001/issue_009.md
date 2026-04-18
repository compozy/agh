---
status: resolved
file: internal/memory/store.go
line: 880
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745a9,comment:PRRC_kwDOR5y4QM65BAQO
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`indexMatchesHeaders` is too weak to detect stale `MEMORY.md` content.**

This only validates that each linked filename appears once. If a document’s title or description changes, `LoadIndex` will still accept a stale on-disk index and return outdated prompt content as long as the filenames match.

<details>
<summary>Stricter comparison</summary>

```diff
 func indexMatchesHeaders(content string, headers []Header) bool {
-	if len(headers) == 0 {
-		return strings.TrimSpace(content) == ""
-	}
-	targets := make(map[string]struct{}, len(headers))
-	for _, header := range headers {
-		targets[header.Filename] = struct{}{}
-	}
-
-	seen := make(map[string]struct{}, len(headers))
-	for line := range strings.SplitSeq(content, "\n") {
-		target, ok := firstMarkdownLinkTarget(line)
-		if !ok {
-			continue
-		}
-		if _, exists := targets[target]; !exists {
-			return false
-		}
-		if _, exists := seen[target]; exists {
-			return false
-		}
-		seen[target] = struct{}{}
-	}
-
-	return len(seen) == len(targets)
+	return strings.TrimSpace(content) == strings.TrimSpace(renderIndex(headers))
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 856 - 880, indexMatchesHeaders
currently only verifies that each filename appears once, allowing stale
titles/descriptions to pass; update it to validate that the on-disk index
content exactly matches the Header metadata (not just filenames). Specifically,
while iterating lines using firstMarkdownLinkTarget, also extract the link text
(the markdown link label) and any adjacent description text and compare them
against Header.Title and Header.Description for the corresponding Header entry;
fail if any title/description differs, if order/occurrence mismatches, or if any
header is missing, so the function enforces a strict canonical match between
content and the slice of Header structs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `indexMatchesHeaders` only checks that each filename appears once, so stale titles, descriptions, or reordered lines can still be accepted as current `MEMORY.md` content.
  - I will tighten the comparison against the canonical rendered index and add a regression test that proves stale metadata gets synthesized instead of reused.

## Resolution

- Tightened `indexMatchesHeaders` to compare against the canonical rendered index content, not just linked filenames.
- Added regression coverage showing stale title/description content is rejected and synthesized from current file metadata.
