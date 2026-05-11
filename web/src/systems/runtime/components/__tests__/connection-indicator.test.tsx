import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/systems/daemon/hooks/use-daemon-connection-status", () => ({
  useDaemonConnectionStatus: () => "connected" as const,
}));

vi.mock("../../hooks/use-nav-counts", () => ({
  useNavCounts: () => ({
    counts: {},
    refresh: () => undefined,
    status: "loading" as const,
  }),
}));

import { RuntimeConnectionIndicator, resolveRuntimeConnectionState } from "../connection-indicator";

describe("resolveRuntimeConnectionState", () => {
  it("Should return success solid when daemon is connected and not degraded", () => {
    expect(resolveRuntimeConnectionState("connected", false)).toEqual({
      tone: "success",
      pulse: false,
      label: "Connected",
    });
  });

  it("Should return success pulse when daemon is connected but degraded", () => {
    expect(resolveRuntimeConnectionState("connected", true)).toEqual({
      tone: "success",
      pulse: true,
      label: "Degraded",
    });
  });

  it("Should return success pulse when daemon is mid-connect", () => {
    expect(resolveRuntimeConnectionState("connecting", false)).toEqual({
      tone: "success",
      pulse: true,
      label: "Connecting",
    });
  });

  it("Should return danger solid when daemon is disconnected", () => {
    expect(resolveRuntimeConnectionState("disconnected", false)).toEqual({
      tone: "danger",
      pulse: false,
      label: "Disconnected",
    });
  });

  it("Should return danger solid with the error label when the query errors", () => {
    expect(resolveRuntimeConnectionState("error", false)).toEqual({
      tone: "danger",
      pulse: false,
      label: "Connection error",
    });
  });
});

describe("RuntimeConnectionIndicator", () => {
  it("Should render the dot and label when variant defaults to footer", () => {
    render(<RuntimeConnectionIndicator status="connected" degraded={false} />);
    const indicator = screen.getByTestId("runtime-connection-indicator");
    expect(indicator).toHaveAttribute("data-tone", "success");
    expect(indicator).toHaveAttribute("data-pulse", "false");
    expect(indicator).toHaveAttribute("data-status", "connected");
    expect(indicator).toHaveAttribute("data-variant", "footer");
    expect(indicator).toHaveAttribute("role", "status");
    expect(indicator.querySelector('[data-slot="connection-indicator-dot"]')).not.toBeNull();
    expect(indicator.querySelector('[data-slot="connection-indicator-label"]')).toHaveTextContent(
      "Connected"
    );
  });

  it("Should render only the dot when dotOnly is true (rail mode)", () => {
    render(<RuntimeConnectionIndicator status="disconnected" degraded={false} dotOnly />);
    const indicator = screen.getByTestId("runtime-connection-indicator");
    expect(indicator).toHaveAttribute("data-variant", "rail-dot");
    expect(indicator.querySelector('[data-slot="connection-indicator-dot"]')).not.toBeNull();
    expect(indicator.querySelector('[data-slot="connection-indicator-label"]')).toBeNull();
  });

  it("Should pulse the dot when daemon is connected but degraded", () => {
    render(<RuntimeConnectionIndicator status="connected" degraded />);
    const indicator = screen.getByTestId("runtime-connection-indicator");
    expect(indicator).toHaveAttribute("data-tone", "success");
    expect(indicator).toHaveAttribute("data-pulse", "true");
    const dot = indicator.querySelector('[data-slot="connection-indicator-dot"]');
    expect(dot).toHaveAttribute("data-pulse", "true");
  });

  it("Should paint the danger tone when daemon is unreachable", () => {
    render(<RuntimeConnectionIndicator status="disconnected" degraded={false} />);
    const indicator = screen.getByTestId("runtime-connection-indicator");
    expect(indicator).toHaveAttribute("data-tone", "danger");
    expect(indicator).toHaveAttribute("data-pulse", "false");
  });
});
