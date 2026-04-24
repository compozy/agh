---
status: resolved
file: web/e2e/tasks.spec.ts
line: 158
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1z,comment:PRRC_kwDOR5y4QM67HMXE
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify whether fixture IDs contain regex metacharacters that could affect dynamic RegExp assertions.
fixture="$(fd -a 'browser_session_lifecycle_fixture.json' | head -n1)"
if [[ -z "${fixture:-}" ]]; then
  echo "browser_session_lifecycle_fixture.json not found"
  exit 1
fi

python - "$fixture" <<'PY'
import json, re, sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)

regex_meta = re.compile(r'[.^$*+?()[\]{}|\\]')
ids = []

def walk(node):
    if isinstance(node, dict):
        for k, v in node.items():
            if k == "id" and isinstance(v, str):
                ids.append(v)
            walk(v)
    elif isinstance(node, list):
        for item in node:
            walk(item)

walk(data)

print(f"Found {len(ids)} ids in {path}")
for value in ids:
    tag = "META_CHAR_PRESENT" if regex_meta.search(value) else "ok"
    print(f"{tag}\t{value}")
PY
```

Repository: compozy/agh

Length of output: 163

---

🏁 Script executed:

```shell
cat -n web/e2e/tasks.spec.ts | head -n 170 | tail -n 30
```

Repository: compozy/agh

Length of output: 1719

---

🏁 Script executed:

```shell
# Search for where 'seeded' is defined or populated in the test file
rg -n "seeded\s*=" web/e2e/tasks.spec.ts | head -n 20
```

Repository: compozy/agh

Length of output: 121

---

🏁 Script executed:

```shell
# Find fixture files that might be used in this test
fd -e json web/e2e/fixtures | head -n 20
```

Repository: compozy/agh

Length of output: 425

---

🏁 Script executed:

```shell
# Find where seedBrowserTasksOperatorFlow is defined
rg -n "seedBrowserTasksOperatorFlow" web/e2e --type ts
```

Repository: compozy/agh

Length of output: 556

---

🏁 Script executed:

```shell
# Search for fixture files
fd . web/e2e/fixtures
```

Repository: compozy/agh

Length of output: 473

---

🏁 Script executed:

```shell
# Check the actual fixture data to see ID formats
find web/e2e -name "*.json" -type f
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Examine the seedBrowserTasksOperatorFlow function
sed -n '735,850p' web/e2e/fixtures/runtime-seed.ts
```

Repository: compozy/agh

Length of output: 3804

---

🏁 Script executed:

```shell
# Check what types of IDs are being generated - look for ID creation patterns
rg -n "id\s*:" web/e2e/fixtures/runtime-seed.ts | head -n 30
```

Repository: compozy/agh

Length of output: 1559

---

🏁 Script executed:

```shell
# Look for task and run creation in the seeding function
rg -B5 -A10 "runningTask\|runningRun" web/e2e/fixtures/runtime-seed.ts | head -n 80
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the createBrowserTask function to see ID format
rg -B5 -A15 "async function createBrowserTask" web/e2e/fixtures/runtime-seed.ts
```

Repository: compozy/agh

Length of output: 632

---

🏁 Script executed:

```shell
# Look for task creation endpoint and response type
rg -B3 -A3 "TaskPayload\|Task\b" web/e2e/fixtures/runtime-seed.ts | head -n 50
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check what TaskRun type looks like - find its definition
rg -n "interface TaskRun\|type TaskRun" --type ts | head -n 5
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Find TaskRecord import in runtime-seed.ts
rg "import.*Task" web/e2e/fixtures/runtime-seed.ts | head -n 20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for TaskRecord type definition across the codebase
fd . --type f \( -name "*.ts" -o -name "*.tsx" \) | xargs rg "interface TaskRecord|type TaskRecord" | head -n 5
```

Repository: compozy/agh

Length of output: 446

---

🏁 Script executed:

```shell
# Check Playwright documentation pattern - look for other uses of toHaveURL with RegExp
rg "toHaveURL\(" web/e2e --type ts | head -n 10
```

Repository: compozy/agh

Length of output: 733

---



</details>

**Make URL assertion consistent with line 163—use `expect.poll()` for pathname comparison instead of dynamic RegExp.**

Line 158 constructs a dynamic RegExp from variable content, which is brittle if IDs ever contain regex metacharacters (`.`, `+`, `*`, etc.). Line 163 in the same test already demonstrates the safer pattern. Apply the same approach here for consistency and robustness.

```diff
-  await expect(appPage).toHaveURL(new RegExp(`${activeRunPath}$`));
+  await expect.poll(() => new URL(appPage.url()).pathname).toBe(activeRunPath);
```

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  const activeRunPath = `/tasks/${seeded.runningTask.id}/runs/${seeded.runningRun.id}`;
  const activeRunLink = tasksUI.dashboardActiveRunLink(seeded.runningRun.id);
  await expect(activeRunLink).toBeVisible();
  await expect(activeRunLink).toHaveAttribute("href", activeRunPath);
  await appPage.goto(runtime.url(activeRunPath), {
    waitUntil: "domcontentloaded",
  });
  await expect(tasksUI.runDetailContent).toBeVisible();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(activeRunPath);
```

</details>

<!-- suggestion_end -->

<details>
<summary>🧰 Tools</summary>

<details>
<summary>🪛 ast-grep (0.42.1)</summary>

[warning] 157-157: Regular expression constructed from variable input detected. This can lead to Regular Expression Denial of Service (ReDoS) attacks if the variable contains malicious patterns. Use libraries like 'recheck' to validate regex safety or use static patterns.
Context: new RegExp(`${activeRunPath}$`)
Note: [CWE-1333] Inefficient Regular Expression Complexity [REFERENCES]
    - https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS
    - https://cwe.mitre.org/data/definitions/1333.html

(regexp-from-variable)

</details>

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/e2e/tasks.spec.ts` around lines 150 - 158, The URL assertion using a
dynamic RegExp is brittle; replace the final expect(appPage).toHaveURL(new
RegExp(`${activeRunPath}$`)) with the same pathname-poll pattern used on line
163: use expect.poll to repeatedly read appPage.url(), parse the URL's pathname,
and assert it equals activeRunPath so the test compares pathnames (referencing
activeRunPath, appPage, and tasksUI.runDetailContent for locating the
assertion).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `web/e2e/tasks.spec.ts` builds `new RegExp(`${activeRunPath}$`)` from a URL path containing runtime-generated task/run IDs. Those IDs are opaque values, so regex metacharacters in an ID would change the assertion semantics instead of matching the literal route.
- Fix approach: compare the current page `pathname` to the literal `activeRunPath` value instead of turning the path into a regex.
- Resolution: replaced the dynamic URL regex assertion with the same literal pathname polling pattern used later in the spec. Targeted Tasks E2E and full `make verify` passed after the code change.
