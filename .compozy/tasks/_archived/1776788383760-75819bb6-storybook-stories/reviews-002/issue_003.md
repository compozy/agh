---
status: resolved
file: web/src/storybook/web-storybook-stories-and-fixtures.test.tsx
line: 6
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wG0,comment:PRRC_kwDOR5y4QM65CLB4
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Extend neutrality checks to session fixture workspace paths.**

This test validates workspace fixture paths but does not validate `sessionFixtures[].workspace_path`, so user-specific paths can still slip through.



<details>
<summary>🔧 Proposed fix</summary>

```diff
-import { uiMessageFixtures } from "@/systems/session/mocks";
+import { sessionFixtures, uiMessageFixtures } from "@/systems/session/mocks";
...
   it("keeps UI message fixture ids unique and workspace paths neutral", () => {
     const ids = uiMessageFixtures.map(message => message.id);
+    const sessionPaths = sessionFixtures
+      .map(session => session.workspace_path)
+      .filter((value): value is string => typeof value === "string");
...
-    for (const path of workspacePaths) {
+    for (const path of [...workspacePaths, ...sessionPaths]) {
       expect(path).not.toMatch(/^\/Users\//);
       expect(path).not.toContain("/pedro/");
     }
   });
```
</details>


Also applies to: 99-123

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/storybook/web-storybook-stories-and-fixtures.test.tsx` around lines 5
- 6, The test currently validates workspace fixture paths but omits checking
session-related fixtures; add neutrality checks for the session fixtures by
iterating over uiMessageFixtures (or the session fixture array used) and
asserting each item’s workspace_path matches one of the known workspace fixtures
(workspaceFixtures[].workspace_path) or equals
workspaceDetailFixture.workspace_path. Update the existing test logic where
workspace paths are validated (the same section that checks workspaceFixtures
and workspaceDetailFixture) to include uiMessageFixtures[].workspace_path so
user-specific paths cannot slip through.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Verified in `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx` and `web/src/systems/session/mocks/fixtures.ts`.
  - The neutrality test only scans workspace fixture roots and skill directories; it never inspects `sessionFixtures[].workspace_path`.
  - Root cause: the regression guard does not cover session fixture paths even though those fixtures are committed mock data and can leak user-specific absolute paths.
  - Fix approach: include the session fixture workspace paths in the neutrality assertions so the test covers all committed session-related path fixtures.
  - Resolved by extending the neutrality assertions to include `sessionFixtures[].workspace_path`.
