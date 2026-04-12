---
status: resolved
file: internal/daemon/channels_test.go
line: 225
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tt61,comment:PRRC_kwDOR5y4QM624Xys
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the expected reload failure, not just “some error”.**

Both checks still pass on unrelated failures, so they don't prove the rollback path was triggered by `Reload`. Keep a sentinel `reloadErr` and assert it with `errors.Is`.

<details>
<summary>Example</summary>

```diff
-	extensions := &fakeExtensionRuntime{reloadErr: errors.New("reload boom")}
+	reloadErr := errors.New("reload boom")
+	extensions := &fakeExtensionRuntime{reloadErr: reloadErr}
 	runtime := newChannelRuntime(db, discardLogger(), func() time.Time { return now }, nil)
 	runtime.setExtensionRuntime(extensions)

 	_, err := runtime.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
 		...
 	})
-	if err == nil {
-		t.Fatal("CreateInstance() error = nil, want reload failure")
+	if !errors.Is(err, reloadErr) {
+		t.Fatalf("CreateInstance() error = %v, want wrapped reload failure", err)
 	}
```
</details>


As per coding guidelines, `**/*_test.go`: `MUST have specific error assertions (ErrorContains, ErrorAs)`.


Also applies to: 567-572

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/channels_test.go` around lines 209 - 225, The test uses a
sentinel reload error but only checks that CreateInstance returned a non-nil
error; change the assertion to verify the error is the sentinel reloadErr using
errors.Is (or testing/require/AssertErrorIs equivalent) so the failure proves
Reload was invoked: keep the fakeExtensionRuntime{reloadErr: errors.New("reload
boom")} sentinel, call runtime.CreateInstance as before, then replace t.Fatal on
nil with an assertion like if !errors.Is(err, extensions.reloadErr) {
t.Fatalf("... expected reloadErr, got %v", err) }; apply the same specific-error
assertion fix to the other test at the referenced location (lines ~567-572) that
currently only checks non-nil.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Both reload-failure tests create a sentinel reload error but only assert that the call returned some non-nil error.
  - That means the tests can pass on unrelated failures and do not prove the reload path triggered the rollback behavior under test.
  - Fix approach: keep explicit sentinel reload errors and assert them with `errors.Is` in both affected tests.

## Resolution

- Updated the create rollback and transition rollback tests to keep sentinel reload errors and assert them with `errors.Is`.
- Verified with `go test ./internal/daemon` and `make verify`.
