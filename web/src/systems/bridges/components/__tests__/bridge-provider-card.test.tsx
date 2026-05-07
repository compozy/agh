import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BridgeProviderCard } from "@/systems/bridges/components/bridge-provider-card";
import type { BridgeProvider } from "@/systems/bridges/types";

function makeProvider(overrides: Partial<BridgeProvider> = {}): BridgeProvider {
  return {
    config_schema: { schema: "provider-config", version: "2026-04-15" },
    description: "Provider-specific runtime settings",
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    platform: "telegram",
    secret_slots: [],
    state: "active",
    ...overrides,
  };
}

describe("BridgeProviderCard", () => {
  it("renders the provider name and default copy without onSelect", () => {
    render(<BridgeProviderCard provider={makeProvider({ description: undefined })} />);
    expect(screen.getByTestId("bridge-provider-card-ext-telegram::telegram")).toBeInTheDocument();
    expect(
      screen.getByText(/Bridge adapter installed and ready for instance configuration/i)
    ).toBeInTheDocument();
  });

  it("calls onSelect when the card is clicked and selectable", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<BridgeProviderCard onSelect={onSelect} provider={makeProvider()} />);

    await user.click(screen.getByTestId("bridge-provider-card-ext-telegram::telegram"));
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("renders UNAVAILABLE badge and disables selection when unhealthy", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <BridgeProviderCard
        onSelect={onSelect}
        provider={makeProvider({ enabled: false, health: "unhealthy" })}
      />
    );

    expect(screen.getByText("UNAVAILABLE")).toBeInTheDocument();
    const card = screen.getByTestId("bridge-provider-card-ext-telegram::telegram");
    expect(card).toBeDisabled();

    await user.click(card);
    expect(onSelect).not.toHaveBeenCalled();
  });
});
