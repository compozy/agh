---
status: resolved
file: internal/extension/host_api.go
line: 768
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:a42c919bcb31
review_hash: a42c919bcb31
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 031: Use h.now() instead of time.Now() for testability.
## Review Comment

The handler has an injectable `now` function but this line uses `time.Now()` directly, making deadline behavior harder to test.

```diff
- deadline := time.Now().Add(seedPollWindow)
+ deadline := h.now().Add(seedPollWindow)
```

---

## Triage

- Decision: `valid`
- Why: `loadPromptSeedEvents` computes its deadline from `time.Now()` even though `HostAPIHandler` already has an injectable clock. That makes time-dependent tests harder to drive deterministically.
- Root cause: The poll window uses the global clock instead of the handler's injected `now` function.
- Fix plan: Compute the deadline from `h.now()` so the whole method follows the handler's clock abstraction.
- Resolution: The seed-event polling loop now uses `h.now()` consistently and the updated handler passed targeted tests and `make verify`.
