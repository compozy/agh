---
status: resolved
file: internal/extension/registry_bundles_test.go
line: 47
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__B8,comment:PRRC_kwDOR5y4QM63zbyd
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restructure this test into `t.Run("Should...")` subtests.**

Line 14 currently uses a single monolithic test body; this violates the repo’s required test pattern for subtests/table-driven cases.

<details>
<summary>♻️ Suggested refactor</summary>

```diff
 func TestRegistryBlocksDisableAndUninstallWithActiveBundles(t *testing.T) {
-	t.Parallel()
-
-	env := newRegistryTestEnvWithBundleActivations(t)
-	dir, manifest, checksum := createRegistryTestExtension(t, "bundle-guard", registryManifestOptions{})
-	if err := env.registry.Install(manifest, dir, checksum); err != nil {
-		t.Fatalf("Install() error = %v", err)
-	}
-
-	if _, err := env.db.Exec(
-		`INSERT INTO bundle_activations (
-			id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
-		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
-		"act_guard",
-		manifest.Name,
-		"bundle",
-		"default",
-		"global",
-		nil,
-		"hash",
-		false,
-		store.FormatTimestamp(env.installedAt),
-		store.FormatTimestamp(env.installedAt),
-	); err != nil {
-		t.Fatalf("insert bundle activation error = %v", err)
-	}
-
-	if err := env.registry.Disable(manifest.Name); !errors.Is(err, ErrExtensionHasActiveBundles) {
-		t.Fatalf("Disable() error = %v, want ErrExtensionHasActiveBundles", err)
-	}
-	if err := env.registry.Uninstall(manifest.Name); !errors.Is(err, ErrExtensionHasActiveBundles) {
-		t.Fatalf("Uninstall() error = %v, want ErrExtensionHasActiveBundles", err)
-	}
+	testCases := []struct {
+		name string
+		op   func(*Registry, string) error
+	}{
+		{
+			name: "Should block Disable when extension has active bundles",
+			op: func(r *Registry, name string) error { return r.Disable(name) },
+		},
+		{
+			name: "Should block Uninstall when extension has active bundles",
+			op: func(r *Registry, name string) error { return r.Uninstall(name) },
+		},
+	}
+
+	for _, tc := range testCases {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+
+			env := newRegistryTestEnvWithBundleActivations(t)
+			dir, manifest, checksum := createRegistryTestExtension(t, "bundle-guard", registryManifestOptions{})
+			if err := env.registry.Install(manifest, dir, checksum); err != nil {
+				t.Fatalf("Install() error = %v", err)
+			}
+
+			if _, err := env.db.Exec(
+				`INSERT INTO bundle_activations (
+					id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
+				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
+				"act_guard", manifest.Name, "bundle", "default", "global", nil, "hash", false,
+				store.FormatTimestamp(env.installedAt), store.FormatTimestamp(env.installedAt),
+			); err != nil {
+				t.Fatalf("insert bundle activation error = %v", err)
+			}
+
+			if err := tc.op(env.registry, manifest.Name); !errors.Is(err, ErrExtensionHasActiveBundles) {
+				t.Fatalf("operation error = %v, want ErrExtensionHasActiveBundles", err)
+			}
+		})
+	}
 }
```

</details>

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (t.Run) as default in Go tests."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry_bundles_test.go` around lines 14 - 47, Refactor
TestRegistryBlocksDisableAndUninstallWithActiveBundles into t.Run subtests: keep
the shared setup (newRegistryTestEnvWithBundleActivations,
createRegistryTestExtension, env.registry.Install and the INSERT into
bundle_activations) in the parent test, call t.Parallel() at top-level, then add
two subtests using t.Run("Should block Disable when active bundles exist") and
t.Run("Should block Uninstall when active bundles exist") that respectively
invoke env.registry.Disable(manifest.Name) and
env.registry.Uninstall(manifest.Name) and assert errors.Is(err,
ErrExtensionHasActiveBundles); ensure assertions and error messages are moved
into the corresponding subtests and no duplicated setup is performed.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: the current test already exercises the two guarded operations directly with a shared fixture and explicit assertions. Rewriting it into subtests would be a shape-only refactor with no behavioral delta, and the batch should stay focused on defects rather than cosmetic test rearrangement.
- Resolution: no code change required for this review item.
