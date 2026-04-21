# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- `task_01` groundwork is already implemented on the current branch in `internal/config` and `internal/session`; dependent tasks should consume the existing normalized capability catalog and session projection instead of adding parallel loaders or digest logic.
- `task_02` now removes `recipe` from the supported network artifact kinds in `internal/network`; follow-up tasks should treat `kind:"capability"` as the only transfer artifact and assume legacy recipe envelopes are hard-rejected.

## Shared Decisions

- For unified-capabilities follow-up work, transferable `kind:"capability"` payloads must reuse the canonical structured capability shape required by task_01 digest computation. Do not introduce a separate reduced wire-only capability schema even if older techspec snippets appear narrower.
- Network integrity checks for transferred capabilities should continue calling the runtime-owned `internal/config.CanonicalCapabilityDigest` helper rather than reimplementing canonicalization in other packages.
- Network interaction bookkeeping must update on outbound send as well as inbound receive for `direct`, `capability`, `receipt`, and `trace` envelopes that carry an `interaction_id`; otherwise multi-router peers lose sender-side lifecycle state and later capability receipts/traces can be ignored.
- Daemon/API discovery contracts now expose capability discovery as typed fields, not raw peer-card `ext` blobs: `peer_card.capabilities` is the brief typed list and peer detail adds `capability_catalog` for rich discovery. Follow-up frontend/docs work should target those fields and should not read `agh.capabilities_brief` or `agh.capability_catalog` from API-visible `ext`.
- Frontend protocol-kind registries (channel-detail `VALID_KINDS`, design-system showcase `KINDS`, UI kit kind-chip story) must stay in sync with the backend envelope kinds; `recipe` is replaced by `capability` everywhere a static list is kept.

## Shared Learnings

- Local peer cards now advertise `artifacts_supported: ["capability"]` even when the brief capability discovery list is empty; transfer support is protocol-level and should stay separate from discovery inventory size.
- Rich `whois` catalogs can be intentionally filtered; when building API-visible brief discovery, merge rich-catalog summaries over the `greet` brief summaries instead of replacing them, otherwise partial `whois` responses blank unrelated brief capability summaries.

## Open Risks

## Handoffs
