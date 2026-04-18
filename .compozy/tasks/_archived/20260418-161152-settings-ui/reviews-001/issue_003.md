---
status: resolved
file: internal/api/core/settings.go
line: 795
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRW,comment:PRRC_kwDOR5y4QM65B6z1
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject or normalize mismatched MCP server names.**

This endpoint accepts the resource key from `:name` and also forwards `server.name` from the body. A request to `/mcp-servers/foo` with `"name":"bar"` can leave the stored key and payload out of sync depending on which field downstream trusts.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 	name, err := requiredSettingsPathValue(c.Param("name"), "name")
 	if err != nil {
 		return settingspkg.CollectionItemPutRequest{}, err
 	}
+	bodyName := strings.TrimSpace(body.Server.Name)
+	if bodyName != "" && bodyName != name {
+		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
+			fmt.Errorf("mcp-servers.server.name must match path name %q", name),
+		)
+	}
 	target, err := parseSettingsTarget(c.Query("target"))
 	if err != nil {
 		return settingspkg.CollectionItemPutRequest{}, err
 	}
 	server := aghconfig.MCPServer{
-		Name:    strings.TrimSpace(body.Server.Name),
+		Name:    name,
 		Command: strings.TrimSpace(body.Server.Command),
 		Args:    cloneStrings(body.Server.Args),
 		Env:     cloneStringMap(body.Server.Env),
 	}
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	name, err := requiredSettingsPathValue(c.Param("name"), "name")
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	bodyName := strings.TrimSpace(body.Server.Name)
	if bodyName != "" && bodyName != name {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(
			fmt.Errorf("mcp-servers.server.name must match path name %q", name),
		)
	}
	target, err := parseSettingsTarget(c.Query("target"))
	if err != nil {
		return settingspkg.CollectionItemPutRequest{}, err
	}
	server := aghconfig.MCPServer{
		Name:    name,
		Command: strings.TrimSpace(body.Server.Command),
		Args:    cloneStrings(body.Server.Args),
		Env:     cloneStringMap(body.Server.Env),
	}
	if err := server.Validate("server"); err != nil {
		return settingspkg.CollectionItemPutRequest{}, NewSettingsValidationError(err)
	}
	return settingspkg.CollectionItemPutRequest{
		CollectionRequest: req,
		Name:              name,
		Target:            target,
		MCPServer:         &server,
	}, nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/settings.go` around lines 773 - 795, The handler currently
accepts :name and body.Server.Name separately, which can desync stored keys;
after obtaining name via requiredSettingsPathValue, enforce consistency by
normalizing server.Name to the path value (set server.Name = name after
trimming) before running server.Validate and before constructing
settingspkg.CollectionItemPutRequest so the returned MCPServer and the request
Name always match; alternatively, if you prefer strictness, reject when
body.Server.Name is non-empty and doesn't equal name by returning a validation
error from the same spot (use requiredSettingsPathValue, server.Name,
server.Validate, and settingspkg.CollectionItemPutRequest to locate where to
apply the change).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `parsePutSettingsMCPServerRequest`: the request path key and `body.Server.Name` are accepted independently, and the constructed `aghconfig.MCPServer` keeps the body name. I will normalize the stored server name to the path value and reject mismatches so the URL key and persisted payload cannot drift apart.
