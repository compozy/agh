---
status: resolved
file: internal/daemon/network_e2e_assertions_test.go
line: 50
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEbj,comment:PRRC_kwDOR5y4QM640qz1
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Match the full correlation tuple inside a single transcript surface.**

These checks search each expected attribute independently across the *combined* transcript, so they can pass when `id`, `kind`, `reply-to`, and `trace-id` are spread across different messages. That weakens the helper enough to miss the correlation regressions it is supposed to catch.

<details>
<summary>💡 Suggested fix</summary>

```diff
 func validateNetworkCorrelationSurfaces(
 	messages []transcript.Message,
 	audit []store.NetworkAuditEntry,
 	expectation networkCorrelationExpectation,
 ) error {
-	content := transcriptContent(messages)
-
-	for _, check := range []struct {
+	checks := []struct {
 		label  string
 		needle string
 	}{
 		{label: "message id", needle: attributeNeedle("id", expectation.MessageID)},
 		{label: "kind", needle: attributeNeedle("kind", expectation.Kind)},
 		{label: "interaction", needle: attributeNeedle("interaction", expectation.InteractionID)},
 		{label: "reply-to", needle: attributeNeedle("reply-to", expectation.ReplyTo)},
 		{label: "trace-id", needle: attributeNeedle("trace-id", expectation.TraceID)},
-	} {
-		if check.needle == "" {
-			continue
-		}
-		if !strings.Contains(content, check.needle) {
-			return fmt.Errorf("transcript missing %s %q", check.label, check.needle)
+	}
+
+	matched := false
+	for _, message := range messages {
+		content := strings.TrimSpace(message.Content)
+		if content == "" {
+			continue
+		}
+
+		ok := true
+		for _, check := range checks {
+			if check.needle != "" && !strings.Contains(content, check.needle) {
+				ok = false
+				break
+			}
+		}
+		if ok {
+			matched = true
+			break
 		}
 	}
+	if !matched {
+		return fmt.Errorf("transcript missing correlated attributes for message %q", expectation.MessageID)
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	checks := []struct {
		label  string
		needle string
	}{
		{label: "message id", needle: attributeNeedle("id", expectation.MessageID)},
		{label: "kind", needle: attributeNeedle("kind", expectation.Kind)},
		{label: "interaction", needle: attributeNeedle("interaction", expectation.InteractionID)},
		{label: "reply-to", needle: attributeNeedle("reply-to", expectation.ReplyTo)},
		{label: "trace-id", needle: attributeNeedle("trace-id", expectation.TraceID)},
	}

	matched := false
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		ok := true
		for _, check := range checks {
			if check.needle != "" && !strings.Contains(content, check.needle) {
				ok = false
				break
			}
		}
		if ok {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("transcript missing correlated attributes for message %q", expectation.MessageID)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/network_e2e_assertions_test.go` around lines 33 - 50, The
current check iterates
attributeNeedle("id"/"kind"/"interaction"/"reply-to"/"trace-id") against the
combined transcriptContent(messages), allowing each attribute to match in
different messages; instead ensure the full correlation tuple appears within a
single message surface by either (a) constructing a single combined needle that
includes all non-empty attributeNeedle values from expectation and asserting
strings.Contains(content, combinedNeedle), or (b) iterate the individual message
strings (from messages) and for each message assert that all non-empty
attributeNeedle(...) values are present in that one message; update the loop
around transcriptContent/messages and use attributeNeedle and expectation to
locate and verify the tuple in a single message rather than across the whole
transcript.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the helper currently searches for each expected attribute in the combined transcript text, so different messages can satisfy different parts of the same correlation tuple.
- Fix plan: require one transcript message to contain the full non-empty attribute set and add a regression test that proves split-message matches are rejected.
- Resolution: tightened correlation matching to require a single transcript surface to carry the full attribute tuple and added a negative regression test for split-message matches.
- Verification: `go test ./internal/daemon` passed. Historical note: the earlier `driver/dist/index.js` blocker was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
