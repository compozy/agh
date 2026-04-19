import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { networkPeersFixture } from "../mocks";
import { NetworkPeersListPanel } from "./network-peers-list-panel";

describe("NetworkPeersListPanel", () => {
  it("Should render the no-results Empty state when the search query does not match any peer", () => {
    render(
      <NetworkPeersListPanel
        onSearchChange={() => undefined}
        onSelectPeer={() => undefined}
        peers={[]}
        searchQuery="missing"
        selectedPeerId={null}
      />
    );

    const empty = screen.getByTestId("network-peers-list-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveTextContent("No peers found");
  });

  it("Should forward typed search input via onSearchChange with the typed value", () => {
    const onSearchChange = vi.fn();
    render(
      <NetworkPeersListPanel
        onSearchChange={onSearchChange}
        onSelectPeer={() => undefined}
        peers={networkPeersFixture}
        searchQuery=""
        selectedPeerId={null}
      />
    );

    fireEvent.change(screen.getByTestId("network-peer-search-input"), {
      target: { value: "remote" },
    });

    expect(onSearchChange).toHaveBeenCalledWith("remote");
  });

  it('Should call onSelectPeer with the peer id and surface data-state="selected"', () => {
    const onSelectPeer = vi.fn();
    const target = networkPeersFixture[1]!;
    render(
      <NetworkPeersListPanel
        onSearchChange={() => undefined}
        onSelectPeer={onSelectPeer}
        peers={networkPeersFixture}
        searchQuery=""
        selectedPeerId={networkPeersFixture[0]!.peer_id}
      />
    );

    const selectedRow = screen.getByTestId(`network-peer-item-${networkPeersFixture[0]!.peer_id}`);
    expect(selectedRow).toHaveAttribute("data-state", "selected");

    fireEvent.click(screen.getByTestId(`network-peer-item-${target.peer_id}`));
    expect(onSelectPeer).toHaveBeenCalledWith(target.peer_id);
  });

  it("Should surface the error Empty state when an errorMessage is provided with an empty list", () => {
    render(
      <NetworkPeersListPanel
        errorMessage="Peer discovery failed"
        onSearchChange={() => undefined}
        onSelectPeer={() => undefined}
        peers={[]}
        searchQuery=""
        selectedPeerId={null}
      />
    );

    const error = screen.getByTestId("network-peers-list-error");
    expect(error).toHaveTextContent("Unable to load peers");
    expect(error).toHaveTextContent("Peer discovery failed");
  });
});
