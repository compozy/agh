import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import { NetworkCoordinationInvitation } from "../coordination-invitation";

describe("NetworkCoordinationInvitation interactions", () => {
  it("Should invoke accept only once while accepting is true for double-click", () => {
    const onAccept = vi.fn();
    const onDismiss = vi.fn();
    const { rerender } = render(
      <NetworkCoordinationInvitation
        accepting={false}
        dismissing={false}
        onAccept={onAccept}
        onDismiss={onDismiss}
        visible
      />
    );

    fireEvent.click(screen.getByTestId("network-coordination-invitation-accept"));
    expect(onAccept).toHaveBeenCalledTimes(1);

    rerender(
      <NetworkCoordinationInvitation
        accepting
        dismissing={false}
        onAccept={onAccept}
        onDismiss={onDismiss}
        visible
      />
    );
    fireEvent.click(screen.getByTestId("network-coordination-invitation-accept"));
    expect(onAccept).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/future runs/i)).toBeInTheDocument();
  });

  it("Should hide when not visible so terminal runs can drop the invite gracefully", () => {
    const { container } = render(
      <NetworkCoordinationInvitation
        onAccept={() => undefined}
        onDismiss={() => undefined}
        visible={false}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });
});
