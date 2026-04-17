---
status: resolved
file: internal/codegen/sdkts/generate.go
line: 268
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y6EJ,comment:PRRC_kwDOR5y4QM644iq5
---

# Issue 005: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Go's `encoding/json` package, for a struct tag like `json:" spaced_name , omitempty ,string"`, are whitespace around the field name and options trimmed, or are they treated literally? Please answer from the Go standard library documentation or source.`

💡 Result:

In Go's encoding/json package, whitespace around the field name and options in struct tags like json:" spaced_name , omitempty ,string" is treated literally, not trimmed. The tag parser splits on the first comma without trimming spaces, so spaces become part of the field name (" spaced_name ") or options (" omitempty ", " string"), causing invalid tag parsing. The tag is ignored if invalid (e.g., due to space after json: or invalid name), falling back to the field name. This behavior is intentional per official statements: "We want field tag parsing to be as simple as possible: no whitespace or any other syntax beyond what is specified." (Go issue `#14142`). The reflect package docs specify no whitespace in keys/values except space-separated key-value pairs. go vet warns: "struct field tag json: \" spaced_name , omitempty ,string\" not compatible with reflect.StructTag.Get: bad syntax for struct tag value". Source code in encoding/json/tags.go: func parseTag(tag string) (string, tagOptions) { tag, opt, _ := strings.Cut(tag, ",") return tag, tagOptions(opt) } No trimming occurs.

Citations:

- 1: https://github.com/golang/go/issues/14142
- 2: https://groups.google.com/g/golang-nuts/c/AepB-TRRLck
- 3: https://go.dev/src/encoding/json/encode.go
- 4: https://pkg.go.dev/reflect
- 5: https://go.dev/src/encoding/json/tags.go
- 6: https://github.com/golang/go/issues/18531

---

🏁 Script executed:

```shell
cat -n internal/codegen/sdkts/generate.go | sed -n '250,270p'
```

Repository: compozy/agh

Length of output: 594

---



</details>

**Remove `strings.TrimSpace` calls from JSON tag parsing.**

This parser must match `encoding/json` behavior. Go's JSON encoder does not trim whitespace in tag field names or options—spaces are treated literally. Trimming here causes the generated TypeScript schema to diverge from actual marshaled payloads. Fields with accidental spaces in tags get silently normalized instead of failing like the encoder would.

<details>
<summary>Fix</summary>

```diff
- name = strings.TrimSpace(name)
  if name == "" {
    name = field.Name
  }
  omitempty := false
  for opts != "" {
    part := opts
    if next, rest, found := strings.Cut(opts, ","); found {
      part = next
      opts = rest
    } else {
      opts = ""
    }
-   if strings.TrimSpace(part) == "omitempty" {
+   if part == "omitempty" {
      omitempty = true
      break
    }
  }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	name, opts, _ := strings.Cut(tag, ",")
	if name == "" {
		name = field.Name
	}
	omitempty := false
	for opts != "" {
		part := opts
		if next, rest, found := strings.Cut(opts, ","); found {
			part = next
			opts = rest
		} else {
			opts = ""
		}
		if part == "omitempty" {
			omitempty = true
			break
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/codegen/sdkts/generate.go` around lines 252 - 268, The JSON tag
parsing in generate.go currently calls strings.TrimSpace on the tag name and
each option which diverges from encoding/json (spaces are significant); in the
block that sets name, opts and iterates options (variables name, opts, part,
omitempty) remove all strings.TrimSpace calls and compare option strings exactly
(e.g., part == "omitempty") and leave name untouched so whitespace is preserved
exactly as encoding/json would treat it; update the loop that extracts parts
(using strings.Cut) to stop trimming and rely on literal string comparisons so
schema generation matches actual marshaled payloads.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `jsonFieldName` trims both the tag name and each option, which diverges from
  `encoding/json`'s `parseTag` and option matching behavior. That normalization
  changes generated schema field names and `omitempty` detection. Plan:
  preserve literal tag content, compare options exactly, and update tests to
  cover whitespace-containing tags.
