import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsRestartBanner } from "../settings-restart-banner";

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

  it("renders warning tone with the restart-required message and trigger button", () => {
    const restart = makeRestart({ isRestartRequired: true });
    render(<SettingsRestartBanner slug="memory" restart={restart} />);

    const banner = screen.getByTestId("settings-page-memory-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "warning");
    expect(banner).toHaveAttribute("role", "status");
    expect(screen.getByTestId("settings-page-memory-restart-banner-message")).toHaveTextContent(
      "Changes saved. Restart the daemon to apply."
    );
    expect(
      screen.queryByTestId("settings-page-memory-restart-banner-active-sessions")
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("settings-page-memory-restart-banner-trigger"));
    expect(restart.trigger).toHaveBeenCalled();
  });

  it("renders polling tone (info) without the trigger button and with status suffix", () => {
    render(
      <SettingsRestartBanner
        slug="observability"
        restart={makeRestart({
          isRestartRequired: true,
          isPolling: true,
          status: "stopping",
          operationId: "op_1",
          activeSessionCount: 2,
        })}
      />
    );

    const banner = screen.getByTestId("settings-page-observability-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "info");
    expect(
      screen.getByTestId("settings-page-observability-restart-banner-message")
    ).toHaveTextContent("Restarting daemon · stopping");
    expect(screen.getByTestId("settings-page-observability-restart-banner-op")).toHaveTextContent(
      "op_1"
    );
    expect(
      screen.getByTestId("settings-page-observability-restart-banner-active-sessions")
    ).toHaveTextContent("2 active sessions");
    expect(
      screen.getByTestId("settings-page-observability-restart-banner-active-sessions")
    ).toHaveClass("font-mono", "text-badge", "font-semibold", "tracking-badge");
    expect(
      screen.getByTestId("settings-page-observability-restart-banner-active-sessions").className
    ).toContain("text-(--muted)");
    expect(
      screen.queryByTestId("settings-page-observability-restart-banner-trigger")
    ).not.toBeInTheDocument();
  });

  it("renders the danger tone with the failure reason suffix and role=alert", () => {
    const restart = makeRestart({
      isRestartRequired: true,
      isFailed: true,
      failureReason: "helper exited non-zero",
    });
    render(<SettingsRestartBanner slug="general" restart={restart} />);

    const banner = screen.getByTestId("settings-page-general-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "danger");
    expect(banner).toHaveAttribute("role", "alert");
    expect(banner).toHaveTextContent("helper exited non-zero");

    fireEvent.click(screen.getByTestId("settings-page-general-restart-banner-dismiss"));
    expect(restart.dismiss).toHaveBeenCalled();
  });

  it("renders the success tone with the Dismiss button", () => {
    const restart = makeRestart({ isRestartRequired: true, isSuccessful: true });
    render(<SettingsRestartBanner slug="general" restart={restart} />);

    const banner = screen.getByTestId("settings-page-general-restart-banner");
    expect(banner).toHaveAttribute("data-tone", "success");
    expect(banner).toHaveTextContent("Daemon restarted successfully");
    expect(screen.getByTestId("settings-page-general-restart-banner-dismiss")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("settings-page-general-restart-banner-dismiss"));
    expect(restart.dismiss).toHaveBeenCalled();
  });

  it("omits the Dismiss button when the banner is only in the warning state", () => {
    render(
      <SettingsRestartBanner slug="general" restart={makeRestart({ isRestartRequired: true })} />
    );
    expect(
      screen.queryByTestId("settings-page-general-restart-banner-dismiss")
    ).not.toBeInTheDocument();
  });
});
