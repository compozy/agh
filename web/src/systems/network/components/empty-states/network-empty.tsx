import { Network as NetworkIcon } from "lucide-react";

import { Button, Empty } from "@agh/ui";

export interface NetworkEmptyProps {
  /** Settings deep-link handler from the parent route. */
  onOpenSettings?: () => void;
  /** When true, Network is administratively disabled; name who can change it. */
  disabledByAdmin?: boolean;
  className?: string;
}

/**
 * Oriented empty for Network discovery (UT-058):
 * where am I, what can I do, who changes this, what is the next action.
 */
export function NetworkEmpty({
  onOpenSettings,
  disabledByAdmin = false,
  className,
}: NetworkEmptyProps) {
  const title = disabledByAdmin ? "Network is disabled." : "Network is ready when you are.";
  const description = disabledByAdmin
    ? "An operator with admin access can enable Network availability. Existing channels and history stay intact."
    : "You are in the Network area. Enable coordination for future multi-agent runs, or open settings to review Live defaults and ceilings. Availability does not opt executions in.";

  return (
    <Empty
      action={
        onOpenSettings ? (
          <Button
            data-testid="network-empty-open-settings"
            onClick={onOpenSettings}
            size="sm"
            type="button"
            variant="outline"
          >
            Open network settings
          </Button>
        ) : null
      }
      className={className}
      data-testid="network-empty"
      description={description}
      icon={NetworkIcon}
      title={title}
    />
  );
}
