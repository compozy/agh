---
status: resolved
file: extensions/bridges/discord/provider.go
line: 1049
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Odcy,comment:PRRC_kwDOR5y4QM64G4Yv
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Apply DM policy to reaction events too.**

`MESSAGE_REACTION_ADD`/`REMOVE` dispatches immediately after dedup, but unlike `MESSAGE_CREATE` it never calls `allowDiscordDirectMessage`. A blocked DM user can still generate inbound reaction events by reacting to a direct message.

<details>
<summary>Suggested fix</summary>

```diff
 		if cfg.dedup.Mark(mapped.Envelope.IdempotencyKey) {
 			return writeWebhookNoContent(w)
 		}
+		if !allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct) {
+			return writeWebhookNoContent(w)
+		}
 		if err := p.dispatchInboundEnvelope(ctx, cfg.instanceID, mapped.Envelope); err != nil {
 			return &bridgesdk.HTTPError{StatusCode: http.StatusInternalServerError, Message: err.Error()}
 		}
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 1040 - 1049, Reaction
events (handled via mapDiscordReactionEvent and the MESSAGE_REACTION_ADD/REMOVE
flow) are missing the DM policy check present for MESSAGE_CREATE; before
deduping (dedup.Mark) and dispatching (dispatchInboundEnvelope) you must invoke
the same allowDiscordDirectMessage check used for messages to block reactions
from disallowed DM users. Update the reaction handling path to call
allowDiscordDirectMessage with the mapped/envelope context (the same inputs used
for MESSAGE_CREATE), return an appropriate HTTP error or no-content response
when disallowed, and only proceed to dedup.Mark and
p.dispatchInboundEnvelope(mapped.Envelope) if the DM policy allows the event.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The reaction webhook path in `handleEventWebhook` currently unmarshals and maps DM reaction events, then deduplicates and dispatches them without calling `allowDiscordDirectMessage`.
  - `MESSAGE_CREATE` and interaction paths already gate direct-message traffic through DM policy, so reactions are the inconsistent hole.
  - Root cause: the reaction branch never applies the same `allowDiscordDirectMessage(cfg, mapped.User, mapped.Direct)` check returned by `mapDiscordReactionEvent`.
  - Outcome: added the DM policy gate to the reaction branch and extended `extensions/bridges/discord/provider_test.go` to prove disallowed DM reactions do not ingest inbound events. Verified with `go test ./extensions/bridges/discord ./extensions/bridges/gchat` and `make verify`.
