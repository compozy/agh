# Issue 4 - Review Thread Comment

**File:** `internal/skills/loader.go:285`
**Date:** 2026-04-08 11:02:40 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: an empty hook command is a malformed declaration, not a runtime execution problem. The loader now rejects that entry during parse with skill/index context and surfaces the error through both disk and bundled loaders.

## Body

_⚠️ Potential issue_ | _🟠 Major_

**Reject hook entries with empty `command` during parse.**

`parseHookDecls` currently appends hooks even when `command` is empty. That defers failure to runtime hook execution instead of failing fast at load time.

<details>
<summary>✅ Proposed fix</summary>

```diff
 		hook := HookDecl{
 			Event:   event,
 			Command: strings.TrimSpace(stringValue(entry["command"])),
 			Args:    stringSliceValue(skill, "metadata.agh.hooks", idx, "args", entry["args"]),
 			Env:     stringMapValue(skill, "metadata.agh.hooks", idx, "env", entry["env"]),
 			Timeout: durationValue(skill, "metadata.agh.hooks", idx, "timeout", entry["timeout"]),
 		}
+		if hook.Command == "" {
+			warnAGHMetadata(skill, "skills: invalid metadata.agh.hooks entry", "index", idx, "reason", "missing command")
+			continue
+		}

 		hooks = append(hooks, hook)
 	}
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		hook := HookDecl{
			Event:   event,
			Command: strings.TrimSpace(stringValue(entry["command"])),
			Args:    stringSliceValue(skill, "metadata.agh.hooks", idx, "args", entry["args"]),
			Env:     stringMapValue(skill, "metadata.agh.hooks", idx, "env", entry["env"]),
			Timeout: durationValue(skill, "metadata.agh.hooks", idx, "timeout", entry["timeout"]),
		}
		if hook.Command == "" {
			warnAGHMetadata(skill, "skills: invalid metadata.agh.hooks entry", "index", idx, "reason", "missing command")
			continue
		}

		hooks = append(hooks, hook)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/loader.go` around lines 276 - 285, parseHookDecls is
currently appending HookDecls even when Command is empty; modify parseHookDecls
to validate the built HookDecl (the Command field created via stringValue) and
fail fast: after creating hook := HookDecl{...}, if hook.Command == "" return a
descriptive error (include skill identifier and hook index) instead of
appending; ensure callers of parseHookDecls handle the returned error. This
enforces rejecting empty command entries at parse time rather than deferring to
runtime hook execution.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKhM`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKhM
```

---

_Generated from PR review - CodeRabbit AI_
