---
status: resolved
file: internal/extension/host_api_test.go
line: 1897
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_S2l,comment:PRRC_kwDOR5y4QM65JSEn
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Make the missing-boundary test independent of the current query shape.**

Right now this stub returns different data depending on `query.Limit`, so the test is coupled to how `submitPrompt` currently fetches events rather than to the business condition being tested. If `submitPrompt` changes its lookup strategy, this can fail for the wrong reason.

<details>
<summary>Suggested change</summary>

```diff
-			eventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
-				if query.Limit == 1 {
-					return nil, nil
-				}
-				return []store.SessionEvent{{
+			eventsFn: func(_ context.Context, _ string, _ store.EventQuery) ([]store.SessionEvent, error) {
+				return []store.SessionEvent{{
 					ID:        "ev-agent",
 					Sequence:  1,
 					TurnID:    "turn-agent",
 					Type:      acp.EventTypeAgentMessage,
 					AgentName: "coder",
 					Content:   `{"schema":"agh.session.event.v1","type":"agent_message","text":"reply"}`,
 					Timestamp: time.Date(2026, 4, 18, 14, 6, 0, 0, time.UTC),
 				}}, nil
 			},
```
</details>

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
			eventsFn: func(_ context.Context, _ string, _ store.EventQuery) ([]store.SessionEvent, error) {
				return []store.SessionEvent{{
					ID:        "ev-agent",
					Sequence:  1,
					TurnID:    "turn-agent",
					Type:      acp.EventTypeAgentMessage,
					AgentName: "coder",
					Content:   `{"schema":"agh.session.event.v1","type":"agent_message","text":"reply"}`,
					Timestamp: time.Date(2026, 4, 18, 14, 6, 0, 0, time.UTC),
				}}, nil
			},
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 1885 - 1897, The stubbed
eventsFn should not branch on query.Limit (which couples the test to
submitPrompt's lookup shape); instead make eventsFn return the same event set
required for the missing-boundary scenario regardless of query.Limit (or detect
the semantic intent via the query's filter/turn id if present) so the
missing-boundary test relies on the returned event content (e.g., SessionEvent
with TurnID "turn-agent" and Type acp.EventTypeAgentMessage) rather than the
numeric Limit; update the eventsFn in the test to remove the query.Limit
conditional and always produce the event(s) needed to exercise the
missing-boundary behavior.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The finding is valid. `submitPrompt()` currently calls `latestSessionSequence()`, which probes `sessions.Events()` with `store.EventQuery{Limit: 1}` before it performs the post-prompt fetch. This test encodes that implementation detail by returning `nil` only when `query.Limit == 1`, so a future change to the lookup shape could make the test fail for reasons unrelated to the missing-boundary behavior being exercised.
  The root condition under test is simpler: after prompt submission, the stored event set lacks any prompt-initiating boundary event and therefore `promptSubmissionFromStoredEvents()` must fail with `turn id not found`.
  Fix approach: update the stubbed `eventsFn` so it provides the same boundary-less event payload independent of the numeric query shape, keeping the assertion focused on the returned stored-event content instead of `EventQuery.Limit`.
  Implemented: `TestHostAPIHandlerSubmitPromptRejectsMissingBoundaryEvents` now ignores the incoming `store.EventQuery` and always returns the same boundary-less stored event payload, so the test outcome depends on missing prompt boundary events rather than the current `submitPrompt()` lookup shape.
  Verification:
  - `go test ./internal/extension -run '^TestHostAPIHandlerSubmitPromptRejects(MissingSessionManager|MissingBoundaryEvents)$' -count=1` → `ok  	github.com/pedronauck/agh/internal/extension	0.012s`
  - `make verify` → exit code `0`; web checks passed (`167` test files, `1173` tests), Go lint reported `0 issues`, the Go suite completed with `DONE 5342 tests in 10.590s`, and package boundary checks ended with `OK: all package boundaries respected`.
