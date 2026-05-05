import { Network as NetworkIcon } from "lucide-react";

import { Button, Empty } from "@agh/ui";

export interface NetworkEmptyProps {
  /** Settings deep-link handler from the parent route. */
  onOpenSettings?: () => void;
  className?: string;
}

/**
 * Empty state for when the embedded network is disabled in config (`_design.md`
 * §7.2).
 */
export function NetworkEmpty({ onOpenSettings, className }: NetworkEmptyProps) {
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
            Open settings
          </Button>
        ) : null
      }
      className={className}
      data-testid="network-empty"
      description="Enable the embedded network in your AGH config to start."
      icon={NetworkIcon}
      title="The network is off."
    />
  );
}
