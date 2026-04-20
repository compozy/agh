---
status: resolved
file: packages/ui/src/components/search-input.tsx
line: 53
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcN,comment:PRRC_kwDOR5y4QM65JoyN
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`value ?? ""` makes the component unintentionally controlled even when `value` is omitted**

`value` is optional, but this always passes a `value` prop to `<input>`. That prevents true uncontrolled usage and can cause confusing typing behavior unless the parent fully controls state.



<details>
<summary>Proposed fix</summary>

```diff
 function SearchInput({
   value,
   onChange,
@@
 }: SearchInputProps) {
+  const isControlled = value !== undefined;
+
   return (
@@
       <input
         type="search"
         data-slot="search-input-control"
         placeholder={placeholder}
-        value={value ?? ""}
+        {...(isControlled ? { value } : {})}
         onChange={event => onChange?.(event.target.value)}
         disabled={disabled}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
function SearchInput({
  value,
  onChange,
  placeholder = "Search…",
  kbd,
  className,
  containerClassName,
  disabled,
  ...props
}: SearchInputProps) {
  const isControlled = value !== undefined;

  return (
    <div
      data-slot="search-input"
      data-disabled={disabled ? "true" : undefined}
      className={cn(
        "flex h-9 min-w-0 items-center gap-2 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 text-[13px] text-[color:var(--color-text-primary)] transition-colors focus-within:border-[color:var(--color-accent)] focus-within:ring-1 focus-within:ring-[color:var(--color-accent)]",
        "data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-60",
        containerClassName
      )}
    >
      <SearchIcon
        aria-hidden="true"
        className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
      />
      <input
        type="search"
        data-slot="search-input-control"
        placeholder={placeholder}
        {...(isControlled ? { value } : {})}
        onChange={event => onChange?.(event.target.value)}
        disabled={disabled}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/search-input.tsx` around lines 23 - 53, The input
is being forced into controlled mode by always passing value={value ?? ""} in
the SearchInput component; change the input so it only receives a value prop
when the parent actually provides one (i.e. treat undefined as uncontrolled).
Concretely, in SearchInput update the <input> props to conditionally pass value
only when value !== undefined (and otherwise leave it uncontrolled or use
defaultValue if you want an initial value), e.g. replace value={value ?? ""}
with logic that spreads value only when defined (or uses defaultValue from
props), keeping onChange/disabled as-is.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `value={value ?? \"\"}` forces the `<input>` into controlled mode even when callers omit `value`, which breaks true uncontrolled usage and is the root cause behind the misleading typing test behavior.
  - Fix by only forwarding `value` when the prop is actually provided and otherwise letting the native input remain uncontrolled.
