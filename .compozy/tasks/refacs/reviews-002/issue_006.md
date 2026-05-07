---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/automation/model/template_test.go
line: 108
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUsj,comment:PRRC_kwDOR5y4QM6-_G3Y
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Missing test cases for the two new `ValidateTriggerPromptTemplate` code paths.**

The new guards added in `template.go` (lines 38–43) are not exercised by any test for `ValidateTriggerPromptTemplate`:

1. **Empty/whitespace input** (lines 38–40 of `template.go`) — no subtest calls `ValidateTriggerPromptTemplate` with `""` or `"   "`.
2. **Plain-text prompt with no delimiters** (lines 41–43 of `template.go`) — no subtest verifies that `ValidateTriggerPromptTemplate("plain text")` returns `nil`.

Both paths are trivial to cover and keeping them untested risks breaching the 80% coverage floor.

<details>
<summary>✅ Suggested additions to the acceptance table and a new rejection table entry</summary>

```diff
 // In TestValidateTriggerPromptTemplateAcceptsSupportedReferences
         {
+            name:   "Should accept plain text without template delimiters",
+            prompt: "Trigger kind: plain text only",
+        },
+        {
```

```diff
+func TestValidateTriggerPromptTemplateRejectsEmptyInput(t *testing.T) {
+    t.Parallel()
+
+    tests := []struct {
+        name   string
+        prompt string
+    }{
+        {name: "Should reject empty prompt", prompt: ""},
+        {name: "Should reject whitespace-only prompt", prompt: "   "},
+    }
+
+    for _, tt := range tests {
+        t.Run(tt.name, func(t *testing.T) {
+            t.Parallel()
+            err := ValidateTriggerPromptTemplate(tt.prompt)
+            requireErrorContains(t, err, "required")
+        })
+    }
+}
```
</details>




As per coding guidelines, the repository enforces an "80% code coverage floor per Go package."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestValidateTriggerPromptTemplateAcceptsSupportedReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "Should accept top level fields",
			prompt: `Trigger kind: {{ .Kind }}`,
		},
		{
			name:   "Should accept data field chains",
			prompt: `Session: {{ .Data.session_id }}`,
		},
		{
			name:   "Should accept data index lookups",
			prompt: `Payload: {{ index .Data "payload" }}`,
		},
		{
			name:   "Should accept data scoped with blocks",
			prompt: `{{ with .Data }}{{ index . "session_id" }}{{ end }}`,
		},
		{
			name:   "Should accept data scoped field lookups",
			prompt: `{{ with .Data }}{{ .session_id }}{{ end }}`,
		},
		{
			name:   "Should accept range variables without variable rooted field lookups",
			prompt: `{{ range $key, $value := .Data }}{{ $key }}{{ end }}`,
		},
		{
			name:   "Should accept chained data expressions",
			prompt: `{{ (.Data).session_id }}`,
		},
		{
			name:   "Should accept defined templates with root envelope invocation",
			prompt: `{{ define "body" }}{{ .Source }}{{ end }}{{ template "body" . }}`,
		},
		{
			name:   "Should accept plain text without template delimiters",
			prompt: "Trigger kind: plain text only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := ValidateTriggerPromptTemplate(tt.prompt); err != nil {
				t.Fatalf("ValidateTriggerPromptTemplate() error = %v", err)
			}
		})
	}
}

func TestValidateTriggerPromptTemplateRejectsUnsupportedReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "Should reject unknown top level fields",
			prompt: `{{ .EnvelopeID }}`,
			want:   []string{"EnvelopeID"},
		},
		{
			name:   "Should reject child fields on scalar values",
			prompt: `{{ .Scope.Name }}`,
			want:   []string{"Scope"},
		},
		{
			name:   "Should reject chained scalar field lookups",
			prompt: `{{ (.Source).Value }}`,
			want:   []string{"Source"},
		},
		{
			name:   "Should reject non data index targets",
			prompt: `{{ index .Kind "anything" }}`,
			want:   []string{".Kind"},
		},
		{
			name:   "Should reject root dot index targets",
			prompt: `{{ index . "payload" }}`,
			want:   []string{"only .Data"},
		},
		{
			name:   "Should reject variable rooted lookups",
			prompt: `{{ range $key, $value := .Data }}{{ $value.name }}{{ end }}`,
			want:   []string{"variable-rooted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTriggerPromptTemplate(tt.prompt)
			requireErrorContains(t, err, tt.want...)
		})
	}
}

func TestValidateTriggerPromptTemplateRejectsEmptyInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prompt string
	}{
		{name: "Should reject empty prompt", prompt: ""},
		{name: "Should reject whitespace-only prompt", prompt: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTriggerPromptTemplate(tt.prompt)
			requireErrorContains(t, err, "required")
		})
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/automation/model/template_test.go` around lines 8 - 108, Add two new
acceptance subtests for ValidateTriggerPromptTemplate in template_test.go
covering the new guards: one subtest calling ValidateTriggerPromptTemplate with
empty string "" and another with whitespace "   " (both should return nil), and
a third subtest calling it with a plain-text prompt like "plain text" (no
template delimiters) which should also return nil; place these entries in the
TestValidateTriggerPromptTemplateAcceptsSupportedReferences table (or add them
as individual t.Run cases) and follow the existing t.Parallel and error-checking
pattern used in other cases, referencing ValidateTriggerPromptTemplate to locate
where to add them.
```

</details>

<!-- fingerprinting:phantom:poseidon:churro -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/automation/model/template.go:37-43` added two behavior branches that are not covered in `template_test.go`: empty/whitespace rejection and plain-text-without-delimiters acceptance.
  - The prose inside the autogenerated agent prompt is inconsistent with the actual code and review text; the code clearly rejects empty input and accepts plain text with no delimiters.
  - Fix plan: add a plain-text acceptance case and explicit empty/whitespace rejection coverage aligned with the current implementation.
  - Resolved: the test file now covers plain-text acceptance plus required-input rejection for `ValidateTriggerPromptTemplate`.
