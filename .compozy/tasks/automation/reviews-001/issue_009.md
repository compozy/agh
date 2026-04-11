---
status: resolved
file: internal/automation/extension.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0P,comment:PRRC_kwDOR5y4QM623e7b
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject surrounding whitespace on `event` instead of silently accepting it.**

`Validate()` trims only for the checks, so `" ext.build "` passes here even though `r.Event` still contains whitespace afterward. That leaves you with a validated request whose event name will not match downstream `ext.*` trigger lookups.

<details>
<summary>Possible fix</summary>

```diff
 func (r ExtensionTriggerRequest) Validate(path string) error {
-	if strings.TrimSpace(r.Event) == "" {
+	event := strings.TrimSpace(r.Event)
+	if event == "" {
 		return errors.New(nestedPath(path, "event") + " is required")
 	}
-	if !strings.HasPrefix(strings.TrimSpace(r.Event), "ext.") {
+	if event != r.Event {
+		return errors.New(nestedPath(path, "event") + " must not contain surrounding whitespace")
+	}
+	if !strings.HasPrefix(event, "ext.") {
 		return errors.New(nestedPath(path, "event") + " must start with \"ext.\"")
 	}
 	if err := ValidateScopeBinding(r.Scope, r.WorkspaceID, path, "workspace_id"); err != nil {
 		return err
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (r ExtensionTriggerRequest) Validate(path string) error {
	event := strings.TrimSpace(r.Event)
	if event == "" {
		return errors.New(nestedPath(path, "event") + " is required")
	}
	if event != r.Event {
		return errors.New(nestedPath(path, "event") + " must not contain surrounding whitespace")
	}
	if !strings.HasPrefix(event, "ext.") {
		return errors.New(nestedPath(path, "event") + " must start with \"ext.\"")
	}
	if err := ValidateScopeBinding(r.Scope, r.WorkspaceID, path, "workspace_id"); err != nil {
		return err
	}
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/extension.go` around lines 17 - 27, The Validate method
on ExtensionTriggerRequest currently trims r.Event only for checks but leaves
the original value intact; change Validate (in ExtensionTriggerRequest.Validate)
to first compute trimmed := strings.TrimSpace(r.Event), then if trimmed == ""
return the existing required error, then if trimmed != r.Event return an error
indicating surrounding whitespace is not allowed (use nestedPath(path, "event")
in the message), and then proceed to use trimmed for the HasPrefix("ext.") check
and the rest of validation (including the existing ValidateScopeBinding call).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `ExtensionTriggerRequest.Validate` trims `Event` only while checking it, which means a whitespace-padded event validates even though the stored value still contains whitespace and will not match trigger registrations. I will reject surrounding whitespace explicitly and keep validation aligned with the exact event string that runtime matching consumes.
