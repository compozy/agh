---
status: resolved
file: internal/registry/github/client.go
line: 334
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564r_T,comment:PRRC_kwDOR5y4QM63phdy
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Return the first published release instead of discarding it.**

Lines 327-334 already fetch the published releases page, but this branch still returns an error even when a usable release exists. That makes `Info()` and versionless `Download()` fail for repos where `/releases/latest` returns `404` but page 1 still has published releases.

<details>
<summary>🛠️ Suggested fix</summary>

```diff
 	case http.StatusNotFound:
 		if err := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)); err != nil {
 			return nil, err
 		}
 		releases, listErr := c.fetchReleasePage(ctx, repo)
 		if listErr != nil {
 			return nil, listErr
 		}
 		if len(releases) == 0 {
 			return nil, fmt.Errorf("github: repository %q has no published releases", repo.full)
 		}
-		return nil, fmt.Errorf("github: latest release not found for %q", repo.full)
+		latest := releases[0]
+		return &latest, nil
 	default:
 		return nil, responseError(response, "latest release", repo.full)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	case http.StatusNotFound:
		if err := closeResponseBody(response.Body, fmt.Sprintf("latest release response for %q", repo.full)); err != nil {
			return nil, err
		}
		releases, listErr := c.fetchReleasePage(ctx, repo)
		if listErr != nil {
			return nil, listErr
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("github: repository %q has no published releases", repo.full)
		}
		latest := releases[0]
		return &latest, nil
	default:
		return nil, responseError(response, "latest release", repo.full)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 323 - 334, The branch
handling http.StatusNotFound after calling closeResponseBody incorrectly returns
an error even when fetchReleasePage(ctx, repo) returns published releases;
modify the http.StatusNotFound case so that after calling
c.fetchReleasePage(ctx, repo) it returns the first publishable release (from the
releases slice) instead of erroring: call fetchReleasePage, check
len(releases)==0 and only then return the "no published releases" error,
otherwise return that first release (the same shape returned by the normal
latest-release path) so Info() and versionless Download() succeed; changes touch
the http.StatusNotFound branch and use fetchReleasePage, closeResponseBody, and
repo.full to locate code.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `fetchLatestRelease` correctly falls back to the paginated releases endpoint on `404`, but then discards the published release it just fetched.
- Evidence: [`internal/registry/github/client.go`](internal/registry/github/client.go) lines 323-334 return `"latest release not found"` even when `fetchReleasePage` returns a non-empty published release slice.
- Fix plan: return the first published release from the fallback path and add regression coverage for `/releases/latest -> 404` with `/releases -> 200`.
- Resolution: The GitHub client now returns the first published release from the `404` fallback path, with regression coverage for that scenario. Verified with package tests and `make verify`.
