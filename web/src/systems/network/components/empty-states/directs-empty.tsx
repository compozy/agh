import { MessageCircle } from "lucide-react";

import { Button, Empty } from "@agh/ui";

export interface DirectsEmptyProps {
  onNewDirect?: () => void;
  className?: string;
}

/**
 * Empty state for the Directs tab (`_design.md` §7.2).
 */
export function DirectsEmpty({ onNewDirect, className }: DirectsEmptyProps) {
  return (
    <Empty
      action={
        onNewDirect ? (
          <Button
            data-testid="network-directs-empty-new"
            onClick={onNewDirect}
            size="sm"
            type="button"
            variant="outline"
          >
            New direct
          </Button>
        ) : null
      }
      className={className}
      data-testid="network-directs-empty"
      description="Open one to talk privately with a peer in this channel."
      icon={MessageCircle}
      title="No direct rooms yet."
    />
  );
}
