---
status: resolved
file: internal/store/meta_test.go
line: 19
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCk,comment:PRRC_kwDOR5y4QM61T6IB
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert `WorkspaceID` in both read-back checks**

`WorkspaceID` is now part of the fixture, but it is never validated in either test. A serialization/deserialization regression on this field would go undetected.



<details>
<summary>Proposed test assertion updates</summary>

```diff
- if readBack.ID != meta.ID || readBack.AgentName != meta.AgentName || readBack.State != meta.State || readBack.SessionType != meta.SessionType {
+ if readBack.ID != meta.ID ||
+    readBack.AgentName != meta.AgentName ||
+    readBack.WorkspaceID != meta.WorkspaceID ||
+    readBack.State != meta.State ||
+    readBack.SessionType != meta.SessionType {
    t.Fatalf("ReadSessionMeta() = %#v, want %#v", readBack, meta)
  }
```

```diff
- if meta.ID != base.ID || meta.AgentName != base.AgentName {
-   t.Fatalf("ReadSessionMeta() = %#v, want id=%q agent=%q", meta, base.ID, base.AgentName)
+ if meta.ID != base.ID || meta.AgentName != base.AgentName || meta.WorkspaceID != base.WorkspaceID {
+   t.Fatalf("ReadSessionMeta() = %#v, want id=%q agent=%q workspace_id=%q", meta, base.ID, base.AgentName, base.WorkspaceID)
  }
```
</details>


Also applies to: 46-46

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/meta_test.go` at line 19, Add assertions to validate the
WorkspaceID field in both read-back checks in the meta_test.go tests: after
creating the fixture with WorkspaceID: "ws-meta", assert that the
persisted/read-back Meta struct's WorkspaceID equals "ws-meta" in both places
where the test currently checks other fields; ensure you reference the
WorkspaceID field on the returned object (e.g., returnedMeta.WorkspaceID or
metaReadBack.WorkspaceID) in each assertion so serialization/deserialization
regressions for WorkspaceID are detected.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Read-back assertions cover other persisted metadata fields but skip `WorkspaceID`
  even though both fixtures populate it. A `workspace_id` serialization or
  deserialization regression would currently pass unnoticed. Plan: extend both
  read-back assertions to include `WorkspaceID`.
