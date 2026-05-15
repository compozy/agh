import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { ModelCommandSelect } from "../model-command-select";
import type { ModelSelectOption } from "../../types";

const options: ModelSelectOption[] = [
  {
    id: "gpt-5.4",
    label: "GPT-5.4",
    availability: { label: "Live", tone: "success", state: "available_live" },
  },
  { id: "gpt-5.4-mini", label: "GPT-5.4 Mini" },
];

function renderSelect(props: Partial<React.ComponentProps<typeof ModelCommandSelect>> = {}) {
  const onChange = props.onChange ?? vi.fn();
  render(
    <UIProvider reducedMotion="always">
      <ModelCommandSelect
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

describe("ModelCommandSelect", () => {
  it("Should show the selected model id in the trigger", () => {
    renderSelect({ value: "gpt-5.4" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("gpt-5.4");
  });

  it("Should show a loading label when loading and no value is selected", () => {
    renderSelect({ value: "", loading: true });
    expect(screen.getByTestId("trigger")).toHaveTextContent("Loading models...");
  });

  it("Should echo the provider default in the trigger and the default item when defaultModel is set", async () => {
    const user = userEvent.setup();
    renderSelect({ value: "", defaultModel: "gpt-5.4" });
    expect(screen.getByTestId("trigger")).toHaveTextContent("gpt-5.4");
    await user.click(screen.getByTestId("trigger"));
    expect(screen.getByTestId("model-command-item-default")).toHaveTextContent("gpt-5.4");
  });

  it("Should clear the value when the provider default item is selected", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelect({ value: "gpt-5.4" });
    await user.click(screen.getByTestId("trigger"));
    await user.click(screen.getByTestId("model-command-item-default"));
    expect(onChange).toHaveBeenCalledWith("");
  });

  it("Should render the availability pill and data-availability for catalog models", async () => {
    const user = userEvent.setup();
    renderSelect();
    await user.click(screen.getByTestId("trigger"));
    expect(screen.getByTestId("model-command-item-gpt-5.4-availability")).toHaveTextContent("Live");
    expect(screen.getByTestId("model-command-item-gpt-5.4")).toHaveAttribute(
      "data-availability",
      "available_live"
    );
  });

  it("Should commit a custom typed model on Enter", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelect();
    await user.click(screen.getByTestId("trigger"));
    await user.type(screen.getByTestId("model-command-input"), "custom-model");
    await user.keyboard("{Enter}");
    expect(onChange).toHaveBeenCalledWith("custom-model");
  });
});
