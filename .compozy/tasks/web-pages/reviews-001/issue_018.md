---
status: resolved
file: web/src/systems/bridges/components/bridge-create-dialog.tsx
line: 115
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4a,comment:PRRC_kwDOR5y4QM63ZMIP
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Render an explicit empty state when no providers are available.**

When `providers` is empty, this section collapses to a blank grid and the dialog just leaves the user with a disabled submit button. Add a dedicated empty state so the failure mode is actionable. 

<details>
<summary>Suggested fix</summary>

```diff
-                <div className="grid gap-3 lg:grid-cols-2">
-                  {providers.map(provider => (
-                    <BridgeProviderCard
-                      key={buildBridgeProviderKey(provider)}
-                      onSelect={() =>
-                        onDraftChange({
-                          ...draft,
-                          displayName:
-                            !draft.displayName.trim() ||
-                            draft.displayName.trim() === selectedProvider?.display_name
-                              ? provider.display_name
-                              : draft.displayName,
-                          selectedProviderKey: buildBridgeProviderKey(provider),
-                        })
-                      }
-                      provider={provider}
-                      selected={buildBridgeProviderKey(provider) === draft.selectedProviderKey}
-                    />
-                  ))}
-                </div>
+                {providers.length === 0 ? (
+                  <div className="rounded-xl border border-dashed border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-8 text-center text-sm text-[color:var(--color-text-secondary)]">
+                    No bridge providers are currently available.
+                  </div>
+                ) : (
+                  <div className="grid gap-3 lg:grid-cols-2">
+                    {providers.map(provider => (
+                      <BridgeProviderCard
+                        key={buildBridgeProviderKey(provider)}
+                        onSelect={() =>
+                          onDraftChange({
+                            ...draft,
+                            displayName:
+                              !draft.displayName.trim() ||
+                              draft.displayName.trim() === selectedProvider?.display_name
+                                ? provider.display_name
+                                : draft.displayName,
+                            selectedProviderKey: buildBridgeProviderKey(provider),
+                          })
+                        }
+                        provider={provider}
+                        selected={buildBridgeProviderKey(provider) === draft.selectedProviderKey}
+                      />
+                    ))}
+                  </div>
+                )}
```
</details>

As per coding guidelines, `web/src/**/components/**/*.tsx`: "Handle all states in components — loading, error, and empty (never assume `data` exists)".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/bridges/components/bridge-create-dialog.tsx` around lines 85
- 115, The Provider selection grid currently renders nothing when providers is
empty, leaving the dialog blank; update the JSX around providers.map (where
BridgeProviderCard is rendered and buildBridgeProviderKey / onDraftChange /
draft.selectedProviderKey are used) to detect providers.length === 0 and render
an explicit empty state UI (e.g., a short message explaining no providers are
available, optional CTA to refresh or navigate to provider setup, and keep
submit disabled) so the component handles the empty-data state instead of
showing an empty grid.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: if the provider list becomes empty while the create dialog is open, the component renders a blank provider section with no explicit explanation or action state.
- Fix approach: render an explicit empty state in the provider section while keeping submission unavailable.
