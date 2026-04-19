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

  it("forwards the error message when provided", () => {
    render(
      <SettingsFieldRow
        label="API key"
        error="required"
        control={<input aria-label="api-key" />}
        data-testid="field-row"
      />
    );

    const row = screen.getByTestId("field-row");
    expect(row).toHaveTextContent("required");
  });

  it("renders inside an @agh/ui Field container (data-slot=field)", () => {
    render(
      <SettingsFieldRow
        label="Session timeout"
        control={<input aria-label="timeout" />}
        data-testid="field-row"
      />
    );

    expect(screen.getByTestId("field-row")).toHaveAttribute("data-slot", "field");
  });
});
