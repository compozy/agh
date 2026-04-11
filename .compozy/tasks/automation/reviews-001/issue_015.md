---
status: resolved
file: internal/automation/model/template.go
line: 171
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0V,comment:PRRC_kwDOR5y4QM623e7h
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's text/template/parse AST, how are `$var.Field` and `index $var "field"` represented, and would a validator that only resolves FieldNode/ChainNode paths rooted at DotNode miss those cases?`

💡 Result:

$var.Field is represented as a VariableNode with Ident containing both "var" and "Field" (split by "."). index $var "field" is represented as a CommandNode whose Args[0] is a VariableNode for $var, and Args[1] is a StringNode for "field". A validator that only resolves FieldNode/ChainNode paths rooted at DotNode would miss both cases, as they use VariableNode (not DotNode) and CommandNode structures respectively.

Citations:

- 1: https://pkg.go.dev/text/template/parse
- 2: https://pkg.go.dev/text/template/parse@go1.24.2
- 3: https://go.dev/src/text/template/parse/node.go?s=20090:20354
- 4: https://github.com/golang/go/blob/master/src/text/template/parse/node.go
- 5: https://go.dev/src/text/template/exec.go?s=5258:5302

---

🏁 Script executed:

```shell
# Read the template.go file around the lines mentioned in the review
cd /tmp/repo 2>/dev/null || cd . && find . -name "template.go" -path "*/automation/model/*" | head -1 | xargs wc -l
```

Repository: compozy/agh

Length of output: 98

---

🏁 Script executed:

```shell
# Get the file path and read the relevant sections
find . -name "template.go" -path "*/automation/model/*" | head -1 | xargs cat -n | sed -n '145,220p'
```

Repository: compozy/agh

Length of output: 2306

---

🏁 Script executed:

```shell
find . -name "template.go" -path "*/automation/model/*" | head -1 | xargs cat -n | sed -n '100,152p'
```

Repository: compozy/agh

Length of output: 1391

---



</details>

**Variable-rooted lookups bypass the activation-field validator.**

The validator only handles paths rooted at `.`. Templates like `{{$root := .}}{{$root.EnvelopeID}}` or `{{index $root "EnvelopeID"}}` encounter `VariableNode` during traversal, which is not handled in `validateTemplateArg` (lines 153–171) or `templateFieldPath` (lines 174–203). Both cases fall through validation:

- `$var.Field` (VariableNode): Returns `nil` from `validateTemplateArg` without checking
- `index $var "field"` (CommandNode with VariableNode arg): `validateIndexArgs` calls `templateFieldPath(args[0])`, which returns `(nil, false)` for VariableNode; the early return at line 145 skips validation

Invalid trigger templates can pass config validation and fail later at execution time. Please either track variable provenance or conservatively reject `VariableNode`-rooted dereferences and index targets.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/model/template.go` around lines 153 - 171, The validator
currently skips VariableNode roots, so update validateTemplateArg to explicitly
handle *parse.VariableNode* and return an error (or otherwise reject) when a
variable is used as a root for field lookups or indexing; also adjust
templateFieldPath and validateIndexArgs to treat a VariableNode root as a
disallowed/invalid path (i.e., return an error or false that causes validation
failure) instead of silently returning ok, ensuring functions named
validateTemplateArg, templateFieldPath, validateIndexArgs,
validateActivationFieldPath (and the callers
validatePipeNode/validateCommandNode) will conservatively reject any $var.Field
or index $var "field" usages.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The template validator only resolves field/index paths rooted at `.`, so variable-rooted lookups like `$root.Field` and `index $root "field"` bypass static validation and can fail later at execution time. I will conservatively reject variable-rooted dereferences/index targets and extend prompt-validation coverage for those forms.
