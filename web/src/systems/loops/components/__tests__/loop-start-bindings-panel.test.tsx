import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { LoopStartBindingsPanel } from "../detail/loop-start-bindings-panel";
import type { LoopBindingRow } from "../../lib/loop-bindings";

const DECLARED = ["manual", "cli", "http", "uds", "native_tool", "schedule"];
const BINDINGS: LoopBindingRow[] = [
  {
    id: "job_nightly",
    name: "nightly",
    kind: "schedule",
    enabled: false,
    meta: "Cron 0 3 * * * · next in 6h",
  },
];

describe("LoopStartBindingsPanel", () => {
  it("Should render the declared start kinds and attached automation rows", () => {
    render(<LoopStartBindingsPanel declaredKinds={DECLARED} bindings={BINDINGS} />);
    const kinds = screen.getAllByTestId("loop-declared-kind").map(node => node.textContent);
    expect(kinds).toEqual(DECLARED);
    const row = screen.getByTestId("loop-binding-row");
    expect(row).toHaveAttribute("data-enabled", "false");
    expect(row).toHaveTextContent("nightly");
    expect(row).toHaveTextContent("Cron 0 3 * * *");
  });

  it("Should show the empty state when no automations are attached", () => {
    render(<LoopStartBindingsPanel declaredKinds={DECLARED} bindings={[]} />);
    expect(screen.getByTestId("loop-bindings-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-binding-row")).not.toBeInTheDocument();
  });

  it("Should gate the Add CTAs to the kinds the allowlist permits", () => {
    const onAddSchedule = vi.fn();
    render(
      <LoopStartBindingsPanel
        declaredKinds={DECLARED}
        bindings={[]}
        onAddSchedule={onAddSchedule}
      />
    );
    // software-delivery declares schedule but not trigger/webhook.
    expect(screen.getByTestId("loop-add-schedule")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-add-trigger")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("loop-add-schedule"));
    expect(onAddSchedule).toHaveBeenCalledTimes(1);
  });

  it("Should offer Add trigger when the loop declares a trigger or webhook start", () => {
    render(<LoopStartBindingsPanel declaredKinds={["manual", "webhook"]} bindings={[]} />);
    expect(screen.getByTestId("loop-add-trigger")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-add-schedule")).not.toBeInTheDocument();
  });
});
