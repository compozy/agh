import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { ReasoningCommandSelect } from "../reasoning-command-select";
import type { ReasoningSelectOption } from "../../types";

const options: ReasoningSelectOption[] = [
  { value: "low", label: "Low", source: "catalog" },
  { value: "high", label: "High", source: "acp" },
];

function renderSelect(props: Partial<React.ComponentProps<typeof ReasoningCommandSelect>> = {}) {
  const onChange = props.onChange ?? vi.fn();
  render(
    <UIProvider reducedMotion="always">
      <ReasoningCommandSelect
        options={options}
        value={props.value ?? ""}
        onChange={onChange}
        triggerTestId="trigger"
        {...props}
      />
    </UIProvider>
  );
  return { onChange };
}

describe("ReasoningCommandSelect", () => {
  it("Should map a known effort value to its descriptive label in the trigger", () => {
    renderSelect({ value: "high" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("High");
  });

  it("Should show the disabled hint in the trigger when disabled with a hint", () => {
    renderSelect({ value: "", disabled: true, disabledHint: "Not supported by this model" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("Not supported by this model");
  });

  it("Should select the provider default and emit an empty value", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelect({ value: "high" });
    await user.click(screen.getByTestId("trigger"));
    await user.click(screen.getByTestId("reasoning-command-item-default"));
    expect(onChange).toHaveBeenCalledWith("");
  });

  it("Should select an effort option and expose its source", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelect();
    await user.click(screen.getByTestId("trigger"));
    const item = screen.getByTestId("reasoning-command-item-low");
    expect(item).toHaveAttribute("data-source", "catalog");
    await user.click(item);
    expect(onChange).toHaveBeenCalledWith("low");
  });
});
