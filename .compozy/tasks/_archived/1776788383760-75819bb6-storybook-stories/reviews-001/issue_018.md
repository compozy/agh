---
status: resolved
file: web/src/storybook/packages-ui-storybook-config.test.ts
line: 9
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:53bc89794703
review_hash: 53bc89794703
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 018: __dirname is undefined in ESM and will cause a runtime error.
## Review Comment

Line 9 uses `__dirname`, which is not available in ESM modules. The web package is configured as ESM (`"type": "module"` in web/package.json). Replace with `import.meta.url` + `fileURLToPath` to resolve the file path safely.

## Triage

- Decision: `VALID`
- Notes:
  - `web/package.json` declares `"type": "module"`, so this Vitest file runs as ESM and `__dirname` is not available.
  - The current test source reads `preview.ts` via `join(__dirname, ...)`, which is a real runtime hazard for this module format.
  - The resolved fix derives an absolute path with `resolve(process.cwd(), "../packages/ui/.storybook/preview.ts")`, which matches how the web verification commands execute and avoids the ESM-only `__dirname` failure.
