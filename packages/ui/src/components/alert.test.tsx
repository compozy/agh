import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Alert, AlertDescription, AlertTitle } from "./alert";

describe("Alert", () => {
  it("renders role=alert by default and forwards class + data-variant", () => {
    render(
      <Alert data-testid="alert" variant="warning" className="ring-1">
        <AlertTitle>Heads up</AlertTitle>
        <AlertDescription>Restart required</AlertDescription>
      </Alert>
    );
    const alert = screen.getByTestId("alert");
    expect(alert).toHaveAttribute("role", "alert");
    expect(alert).toHaveAttribute("data-variant", "warning");
    expect(alert).toHaveAttribute("data-slot", "alert");
    expect(alert).toHaveClass("ring-1");
  });

  it("accepts a role override for non-danger tones (e.g. status)", () => {
    render(
      <Alert data-testid="alert" variant="success" role="status">
        <AlertTitle>OK</AlertTitle>
      </Alert>
    );
    expect(screen.getByTestId("alert")).toHaveAttribute("role", "status");
  });

  it("supports the new semantic variants (success/warning/info/accent/destructive)", () => {
    for (const variant of ["success", "warning", "info", "accent", "destructive"] as const) {
      const { unmount } = render(
        <Alert data-testid={`alert-${variant}`} variant={variant}>
          <AlertTitle>ok</AlertTitle>
        </Alert>
      );
      expect(screen.getByTestId(`alert-${variant}`)).toHaveAttribute("data-variant", variant);
      unmount();
    }
  });
});
