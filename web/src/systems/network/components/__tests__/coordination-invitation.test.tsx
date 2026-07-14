import { describe, expect, it } from "vitest";

import {
  shouldShowCoordinationInvitation,
  type NetworkCoordinationInvitationGates,
} from "../../lib/coordination-invitation-gates";

const baseGates: NetworkCoordinationInvitationGates = {
  hasActiveRun: true,
  isCoordinator: true,
  workerCount: 2,
  coordinationEnabled: false,
  networkAvailable: true,
  invitationDismissed: false,
};

describe("shouldShowCoordinationInvitation", () => {
  it("Should show when every gate is true", () => {
    expect(shouldShowCoordinationInvitation(baseGates)).toBe(true);
  });

  it.each([
    ["hasActiveRun", { hasActiveRun: false }],
    ["isCoordinator", { isCoordinator: false }],
    ["workerCount", { workerCount: 1 }],
    ["coordinationEnabled", { coordinationEnabled: true }],
    ["networkAvailable", { networkAvailable: false }],
    ["invitationDismissed", { invitationDismissed: true }],
  ] as const)("Should hide when %s is false for the invitation matrix", (_name, override) => {
    expect(
      shouldShowCoordinationInvitation({
        ...baseGates,
        ...override,
      })
    ).toBe(false);
  });
});
