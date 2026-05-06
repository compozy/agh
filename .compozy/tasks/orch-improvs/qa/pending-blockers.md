# QA Re-Scope: Slack Bridge Terminal Delivery

## Status

`RESCOPED / NO OPEN BLOCKER FOR THIS QA RUN`.

The original Slack terminal-delivery lane was not validated. The operator explicitly
re-scoped this QA run with the instruction: `não temos slack para testar, pule isso`.
Therefore Slack is recorded as skipped by operator decision, not as passed.

## Skipped Lane

- Test cases: `TC-INT-003` and `TC-SCEN-003`.
- Production bridge target: Slack.
- Bridge instance: `brg-7b3e5ec6ee5cc52e`.
- Task notification subscription: `subscription-bridge-task-terminal-prime-agent`.
- Task: `task-orch-prime-agent`.

Historical evidence remains in the QA lab showing why Slack could not be exercised:
the bridge reached `auth_required`, secret bindings were empty, the `bridges` vault
namespace was empty, and the Slack cursor remained at `last_sequence: 0`.

## Replacement Evidence For This Run

The required bridge terminal-delivery/cursor primitive was exercised through a
reachable non-Slack bridge provider in the same QA lab:

- Provider: Telegram local bridge adapter.
- Bridge instance: `brg-068c83126dbe2010`.
- Successful task: `task-telegram-delivery-qa-r2`.
- Successful run: `run-74ea41b1513bcc00`.
- Successful subscription: `subscription-telegram-terminal-delivery-qa-r2`.
- Cursor evidence: `qa-artifacts/qa/cli-telegram-task-notification-list-after-delivery-qa-r2.json`.
- Provider call evidence: `qa-artifacts/qa/mock-telegram-calls-after-delivery-qa-r2.json`.

Observed result:

- The Telegram bridge was `ready`.
- The terminal task completion produced a real extension-backed `sendMessage`.
- The durable cursor advanced to `last_sequence: 68`.
- The delivery id was `notif:subscription-telegram-terminal-delivery-qa-r2:68`.

Failure behavior was also probed:

- Failure task: `task-telegram-delivery-fail-qa`.
- Failure run: `run-5aefeb61847b80ec`.
- Failure subscription: `subscription-telegram-terminal-failure-probe`.
- Cursor evidence: `qa-artifacts/qa/cli-telegram-task-notification-list-after-failure-probe.json`.

Observed result:

- When the bridge delivery was not accepted, the cursor stayed at `last_sequence: 0`.
- A `last_error` was recorded for the subscription.

## Future Slack Validation

This file no longer blocks the current re-scoped QA run. A future Slack-specific
validation still requires:

```yaml
bridge_instance_id: brg-7b3e5ec6ee5cc52e
secret_bindings:
  bot_token: "Slack bot OAuth token with chat:write access"
  signing_secret: "Slack app signing secret"
target:
  peer_id: UQAOPS
  thread_id: launch-agent
  mode: reply
conditional_inbound_validation:
  public_https_webhook_or_tunnel: "required only if inbound Slack callback validation is included"
```

Do not claim Slack itself is operational until that future provider-specific
run captures a confirmed Slack delivery and corresponding cursor advancement.
