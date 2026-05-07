import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsPageActions } from "../settings-page-actions";
import type { RestartBannerState } from "../settings-restart-banner";

function makeRestart(overrides: Partial<RestartBannerState> = {}): RestartBannerState {
  return {
    isVisible: true,
    isRestartRequired: false,
    isPolling: false,
    isSuccessful: false,
    isFailed: false,
    operationId: null,
    status: null,
    failureReason: undefined,
    activeSessionCount: 0,
    trigger: vi.fn(),
    isTriggerPending: false,
    triggerError: null,
    dismiss: vi.fn(),
    ...overrides,
  };
}

describe("SettingsPageActions", () => {
  it("renders Restart daemon with the outline Button variant and calls restart.trigger on click", () => {
    const restart = makeRestart();
    render(<SettingsPageActions slug="general" restart={restart} />);
    const btn = screen.getByTestId("settings-page-general-restart-action");
    expect(btn).toHaveTextContent("Restart daemon");
    expect(btn).not.toBeDisabled();
    expect(btn).toHaveAttribute("data-slot", "button");
    // buttonVariants outline variant injects the `border` class.
    expect(btn.className).toContain("border");
    fireEvent.click(btn);
    expect(restart.trigger).toHaveBeenCalledTimes(1);
  });

  it("disables the Restart action while triggering", () => {
    render(
      <SettingsPageActions slug="general" restart={makeRestart({ isTriggerPending: true })} />
    );
    const btn = screen.getByTestId("settings-page-general-restart-action");
    expect(btn).toBeDisabled();
    expect(btn).toHaveTextContent("Restarting…");
  });

  it("disables the Restart action while polling", () => {
    render(<SettingsPageActions slug="general" restart={makeRestart({ isPolling: true })} />);
    expect(screen.getByTestId("settings-page-general-restart-action")).toBeDisabled();
  });

  it("renders an optional secondary action before the restart button", () => {
    render(
      <SettingsPageActions
        slug="general"
        restart={makeRestart()}
        secondaryAction={<button data-testid="page-secondary-action">View logs</button>}
      />
    );
    expect(screen.getByTestId("page-secondary-action")).toBeInTheDocument();
  });
});
