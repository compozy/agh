import { Pencil, Play } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  TopbarOverflowIcon,
} from "@agh/ui";

import type { AutomationJob, AutomationTrigger } from "../types";

interface AutomationDetailActionsProps {
  item: AutomationJob | AutomationTrigger;
  onEdit: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onTriggerNow?: () => void;
  state: {
    togglePending: boolean;
    triggerDisabled: boolean;
    triggerPending: boolean;
  };
}

function AutomationDetailActions({
  item,
  onEdit,
  onToggleEnabled,
  onTriggerNow,
  state,
}: AutomationDetailActionsProps) {
  const isDynamic = item.source === "dynamic";
  const showOverflow = isDynamic || Boolean(onTriggerNow);
  return (
    <div className="flex items-center gap-2" data-testid="automation-detail-actions">
      {isDynamic ? (
        <Button
          data-testid="edit-automation-btn"
          onClick={onEdit}
          size="sm"
          type="button"
          variant="neutral"
        >
          <Pencil className="size-3" />
          Edit
        </Button>
      ) : null}
      {onTriggerNow ? (
        <Button
          data-testid="trigger-job-btn"
          disabled={state.triggerDisabled || state.triggerPending}
          onClick={onTriggerNow}
          size="sm"
          type="button"
        >
          <Play className="size-3" />
          {state.triggerPending ? "Queuing..." : "Run now"}
        </Button>
      ) : null}
      {!showOverflow ? (
        <Button
          data-testid="toggle-automation-btn"
          disabled={state.togglePending}
          onClick={() => onToggleEnabled(!item.enabled)}
          size="sm"
          type="button"
          variant={item.enabled ? "neutral" : "default"}
        >
          {state.togglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
        </Button>
      ) : null}
    </div>
  );
}

interface AutomationDetailOverflowProps {
  isTogglePending: boolean;
  item: AutomationJob | AutomationTrigger;
  kind: "jobs" | "triggers";
  onDelete: () => void;
  onToggleEnabled: (enabled: boolean) => void;
}

function AutomationDetailOverflow({
  isTogglePending,
  item,
  kind,
  onDelete,
  onToggleEnabled,
}: AutomationDetailOverflowProps) {
  const isDynamic = item.source === "dynamic";
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More actions"
        data-testid="automation-detail-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="automation-detail-overflow-menu">
        <DropdownMenuItem
          data-testid="toggle-automation-btn"
          disabled={isTogglePending}
          onClick={() => onToggleEnabled(!item.enabled)}
        >
          {isTogglePending ? "Saving..." : item.enabled ? "Disable" : "Enable"}
        </DropdownMenuItem>
        {isDynamic ? (
          <DropdownMenuItem
            data-testid="delete-automation-btn"
            onClick={onDelete}
            variant="destructive"
          >
            Delete {kind === "jobs" ? "job" : "trigger"}
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export { AutomationDetailActions, AutomationDetailOverflow };
