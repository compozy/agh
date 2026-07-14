import { NetworkCoordinationInvitation } from "./coordination-invitation";
import {
  useAcceptNetworkCoordinationInvitation,
  useDismissNetworkCoordinationInvitation,
  useNetworkCoordination,
} from "../hooks/use-network-coordination";
import { useNetworkPage } from "../hooks/use-network-page";
import { shouldShowCoordinationInvitation } from "../lib/coordination-invitation-gates";

export interface TaskRunCoordinationInvitationHostProps {
  taskId: string;
  hasActiveRun: boolean;
  isCoordinator: boolean;
  workerCount: number;
}

export function TaskRunCoordinationInvitationHost({
  taskId,
  hasActiveRun,
  isCoordinator,
  workerCount,
}: TaskRunCoordinationInvitationHostProps) {
  const page = useNetworkPage();
  const coordination = useNetworkCoordination(taskId);
  const accept = useAcceptNetworkCoordinationInvitation(taskId);
  const dismiss = useDismissNetworkCoordinationInvitation("task", taskId);

  const networkAvailable = Boolean(page.status?.enabled);
  const invitation = coordination.data?.invitation;
  const visible = shouldShowCoordinationInvitation({
    hasActiveRun,
    isCoordinator,
    workerCount,
    coordinationEnabled: Boolean(coordination.data?.enabled),
    networkAvailable,
    invitationDismissed: Boolean(invitation?.dismissed),
  });

  return (
    <NetworkCoordinationInvitation
      accepting={accept.isPending}
      dismissing={dismiss.isPending}
      onAccept={() => {
        if (accept.isPending || dismiss.isPending) return;
        void accept.mutateAsync();
      }}
      onDismiss={() => {
        if (accept.isPending || dismiss.isPending) return;
        void dismiss.mutateAsync();
      }}
      visible={visible}
    />
  );
}
