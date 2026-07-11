import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { AgentFleetToolbar } from "../agent-fleet-toolbar";

vi.mock("@agh/ui", async importOriginal => {
  const actual = await importOriginal<typeof import("@agh/ui")>();
  return {
    ...actual,
    Filters: ({
      onChange,
      "data-testid": testId,
    }: {
      onChange: (next: { id: string; field: string; operator: string; values: string[] }[]) => void;
      "data-testid"?: string;
    }) => (
      <div data-testid={testId}>
        <button
          data-testid="mock-set-category"
          onClick={() =>
            onChange([
              {
                id: "agent-fleet-filter-category",
                field: "category",
                operator: "is",
                values: ["Ops"],
              },
            ])
          }
          type="button"
        >
          Set category
        </button>
        <button
          data-testid="mock-set-status"
          onClick={() =>
            onChange([
              {
                id: "agent-fleet-filter-status",
                field: "status",
                operator: "is",
                values: ["active"],
              },
            ])
          }
          type="button"
        >
          Set status
        </button>
      </div>
    ),
  };
});

describe("AgentFleetToolbar", () => {
  it("Should forward filter chip changes to category and status handlers", async () => {
    const user = userEvent.setup();
    const onFiltersChange = vi.fn();
    render(
      <AgentFleetToolbar
        categoryOptions={["Ops"]}
        draftQuery=""
        onDraftQueryChange={vi.fn()}
        onFiltersChange={onFiltersChange}
        search={{}}
        searchInputRef={createRef<HTMLInputElement>()}
      />
    );

    await user.click(screen.getByTestId("mock-set-category"));
    expect(onFiltersChange).toHaveBeenCalledWith({ category: "Ops", status: undefined });

    await user.click(screen.getByTestId("mock-set-status"));
    expect(onFiltersChange).toHaveBeenLastCalledWith({ category: undefined, status: "active" });
  });
});
