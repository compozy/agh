import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as React from "react";
import { describe, expect, it, vi } from "vitest";

import { DIALOG_TOUCH_TARGET_CLASS } from "../../../lib/dialog-shell";
import { SecretField, type SecretFieldProps } from "../secret-field";

/**
 * Controlled harness — the component is fully controlled, so the value and
 * editing flags must round-trip through state for the rotate/cancel cycle to be
 * observable.
 */
function SecretFieldHarness({
  onValueChangeSpy,
  ...props
}: Partial<SecretFieldProps> & { onValueChangeSpy?: (next: string) => void }) {
  const [value, setValue] = React.useState("");
  const [editing, setEditing] = React.useState(false);
  return (
    <SecretField
      editing={editing}
      id="api-key"
      label="API key"
      onEditingChange={setEditing}
      onValueChange={next => {
        setValue(next);
        onValueChangeSpy?.(next);
      }}
      testIdPrefix="secret"
      value={value}
      {...props}
    />
  );
}

describe("SecretField", () => {
  it("Should render a write-only input on the create path", async () => {
    const user = userEvent.setup();
    render(<SecretFieldHarness />);

    const root = screen.getByTestId("secret");
    expect(root).toHaveAttribute("data-state", "absent");

    const input = screen.getByLabelText("API key");
    expect(input).toHaveAttribute("type", "password");
    expect(input).toHaveValue("");

    await user.type(input, "sk-live");
    expect(screen.getByLabelText("API key")).toHaveValue("sk-live");
    expect(screen.queryByTestId("secret-presence")).not.toBeInTheDocument();
  });

  it("Should expose the write input under a caller-supplied test id", () => {
    // Domains that absorb an existing control keep their bound e2e selectors
    // rather than forcing a spec rewrite.
    render(<SecretFieldHarness inputTestId="vault-secret-value-input" />);

    const input = screen.getByTestId("vault-secret-value-input");
    expect(input).toHaveAttribute("type", "password");
    expect(input).toBe(screen.getByLabelText("API key"));
  });

  it("Should cycle present to editing and back to present on cancel", async () => {
    const user = userEvent.setup();
    const onValueChangeSpy = vi.fn();
    render(
      <SecretFieldHarness
        onValueChangeSpy={onValueChangeSpy}
        present
        presenceLabel="vault:mcp/github"
      />
    );

    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "present");
    expect(screen.getByTestId("secret-presence")).toHaveTextContent("vault:mcp/github");
    expect(screen.queryByLabelText("API key")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("secret-replace"));
    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "editing");
    await user.type(screen.getByLabelText("API key"), "rotated");

    await user.click(screen.getByTestId("secret-cancel"));
    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "present");
    expect(screen.getByTestId("secret-presence")).toHaveTextContent("vault:mcp/github");
    // Cancelling clears the draft so the existing binding is preserved untouched.
    expect(onValueChangeSpy).toHaveBeenLastCalledWith("");
  });

  it("Should never expose a stored value as text", () => {
    render(<SecretFieldHarness present presenceLabel="vault:mcp/github" />);

    const presence = screen.getByTestId("secret-presence");
    expect(presence).toHaveTextContent("••••••••");
    expect(presence.textContent).not.toContain("sk-");
  });

  it("Should report the invalid, saving, and rotated states", () => {
    const { rerender } = render(<SecretFieldHarness error="Value is required." />);
    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "invalid");
    expect(screen.getByText("Value is required.")).toBeInTheDocument();

    rerender(<SecretFieldHarness saving />);
    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "saving");
    expect(screen.getByLabelText("API key")).toBeDisabled();

    rerender(<SecretFieldHarness present rotated />);
    expect(screen.getByTestId("secret")).toHaveAttribute("data-state", "rotated");
    expect(screen.getByText("rotated")).toBeInTheDocument();
  });

  it("Should associate description and error through aria-describedby", () => {
    const { rerender } = render(
      <SecretFieldHarness description="Required by this server." required />
    );

    const input = screen.getByLabelText("API key");
    expect(input).toHaveAttribute("aria-describedby", "api-key-description");
    expect(input).not.toHaveAttribute("aria-invalid");

    rerender(
      <SecretFieldHarness
        description="Required by this server."
        error="Choose a present ref or enter a value."
        required
      />
    );
    const invalidInput = screen.getByLabelText("API key");
    expect(invalidInput).toHaveAttribute("aria-describedby", "api-key-description api-key-error");
    expect(invalidInput).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByText("Choose a present ref or enter a value.")).toHaveAttribute(
      "id",
      "api-key-error"
    );
  });

  it("Should bind an existing reference instead of a typed value", async () => {
    const user = userEvent.setup();
    const onModeChange = vi.fn();
    const onSelectRef = vi.fn();
    render(
      <SecretFieldHarness
        binding={{
          namespaceLabel: "namespace=mcp",
          onSelectRef,
          selectedRef: "",
          sources: [
            { present: true, ref: "vault:mcp/github" },
            { present: false, ref: "vault:mcp/stale" },
          ],
        }}
        mode="source"
        onModeChange={onModeChange}
        sourceModeLabel="Use Vault"
      />
    );

    expect(screen.getByTestId("secret-sources")).toBeInTheDocument();
    expect(screen.queryByLabelText("API key")).not.toBeInTheDocument();

    const options = screen.getAllByRole("radio");
    expect(options).toHaveLength(2);
    expect(options[1]).toBeDisabled();

    await user.click(options[0]);
    expect(onSelectRef).toHaveBeenCalledWith("vault:mcp/github");

    await user.click(screen.getByTestId("secret-mode-value"));
    expect(onModeChange).toHaveBeenCalledWith("value");
  });

  it("Should mark the selected source neutrally, never with accent", () => {
    render(
      <SecretFieldHarness
        binding={{
          onSelectRef: vi.fn(),
          selectedRef: "vault:mcp/github",
          sources: [
            { present: true, ref: "vault:mcp/github" },
            { present: true, ref: "vault:mcp/other" },
          ],
        }}
        mode="source"
        onModeChange={vi.fn()}
      />
    );

    const [selected, unselected] = screen.getAllByRole("radio");
    expect(selected).toHaveAttribute("aria-checked", "true");
    expect(unselected).toHaveAttribute("aria-checked", "false");
    // Accent is reserved for true CTAs; selection reads as glaze + rim.
    expect(selected.className).toContain("data-[selected]:bg-row-selected");
    expect(selected.className).toContain("data-[selected]:border-line-strong");
    expect(selected.className).not.toMatch(/accent/);
    expect(selected).toHaveClass(DIALOG_TOUCH_TARGET_CLASS);
  });
});
