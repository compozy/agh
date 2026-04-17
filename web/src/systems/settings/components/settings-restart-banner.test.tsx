import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsRestartBanner } from "./settings-restart-banner";

type BannerOverrides = Parameters<typeof SettingsRestartBanner>[0]["restart"];

function makeRestart(overrides: Partial<BannerOverrides> = {}): BannerOverrides {
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

describe("SettingsRestartBanner", () => {
  it("renders nothing when the banner state is not visible", () => {
    render(<SettingsRestartBanner slug="general" restart={makeRestart({ isVisible: false })} />);

    expect(screen.queryByTestId("settings-page-general-restart-banner")).not.toBeInTheDocument();
  });

  it("renders the restart-required warning state with a trigger button when required", () => {
    const restart = makeRestart({ isRestartRequired: true });
    render(<SettingsRestartBanner slug="memory" restart={restart} />);

    const banner = screen.getByTestId("settings-page-memory-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "warning");
    fireEvent.click(screen.getByTestId("settings-page-memory-restart-banner-trigger"));
    expect(restart.trigger).toHaveBeenCalled();
  });

  it("renders the polling state without a trigger button while the restart is running", () => {
    render(
      <SettingsRestartBanner
        slug="observability"
        restart={makeRestart({
          isRestartRequired: true,
          isPolling: true,
          status: "stopping",
          operationId: "op_1",
        })}
      />
    );

    expect(
      screen.getByTestId("settings-page-observability-restart-banner-message")
    ).toHaveTextContent("Restarting daemon · stopping");
    expect(
      screen.queryByTestId("settings-page-observability-restart-banner-trigger")
    ).not.toBeInTheDocument();
  });

  it("renders the failure banner with the failure reason and a dismiss action", () => {
    const restart = makeRestart({
      isRestartRequired: true,
      isFailed: true,
      failureReason: "helper exited non-zero",
    });
    render(<SettingsRestartBanner slug="general" restart={restart} />);

    const banner = screen.getByTestId("settings-page-general-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "danger");
    expect(banner).toHaveTextContent("helper exited non-zero");

    fireEvent.click(screen.getByTestId("settings-page-general-restart-banner-dismiss"));
    expect(restart.dismiss).toHaveBeenCalled();
  });

  it("renders the success banner once the restart completes", () => {
    render(
      <SettingsRestartBanner
        slug="general"
        restart={makeRestart({ isRestartRequired: true, isSuccessful: true })}
      />
    );

    const banner = screen.getByTestId("settings-page-general-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "success");
    expect(banner).toHaveTextContent("Daemon restarted successfully");
  });
});
