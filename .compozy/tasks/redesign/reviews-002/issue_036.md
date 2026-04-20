---
status: resolved
file: packages/ui/src/components/sonner.test.tsx
line: 31
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcS,comment:PRRC_kwDOR5y4QM65JoyU
---

# Issue 036: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "Checking whether global test setup already performs RTL cleanup:"
fd -i 'setup*' .
rg -n -C2 'afterEach|cleanup\(|@testing-library/react' --iglob '*setup*.ts' --iglob '*setup*.tsx' --iglob '*vitest*.ts' --iglob '*vitest*.tsx'
```

Repository: compozy/agh

Length of output: 1385

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Checking packages/ui/src/test-setup.ts ==="
cat -n packages/ui/src/test-setup.ts

echo ""
echo "=== Checking packages/ui/src/components/sonner.test.tsx ==="
cat -n packages/ui/src/components/sonner.test.tsx
```

Repository: compozy/agh

Length of output: 2375

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Looking for vitest config files ==="
fd -i 'vitest.config*' .

echo ""
echo "=== Checking sonner.ts to verify toast is a singleton ==="
cat -n packages/ui/src/components/sonner.ts | head -40

echo ""
echo "=== Checking if there's a global vitest setup file ==="
rg -n 'setupFiles|setupFilesAfterEnv' vitest.config.ts vitest.config.js 2>/dev/null || echo "No global setup found in vitest config"

echo ""
echo "=== Checking packages/ui vitest config specifically ==="
fd -i 'vitest*' packages/ui/
</parameter>
```

Repository: compozy/agh

Length of output: 419

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Looking for sonner files in packages/ui/src/components ==="
fd 'sonner' packages/ui/src/components/

echo ""
echo "=== Checking packages/ui/vitest.config.ts ==="
cat -n packages/ui/vitest.config.ts

echo ""
echo "=== Looking for index.ts in sonner test directory ==="
ls -la packages/ui/src/components/ | grep -E '\.(ts|tsx)$' | head -20
</parameter>
```

Repository: compozy/agh

Length of output: 2596

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Checking sonner.tsx ==="
cat -n packages/ui/src/components/sonner.tsx
```

Repository: compozy/agh

Length of output: 1513

---



</details>

**Add explicit toast cleanup to reduce cross-test flakiness.**

The `toast` singleton from the sonner library persists across tests. Without calling `toast.dismiss()` between tests, previous test toasts remain in the DOM and can interfere with assertions in subsequent tests. RTL's `cleanup()` must also be explicitly invoked—it is not automatic.

<details>
<summary>Suggested refactor</summary>

```diff
-import { act, render, waitFor } from "@testing-library/react";
-import { describe, expect, it } from "vitest";
+import { act, cleanup, render, waitFor } from "@testing-library/react";
+import { afterEach, describe, expect, it } from "vitest";
 import { Toaster, toast } from "./sonner";
+
+afterEach(() => {
+  toast.dismiss();
+  cleanup();
+});
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/sonner.test.tsx` around lines 13 - 31, Add
explicit cleanup between tests by registering an afterEach hook that calls the
sonner toast singleton's dismiss method and RTL's cleanup to remove lingering
toasts and DOM: add an afterEach that invokes toast.dismiss() and cleanup() so
tests that render <Toaster /> (and use toast.success / toast.error) cannot leak
toasts into subsequent tests; target the toast symbol and the Toaster/render
tests in this file when applying the change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The package runs Vitest with `globals: true` and `@testing-library/react` setup, and the targeted sonner/sidebar/search-input/split-pane/page-header test run passed cleanly without evidence of DOM or toast leakage.
  - The review’s claim that RTL cleanup is not automatic in this setup is inaccurate here, so adding explicit `cleanup()` and `toast.dismiss()` would be redundant without a reproduced flake.
