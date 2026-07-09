import { Check, X } from "lucide-react";

import type { MCPLastAction } from "@/hooks/routes/use-mcp-page";
import { Alert, AlertAction, AlertDescription, Button } from "@agh/ui";

import { mcpWriteTargetLabel } from "./mcp-server-labels";

export function MCPActionResultBanner({
  action,
  onDismiss,
}: {
  action: MCPLastAction;
  onDismiss: () => void;
}) {
  const isSaved = action.kind === "saved";
  const restartBadge = action.result.restart_required
    ? "restart required to apply"
    : "applied immediately";
  const writeTargetLabel = action.result.write_target
    ? ` · persisted to ${mcpWriteTargetLabel(action.result.write_target)}`
    : "";

  const message = isSaved
    ? `Saved "${action.name}"${writeTargetLabel} · ${restartBadge}.`
    : action.remainingShadowed > 0
      ? `Deleted "${action.name}" · ${action.remainingShadowed} shadowed source${action.remainingShadowed === 1 ? "" : "s"} may become effective on reload · ${restartBadge}.`
      : `Deleted "${action.name}" · no other sources remained · ${restartBadge}.`;

  return (
    <Alert
      variant={isSaved ? "success" : "info"}
      role="status"
      data-testid="settings-page-mcp-servers-action-result"
      data-kind={action.kind}
    >
      <Check className="size-3" />
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="settings-page-mcp-servers-action-result-dismiss"
        >
          <X className="size-3" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
