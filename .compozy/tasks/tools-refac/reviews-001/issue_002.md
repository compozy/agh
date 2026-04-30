---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/acp/client_test.go
line: 779
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulI4,comment:PRRC_kwDOR5y4QM680KHA
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this case in a `t.Run("Should ...")` subtest.**

Line 731 defines a direct top-level test body; this repo’s test convention requires a `Should ...` subtest wrapper.

<details>
<summary>🔧 Proposed update</summary>

```diff
 func TestStartMCPServersSkipsRemoteTransports(t *testing.T) {
-	t.Parallel()
-
-	driver := New()
-	captureFile := filepath.Join(t.TempDir(), "session-new-mcp.jsonl")
-	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{
-		Cwd: t.TempDir(),
-		Env: helperEnvWithCapture("stream_updates", "", captureFile),
-		MCPServers: []aghconfig.MCPServer{
-			{
-				Name:      "agh-hosted-tools",
-				Transport: aghconfig.MCPServerTransportStdio,
-				Command:   "/bin/agh",
-				Args:      []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"},
-				Env:       map[string]string{"AGH_HOME": "/tmp/agh-home"},
-			},
-			{
-				Name:      "remote-http",
-				Transport: aghconfig.MCPServerTransportHTTP,
-				URL:       "https://example.test/mcp",
-			},
-			{
-				Name:      "remote-sse",
-				Transport: aghconfig.MCPServerTransportSSE,
-				URL:       "https://example.test/sse",
-			},
-		},
-	})
-	defer stopProcess(t, driver, proc)
-
-	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionNew)
-	request := decodeCapturedNewSessionRequest(t, params)
-	if got, want := len(request.MCPServers), 1; got != want {
-		t.Fatalf("session/new mcpServers = %#v, want only hosted stdio entry", request.MCPServers)
-	}
-	stdio := request.MCPServers[0].Stdio
-	if stdio == nil {
-		t.Fatalf("session/new mcpServers[0] = %#v, want stdio variant", request.MCPServers[0])
-	}
-	if stdio.Name != "agh-hosted-tools" || stdio.Command != "/bin/agh" {
-		t.Fatalf("hosted stdio entry = %#v, want hosted command", stdio)
-	}
-	if !slices.Equal(stdio.Args, []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"}) {
-		t.Fatalf("hosted stdio args = %#v, want tool mcp bind args", stdio.Args)
-	}
-	if got, want := len(stdio.Env), 1; got != want || stdio.Env[0].Name != "AGH_HOME" {
-		t.Fatalf("hosted stdio env = %#v, want AGH_HOME only", stdio.Env)
-	}
+	t.Run("ShouldSkipRemoteTransportsAndKeepHostedStdio", func(t *testing.T) {
+		t.Parallel()
+
+		driver := New()
+		captureFile := filepath.Join(t.TempDir(), "session-new-mcp.jsonl")
+		proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{
+			Cwd: t.TempDir(),
+			Env: helperEnvWithCapture("stream_updates", "", captureFile),
+			MCPServers: []aghconfig.MCPServer{
+				{
+					Name:      "agh-hosted-tools",
+					Transport: aghconfig.MCPServerTransportStdio,
+					Command:   "/bin/agh",
+					Args:      []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"},
+					Env:       map[string]string{"AGH_HOME": "/tmp/agh-home"},
+				},
+				{
+					Name:      "remote-http",
+					Transport: aghconfig.MCPServerTransportHTTP,
+					URL:       "https://example.test/mcp",
+				},
+				{
+					Name:      "remote-sse",
+					Transport: aghconfig.MCPServerTransportSSE,
+					URL:       "https://example.test/sse",
+				},
+			},
+		})
+		defer stopProcess(t, driver, proc)
+
+		params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionNew)
+		request := decodeCapturedNewSessionRequest(t, params)
+		if got, want := len(request.MCPServers), 1; got != want {
+			t.Fatalf("session/new mcpServers = %#v, want only hosted stdio entry", request.MCPServers)
+		}
+		stdio := request.MCPServers[0].Stdio
+		if stdio == nil {
+			t.Fatalf("session/new mcpServers[0] = %#v, want stdio variant", request.MCPServers[0])
+		}
+		if stdio.Name != "agh-hosted-tools" || stdio.Command != "/bin/agh" {
+			t.Fatalf("hosted stdio entry = %#v, want hosted command", stdio)
+		}
+		if !slices.Equal(stdio.Args, []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"}) {
+			t.Fatalf("hosted stdio args = %#v, want tool mcp bind args", stdio.Args)
+		}
+		if got, want := len(stdio.Env), 1; got != want || stdio.Env[0].Name != "AGH_HOME" {
+			t.Fatalf("hosted stdio env = %#v, want AGH_HOME only", stdio.Env)
+		}
+	})
 }
```
</details>


As per coding guidelines, `**/*_test.go`: Use `t.Run("Should ...")` subtests with `t.Parallel` by default (opt-out with `t.Setenv`).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client_test.go` around lines 731 - 779, The test
TestStartMCPServersSkipsRemoteTransports is a top-level test body but must
follow the repo convention to use a t.Run("Should ...") subtest; modify the
function to call t.Run("Should skip remote transports when starting MCP
servers", func(t *testing.T) { t.Parallel(); /* existing test body */ }) so the
existing assertions and helper calls (New, startHelperProcess,
captureRequestParams, decodeCapturedNewSessionRequest, stopProcess) move inside
that subtest and retain t.Parallel; ensure you do not duplicate defer
stopProcess outside the subtest and keep helperEnvWithCapture, StartOpts and
MCPServers setup inside the subtest scope.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestStartMCPServersSkipsRemoteTransports` has assertions directly in the top-level test body. AGH test conventions require independent cases inside `t.Run("Should ...")` with `t.Parallel()` unless a legitimate opt-out applies. This test does not use `t.Setenv` or shared mutable state, so the body should move into a named subtest.
