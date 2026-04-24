---
status: resolved
file: web/src/systems/network/mocks/fixtures.ts
line: 243
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIev,comment:PRRC_kwDOR5y4QM66CAla
---

# Issue 025: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Keep the mock message payloads aligned with the server mapper.**

`NetworkChannelMessagePayloadFromEntry` only exposes `session_id` for local/sent messages, and it prefers the session name for local `display_name`s when a session is known. These fixtures give received messages remote `session_id`s and keep local `display_name`s as peer-card names, so Storybook/tests can pass against payloads the API never returns.



Also applies to: 245-276

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/mocks/fixtures.ts` around lines 191 - 243, The
fixture networkChannelMessagesFixture contains entries whose fields don't match
the server mapper: update each message so only local/sent messages include
session_id and their display_name uses the session name (not the peer card name)
when a session is known, and ensure received messages (direction: "received") do
NOT carry session_id; adjust the three entries (message_id "msg_storybook_1",
"msg_storybook_2", "msg_storybook_3") accordingly and apply the same corrections
to the later fixtures in the 245-276 range so the NetworkChannelMessage shape
matches NetworkChannelMessagePayloadFromEntry and the server-mapped payloads the
app expects.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- The mock payloads in `web/src/systems/network/mocks/fixtures.ts:191-276` diverge from `internal/api/core/network_details.go:986-1033`, which only exposes `session_id` for local/sent messages and rewrites local display names from the owning session when available.
- The current fixtures assign remote `session_id` values and keep local display names at the peer-card label, so Storybook/tests can pass against payloads the API never emits.
- Fix approach: align both channel and peer message fixtures with the server mapper semantics and add regression coverage that asserts the fixture contract directly.

## Resolution

- Updated the network message fixtures so local/sent entries use the session name (`Storybook rollout`) and only local/sent entries retain `session_id`.
- Added minimal out-of-scope regression coverage in `web/src/systems/network/mocks/network-mocks.test.ts` because the batch scope did not include an existing mock-contract test file.
- Verified with `bun x vitest run src/systems/network/mocks/network-mocks.test.ts`, `make web-test`, and `make verify`.
