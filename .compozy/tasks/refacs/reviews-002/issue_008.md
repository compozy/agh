---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/automation/trigger_filter.go
line: 61
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUs2,comment:PRRC_kwDOR5y4QM6-_G3u
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Invalid filter keys are silently dropped, which can produce false-positive matches.**

Right now, unsupported paths are skipped during compilation. If all paths are invalid, `entries` becomes empty and `matches()` returns `true`, effectively converting an invalid filter into match-all behavior.

 

<details>
<summary>🔧 Suggested fail-closed fix</summary>

```diff
 type triggerFilter struct {
 	entries []triggerFilterEntry
+	invalid bool
 }

 func (f triggerFilter) matches(envelope ActivationEnvelope) bool {
+	if f.invalid {
+		return false
+	}
 	for _, entry := range f.entries {
 		if !entry.matches(envelope) {
 			return false
 		}
 	}
 	return true
 }

 func compileTriggerFilter(filter map[string]string) triggerFilter {
 	if len(filter) == 0 {
 		return triggerFilter{}
 	}
 	entries := make([]triggerFilterEntry, 0, len(filter))
 	for rawPath, rawWant := range filter {
 		entry, ok := triggerFilterEntryFromPath(rawPath, rawWant)
-		if ok {
-			entries = append(entries, entry)
-		}
+		if !ok {
+			return triggerFilter{invalid: true}
+		}
+		entries = append(entries, entry)
 	}
 	return triggerFilter{entries: entries}
 }

 func cloneTriggerFilter(src triggerFilter) triggerFilter {
 	if len(src.entries) == 0 {
-		return triggerFilter{}
+		return triggerFilter{invalid: src.invalid}
 	}
 	entries := make([]triggerFilterEntry, len(src.entries))
 	copy(entries, src.entries)
-	return triggerFilter{entries: entries}
+	return triggerFilter{entries: entries, invalid: src.invalid}
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/automation/trigger_filter.go` around lines 49 - 61,
compileTriggerFilter silently drops invalid keys which can turn an all-invalid
filter into a match-all; update compileTriggerFilter to fail-closed by detecting
unsupported paths from triggerFilterEntryFromPath and returning an error (change
signature to compileTriggerFilter(...)(triggerFilter, error)) or, if changing
signatures is undesirable, return a triggerFilter flagged as invalid that causes
matches() to always return false; ensure you surface the invalid key names in
the returned error or store them on the triggerFilter so callers can log/handle
them instead of getting a silent match-all.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The claimed match-all bug does not reproduce in the current code path.
  - `internal/automation/trigger.go:894-897` falls back to `exactFilterMatch(trigger.Filter, envelope)` whenever `compileTriggerFilter` drops any path, and `exactFilterMatch` fail-closes on unsupported keys because `triggerFilterEntryFromPath` returns `ok=false`.
  - That means an all-invalid compiled filter does not become match-all at runtime; the registration simply does not match. No code change is needed.
  - Resolved as invalid: analysis confirmed the fallback path already fail-closes on invalid filter keys.
