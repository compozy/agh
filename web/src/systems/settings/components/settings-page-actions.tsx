import { RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@agh/ui";
import type { RestartBannerState } from "./settings-restart-banner";

interface SettingsPageActionsProps {
  slug: string;
  restart: RestartBannerState;
  secondaryAction?: ReactNode;
}

function SettingsPageActions({ slug, restart, secondaryAction }: SettingsPageActionsProps) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-2">
      {secondaryAction}
      <Button
        type="button"
        variant="outline"
        size="sm"
        data-testid={`settings-page-${slug}-restart-action`}
        disabled={restart.isTriggerPending || restart.isPolling}
        onClick={() => restart.trigger()}
      >
        <RefreshCw className="size-3.5" />
        {restart.isTriggerPending || restart.isPolling ? "Restarting..." : "Restart daemon"}
      </Button>
    </div>
  );
}

export { SettingsPageActions };
