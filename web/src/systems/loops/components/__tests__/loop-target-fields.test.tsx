import { useState } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { vi } from "vitest";

vi.mock("../../hooks/use-loops", async () => {
  const { loopCatalogFixtures } = await import("../../mocks/fixtures");
  return { useLoops: () => ({ data: loopCatalogFixtures, isLoading: false }) };
});

const { LoopTargetFields } = await import("../target/loop-target-fields");
type LoopTargetDraft = import("../../lib/loop-target").LoopTargetDraft;

function Harness({ showMapping }: { showMapping?: boolean }) {
  const [value, setValue] = useState<LoopTargetDraft>({
    loop_name: "",
    inputs: {},
    input_mapping: {},
  });
  return (
    <LoopTargetFields
      workspaceId="ws"
      value={value}
      onChange={setValue}
      showMapping={showMapping}
    />
  );
}

describe("LoopTargetFields", () => {
  it("Should list selectable loops and auto-generate a typed input form for the chosen loop", () => {
    render(<Harness />);
    const select = screen.getByTestId("loop-target-select");
    expect(select).toHaveTextContent("software-delivery");
    // No inputs render until a loop is picked.
    expect(screen.queryByTestId("loop-target-inputs")).not.toBeInTheDocument();

    fireEvent.change(select, { target: { value: "software-delivery" } });
    const controls = screen.getAllByTestId("loop-input-control").map(node => node.dataset.input);
    expect(controls).toEqual(["goal", "max_files"]);
    expect(screen.getByTestId("loop-input-field-goal")).toBeInTheDocument();
    expect(screen.getByTestId("loop-input-field-max_files")).toHaveAttribute("type", "number");
  });

  it("Should write static input values onto the loop target", () => {
    render(<Harness />);
    fireEvent.change(screen.getByTestId("loop-target-select"), {
      target: { value: "software-delivery" },
    });
    fireEvent.change(screen.getByTestId("loop-input-field-goal"), {
      target: { value: "ship it" },
    });
    expect((screen.getByTestId("loop-input-field-goal") as HTMLInputElement).value).toBe("ship it");
  });

  it("Should render the event-payload mapping table only when enabled", () => {
    const { rerender } = render(<Harness />);
    fireEvent.change(screen.getByTestId("loop-target-select"), {
      target: { value: "software-delivery" },
    });
    expect(screen.queryByTestId("loop-input-mapping")).not.toBeInTheDocument();

    rerender(<Harness showMapping />);
    fireEvent.change(screen.getByTestId("loop-target-select"), {
      target: { value: "software-delivery" },
    });
    expect(screen.getByTestId("loop-input-mapping")).toBeInTheDocument();
    expect(screen.getByTestId("loop-mapping-field-goal")).toBeInTheDocument();
  });
});
