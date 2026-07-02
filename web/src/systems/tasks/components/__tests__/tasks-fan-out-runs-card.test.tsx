import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { TasksFanOutRunsCard } from "../tasks-fan-out-runs-card";

describe("TasksFanOutRunsCard", () => {
  it("submits network channel and one designation per non-empty line", async () => {
    const user = userEvent.setup();
    const onFanOut = vi.fn().mockResolvedValue({ designation_group_id: "desig_test", runs: [] });

    render(<TasksFanOutRunsCard defaultNetworkChannel="general" onFanOut={onFanOut} />);

    await user.click(screen.getByTestId("tasks-fan-out-runs-trigger"));
    await user.clear(screen.getByTestId("tasks-fan-out-designations"));
    await user.type(
      screen.getByTestId("tasks-fan-out-designations"),
      "Investigate data path\n\nValidate UI"
    );
    await user.click(screen.getByTestId("tasks-fan-out-runs-submit"));

    expect(onFanOut).toHaveBeenCalledWith({
      network_channel: "general",
      designations: [{ brief: "Investigate data path" }, { brief: "Validate UI" }],
    });
  });

  it("keeps the dialog open when no assignment is provided", async () => {
    const user = userEvent.setup();
    const onFanOut = vi.fn();

    render(<TasksFanOutRunsCard onFanOut={onFanOut} />);

    await user.click(screen.getByTestId("tasks-fan-out-runs-trigger"));
    await user.clear(screen.getByTestId("tasks-fan-out-designations"));
    await user.click(screen.getByTestId("tasks-fan-out-runs-submit"));

    expect(onFanOut).not.toHaveBeenCalled();
    expect(screen.getByTestId("tasks-fan-out-runs-error")).toHaveTextContent(
      "Add at least one assignment."
    );
  });
});
