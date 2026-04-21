---
status: resolved
file: internal/daemon/daemon_test.go
line: 3996
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUF,comment:PRRC_kwDOR5y4QM65IlPG
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don’t silently drop synthetic event marshal failures in the fake manager.**

Line 4029-4031 returns early on marshal failure without surfacing an error, which can hide real test breakage.



<details>
<summary>Proposed fix</summary>

```diff
 func (f *fakeSessionManager) PromptSynthetic(
 	ctx context.Context,
 	id string,
 	opts session.SyntheticPromptOpts,
 ) (<-chan acp.AgentEvent, error) {
@@
-	f.recordSyntheticEvent(id, info, opts)
+	if err := f.recordSyntheticEvent(id, info, opts); err != nil {
+		return nil, err
+	}
 	ch := make(chan acp.AgentEvent)
 	close(ch)
 	return ch, nil
 }
 
 func (f *fakeSessionManager) recordSyntheticEvent(
 	sessionID string,
 	info *session.Info,
 	opts session.SyntheticPromptOpts,
-) {
+) error {
 	if info == nil {
-		return
+		return nil
 	}
@@
 	payload, err := json.Marshal(acp.AgentEvent{
@@
 	})
 	if err != nil {
-		return
+		return fmt.Errorf("record synthetic event payload: %w", err)
 	}
@@
 	f.sessionEvents[sessionID] = append(f.sessionEvents[sessionID], store.SessionEvent{
@@
 	})
+	return nil
 }
```
</details>


Also applies to: 4002-4043

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3970 - 3996, In PromptSynthetic
of fakeSessionManager, don't swallow JSON marshal errors when preparing the
synthetic event: check the error returned by whatever serialization step (used
in recordSyntheticEvent or the marshal call inside that flow) and return that
error from PromptSynthetic instead of returning a closed channel; update the
logic around syntheticPromptHook/recordSyntheticEvent so that if marshaling or
event creation fails you propagate the error (return nil, err) and still record
the call in syntheticPromptCalls, preserving the existing hook behavior
(hook(ctx, id, opts) remains untouched).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  I verified the `recordSyntheticEvent` path against the current `acp.AgentEvent` and `acp.PromptSyntheticMeta` shapes. The marshaled payload only contains strings, `time.Time`, and a nil-or-JSON-safe `json.RawMessage`, so `json.Marshal` is not a realistic failure mode in this fake. Threading a new error return through the fake manager would add dead plumbing without improving correctness or coverage, so I am treating this as non-actionable for the current code.
