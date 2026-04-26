---
status: resolved
file: internal/daemon/harness_context_test.go
line: 458
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vK,comment:PRRC_kwDOR5y4QM67Z0NE
---

# Issue 006: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap this new test scenario in `t.Run("Should...")`.**

The added test should use the required explicit subtest pattern.

<details>
<summary>Proposed diff</summary>

```diff
 func TestSectionSelectorAcceptsCoordinatorStartupSession(t *testing.T) {
     t.Parallel()
-
-    resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
-        MemoryPromptSectionEnabled: true,
-        SkillsPromptSectionEnabled: true,
-    })
-    selector := NewSectionSelector(resolver, nil)
-    descriptors := defaultStartupPromptSectionDescriptors(
-        promptSectionProviderFunc(
-            func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) { return "memory", nil },
-        ),
-        promptSectionProviderFunc(
-            func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) { return "skills", nil },
-        ),
-        nil,
-    )
-
-    selected, resolved, err := selector.Select(session.StartupPromptContext{
-        SessionType: session.SessionTypeCoordinator,
-        Channel:     "coord-run-1",
-    }, descriptors)
-    if err != nil {
-        t.Fatalf("Select(coordinator) error = %v", err)
-    }
-
-    if resolved.Session.SessionClass != SessionClassCoordinator {
-        t.Fatalf("SessionClass = %q, want %q", resolved.Session.SessionClass, SessionClassCoordinator)
-    }
-    wantNames := []string{
-        string(HarnessPromptSectionMemory),
-        string(HarnessPromptSectionSkills),
-        string(HarnessPromptSectionNetwork),
-    }
-    gotNames := make([]string, 0, len(selected))
-    for _, descriptor := range selected {
-        gotNames = append(gotNames, descriptor.Name)
-    }
-    if !slices.Equal(gotNames, wantNames) {
-        t.Fatalf("selected section names = %#v, want %#v", gotNames, wantNames)
-    }
+    t.Run("Should accept coordinator startup session and include coordinator sections", func(t *testing.T) {
+        t.Parallel()
+
+        resolver := NewHarnessContextResolver(HarnessRuntimeSignals{
+            MemoryPromptSectionEnabled: true,
+            SkillsPromptSectionEnabled: true,
+        })
+        selector := NewSectionSelector(resolver, nil)
+        descriptors := defaultStartupPromptSectionDescriptors(
+            promptSectionProviderFunc(
+                func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) { return "memory", nil },
+            ),
+            promptSectionProviderFunc(
+                func(context.Context, *workspacepkg.ResolvedWorkspace) (string, error) { return "skills", nil },
+            ),
+            nil,
+        )
+
+        selected, resolved, err := selector.Select(session.StartupPromptContext{
+            SessionType: session.SessionTypeCoordinator,
+            Channel:     "coord-run-1",
+        }, descriptors)
+        if err != nil {
+            t.Fatalf("Select(coordinator) error = %v", err)
+        }
+
+        if resolved.Session.SessionClass != SessionClassCoordinator {
+            t.Fatalf("SessionClass = %q, want %q", resolved.Session.SessionClass, SessionClassCoordinator)
+        }
+        wantNames := []string{
+            string(HarnessPromptSectionMemory),
+            string(HarnessPromptSectionSkills),
+            string(HarnessPromptSectionNetwork),
+        }
+        gotNames := make([]string, 0, len(selected))
+        for _, descriptor := range selected {
+            gotNames = append(gotNames, descriptor.Name)
+        }
+        if !slices.Equal(gotNames, wantNames) {
+            t.Fatalf("selected section names = %#v, want %#v", gotNames, wantNames)
+        }
+    })
 }
```
</details>



As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_context_test.go` around lines 417 - 458, Wrap the new
test function TestSectionSelectorAcceptsCoordinatorStartupSession in an explicit
subtest using t.Run with a "Should..." description; e.g., inside
TestSectionSelectorAcceptsCoordinatorStartupSession call t.Run("Should select
coordinator startup sections", func(t *testing.T) { ... }) and move all existing
test logic (resolver/selector/descriptors creation, selector.Select call,
assertions on resolved.Session.SessionClass and selected names) into that
subtest so the test follows the required t.Run("Should...") subtest pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestSectionSelectorAcceptsCoordinatorStartupSession` has a direct body and no `Should ...` subtest. Fix by moving the existing selector setup and assertions into a `Should select coordinator startup sections` subtest.
