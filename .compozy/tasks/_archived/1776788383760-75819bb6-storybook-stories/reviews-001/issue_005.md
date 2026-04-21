---
status: resolved
file: web/src/components/ui/stories/combobox.stories.tsx
line: 30
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:64a7c90ddbc8
review_hash: 64a7c90ddbc8
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 005: Use an interface for the option object shape.
## Review Comment

This file defines an object shape with a `type` alias; the project standard is to use `interface` for object shapes in TypeScript.

## Triage

- Decision: `INVALID`
- Notes:
  - There is no repository instruction in scope that mandates `interface` over `type` for every local object shape.
  - `CityOption` is a small local alias used only inside this story file; changing it to an `interface` does not address a correctness, maintainability, or tooling problem.
  - The review comment is preference-only, so it does not justify a production code change in this batch.
