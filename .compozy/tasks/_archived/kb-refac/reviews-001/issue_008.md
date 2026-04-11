---
status: resolved
file: internal/cli/skill_marketplace.go
line: 570
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrXB,comment:PRRC_kwDOR5y4QM62twcW
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/cli/skill_marketplace.go | sed -n '540,575p'
```

Repository: compozy/agh

Length of output: 1229

---

🏁 Script executed:

```shell
# Find related test files
fd -t f "test" internal/cli/ -o fd -t f "*_test.go" internal/cli/
```

Repository: compozy/agh

Length of output: 178

---

🏁 Script executed:

```shell
# Search for tests or usages of versionIsNewer
rg "versionIsNewer|normalizeVersion|parseVersionParts" internal/cli/ -A 2 -B 2
```

Repository: compozy/agh

Length of output: 2581

---

🏁 Script executed:

```shell
# Get the complete implementation of helper functions
sed -n '573,650p' internal/cli/skill_marketplace.go
```

Repository: compozy/agh

Length of output: 777

---

🏁 Script executed:

```shell
# Get parseVersionParts implementation
sed -n '600,640p' internal/cli/skill_marketplace.go
```

Repository: compozy/agh

Length of output: 179

---

🏁 Script executed:

```shell
# Let's trace through the logic with a test case
python3 << 'EOF'
# Simulate the parseVersionParts logic
def parse_version_parts(version):
    segments = version.split(".")
    if len(segments) == 0:
        return None, False
    
    parts = []
    for segment in segments:
        if segment == "":
            return None, False
        try:
            value = int(segment)
            parts.append(value)
        except ValueError:
            return None, False
    return parts, True

# Test with pre-release version
test_cases = [
    ("1.0.0", "1.0.0"),
    ("1.0.0", "1.0.0-rc1"),
    ("1.0.0-rc1", "1.0.0"),
]

for current, latest in test_cases:
    curr_parts, curr_numeric = parse_version_parts(current)
    latest_parts, latest_numeric = parse_version_parts(latest)
    
    print(f"\nComparing '{current}' vs '{latest}':")
    print(f"  current: parts={curr_parts}, numeric={curr_numeric}")
    print(f"  latest:  parts={latest_parts}, numeric={latest_numeric}")
    
    if curr_numeric and latest_numeric:
        print(f"  → Uses numeric comparison")
    else:
        print(f"  → Falls back to string comparison")
        string_result = latest > current
        print(f"    '{latest}' > '{current}' = {string_result}")
        
        # In semver, 1.0.0 should be > 1.0.0-rc1
        if current == "1.0.0-rc1" and latest == "1.0.0":
            print(f"    Semantic expectation: latest (1.0.0) > current (1.0.0-rc1) = True")
            print(f"    Actual result matches expectation: {string_result == True}")
EOF
```

Repository: compozy/agh

Length of output: 675

---

🏁 Script executed:

```shell
# Check the existing tests more thoroughly
sed -n '/func TestCriticalWarnings/,/^}/p' internal/cli/skill_test.go | head -50
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for any version comparison tests
rg -B5 -A5 "versionIsNewer" internal/cli/skill_test.go
```

Repository: compozy/agh

Length of output: 720

---

🏁 Script executed:

```shell
# Check where versionIsNewer is used and if there's any context
rg -B10 -A5 "versionIsNewer" internal/cli/skill_marketplace.go | grep -A10 -B10 "versionIsNewer(currentVersion"
```

Repository: compozy/agh

Length of output: 695

---

🏁 Script executed:

```shell
# Check if there are any handling of pre-release versions elsewhere
rg "rc|alpha|beta|pre|release" internal/cli/skill_marketplace.go
```

Repository: compozy/agh

Length of output: 1750

---



</details>

**The string comparison fallback is semantically incorrect for pre-release versions.**

When either version contains a non-numeric suffix (e.g., pre-release tags like `-rc1`), the comparison falls back to lexical string ordering. This produces wrong results for semantic versioning—specifically, `"1.0.0" > "1.0.0-rc1"` evaluates to `false` when semantically a release should be newer than its pre-release. This prevents users with pre-release versions from upgrading to released versions.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_marketplace.go` around lines 544 - 570, The fallback
string comparison in versionIsNewer is incorrect for pre-release semantics;
update versionIsNewer (and helper parseVersionParts/normalizeVersion if needed)
to detect and handle pre-release suffixes instead of blindly comparing
normalized strings: after comparing numeric parts, if one version has a
pre-release tag and the other does not, treat the one without the tag as newer;
if both have pre-release tags, compare those tags using semver rules (split on
dots/hyphens and compare identifiers numerically when numeric, lexically
otherwise). Ensure versionPartAt/parseVersionParts expose or return the
pre-release portion so versionIsNewer can apply this logic rather than using
normalizedLatest > normalizedCurrent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `versionIsNewer` falls back to lexical string comparison when parsing fails, which is incorrect for semantic-version prerelease rules. That makes released versions compare incorrectly against prerelease builds such as `1.0.0-rc1`.
- Fix approach: Implement semver-aware prerelease comparison, keep the simple numeric comparison for plain dotted versions, and extend the CLI tests with prerelease coverage.
