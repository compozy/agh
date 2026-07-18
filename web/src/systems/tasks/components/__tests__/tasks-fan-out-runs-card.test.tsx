import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { TasksFanOutRunsCard } from "../tasks-fan-out-runs-card";

describe("TasksFanOutRunsCard", () => {
  it("Should submit one designation per non-empty line with explicit Local participation", async () => {
    const user = userEvent.setup();
    const onFanOut = vi.fn().mockResolvedValue({ designation_group_id: "desig_test", runs: [] });

    render(<TasksFanOutRunsCard onFanOut={onFanOut} />);

    await user.click(screen.getByTestId("tasks-fan-out-runs-trigger"));
    expect(screen.queryByTestId("tasks-fan-out-network-channel")).not.toBeInTheDocument();
    await user.clear(screen.getByTestId("tasks-fan-out-designations"));
    await user.type(
      screen.getByTestId("tasks-fan-out-designations"),
      "Investigate data path\n\nValidate UI"
    );
    await user.click(screen.getByTestId("tasks-fan-out-runs-submit"));

    expect(onFanOut).toHaveBeenCalledWith({
      designations: [{ brief: "Investigate data path" }, { brief: "Validate UI" }],
      network_participation: { mode: "local" },
    });
    expect(onFanOut.mock.calls[0]?.[0]).not.toHaveProperty("network_channel");
  });

  it("Should serialize one valid Live participation override for every sibling", async () => {
    const user = userEvent.setup();
    const onFanOut = vi.fn().mockResolvedValue({ designation_group_id: "desig_test", runs: [] });

    render(<TasksFanOutRunsCard onFanOut={onFanOut} />);

    await user.click(screen.getByTestId("tasks-fan-out-runs-trigger"));
    await user.selectOptions(screen.getByTestId("tasks-fan-out-network-mode"), "live");
    await user.type(screen.getByTestId("tasks-fan-out-network-channel"), "release-room");
    await user.click(screen.getByTestId("tasks-fan-out-runs-submit"));

    expect(onFanOut).toHaveBeenCalledWith(
      expect.objectContaining({
        network_participation: {
          mode: "live",
          channel_id: "release-room",
          channel_strategy: "named",
        },
      })
    );
  });

  it("Should keep the dialog open when no assignment is provided", async () => {
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
