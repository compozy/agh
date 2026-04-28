import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PillGroup } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";

describe("SettingsFieldRow", () => {
  it("renders labels, descriptions, controls, and responsive hint copies", () => {
    render(
      <SettingsFieldRow
        label="Default provider"
        description="Used for new sessions"
        hint="CONFIG.TOML"
        control={<input />}
        data-testid="field-row"
      />
    );

    const row = screen.getByTestId("field-row");
    expect(row).toHaveTextContent("Default provider");
    expect(row).toHaveTextContent("Used for new sessions");
    expect(screen.getByLabelText("Default provider")).toBeInTheDocument();
    expect(row).toHaveTextContent("CONFIG.TOML");
  });

  it("forwards the error message when provided", () => {
    render(
      <SettingsFieldRow
        label="API key"
        error="required"
        control={<input />}
        data-testid="field-row"
      />
    );

    const row = screen.getByTestId("field-row");
    expect(row).toHaveTextContent("required");
    expect(screen.getByLabelText("API key")).toHaveAttribute("aria-invalid", "true");
  });

  it("renders inside an @agh/ui Field container (data-slot=field)", () => {
    render(
      <SettingsFieldRow label="Session timeout" control={<input />} data-testid="field-row" />
    );

    expect(screen.getByTestId("field-row")).toHaveAttribute("data-slot", "field");
  });

  it("labels composite control groups with the field label", () => {
    render(
      <SettingsFieldRow
        label="Burst limit"
        description="Applies to queue and request windows"
        control={
          <div>
            <input aria-label="requests" />
            <input aria-label="queue" />
          </div>
        }
        data-testid="field-row"
      />
    );

    expect(screen.getByRole("group", { name: "Burst limit" })).toHaveAttribute(
      "aria-describedby",
      expect.stringContaining("description")
    );
  });

  it("labels custom grouped controls through aria-labelledby", () => {
    render(
      <SettingsFieldRow
        label="Catalog scope"
        control={
          <PillGroup
            items={[
              { value: "global", label: "Global" },
              { value: "workspace", label: "Workspace" },
            ]}
            onChange={() => undefined}
            value="global"
          />
        }
        data-testid="field-row"
      />
    );

    expect(screen.getByRole("group", { name: "Catalog scope" })).toBeInTheDocument();
  });
});
