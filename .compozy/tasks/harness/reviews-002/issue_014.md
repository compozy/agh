---
status: resolved
file: internal/session/stop_reason_test.go
line: 307
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUM,comment:PRRC_kwDOR5y4QM65IlPR
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
fd -t f "stop_reason_test.go" internal/session/
```

Repository: compozy/agh

Length of output: 91

---

🏁 Script executed:

```shell
cat -n internal/session/stop_reason_test.go | sed -n '217,320p'
```

Repository: compozy/agh

Length of output: 3620

---

🏁 Script executed:

```shell
# Also check the prepareStopWithCause implementation to understand what errors it wraps
rg -A 10 "func.*prepareStopWithCause" internal/session/ -t go
```

Repository: compozy/agh

Length of output: 699

---

🏁 Script executed:

```shell
# Get more of the prepareStopWithCause implementation to see error wrapping
rg -B 5 -A 50 "func.*prepareStopWithCause" internal/session/stop_reason.go
```

Repository: compozy/agh

Length of output: 1918

---

🏁 Script executed:

```shell
# Look for error wrapping patterns in the file, especially for hooks and metadata write
rg "hook boom|metadata write" internal/session/stop_reason.go -B 3 -A 3
```

Repository: compozy/agh

Length of output: 314

---

🏁 Script executed:

```shell
# Search for the actual hook dispatch and metadata write error handling
rg "dispatchSessionPreStop|metaPath" internal/session/stop_reason.go -B 2 -A 5
```

Repository: compozy/agh

Length of output: 400

---



</details>

**Add explicit root-cause assertions for all failure-stage rows.**

Two rows currently validate only stage text (`pre-stop hook failures` and `metadata write failures`) using `strings.Contains`. This allows tests to pass even if wrapped causes regress. Per the coding guidelines, tests must use specific error assertions like `errors.Is()` and `errors.As()`. Assert wrapped causes for these rows too, not only stage context.

<details>
<summary>Suggested patch</summary>

```diff
 func TestPrepareStopWithCauseWrapsStageFailures(t *testing.T) {
 	t.Parallel()
 
+	hookErr := errors.New("hook boom")
+
 	tests := []struct {
 		name      string
 		setup     func(t *testing.T) (*Manager, *Session, context.Context)
 		wantStage string
 		wantErr   error
+		wantAsPathErr bool
 	}{
 		{
 			name: "Should wrap pre-stop hook failures",
 			setup: func(t *testing.T) (*Manager, *Session, context.Context) {
 				t.Helper()
 
 				dispatcher := &spyHookDispatcher{
 					dispatchSessionPreStopFn: func(
 						_ context.Context,
 						payload hookspkg.SessionPreStopPayload,
 					) (hookspkg.SessionPreStopPayload, error) {
-						return payload, errors.New("hook boom")
+						return payload, hookErr
 					},
 				}
 				h := newHarness(t, WithHookSet(fullHookSet(dispatcher)))
 				return h.manager, createSession(t, h), testutil.Context(t)
 			},
 			wantStage: "prepare stop pre-stop hooks",
+			wantErr:   hookErr,
 		},
@@
 		{
 			name: "Should wrap metadata write failures",
@@
 			wantStage: "prepare stop metadata write",
+			wantAsPathErr: true,
 		},
@@
 			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
 				t.Fatalf("prepareStopWithCause() error = %v, want wrapped %v", err, tc.wantErr)
 			}
+			if tc.wantAsPathErr {
+				var pathErr *os.PathError
+				if !errors.As(err, &pathErr) {
+					t.Fatalf("prepareStopWithCause() error = %v, want wrapped *os.PathError", err)
+				}
+			}
 		})
 	}
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestPrepareStopWithCauseWrapsStageFailures(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook boom")

	tests := []struct {
		name      string
		setup     func(t *testing.T) (*Manager, *Session, context.Context)
		wantStage string
		wantErr   error
		wantAsPathErr bool
	}{
		{
			name: "Should wrap pre-stop hook failures",
			setup: func(t *testing.T) (*Manager, *Session, context.Context) {
				t.Helper()

				dispatcher := &spyHookDispatcher{
					dispatchSessionPreStopFn: func(
						_ context.Context,
						payload hookspkg.SessionPreStopPayload,
					) (hookspkg.SessionPreStopPayload, error) {
						return payload, hookErr
					},
				}
				h := newHarness(t, WithHookSet(fullHookSet(dispatcher)))
				return h.manager, createSession(t, h), testutil.Context(t)
			},
			wantStage: "prepare stop pre-stop hooks",
			wantErr:   hookErr,
		},
		{
			name: "Should wrap state synchronization failures",
			setup: func(t *testing.T) (*Manager, *Session, context.Context) {
				t.Helper()

				h := newHarness(t)
				session := createSession(t, h)
				session.mu.Lock()
				session.State = StateStarting
				session.mu.Unlock()
				return h.manager, session, testutil.Context(t)
			},
			wantStage: "prepare stop state sync",
			wantErr:   ErrInvalidStateTransition,
		},
		{
			name: "Should wrap metadata write failures",
			setup: func(t *testing.T) (*Manager, *Session, context.Context) {
				t.Helper()

				h := newHarness(t)
				session := createSession(t, h)
				blockingPath := filepath.Join(t.TempDir(), "meta-parent")
				if err := os.WriteFile(blockingPath, []byte("block"), 0o644); err != nil {
					t.Fatalf("WriteFile(blockingPath) error = %v", err)
				}
				session.mu.Lock()
				session.metaPath = filepath.Join(blockingPath, "session.json")
				session.mu.Unlock()
				return h.manager, session, testutil.Context(t)
			},
			wantStage: "prepare stop metadata write",
			wantAsPathErr: true,
		},
		{
			name: "Should wrap prompt setup wait failures",
			setup: func(t *testing.T) (*Manager, *Session, context.Context) {
				t.Helper()

				h := newHarness(t)
				session := createSession(t, h)
				if _, err := session.beginPromptSetup(); err != nil {
					t.Fatalf("beginPromptSetup() error = %v", err)
				}
				ctx, cancel := context.WithCancel(testutil.Context(t))
				cancel()
				return h.manager, session, ctx
			},
			wantStage: "prepare stop prompt setup wait",
			wantErr:   context.Canceled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager, session, ctx := tc.setup(t)
			_, _, _, _, _, err := manager.prepareStopWithCause(ctx, session.ID, CauseUserRequested, "")
			if err == nil {
				t.Fatal("prepareStopWithCause() error = nil, want wrapped stage failure")
			}
			if !strings.Contains(err.Error(), tc.wantStage) {
				t.Fatalf("prepareStopWithCause() error = %v, want stage context %q", err, tc.wantStage)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("prepareStopWithCause() error = %v, want wrapped %v", err, tc.wantErr)
			}
			if tc.wantAsPathErr {
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) {
					t.Fatalf("prepareStopWithCause() error = %v, want wrapped *os.PathError", err)
				}
			}
		})
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/stop_reason_test.go` around lines 217 - 307, The two
table-driven tests ("Should wrap pre-stop hook failures" and "Should wrap
metadata write failures") only check for stage text; add explicit expected
root-cause errors (set wantErr to the concrete error values, e.g. the hook error
and the underlying file/write error) and update the verification logic after
calling manager.prepareStopWithCause to always assert the wrapped cause using
errors.Is(err, tc.wantErr) (keep the existing strings.Contains check for
tc.wantStage). Locate the tests in stop_reason_test.go where the tests slice and
the call to prepareStopWithCause are defined and update the two test case
entries to include appropriate wantErr values and ensure the final error checks
use tc.wantErr for all rows.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestPrepareStopWithCauseWrapsStageFailures` currently asserts wrapped causes for the state-sync and prompt-setup rows, but the pre-stop hook and metadata-write rows only check stage text.
  - `prepareStopWithCause` wraps the underlying hook error and the metadata write failure, so the test should assert those root causes explicitly as well.
  - Root cause: the regression coverage is incomplete and can miss wrapped-cause regressions while still passing on stage-string matches.
  - Fix approach: use a stable sentinel error for the hook row plus `errors.Is`, and assert `errors.As(..., *os.PathError)` for the metadata-write row.
  - Resolved in `internal/session/stop_reason_test.go` and verified with targeted tests plus the full repository gate.
