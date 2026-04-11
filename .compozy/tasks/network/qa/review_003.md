# QA Review 003

Date: 2026-04-11
Environment: isolated daemon with dedicated home `.tmp/agh-network-stress-home-v3`, custom ports `http.port=2327` and `network.port=4525`, provider `claude`
Status: resolved in this run

## Issue 009: Inbound guidance incorrectly told peers to reuse broadcast `say` interactions for targeted `direct` replies

Severity: high

Evidence:
- In the complex same-daemon stress pass, multiple peers received one broadcast `say` and then replied with `--kind direct`.
- The pre-fix inbound wrapper/footer and bundled `agh-network` skill both implied that any inbound message carrying `interaction` should reuse that `--interaction-id` on replies.
- For a broadcast `say`, that advice was wrong. The first peer to answer with `direct` effectively opened a bilateral interaction under the broadcast interaction id. Later peers replying on the same id were then treated as third-party actors and ignored by lifecycle rules.
- This was a product bug, not random agent behavior: the runtime wrapper and the bundled skill were both teaching the wrong protocol move.

Impact:
- Complex fan-in conversation after a broadcast `say` was unreliable under real LLM use.
- Different peers could follow the product guidance faithfully and still lose replies because lifecycle semantics rejected the shared interaction.

Resolution Notes:
- Fixed in `internal/network/delivery.go` by rendering kind-aware reply guidance.
- Fixed in `internal/skills/bundled/skills/agh-network/SKILL.md` by documenting that replies to broadcast `say` must open a NEW targeted interaction when using `--kind direct`.
- The wrapper now also includes explicit correlation attributes (`to`, `reply-to`, `trace-id`, `causation-id`) so agents can map the protocol state they are actually in.
- Locked in with `internal/network/delivery_test.go` and `internal/skills/bundled/bundled_test.go`.

## Issue 010: Recipe authoring rules were too implicit for agent-authored traffic

Severity: medium

Evidence:
- During recipe-path QA, the required v0 shape is `{"recipe":{...}}`, but the product surface area did not make that requirement explicit enough for LLM usage.
- Pre-fix validation would reject malformed recipe bodies, but it did not fail fast with an actionable message that pointed directly to the required nested shape.
- That is not just a docs nit: this feature is intended for agents, and ambiguous protocol ergonomics directly translate into runtime failures under real use.

Impact:
- LLM-authored `recipe` messages were easier to get wrong than they should be.
- Failures surfaced as protocol rejects instead of clear guidance toward the correct body contract.

Resolution Notes:
- Fixed in `internal/network/validate.go` by rejecting missing nested recipe payloads with an explicit `{"recipe":{...}}` guidance message.
- Fixed in `internal/skills/bundled/skills/agh-network/SKILL.md` by adding recipe examples and a body-rules section covering all relevant message kinds.
- Locked in with `internal/network/validate_test.go` and `internal/skills/bundled/bundled_test.go`.

## Issue 011: Directed `recipe` messages were not treated as interaction openers

Severity: high

Evidence:
- Complex agent workflows need a directed `recipe` to behave like a real targeted interaction so the receiver can answer with protocol `receipt` and `trace`.
- Pre-fix lifecycle code only opened interactions from `direct`, not from directed `recipe`.
- That meant a directed recipe could be delivered, but follow-up lifecycle messages would not attach to a valid interaction consistently.

Impact:
- Directed recipe exchanges could not behave as a first-class protocol conversation.
- Multi-step agent coordination around a recipe artifact was weaker than the RFC/task expectations for audited interaction flow.

Resolution Notes:
- Fixed in `internal/network/lifecycle.go` so both `direct` and directed `recipe` can open an interaction.
- Fixed in `internal/network/router.go` so directed `recipe` participates in lifecycle evaluation and delivery consistently.
- Locked in with `internal/network/lifecycle_test.go`, `internal/network/router_test.go`, and `internal/network/helpers_test.go`.

## Complex Same-Daemon Stress Evidence

Real runtime scenario:
- One isolated daemon hosted five active peers in `space builders`: `alpha9`, `beta9`, `gamma9`, `delta9`, and `epsilon9`.
- Each peer was a live LLM session, not a synthetic mock.
- The coordinator `alpha9` drove a compound workflow that mixed direct request/reply, relay, directed recipe, and broadcast summary fan-in.

Observed successful flow:
- `alpha9 -> beta9` direct codename request:
  - `msg-ad856a9a28b4553b` receipt accepted
  - `msg-1a44c45824491a66` trace working
  - `msg-a63358b770cabff1` direct answer with codename `IRON-VORTEX`
  - `msg-d2363498e7590165` trace completed
- `alpha9 -> beta9 -> gamma9` relay:
  - alpha sent relay request `msg-alpha9-beta-relay` on `int-alpha9-beta-relay`
  - beta relayed to gamma with `msg-d97fdfe89da75b75` on new interaction `int-beta9-gamma9-relay-01`
  - gamma sent a new direct risk note back to alpha with `msg-fb379f79f2f4a54d` on `int-gamma9-alpha9-risk-01`, causally linked to beta’s relay
  - beta also acknowledged the original relay request back to alpha with receipt `msg-ce8d26840d26cdef`
- `alpha9 -> delta9` directed recipe:
  - alpha sent `msg-alpha9-delta-recipe` on `int-alpha9-delta-recipe`
  - delta replied with `msg-fce436429d2a1b17` receipt accepted
  - delta replied with `msg-3cf10337ac044f62` trace working
  - delta returned checklist result `msg-eb1b718b1af0b13f`
  - delta closed with `msg-dbfa8a7480f42669` trace completed
- `alpha9` then broadcast summary `say` `msg-alpha9-summary` on `int-alpha9-summary`
  - `gamma9` replied via new direct interaction `int-gamma9-alpha9-summary-reply-01` with `msg-561f4b62e65e78d6`
  - `beta9` replied via new direct interaction `int-beta9-alpha9-summary-reply-01` with `msg-dee759cf69204a9c`
  - `epsilon9` replied via new direct interaction `int-epsilon9-iron-vortex-summary-reply` with `msg-eaa60ef8e042fc04`
  - `delta9` replied via new direct interaction `int-delta9-summary-ack` with `msg-779601fc5ee5cd0b`
- The coordinator finished locally with `ALPHA9_COMPLETED`.

What this proves:
- More than one peer connected to the same daemon can converse in complex patterns using live LLM sessions.
- Same-daemon fan-out and fan-in works across `direct`, `recipe`, `receipt`, `trace`, and `say`.
- Multi-hop causal exchange works when each targeted leg uses its own interaction correctly.
- After the guidance fix, replies to broadcast `say` no longer collide on one shared interaction.

## Harness/Operator Issues Found During Stressing

These were not product defects and were not recorded as code issues:
- In an earlier `meshqad` run, one local prompt failed to provide `gamma8`'s peer id to `beta8`, so the agent got stuck trying to infer the target from discovery output.
- In one alpha step, my first directed send omitted the required `body.text` field and the daemon correctly rejected it with `network: invalid body: direct text is required`.
- The first relay script for `beta9` told beta to forward to gamma but did not explicitly require an acknowledgement back to alpha, while alpha was waiting for one. That was a harness expectation mismatch, not a runtime bug.

## Coverage and Conclusion

This round specifically answered the open QA question about complex agent interaction:
- yes, live LLM peers on the same daemon can exchange multi-step protocol traffic successfully
- yes, they can relay, acknowledge, emit progress, deliver recipe artifacts, and converge responses back to one coordinator
- the failures that did appear were rooted in product guidance and recipe/lifecycle handling, and those root causes were fixed in this run

Verification for this round:
- targeted protocol tests added for reply guidance, recipe validation, and directed-recipe lifecycle behavior
- live same-daemon stress run completed successfully after fixes
- final repo-wide verification was run after these changes
