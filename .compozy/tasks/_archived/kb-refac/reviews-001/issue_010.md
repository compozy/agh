---
status: resolved
file: internal/cli/skill_workspace.go
line: 438
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrXa,comment:PRRC_kwDOR5y4QM62twcy
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Escape the skill body before embedding it in XML.**

`content` is written verbatim between XML tags. Any `<`, `&`, or embedded HTML in `SKILL.md` will produce invalid XML and break `skill view` consumers.



<details>
<summary>Proposed fix</summary>

```diff
 	builder.WriteString(`">`)
 	builder.WriteString("\n")
-	builder.WriteString(content)
+	builder.WriteString(skillXMLTextReplacer.Replace(content))
 	if !strings.HasSuffix(content, "\n") {
 		builder.WriteString("\n")
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	var builder strings.Builder
	builder.WriteString(`<skill_content name="`)
	builder.WriteString(skillXMLAttributeReplacer.Replace(skill.Meta.Name))
	builder.WriteString(`">`)
	builder.WriteString("\n")
	builder.WriteString(skillXMLTextReplacer.Replace(content))
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("\n<skill_resources>\n")
	for _, resource := range resources {
		builder.WriteString("  <file>")
		builder.WriteString(skillXMLTextReplacer.Replace(resource))
		builder.WriteString("</file>\n")
	}
	builder.WriteString("</skill_resources>\n")
	builder.WriteString("</skill_content>")
	return builder.String(), nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_workspace.go` around lines 421 - 438, The skill body
(variable content) is embedded verbatim into XML causing unescaped characters
like '<' or '&' to break consumers; update the XML assembly to escape the skill
body before writing it (use the existing skillXMLTextReplacer or an equivalent
XML-escaping function) where content is appended to the strings.Builder (the
same way resources use skillXMLTextReplacer.Replace(resource)), ensuring you
replace/escape content prior to checking/adding trailing newlines and before
builder.WriteString(content) so the produced XML is always well-formed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `renderSkillXML` escapes the attribute and resource file nodes, but it writes the skill body text verbatim inside XML. Any `<`, `&`, or similar markup in `SKILL.md` content produces malformed XML for `skill view`.
- Fix approach: Escape the body content with the existing XML text replacer and add a regression test covering embedded XML-sensitive characters.
