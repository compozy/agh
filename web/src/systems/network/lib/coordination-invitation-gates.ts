export interface NetworkCoordinationInvitationGates {
  hasActiveRun: boolean;
  isCoordinator: boolean;
  workerCount: number;
  coordinationEnabled: boolean;
  networkAvailable: boolean;
  invitationDismissed: boolean;
}

/** Exact invitation visibility matrix for UT-055. */
export function shouldShowCoordinationInvitation(
  gates: NetworkCoordinationInvitationGates
): boolean {
  return (
    gates.hasActiveRun &&
    gates.isCoordinator &&
    gates.workerCount >= 2 &&
    !gates.coordinationEnabled &&
    gates.networkAvailable &&
    !gates.invitationDismissed
  );
}
