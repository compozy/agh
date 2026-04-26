---
status: resolved
file: internal/api/core/tasks_surface_integration_test.go
line: 404
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tj,comment:PRRC_kwDOR5y4QM67YhqJ
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

if [ -f go.mod ]; then
  echo "== go.mod =="
  sed -n '1,20p' go.mod | sed -n '/^go /p'
else
  fd '^go\.mod$' -a -x sh -c '
    echo "== $1 =="
    sed -n "1,20p" "$1" | sed -n "/^go /p"
  ' sh {}
fi
```

Repository: compozy/agh

Length of output: 78

---



</details>

**Add assertions for HTTP response body payloads in execution routes.**

The test asserts HTTP status codes and forwarded `taskpkg.ExecutionRequest` properties, but does not validate the response bodies returned by handlers for publish/start/approve routes. Routes like reject/read/archive/dismiss are tested only for status code with no payload verification. A handler regression that returns unexpected response body content would pass undetected.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_integration_test.go` around lines 335 - 404,
The test currently only checks status codes and the recorded executionRequests
map (via performRequest and executionRequests[tc.call]) but does not assert the
HTTP response body for publish/start/approve ("/tasks/task-1/publish",
"/tasks/task-1/start", "/tasks/task-1/approve") nor for the triage/reject
routes; update the table-driven test in Test (the loop that uses performRequest,
resp and tc) to decode/inspect resp.Body and add assertions: for calls
"publish","start","approve" assert the returned JSON matches the expected
ExecutionRequest fields (idempotency_key, network_channel, metadata) or the
handler's documented response schema, and for
"reject","read","archive","dismiss" assert either the expected empty body or
specific JSON message; use tc.call and tc.wantMetadata/tc.wantKey/tc.wantChannel
to drive the expected body checks so the assertions fail if handlers return
unexpected payloads.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

---

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the execution response payload, not just the status code.**

`publish`, `start`, and `approve` now return a `TaskExecutionResponse`, but these subtests only validate status and the forwarded request. A handler regression that drops the `run` payload, returns the wrong shape, or points the run at the wrong task would still pass here. As per coding guidelines, `MUST test meaningful business logic, not trivial operations` and `Ensure tests verify behavior outcomes, not just function calls`.


<details>
<summary>Suggested test hardening</summary>

```diff
 		t.Run(tc.name, func(t *testing.T) {
 			resp := performRequest(t, fixture.Engine, http.MethodPost, tc.path, tc.body)
 			if resp.Code != tc.want {
 				t.Fatalf("%s status = %d, want %d; body=%s", tc.path, resp.Code, tc.want, resp.Body.String())
 			}
+			if tc.wantKey != "" {
+				var execution contract.TaskExecutionResponse
+				testutil.DecodeJSONResponse(t, resp, &execution)
+				if execution.Task.ID != "task-1" ||
+					execution.Run.TaskID != "task-1" ||
+					execution.Run.ID == "" ||
+					execution.Run.Status != taskpkg.TaskRunStatusQueued {
+					t.Fatalf("%s execution response = %#v", tc.path, execution)
+				}
+			}
 			if tc.wantKey == "" {
 				return
 			}
 			got, ok := executionRequests[tc.call]
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_integration_test.go` around lines 335 - 404,
The tests currently only assert status codes and the forwarded
executionRequests; update each subtest (inside the loop that uses performRequest
and checks executionRequests[tc.call]) to also unmarshal resp.Body into a
TaskExecutionResponse and assert its fields match the expected values
(idempotency key == tc.wantKey, network_channel == tc.wantChannel, metadata ==
tc.wantMetadata, and that the response's run references the correct task ID
"task-1" and/or run/task identifier expected for publish/start/approve). Keep
the existing checks against executionRequests[tc.call] but add these
response-body assertions so handlers returning wrong payload shapes or wrong run
targets will fail; reference the performRequest call, the executionRequests map
lookup, and the TaskExecutionResponse type when locating where to add the
additional assertions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The task mutation integration test asserts status codes and forwarded execution requests, but it does not decode the HTTP response payload for publish/start/approve or the task/triage responses for reject/read/archive/dismiss. Handler regressions that return an empty or wrong JSON body would pass. The fix is to decode and assert response bodies in each subtest.
- Resolution: Added response-body decoding/assertions for execution, task, and triage route responses; verified with the focused integration test and full `make verify`.
