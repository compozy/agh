---
status: resolved
file: web/src/systems/session/lib/tool-labels.ts
line: 88
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57IvOJ,comment:PRRC_kwDOR5y4QM63_P0M
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Check existing test coverage for getToolIcon
rg -A 10 'describe.*getToolIcon' web/src/systems/session/lib/tool-labels.test.ts

# Check if any tests pass toolInput parameter
rg 'getToolIcon\([^)]+,' web/src/systems/session/lib/
```

Repository: compozy/agh

Length of output: 512

---

🏁 Script executed:

```shell
cat web/src/systems/session/lib/tool-labels.test.ts
```

Repository: compozy/agh

Length of output: 3770

---

</details>

**Add tests for semantic fallback behavior in `getToolIcon`.**

The `getToolIcon` tests only call the function with a tool name argument. The semantic fallbacks based on `toolInput` properties (lines 80-85: checking for "command", "file_path"/"filePath", "pattern", "url"/"query") are untested. Add test cases that exercise these fallbacks with mock `toolInput` objects.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/lib/tool-labels.ts` around lines 75 - 88, Tests do
not cover semantic fallback branches in getToolIcon; add unit tests that call
getToolIcon with a missing TOOL_ICONS entry and a toolInput object to assert the
correct icon is returned: supply toolInput containing "command" to expect
Terminal, "file_path" and "filePath" to expect FileText, "pattern" to expect
Search, and "url" and "query" to expect Globe; also include a case with no
matching properties to assert the default Wrench is returned. Mock or stub
TOOL_ICONS so the direct lookup misses, import getToolIcon and the Lucide icon
symbols (Terminal, FileText, Search, Globe, Wrench) and add explicit assertions
for each branch.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `getToolIcon` has semantic fallback branches for unknown tools keyed off `toolInput`, but the current unit tests only exercise direct name lookups and the final default branch.
- Fix approach: Extend `tool-labels.test.ts` with explicit cases for the `command`, `file_path`, `filePath`, `pattern`, `url`, `query`, and no-match fallback branches.
- Resolution: Added explicit `getToolIcon` tests for each semantic fallback branch and the no-match default case.
