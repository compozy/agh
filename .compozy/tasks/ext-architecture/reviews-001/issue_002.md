---
status: resolved
file: cmd/agh-codegen/main.go
line: 96
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAZu,comment:PRRC_kwDOR5y4QM62zlr0
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Unused `doc` parameter makes the function signature misleading.**

The `doc any` parameter is never used — the function calls `spec.WriteFile()` which internally regenerates the document. This creates a confusing API where callers pass a document that gets ignored.

Either use the passed document for serialization or remove the parameter entirely.

<details>
<summary>Proposed fix: remove unused parameter</summary>

```diff
-func marshalOpenAPI(doc any) ([]byte, error) {
+func marshalOpenAPI() ([]byte, error) {
 	file, err := os.CreateTemp("", "agh-openapi-*.json")
 	if err != nil {
 		return nil, err
```

And update the caller:

```diff
-	want, err := marshalOpenAPI(doc)
+	want, err := marshalOpenAPI()
```

</details>

As per coding guidelines: "Never use `interface{}`/`any` when a concrete type is known".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func marshalOpenAPI() ([]byte, error) {
	file, err := os.CreateTemp("", "agh-openapi-*.json")
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()

	if err := spec.WriteFile(file.Name()); err != nil {
		return nil, err
	}
	return os.ReadFile(file.Name())
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@cmd/agh-codegen/main.go` around lines 82 - 96, The marshalOpenAPI function
currently declares a doc any parameter that is never used (the body calls
spec.WriteFile instead), so update the API to remove the unused parameter:
change the marshalOpenAPI signature to take no arguments, update all callers to
stop passing a document, and keep the existing behavior that writes spec via
spec.WriteFile and returns the file bytes; alternatively if you prefer to
serialize the provided document instead, replace the body to marshal the doc
(e.g., json.Marshal) and remove spec.WriteFile—choose one approach and apply it
consistently to marshalOpenAPI and all call sites that reference marshalOpenAPI.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `marshalOpenAPI` does not use the `doc` parameter at all and always regenerates the spec via `spec.WriteFile`. I will remove the misleading parameter, update the call site, and keep the current file-based serialization behavior.
