import { Button } from "@agh/ui";

export interface NetworkCoordinationInvitationProps {
  visible: boolean;
  accepting?: boolean;
  dismissing?: boolean;
  onAccept: () => void;
  onDismiss: () => void;
  className?: string;
}

export function NetworkCoordinationInvitation({
  visible,
  accepting = false,
  dismissing = false,
  onAccept,
  onDismiss,
  className,
}: NetworkCoordinationInvitationProps) {
  if (!visible) {
    return null;
  }

  return (
    <div
      className={className}
      data-testid="network-coordination-invitation"
      role="region"
      aria-label="Coordination invitation"
    >
      <p className="text-sm text-foreground">
        Turn on coordination conversations for future runs in this workspace? Accepting does not
        change the active run.
      </p>
      <div className="mt-3 flex gap-2">
        <Button
          data-testid="network-coordination-invitation-accept"
          disabled={accepting || dismissing}
          onClick={onAccept}
          size="sm"
          type="button"
        >
          {accepting ? "Enabling…" : "Enable for future runs"}
        </Button>
        <Button
          data-testid="network-coordination-invitation-dismiss"
          disabled={accepting || dismissing}
          onClick={onDismiss}
          size="sm"
          type="button"
          variant="outline"
        >
          {dismissing ? "Dismissing…" : "Dismiss"}
        </Button>
      </div>
    </div>
  );
}
