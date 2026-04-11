---
status: resolved
file: docs/ideas/anp/index.html
line: 227
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:2ea11244ff5d
review_hash: 2ea11244ff5d
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 003: Accepted upload types and parser behavior are inconsistent.
## Review Comment

Line 227 accepts `.json`, but `parseJsonl` (Line 269 onward) only handles JSONL lines. Regular JSON files will fail with a parse error. Either narrow accepted types or add JSON-document fallback parsing.

Also applies to: 269-280

## Triage

- Decision: `valid`
- Notes: The file picker explicitly accepts `.json`, but the parser only supports JSONL lines today. I will make the loader accept both JSONL and regular JSON documents so the UI matches the advertised upload contract.
