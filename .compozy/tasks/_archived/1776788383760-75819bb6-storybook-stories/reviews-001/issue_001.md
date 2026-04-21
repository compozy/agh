---
status: resolved
file: packages/ui/.storybook/preview.css
line: 3
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:82d242ad3de2
review_hash: 82d242ad3de2
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 001: Update Stylelint configuration to allow Tailwind v4 at-rules (@source, @import).
## Review Comment

The `@source` directive on line 3 is valid Tailwind v4 syntax, but Stylelint's inherited `at-rule-no-unknown` rule from `stylelint-config-standard-scss` will flag it as unknown and block CI. Add the `stylelint-tailwindcss` plugin or explicitly configure `ignoreAtRules` in `.stylelintrc.json` to allow Tailwind directives:

Option 1: Install and use the Tailwind plugin:
```json
{
"extends": ["stylelint-config-standard-scss"],
"plugins": ["stylelint-tailwindcss"]
}
```

Option 2: Disable the rule or allow custom at-rules:
```json
{
"extends": ["stylelint-config-standard-scss"],
"rules": {
"at-rule-no-unknown": ["error", { "ignoreAtRules": ["source", "tailwindcss", "layer"] }]
}
}
```

## Triage

- Decision: `INVALID`
- Notes:
  - The comment assumes Stylelint is part of this repository's verification path, but this repo has no Stylelint config, no Stylelint dependency, and no Stylelint command in `make verify` or the web/package scripts.
  - `packages/ui/.storybook/preview.css` is consumed by Tailwind v4 through Storybook/Vite, where `@source` is valid syntax and already supported by the active toolchain.
  - There is no actual CI or local failure mode to fix here, so changing unrelated lint configuration would be speculative and out of scope.
