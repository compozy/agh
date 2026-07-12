import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Button, Filters, type FilterFieldsConfig } from "@agh/ui";

const FIELDS: FilterFieldsConfig<boolean> = [
  {
    key: "has_work",
    label: "Has work",
    type: "toggle",
  },
];

describe("Filters public surface", () => {
  it("Should resolve Filters from @agh/ui and render an accessible add-filter control", () => {
    const onChange = vi.fn();

    render(
      <Filters<boolean>
        allowMultiple={false}
        fields={FIELDS}
        filters={[]}
        onChange={onChange}
        showSearchInput={false}
        size="sm"
        trigger={
          <Button size="sm" variant="ghost" aria-label="Add filter">
            Filter
          </Button>
        }
      />
    );

    expect(screen.getByRole("button", { name: "Add filter" })).toBeInTheDocument();
  });
});
