import { useState } from "react";
import { RefreshCw } from "lucide-react";
import { toast } from "sonner";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

import { useDaemonConnectionStatus } from "@/systems/status/hooks/use-daemon-connection-status";
import { useSettingsRestart } from "@/systems/settings";

export interface RestartDaemonButtonProps {
  activeSessionCount: number;
}

function describeImpact(activeSessionCount: number): string {
  if (activeSessionCount <= 0) {
    return "No active sessions will be interrupted.";
  }
  if (activeSessionCount === 1) {
    return "1 active session will be interrupted.";
  }
  return `${activeSessionCount} active sessions will be interrupted.`;
}

function RestartDaemonButton({ activeSessionCount }: RestartDaemonButtonProps) {
  const [open, setOpen] = useState(false);
  const connectionStatus = useDaemonConnectionStatus();
  const { triggerAsync, isTriggerPending, isPolling } = useSettingsRestart();

  const isRestarting = isTriggerPending || isPolling;
  const isDisabled = isRestarting || connectionStatus !== "connected";

  const handleConfirm = async () => {
    setOpen(false);
    try {
      await triggerAsync();
    } catch {
      toast.error("Failed to restart daemon.");
    }
  };

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="ml-auto"
        aria-label="Restart daemon"
        data-testid="sidebar-restart-daemon"
        disabled={isDisabled}
        onClick={() => setOpen(true)}
      >
        <RefreshCw aria-hidden="true" />
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent data-testid="sidebar-restart-confirm" showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Restart daemon?</DialogTitle>
            <DialogDescription data-testid="sidebar-restart-confirm-detail">
              {describeImpact(activeSessionCount)}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => setOpen(false)}
              data-testid="sidebar-restart-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick={handleConfirm}
              disabled={isRestarting}
              data-testid="sidebar-restart-confirm-button"
            >
              Restart daemon
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export { RestartDaemonButton, describeImpact };
