---
status: resolved
file: packages/site/components/docs/mermaid.tsx
line: 34
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:ffe909a18f58
review_hash: ffe909a18f58
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 016: Secure the SVG injection with additional sanitization for defense-in-depth.
## Review Comment

The code injects Mermaid-rendered SVG via `dangerouslySetInnerHTML` (line 56). While `securityLevel: "strict"` is configured, it does not fully mitigate XSS risks—security advisories show past vulnerabilities even in strict mode. Although current usage involves static documentation, the component accepts untrusted input structurally. Add DOMPurify sanitization before injection as a defense-in-depth measure:

## Triage

- Decision: `invalid`
- Notes:
  - The component renders repository-authored MDX diagrams only, not user-supplied Mermaid input, and Mermaid is already initialized with `securityLevel: "strict"`.
  - Adding a second sanitizer here would require extra client-side dependency surface and can strip legitimate SVG features without a concrete exploit in the current trust model.
  - If this component ever accepts untrusted diagrams at runtime, the right fix should revisit the trust boundary explicitly instead of layering a speculative sanitizer into this batch.
