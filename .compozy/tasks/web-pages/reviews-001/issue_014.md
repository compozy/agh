---
status: resolved
file: web/src/routes/_app/automation.tsx
line: 289
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4T,comment:PRRC_kwDOR5y4QM63ZMIG
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Capture the edited record ID in `editor` state.**

Both update paths recompute the target ID from `effectiveSelectedJobId` / `effectiveSelectedTriggerId` at submit time. If the list selection changes while the dialog is open — search, scope switch, refetch, or deletion — the mutation can submit `""` or hit the wrong record.


<details>
<summary>💡 Suggested fix</summary>

```diff
 type AutomationEditorState =
   | {
       draft: CreateAutomationJobRequest;
       kind: "jobs";
       mode: "create" | "edit";
     }
   | {
       draft: CreateAutomationTriggerRequest;
       kind: "triggers";
       mode: "create" | "edit";
     };
+// Prefer separate edit variants with a stable `id`, e.g.
+// { id: string; draft: CreateAutomationJobRequest; kind: "jobs"; mode: "edit" }
+// { id: string; draft: CreateAutomationTriggerRequest; kind: "triggers"; mode: "edit" }

     setEditor(
       activeTab === "jobs" && selectedJob
         ? {
+            id: selectedJob.id,
             draft: automationJobToDraft(selectedJob),
             kind: "jobs",
             mode: "edit",
           }
         : selectedTrigger
           ? {
+              id: selectedTrigger.id,
               draft: automationTriggerToDraft(selectedTrigger),
               kind: "triggers",
               mode: "edit",
             }

       const job =
         editor.mode === "create"
           ? await createJobMutation.mutateAsync(editor.draft)
           : await updateJobMutation.mutateAsync({
               data: editor.draft,
-              id: effectiveSelectedJobId ?? "",
+              id: editor.id,
             });

       const trigger =
         editor.mode === "create"
           ? await createTriggerMutation.mutateAsync(editor.draft)
           : await updateTriggerMutation.mutateAsync({
               data: editor.draft,
-              id: effectiveSelectedTriggerId ?? "",
+              id: editor.id,
             });
```
</details>


Also applies to: 306-313

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/automation.tsx` around lines 282 - 289, The submit flow
recomputes the target ID at submit time (using effectiveSelectedJobId /
effectiveSelectedTriggerId), which can change while the dialog is open; fix by
capturing and storing the target record ID on editor open (e.g., add
editor.editedId or editor.originalId when opening the editor for a job/trigger)
and use that stable ID in the update paths instead of recomputing: update the
code that opens the editor to set editor.editedId = selectedId (or derive from
editor.draft.id), and change updateJobMutation.mutateAsync and
updateTriggerMutation.mutateAsync calls to use editor.editedId (falling back to
editor.draft.id) so the mutation always targets the originally edited record.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: edit submissions derive the target job/trigger ID from the current selection instead of the record that was originally opened in the editor, so list changes while the dialog is open can retarget or blank out the mutation.
- Fix approach: store the edited record ID in the editor state when opening edit mode and use that stable ID in update mutations.
