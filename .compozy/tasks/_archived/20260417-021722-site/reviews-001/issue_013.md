---
status: resolved
file: packages/site/app/protocol/[[...slug]]/page.tsx
line: 45
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:afe7254ef24e
review_hash: afe7254ef24e
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 013: Consider adding the same empty-slug check in generateMetadata.
## Review Comment

The `Page` component redirects when `slug` is empty, but `generateMetadata` is invoked independently by Next.js and might receive an empty slug. If `protocolDocs.getPage([])` returns `undefined`, `notFound()` is called, which works but could be made consistent with the redirect behavior.

## Triage

- Decision: `invalid`
- Notes:
  - I did not find a user-visible bug here: the page component already redirects empty protocol slugs to `/protocol/overview/`, which is the authoritative route behavior.
  - `generateMetadata()` using `notFound()` for a missing page is a safe standalone fallback if Next invokes metadata resolution independently of the page redirect.
  - Adding redirect logic to the metadata path would increase branching without changing the actual route experience, so no code change is needed.
