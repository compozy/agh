---
status: resolved
file: internal/extension/bundle.go
line: 220
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57AO3j,comment:PRRC_kwDOR5y4QM63zyPq
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Check how jobs/triggers/bridges are resolved at activation time
# Looking for EqualFold or case-insensitive lookups

rg -n -C3 'EqualFold|(?i)job.*name|trigger.*name|bridge.*name' --type=go -g '!*_test.go' | head -100
```

Repository: compozy/agh

Length of output: 6724

---

🏁 Script executed:

```shell
#!/bin/bash
# Examine the bundleLookupKey function and lines 165, 196
cat -n internal/extension/bundle.go | sed -n '160,200p'
```

Repository: compozy/agh

Length of output: 1704

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at lines 210-220 and surrounding context for channels
cat -n internal/extension/bundle.go | sed -n '205,230p'
```

Repository: compozy/agh

Length of output: 1318

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at jobs (line 232-245), triggers (247-260), bridges (262-275)
cat -n internal/extension/bundle.go | sed -n '230,280p'
```

Repository: compozy/agh

Length of output: 2339

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for bundleLookupKey definition
rg -n 'func bundleLookupKey' --type=go
```

Repository: compozy/agh

Length of output: 130

---

🏁 Script executed:

```shell
#!/bin/bash
# Get the bundleLookupKey function implementation
cat -n internal/extension/bundle.go | sed -n '371,380p'
```

Repository: compozy/agh

Length of output: 405

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for how channels, jobs, triggers, bridges are resolved/looked up at runtime
rg -n 'channelNames|seenJobs|seenTriggers|seenBridges' internal/extension/ --type=go | head -50
```

Repository: compozy/agh

Length of output: 1512

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for case-insensitive resolution in activation/runtime logic
rg -n -A5 'EqualFold.*job|EqualFold.*trigger|EqualFold.*bridge|EqualFold.*channel' --type=go | head -100
```

Repository: compozy/agh

Length of output: 3133

---

🏁 Script executed:

```shell
#!/bin/bash
# Check how channelNames map is used (line 227, 300)
cat -n internal/extension/bundle.go | sed -n '220,310p'
```

Repository: compozy/agh

Length of output: 4138

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for profile/bundle activation and how channels are looked up at runtime
rg -n 'primary.*channel|channel.*lookup|ResolveChannel' internal/extension/ --type=go | head -50
```

Repository: compozy/agh

Length of output: 516

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for job/trigger/bridge name lookups at runtime
rg -n -B2 -A2 'job.*name.*==|trigger.*name.*==|bridge.*name.*==' internal/extension/ --type=go | head -100
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for bundle/profile activation code - how entities are resolved
rg -n 'ActivateBundle|ApplyBundle|LoadBundle' internal/extension/ --type=go | head -30
```

Repository: compozy/agh

Length of output: 714

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for any EqualFold usage with channel/job names in the entire codebase
rg -n 'EqualFold.*Channel|EqualFold.*Job|EqualFold.*Trigger|EqualFold.*Bridge' --type=go | head -50
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for string comparisons with "==" on channel, job, trigger, bridge names
rg -n 'channel.*==|job.*==.*name|trigger.*==.*name|bridge.*==.*name' internal/extension/ --type=go -i | head -50
```

Repository: compozy/agh

Length of output: 133

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the test that validates case-insensitive bundle detection
cat -n internal/extension/registry_bundles_test.go | sed -n '92,140p'
```

Repository: compozy/agh

Length of output: 1697

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for bundle/profile/channel usage in manager or other runtime code
rg -n 'profile\.Name|channel\.Name|job\.Name|trigger\.Name|bridge\.Name' internal/extension/ --type=go -A2 -B2 | head -100
```

Repository: compozy/agh

Length of output: 2592

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for any case-insensitive comparisons in bundle resolution
rg -n 'strings\.EqualFold' internal/extension/bundle.go --type=go -C3
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the test for case-insensitive profile validation
cat -n internal/extension/registry_bundles_test.go | sed -n '136,200p'
```

Repository: compozy/agh

Length of output: 2098

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for how bundles are activated and how channel/job names are matched
rg -n 'ActivateBundle|ApplyBundle|GetBundle|FindBundle' internal/ --type=go | head -30
```

Repository: compozy/agh

Length of output: 2546

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for any runtime matching of channel/job/trigger/bridge names
rg -n 'strings\.Contains|strings\.Index|strings\.Compare' internal/extension/ --type=go | grep -i 'channel\|job\|trigger\|bridge' | head -30
```

Repository: compozy/agh

Length of output: 445

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for tests related to channel/job/trigger/bridge duplicate detection
rg -n 'duplicate.*channel|duplicate.*job|duplicate.*trigger|duplicate.*bridge|case.*channel|case.*job' internal/extension/registry_bundles_test.go --type=go -i
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Check if there are any tests for case-sensitivity of channels, jobs, triggers, bridges
rg -n 'TestBundle.*Channel|TestBundle.*Job|TestBundle.*Trigger|TestBundle.*Bridge' internal/extension/ --type=go | head -30
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at bundle validation to understand if channel name matching is case-sensitive at runtime
cat -n internal/extension/bundle.go | sed -n '220,240p'
```

Repository: compozy/agh

Length of output: 1053

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for any case-insensitive matching in automation or bundle activation
rg -n 'bundleLookupKey' internal/extension/ --type=go
```

Repository: compozy/agh

Length of output: 278

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at how bundles are activated - search in bundles service
cat -n internal/bundles/service.go | sed -n '1,100p'
```

Repository: compozy/agh

Length of output: 3566

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for where bundle specs are used at runtime
rg -n 'BundleSpec|BundleProfile|BundleJob|BundleChannel|BundleTrigger|BundleBridge' internal/bundles/ --type=go -B2 -A2 | head -80
```

Repository: compozy/agh

Length of output: 4835

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for profile matching or selection logic
rg -n 'profile.*select|profile.*match|profile.*lookup' internal/bundles/ --type=go -i | head -30
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for profile selection/matching logic in bundles service
cat -n internal/bundles/service.go | sed -n '100,300p'
```

Repository: compozy/agh

Length of output: 6568

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for any case-insensitive profile name matching
rg -n 'profile.*name|profileName' internal/bundles/service.go --type=go -B2 -A2 | head -100
```

Repository: compozy/agh

Length of output: 581

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for the resolveActivation function or similar that selects profiles
rg -n 'resolveActivation|selectProfile|findProfile' internal/bundles/service.go --type=go -i
```

Repository: compozy/agh

Length of output: 533

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the findProfile function implementation
cat -n internal/bundles/service.go | sed -n '954,970p'
```

Repository: compozy/agh

Length of output: 745

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at resolveActivation function
cat -n internal/bundles/service.go | sed -n '616,680p'
```

Repository: compozy/agh

Length of output: 2913

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the findBundle function implementation
rg -n 'func findBundle' internal/bundles/service.go --type=go -A10
```

Repository: compozy/agh

Length of output: 509

---



</details>

**Use `bundleLookupKey` for case-insensitive duplicate detection in channels, jobs, triggers, and bridges.**

The duplicate detection for channels (line 216), jobs (line 238), triggers (line 253), and bridges (line 268) uses raw trimmed names with case-sensitive comparison, while bundles (line 165) and profiles (line 196) use `bundleLookupKey` for case-insensitive normalization. This inconsistency allows `MyJob` and `myjob` to coexist in the same bundle during validation, which could create maintenance issues.

Since runtime profile resolution uses `strings.EqualFold()` (line 956), consistency across all entity types would improve clarity and prevent edge cases where configuration with differing cases loads but behaves unexpectedly.

<details>
<summary>Current implementation (case-sensitive for channels)</summary>

```
channelNames := make(map[string]struct{}, len(p.Channels.Items))
for idx, item := range p.Channels.Items {
    name := strings.TrimSpace(item.Name)
    if name == "" {
        return fmt.Errorf("%w: bundle %q profile %q channels[%d].name is required", ErrBundleInvalid, bundleName, p.Name, idx)
    }
    if _, exists := channelNames[name]; exists {
        return fmt.Errorf("%w: bundle %q profile %q channel %q is duplicated", ErrBundleInvalid, bundleName, p.Name, name)
    }
    channelNames[name] = struct{}{}
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/bundle.go` around lines 210 - 220, The
channel/jobs/triggers/bridges duplicate checks currently use raw trimmed names;
change them to normalize keys with bundleLookupKey(name) before storing/checking
in maps (e.g., replace usage in channelNames, jobNames, triggerNames,
bridgeNames checks) so comparisons are case-insensitive and consistent with
bundle/profile validation; keep the same empty-name validation and error
messages (ErrBundleInvalid, bundleName, p.Name) but report the original trimmed
name in the error text while using bundleLookupKey(name) for map lookup.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: bundle and profile names are intentionally matched case-insensitively because they are selected by external request input, but channels, jobs, triggers, and bridges are profile-internal identifiers and are currently validated and referenced with exact trimmed names. The runtime paths in `internal/bundles/service.go` only use case-insensitive lookup for bundle/profile selection, not for these inner resources.
- Why not fixing: changing the inner duplicate checks to be case-insensitive would alter accepted bundle semantics rather than fix a demonstrated bug. It would be a policy change, not a scoped review remediation. There is no evidence in the current runtime that `MyJob` and `myjob` collide or resolve ambiguously today.
- Resolution: no code change. Analysis completed and the existing exact-match semantics were preserved.
