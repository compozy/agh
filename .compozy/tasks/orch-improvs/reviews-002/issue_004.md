---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/cli/task.go
line: 1840
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-R,comment:PRRC_kwDOR5y4QM6-VcCs
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Validate that `--missing-work-json` is actually an array.**

The flag help says “JSON array”, but this branch accepts any JSON value. Inputs like `{}` or `"todo"` will pass client-side validation and only fail later, or worse, persist malformed review data.
 
<details>
<summary>Suggested fix</summary>

```diff
 	if hasRaw {
 		payload, err := parseJSONFlag("missing-work-json", raw)
 		if err != nil {
 			return nil, err
 		}
+		var items []json.RawMessage
+		if err := json.Unmarshal(payload, &items); err != nil {
+			return nil, errors.New("cli: --missing-work-json must be a JSON array")
+		}
 		return payload, nil
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func missingWorkFromFlags(items []string, raw string) (json.RawMessage, error) {
	hasRaw := strings.TrimSpace(raw) != ""
	if hasRaw && len(items) > 0 {
		return nil, errors.New("cli: --missing-work-json cannot be combined with --missing-work")
	}
	if hasRaw {
		payload, err := parseJSONFlag("missing-work-json", raw)
		if err != nil {
			return nil, err
		}
		var items []json.RawMessage
		if err := json.Unmarshal(payload, &items); err != nil {
			return nil, errors.New("cli: --missing-work-json must be a JSON array")
		}
		return payload, nil
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/task.go` around lines 1830 - 1840, In missingWorkFromFlags,
after parseJSONFlag("missing-work-json", raw) returns payload, validate that
payload is a JSON array (not an object or scalar) before returning; e.g.
unmarshal payload into []interface{} (or check trimmed first rune is '[' and
json.Valid) and return a clear error like "cli: --missing-work-json must be a
JSON array" if it fails, so that missingWorkFromFlags enforces the flag help
promise.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `missingWorkFromFlags` validates that `--missing-work-json` is syntactically valid JSON but does not enforce the documented array shape.
- Fix approach: Reject non-array JSON payloads early in the CLI layer and add focused coverage in `internal/cli/task_test.go`.
