---
status: resolved
file: internal/session/session_test.go
line: 213
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azz,comment:PRRC_kwDOR5y4QM66RFPf
---

# Issue 015: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this new case in a `t.Run("Should...")` subtest to follow test policy.**

The assertions are good, but this new test case should use the required subtest naming pattern.



<details>
<summary>Proposed adjustment</summary>

```diff
 func TestSessionMetaRoundTripIncludesProvider(t *testing.T) {
 	t.Parallel()
-
-	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
-	session := &Session{
-		ID:          "sess-provider",
-		Name:        "Provider Session",
-		AgentName:   "coder",
-		Provider:    "codex",
-		WorkspaceID: "ws-provider",
-		Workspace:   t.TempDir(),
-		State:       StateActive,
-		CreatedAt:   now,
-		UpdatedAt:   now,
-	}
-
-	meta := session.Meta()
-	if got := meta.Provider; got != "codex" {
-		t.Fatalf("Meta().Provider = %q, want %q", got, "codex")
-	}
-	if got := session.Info().Provider; got != "codex" {
-		t.Fatalf("Info().Provider = %q, want %q", got, "codex")
-	}
-
-	metaPath := filepath.Join(t.TempDir(), "meta.json")
-	if err := store.WriteSessionMeta(metaPath, meta); err != nil {
-		t.Fatalf("WriteSessionMeta() error = %v", err)
-	}
-
-	readBack, err := store.ReadSessionMeta(metaPath)
-	if err != nil {
-		t.Fatalf("ReadSessionMeta() error = %v", err)
-	}
-	if got := readBack.Provider; got != "codex" {
-		t.Fatalf("ReadSessionMeta().Provider = %q, want %q", got, "codex")
-	}
+	t.Run("Should persist and reload provider in session metadata", func(t *testing.T) {
+		t.Parallel()
+		now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
+		session := &Session{
+			ID:          "sess-provider",
+			Name:        "Provider Session",
+			AgentName:   "coder",
+			Provider:    "codex",
+			WorkspaceID: "ws-provider",
+			Workspace:   t.TempDir(),
+			State:       StateActive,
+			CreatedAt:   now,
+			UpdatedAt:   now,
+		}
+
+		meta := session.Meta()
+		if got := meta.Provider; got != "codex" {
+			t.Fatalf("Meta().Provider = %q, want %q", got, "codex")
+		}
+		if got := session.Info().Provider; got != "codex" {
+			t.Fatalf("Info().Provider = %q, want %q", got, "codex")
+		}
+
+		metaPath := filepath.Join(t.TempDir(), "meta.json")
+		if err := store.WriteSessionMeta(metaPath, meta); err != nil {
+			t.Fatalf("WriteSessionMeta() error = %v", err)
+		}
+
+		readBack, err := store.ReadSessionMeta(metaPath)
+		if err != nil {
+			t.Fatalf("ReadSessionMeta() error = %v", err)
+		}
+		if got := readBack.Provider; got != "codex" {
+			t.Fatalf("ReadSessionMeta().Provider = %q, want %q", got, "codex")
+		}
+	})
 }
```
</details>

As per coding guidelines `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestSessionMetaRoundTripIncludesProvider(t *testing.T) {
	t.Parallel()
	t.Run("Should persist and reload provider in session metadata", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
		session := &Session{
			ID:          "sess-provider",
			Name:        "Provider Session",
			AgentName:   "coder",
			Provider:    "codex",
			WorkspaceID: "ws-provider",
			Workspace:   t.TempDir(),
			State:       StateActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		meta := session.Meta()
		if got := meta.Provider; got != "codex" {
			t.Fatalf("Meta().Provider = %q, want %q", got, "codex")
		}
		if got := session.Info().Provider; got != "codex" {
			t.Fatalf("Info().Provider = %q, want %q", got, "codex")
		}

		metaPath := filepath.Join(t.TempDir(), "meta.json")
		if err := store.WriteSessionMeta(metaPath, meta); err != nil {
			t.Fatalf("WriteSessionMeta() error = %v", err)
		}

		readBack, err := store.ReadSessionMeta(metaPath)
		if err != nil {
			t.Fatalf("ReadSessionMeta() error = %v", err)
		}
		if got := readBack.Provider; got != "codex" {
			t.Fatalf("ReadSessionMeta().Provider = %q, want %q", got, "codex")
		}
	})
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/session_test.go` around lines 177 - 213, Wrap the test body
of TestSessionMetaRoundTripIncludesProvider in a t.Run subtest using the
"Should..." naming pattern (e.g. t.Run("Should include provider in meta
roundtrip", func(t *testing.T) { ... })), and move the existing t.Parallel()
call inside that subtest; keep the existing assertions and variables (session,
meta, store.WriteSessionMeta, store.ReadSessionMeta) unchanged so the test still
verifies meta.Provider and Info().Provider and the read-back provider value.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the new provider round-trip test is a direct top-level body instead of using the required `t.Run("Should ...")` pattern.
- Fix plan: wrap the existing assertions in a named subtest without changing the behavior under test.
