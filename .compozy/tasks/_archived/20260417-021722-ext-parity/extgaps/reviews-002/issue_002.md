---
status: resolved
file: internal/api/core/handlers_internal_test.go
line: 361
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57AO3b,comment:PRRC_kwDOR5y4QM63zyPg
---

# Issue 002: _🛠️ Refactor suggestion_ | _🟠 Major_

## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify top-level tests in this file contain at least one t.Run call in their body.
python - <<'PY'
import re
from pathlib import Path

p = Path("internal/api/core/handlers_internal_test.go")
src = p.read_text()

for m in re.finditer(r'func\s+(Test\w+)\(t \*testing\.T\)\s*\{', src):
    name = m.group(1)
    i = m.end()
    depth = 1
    while i < len(src) and depth:
        if src[i] == '{':
            depth += 1
        elif src[i] == '}':
            depth -= 1
        i += 1
    body = src[m.end():i-1]
    if "t.Run(" not in body:
        print(f"{name}: missing t.Run")
PY
```

Repository: compozy/agh

Length of output: 181

---

</details>

**Wrap test case in `t.Run("Should...")` to satisfy subtest requirements.**

All test cases must use the `t.Run()` pattern with subtests per coding guidelines. The test logic is sound—only the structural pattern needs correction.

<details>
<summary>Refactoring diff</summary>

```diff
 func TestNetworkStatusPayloadWrapsBundleSettingsErrors(t *testing.T) {
 	t.Parallel()
-
-	settingsErr := errors.New("settings boom")
-	handlers := &BaseHandlers{
-		Config: aghconfig.Config{
-			Network: aghconfig.NetworkConfig{Enabled: true},
-		},
-		Network: networkServiceStub{
-			statusFn: func(context.Context) (*network.NetworkStatus, error) {
-				return &network.NetworkStatus{}, nil
-			},
-		},
-		Bundles: bundleServiceStub{
-			networkSettingsFn: func(context.Context) (bundlepkg.NetworkSettings, error) {
-				return bundlepkg.NetworkSettings{}, settingsErr
-			},
-		},
-	}
-
-	_, err := handlers.networkStatusPayload(context.Background())
-	if !errors.Is(err, settingsErr) {
-		t.Fatalf("networkStatusPayload() error = %v, want wrapped settings error", err)
-	}
-	if !strings.Contains(err.Error(), "api: load bundle network settings") {
-		t.Fatalf("networkStatusPayload() error = %q, want bundle settings context", err.Error())
-	}
+	t.Run("ShouldWrapBundleSettingsErrors", func(t *testing.T) {
+		t.Parallel()
+
+		settingsErr := errors.New("settings boom")
+		handlers := &BaseHandlers{
+			Config: aghconfig.Config{
+				Network: aghconfig.NetworkConfig{Enabled: true},
+			},
+			Network: networkServiceStub{
+				statusFn: func(context.Context) (*network.NetworkStatus, error) {
+					return &network.NetworkStatus{}, nil
+				},
+			},
+			Bundles: bundleServiceStub{
+				networkSettingsFn: func(context.Context) (bundlepkg.NetworkSettings, error) {
+					return bundlepkg.NetworkSettings{}, settingsErr
+				},
+			},
+		}
+
+		_, err := handlers.networkStatusPayload(context.Background())
+		if !errors.Is(err, settingsErr) {
+			t.Fatalf("networkStatusPayload() error = %v, want wrapped settings error", err)
+		}
+		if !strings.Contains(err.Error(), "api: load bundle network settings") {
+			t.Fatalf("networkStatusPayload() error = %q, want bundle settings context", err.Error())
+		}
+	})
 }
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers_internal_test.go` around lines 334 - 361, Wrap the
existing TestNetworkStatusPayloadWrapsBundleSettingsErrors body in a t.Run
subtest (e.g., t.Run("Should wrap bundle settings errors", func(t *testing.T) {
... })) and move the t.Parallel() call inside that subtest so the test still
runs in parallel; keep all existing setup and assertions (handlers and the call
to handlers.networkStatusPayload) unchanged but placed inside the t.Run closure
to satisfy the subtest pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: this is a style-only suggestion, not a correctness issue. The existing test already exercises one concrete behavior and uses `t.Parallel()` appropriately. Wrapping a single test body in a one-off `t.Run(...)` would add ceremony without increasing coverage, isolation, or diagnostic value.
- Why not fixing: repository guidance says table-driven tests with subtests are the default pattern, not a blanket requirement for every standalone unit test. I will preserve the current structure unless a behavior change requires new subcases.
- Resolution: no code change. Analysis completed and the file remains intentionally unchanged.
