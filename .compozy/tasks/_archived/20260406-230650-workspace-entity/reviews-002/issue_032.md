---
status: resolved
file: web/src/systems/agent/components/agent-sidebar-group.test.tsx
line: 106
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoC_,comment:PRRC_kwDOR5y4QM61T6Ii
---

# Issue 032: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Add a no-callback assertion for disabled state.**

Nice coverage for the disabled attribute. Please also assert that clicking does **not** call `onNewSession`, so behavior (not just presentation) is locked down.


<details>
<summary>🧪 Suggested test enhancement</summary>

```diff
 it("disables the new-session action when requested", () => {
-  render(<AgentSidebarGroup agent={mockAgent} newSessionDisabled={true} />);
+  const onNewSession = vi.fn();
+  render(
+    <AgentSidebarGroup
+      agent={mockAgent}
+      onNewSession={onNewSession}
+      newSessionDisabled={true}
+    />
+  );

-  expect(screen.getByTestId("sidebar-group-action")).toBeDisabled();
+  const action = screen.getByTestId("sidebar-group-action");
+  expect(action).toBeDisabled();
+  action.click();
+  expect(onNewSession).not.toHaveBeenCalled();
 });
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  it("disables the new-session action when requested", () => {
    const onNewSession = vi.fn();
    render(
      <AgentSidebarGroup
        agent={mockAgent}
        onNewSession={onNewSession}
        newSessionDisabled={true}
      />
    );

    const action = screen.getByTestId("sidebar-group-action");
    expect(action).toBeDisabled();
    action.click();
    expect(onNewSession).not.toHaveBeenCalled();
  });
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/agent-sidebar-group.test.tsx` around lines
102 - 106, The test for AgentSidebarGroup currently only asserts the button is
disabled; update it to also verify that no callback is invoked by passing a
jest.fn() as onNewSession to the rendered <AgentSidebarGroup agent={mockAgent}
newSessionDisabled={true} onNewSession={onNewSessionMock} /> and then simulate a
click on the element with test id "sidebar-group-action" (e.g., via
userEvent.click or fireEvent.click) and assert that onNewSessionMock was not
called; this ensures both presentation and behavior are covered.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The disabled-state test only checks that the action is visually disabled. It
  does not verify that clicking the control leaves `onNewSession` untouched.
  Plan: pass a spy callback, trigger the click, and assert that no call occurs.
