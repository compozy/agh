---
status: resolved
file: web/src/systems/session/mocks/fixtures.ts
line: 17
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wHE,comment:PRRC_kwDOR5y4QM65CLCH
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Remove user-specific absolute paths from committed fixtures.**

Line 17 and Line 33 include `/Users/pedro/...`, which leaks local user/environment details and makes fixtures non-portable.



<details>
<summary>🔧 Proposed fix</summary>

```diff
-    workspace_path: "/Users/pedro/Dev/compozy/agh2",
+    workspace_path: "/workspaces/agh2",
...
-    workspace_path: "/Users/pedro/Dev/compozy/agh2",
+    workspace_path: "/workspaces/agh2",
```
</details>


Also applies to: 33-33

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/mocks/fixtures.ts` at line 17, Remove the hard-coded
user-specific absolute paths in the fixtures by replacing the value of
workspace_path and the second absolute path occurrence with a portable,
repo-relative or placeholder value; update the fixture to use a relative path or
a helper that derives the path at runtime (e.g., based on __dirname or
process.cwd()) so tests don't leak /Users/pedro and remain portable — change the
workspace_path entry and the other absolute path entry on line 33 accordingly
and adjust any tests that assume the old absolute value.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Verified in `web/src/systems/session/mocks/fixtures.ts`.
  - Two committed session fixtures hard-code `/Users/pedro/Dev/compozy/agh2` in `workspace_path`, which leaks a local machine path and makes the fixtures non-portable.
  - Root cause: story/session fixture data reused a developer-local absolute path instead of a neutral fixture path.
  - Fix approach: replace the user-specific values with a neutral workspace path and extend the regression test to assert that session fixture paths remain portable.
  - Resolved by switching the fixture paths to `/workspaces/agh2` and adding session-path neutrality assertions.
