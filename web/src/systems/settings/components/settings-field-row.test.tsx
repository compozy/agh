import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SettingsFieldRow } from "./settings-field-row";

describe("SettingsFieldRow", () => {
  it("renders labels, descriptions, controls, and responsive hint copies", () => {
    render(
      <SettingsFieldRow
        label="Default provider"
        description="Used for new sessions"
        hint="CONFIG.TOML"
        control={<input aria-label="provider" />}
        data-testid="field-row"
      />
    );

    const row = screen.getByTestId("field-row");
    expect(row).toHaveTextContent("Default provider");
    expect(row).toHaveTextContent("Used for new sessions");
    expect(screen.getByLabelText("provider")).toBeInTheDocument();
    expect(row).toHaveTextContent("CONFIG.TOML");
  });
});
