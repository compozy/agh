---
status: resolved
file: web/e2e/bridges.spec.ts
line: 29
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcs,comment:PRRC_kwDOR5y4QM640q1T
---

# Issue 035: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't forward the whole host environment into this test runtime.**

Spreading `process.env` here makes the scenario depend on ambient developer/CI variables and can leak unrelated secrets into the seeded daemon. This should stay allow-listed to the few variables the fixture actually needs.

<details>
<summary>🔐 Suggested fix</summary>

```diff
     env: {
-      ...process.env,
       AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
     },
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    env: {
      AGH_TEST_TELEGRAM_TOKEN: "telegram-bot-token",
    },
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/e2e/bridges.spec.ts` around lines 26 - 29, The test is leaking the host
environment by spreading process.env into the env object; remove the spread and
explicitly set only the required allow-listed variables (e.g.,
AGH_TEST_TELEGRAM_TOKEN) in the env object used by the fixture in
bridges.spec.ts (where env is defined), avoiding ...process.env so no unrelated
CI/developer secrets are forwarded.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Spreading `process.env` into the bridge browser runtime leaks unrelated host
  variables into the seeded daemon and makes the test ambient-environment
  dependent. The scenario still needs a narrow allow-list for the Telegram
  token plus the launch prerequisites used by the seeded mock-agent runtime.

## Resolution

- Replaced the host-environment spread with an explicit allow-list that keeps
  the Telegram token and only the minimal process-launch variables needed by
  the seeded bridge test runtime.
