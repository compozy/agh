# BUG-20260713-telegram-route-shapes: Telegram guided setup rejects documented direct-message and ordinary-group routes

- **Status:** open
- **Impact (user-side):** Blocks-Completion
- **Severity:** High · **Priority:** P1
- **Persona Affected:** Tessa and Maya while connecting Telegram through the supported setup flow
- **Journey Step:** J-connect-bridge-provider, connect one supported provider with correct secrets and routing
- **Scenarios:** NB-bridge-provider-setup; NB-025; NB-029
- **Found:** 2026-07-13 · **Report:** docs/qa/reports/2026-07-12-hermes-bridge.md
- **Origin:** n/a

## Summary

`agh bridge setup telegram` always creates a routing policy that requires both `group_id` and `thread_id`. The public Telegram guide separately documents a direct-message send with only `--peer-id` and an ordinary group flow where Telegram may provide a group without a forum topic. Both valid provider shapes are rejected before delivery because every dimension enabled by the fixed policy is mandatory.

One Telegram bridge therefore cannot accept the provider's documented direct-message, ordinary-group, and forum-topic shapes. Changing the instance to a peer-only or group-only policy merely moves the failure to another supported Telegram context.

## Reproduction

- **Charter:** CH-guided-setup-credentials / CH-structured-telegram-setup · **Tour:** Task Tour and Integration Tour
- **Environment:** isolated local daemon, current-source CLI, rebuilt Telegram extension, deterministic fake Telegram Bot API

1. Run `agh bridge setup telegram --json` with valid bot and webhook inputs.
2. Inspect the created instance. Its routing policy is `include_group=true`, `include_thread=true`, `include_peer=false`.
3. Enable the bridge and run the public-guide direct-message command:

   ```bash
   agh bridge send-test "$BRIDGE_ID" \
     --message "AGH Telegram connection check" \
     --peer-id "123456789" \
     --json
   ```

4. Observe that AGH rejects the request before the provider call.
5. Repeat with a non-forum group target that has a group ID but no `message_thread_id`.

**Expected:** Guided setup produces a routing contract that accepts Telegram direct messages, ordinary groups, and forum topics while preserving isolation between their actual identities.

**Actual:** The direct-message send fails with `bridges: routing policy requires thread id`; an ordinary group without a topic also lacks the required thread dimension. Only a group-plus-topic target succeeds.

## Evidence

- Guided policy owner: `internal/cli/bridge_setup_helpers.go` returns `RoutingPolicy{IncludeThread: true, IncludeGroup: true}` for Telegram.
- Core validation owner: `internal/bridges/dimensions.go` requires every enabled policy dimension on every route.
- Public direct-message command: `packages/site/content/runtime/core/bridges/setup-telegram.mdx` uses only `--peer-id`.
- QA instance `brg-d7bb61e3599d428a` reproduced the direct-message failure; the same instance delivered only after using group `424242` plus thread `1` (`del-1f277481f33d99a4`).
- The HTTP/UDS parity instance `brg-b10140d065561772` likewise delivered only with group `515151` plus thread `2` (`del-4fbd9bf86c6ae814`).
- No provider request was emitted for the rejected direct-message target.

## Fix

- **Root cause:** `RoutingPolicy` models enabled dimensions as a single conjunction, but Telegram has alternative route shapes: peer, group, or group plus optional topic. The guided setup selected the most specific conjunction and then advertised all provider shapes as supported.
- **Fix commit:** pending. This requires a structural routing-contract design rather than changing the wizard to a different fixed conjunction.
- **Regression test:** extend the canonical routing/CLI setup suites to prove one guided Telegram instance accepts and isolates direct-message, ordinary-group, and forum-topic identities without fabricating missing dimensions.

## Verification

- **Retested:** not yet
- **Result:** Open. Group-plus-topic delivery is green; the documented direct-message and ordinary-group paths remain blocked by the instance policy.

