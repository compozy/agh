import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { ProviderCommandSelect } from "../provider-command-select";
import type { ProviderSelectOption } from "../../types";

const options: ProviderSelectOption[] = [
  { name: "codex", display_name: "Codex", harness: "openai", runtime_provider: "openai" },
  { name: "claude", display_name: "Claude", harness: "anthropic" },
  { name: "local-acp" },
  { name: "trimmed-acp", display_name: "Trimmed ACP", harness: "   " },
];

function renderSelect(props: Partial<React.ComponentProps<typeof ProviderCommandSelect>> = {}) {
  const onChange = props.onChange ?? vi.fn();
  render(
    <UIProvider reducedMotion="always">
      <ProviderCommandSelect
        options={options}
        value={props.value ?? null}
        onChange={onChange}
        triggerTestId="trigger"
        {...props}
      />
    </UIProvider>
  );
  return { onChange };
}

describe("ProviderCommandSelect", () => {
  it("Should render the selected provider display name and harness in the trigger", () => {
    renderSelect({ value: "codex" });
    const trigger = screen.getByTestId("trigger");
    expect(trigger).toHaveTextContent("Codex");
    expect(trigger).toHaveTextContent("openai");
  });

  it("Should show the placeholder when nothing is selected", () => {
    renderSelect({ value: null, placeholder: "Pick a provider" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("Pick a provider");
  });

  it("Should bucket providers by harness and place harness-less providers under the fallback group", async () => {
    const user = userEvent.setup();
    renderSelect();
    await user.click(screen.getByTestId("trigger"));
    expect(screen.getByTestId("provider-command-group-anthropic")).toHaveTextContent("ANTHROPIC");
    expect(screen.getByTestId("provider-command-group-openai")).toHaveTextContent("OPENAI");
    expect(screen.getByTestId("provider-command-group-general")).toHaveTextContent("Providers");
  });

  it("Should call onChange with the provider name and close the popover on select", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelect();
    await user.click(screen.getByTestId("trigger"));
    await user.click(screen.getByTestId("provider-command-item-claude"));
    expect(onChange).toHaveBeenCalledWith("claude");
    expect(screen.getByTestId("trigger")).toHaveAttribute("aria-expanded", "false");
  });

  it("Should honor a custom testIdPrefix for input and items", async () => {
    const user = userEvent.setup();
    renderSelect({ testIdPrefix: "agent-create-provider" });
    await user.click(screen.getByTestId("trigger"));
    expect(screen.getByTestId("agent-create-provider-input")).toBeInTheDocument();
    expect(screen.getByTestId("agent-create-provider-item-codex")).toBeInTheDocument();
  });

  it("Should render the ACP fallback label for blank harness values", () => {
    renderSelect({ value: "trimmed-acp" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("acp");
  });
});
