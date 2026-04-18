---
status: resolved
file: internal/api/core/settings.go
line: 861
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRe,comment:PRRC_kwDOR5y4QM65B60B
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep the hook declaration name aligned with the URL key.**

`hooks/:name` is the resource identifier, but `decl.Name` still comes from the request body. That lets callers update one hook while persisting metadata for another name.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 	name, err := requiredSettingsPathValue(c.Param("name"), "name")
 	if err != nil {
 		return settingspkg.CollectionItemPutRequest{}, err
 	}
-	decl, err := hookDeclarationFromPayload(*body.Declaration)
+	bodyName := strings.TrimSpace(body.Declaration.Name)
+	if bodyName != "" && bodyName != name {
+		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
+			fmt.Errorf("hooks.declaration.name must match path name %q", name),
+		)
+	}
+	declaration := *body.Declaration
+	declaration.Name = name
+	decl, err := hookDeclarationFromPayload(declaration)
 	if err != nil {
 		return settingspkg.CollectionItemPutRequest{}, err
 	}
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/settings.go` around lines 849 - 861, The hook declaration's
name must be taken from the URL param instead of trusting the request body:
after obtaining name via requiredSettingsPathValue and parsing decl with
hookDeclarationFromPayload, set decl.Name = name (or the appropriate field on
the parsed declaration struct) before returning the
settingspkg.CollectionItemPutRequest so the returned Hook always uses the URL
key; adjust any validation if you prefer to error on mismatch rather than
silently overwrite.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `parsePutSettingsHookRequest`: the collection item key comes from `:name`, but the parsed hook declaration still trusts `body.Declaration.Name`. I will align the persisted declaration name with the path key and reject mismatches so hook metadata cannot be written under a different identifier.
