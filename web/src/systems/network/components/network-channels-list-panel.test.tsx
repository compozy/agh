import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { networkChannelsFixture } from "../mocks";
import { NetworkChannelsListPanel } from "./network-channels-list-panel";

describe("NetworkChannelsListPanel", () => {
  it("Should render the no-results Empty state when the search query does not match any channel", () => {
    render(
      <NetworkChannelsListPanel
        channels={[]}
        onSearchChange={() => undefined}
        onSelectChannel={() => undefined}
        searchQuery="missing"
        selectedChannel={null}
      />
    );

    const empty = screen.getByTestId("network-channels-list-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveTextContent("No channels found");
  });

  it("Should forward typed search input via onSearchChange with the typed value", () => {
    const onSearchChange = vi.fn();
    render(
      <NetworkChannelsListPanel
        channels={networkChannelsFixture.channels}
        onSearchChange={onSearchChange}
        onSelectChannel={() => undefined}
        searchQuery=""
        selectedChannel={null}
      />
    );

    fireEvent.change(screen.getByTestId("network-channel-search-input"), {
      target: { value: "coord" },
    });

    expect(onSearchChange).toHaveBeenCalledWith("coord");
  });

  it('Should mark the selected row via data-state="selected" and call onSelectChannel when clicked', () => {
    const onSelectChannel = vi.fn();
    const firstChannel = networkChannelsFixture.channels[0]!;
    render(
      <NetworkChannelsListPanel
        channels={networkChannelsFixture.channels}
        onSearchChange={() => undefined}
        onSelectChannel={onSelectChannel}
        searchQuery=""
        selectedChannel={firstChannel.channel}
      />
    );

    const selectedRow = screen.getByTestId(`network-channel-item-${firstChannel.channel}`);
    expect(selectedRow).toHaveAttribute("data-state", "selected");
    expect(selectedRow).toHaveAttribute("aria-pressed", "true");

    const otherChannel = networkChannelsFixture.channels[1]!;
    fireEvent.click(screen.getByTestId(`network-channel-item-${otherChannel.channel}`));
    expect(onSelectChannel).toHaveBeenCalledWith(otherChannel.channel);
  });

  it("Should surface the error Empty state when an errorMessage is provided with an empty list", () => {
    render(
      <NetworkChannelsListPanel
        channels={[]}
        errorMessage="Network unavailable"
        onSearchChange={() => undefined}
        onSelectChannel={() => undefined}
        searchQuery=""
        selectedChannel={null}
      />
    );

    const error = screen.getByTestId("network-channels-list-error");
    expect(error).toHaveTextContent("Unable to load channels");
    expect(error).toHaveTextContent("Network unavailable");
  });
});
