---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/bundles/lookup.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYRsp,comment:PRRC_kwDOR5y4QM6-67E2
---

# Issue 029: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep the first normalized record instead of overwriting it.**

`exact[key] = idx` changes duplicate handling from first-match-wins to last-match-wins for records that normalize to the same `(extensionName, bundleName)` pair. That can make the indexed path return a different record than the fallback scan and any prior linear lookup behavior.

 
<details>
<summary>💡 Suggested fix</summary>

```diff
 func newBundleRecordLookup(records []resources.Record[BundleResourceSpec]) bundleRecordLookup {
 	exact := make(map[bundleRecordKey]int, len(records))
 	for idx, record := range records {
 		key := newBundleRecordKey(record.Spec.ExtensionName, record.Spec.Bundle.Name)
 		if key.extensionName == "" || key.bundleName == "" {
 			continue
 		}
-		exact[key] = idx
+		if _, exists := exact[key]; !exists {
+			exact[key] = idx
+		}
 	}
 	return bundleRecordLookup{
 		exact:   exact,
 		records: records,
 	}
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	exact := make(map[bundleRecordKey]int, len(records))
	for idx, record := range records {
		key := newBundleRecordKey(record.Spec.ExtensionName, record.Spec.Bundle.Name)
		if key.extensionName == "" || key.bundleName == "" {
			continue
		}
		if _, exists := exact[key]; !exists {
			exact[key] = idx
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bundles/lookup.go` around lines 21 - 27, The map assignment
exact[key] = idx overwrites earlier entries and makes last-match-wins; change it
to preserve the first normalized record by checking for existence before
assignment (use newBundleRecordKey(record.Spec.ExtensionName,
record.Spec.Bundle.Name) as the key and only set exact[key] = idx if key is not
already present), ensuring bundleRecordKey/newBundleRecordKey and the records
iteration keep the first matching index rather than replacing it.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `newBundleRecordLookup` overwrites earlier normalized entries, which makes indexed lookups last-match-wins while the fallback scan remains first-match-wins. That inconsistency can change which bundle record is returned for duplicate normalized keys.
- Fix plan: preserve the first indexed record and add regression coverage. This likely requires a minimal out-of-scope test update in `internal/bundles/service_test.go`, which will be kept tightly scoped to this lookup behavior.
- Resolution: implemented, including the minimal lookup regression test in `internal/bundles/service_test.go`, and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
