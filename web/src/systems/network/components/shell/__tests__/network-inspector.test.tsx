// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NetworkInspector } from "../network-inspector";

function renderInspector() {
  return render(
    <NetworkInspector
      activeTab="members"
      channel="ops"
      directs={[]}
      isActivityLoading={false}
      isMembersLoading={false}
      isWorkLoading={false}
      members={[]}
      onClose={() => undefined}
      onTabChange={() => undefined}
      threads={[]}
      workCount={0}
      workEntries={[]}
    />
  );
}

describe("NetworkInspector", () => {
  it("Should NOT render the overflow menu (coming-soon affordance) per ADR-013 §5", () => {
    renderInspector();
    expect(screen.queryByTestId("network-inspector-overflow")).toBeNull();
    expect(screen.queryByRole("button", { name: /more actions/i })).toBeNull();
  });

  it("Should expose the close button", () => {
    renderInspector();
    expect(screen.getByTestId("network-inspector-close")).toBeInTheDocument();
  });
});
