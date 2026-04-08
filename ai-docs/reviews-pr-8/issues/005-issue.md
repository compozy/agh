# Issue 5 - Review Thread Comment

**File:** `internal/skills/provenance.go:79`
**Date:** 2026-04-08 11:02:41 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: `{}` was accepted as a provenance sidecar and could silently produce a zero-value `Provenance`. The loader now validates all required marketplace-management fields immediately after JSON decode.

## Body

_⚠️ Potential issue_ | _🟡 Minor_

**Reject sidecars with missing required provenance fields.**

`json.Unmarshal` accepts `{}` here, so `ReadSidecar` can return a zero-value `Provenance`. Because `internal/skills/registry.go` Lines 366-380 use sidecar-backed provenance to treat a skill as marketplace-managed, a partial `.agh-meta.json` can misclassify a local skill and break later hash/update/remove flows.

<details>
<summary>🩹 Suggested validation</summary>

```diff
 	var provenance Provenance
 	if err := json.Unmarshal(payload, &provenance); err != nil {
 		return nil, fmt.Errorf("skills: parse provenance sidecar %q: %w", sidecarPath, err)
 	}
+	if strings.TrimSpace(provenance.Hash) == "" ||
+		strings.TrimSpace(provenance.Registry) == "" ||
+		strings.TrimSpace(provenance.Slug) == "" ||
+		strings.TrimSpace(provenance.Version) == "" ||
+		provenance.InstalledAt.IsZero() {
+		return nil, fmt.Errorf("skills: invalid provenance sidecar %q: missing required fields", sidecarPath)
+	}

 	return &provenance, nil
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/provenance.go` around lines 74 - 79, After json.Unmarshal
into the provenance variable in ReadSidecar, validate that the required
provenance fields are present (non-zero/empty) and return a descriptive error if
any are missing instead of returning a zero-value Provenance; update ReadSidecar
to check the provenance struct fields that are required by registry.go (e.g.,
the provenance identity/ownership fields used to mark marketplace-managed
skills) and fail fast with fmt.Errorf("skills: invalid provenance sidecar %q:
missing %s", sidecarPath, "<field>") when a required field is empty.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKhW`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKhW
```

---

_Generated from PR review - CodeRabbit AI_
