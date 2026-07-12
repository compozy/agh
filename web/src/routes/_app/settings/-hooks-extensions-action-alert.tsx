import { Check, Info, X } from "lucide-react";

import { useSettingsHooksExtensionsPage } from "@/hooks/routes/use-settings-hooks-extensions-page";
import { Alert, AlertAction, AlertDescription, Button } from "@agh/ui";

export function LastActionAlert({
  action,
  onDismiss,
}: {
  action: NonNullable<ReturnType<typeof useSettingsHooksExtensionsPage>["lastAction"]>;
  onDismiss: () => void;
}) {
  const { message, tone } = describeAction(action);
  return (
    <Alert
      variant={tone === "success" ? "success" : "info"}
      role="status"
      data-testid="settings-page-hooks-extensions-action-result"
      data-kind={action.kind}
    >
      {tone === "success" ? <Check className="size-3" /> : <Info className="size-3" />}
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="settings-page-hooks-extensions-action-result-dismiss"
        >
          <X className="size-3" />
        </Button>
      </AlertAction>
    </Alert>
  );
}

function describeAction(
  action: NonNullable<ReturnType<typeof useSettingsHooksExtensionsPage>["lastAction"]>
): { message: string; tone: "success" | "info" } {
  if (action.kind === "saved") {
    const restartBadge = action.result.restart_required
      ? "restart required to apply"
      : "applied immediately";
    return { message: `Policy saved · ${restartBadge}.`, tone: "success" };
  }
  if (action.kind === "hook-toggled") {
    const state = action.enabled ? "enabled" : "disabled";
    const restartBadge = action.result.restart_required
      ? "restart required to reload"
      : "applied immediately";
    return {
      message: `Hook "${action.name}" ${state} · ${restartBadge}.`,
      tone: "success",
    };
  }
  if (action.kind === "extension-installed") {
    return {
      message: `Extension "${action.name}" installed · trust decision recorded.`,
      tone: "info",
    };
  }
  if (action.kind === "extension-updated") {
    return {
      message: `Extension "${action.name}" update ${action.status}.`,
      tone: "info",
    };
  }
  if (action.kind === "extension-removed") {
    return {
      message: `Extension "${action.name}" removed.`,
      tone: "info",
    };
  }
  if (action.kind === "notification-preset-created") {
    return {
      message: "Notification preset " + action.name + " created.",
      tone: "success",
    };
  }
  if (action.kind === "notification-preset-toggled") {
    const state = action.enabled ? "enabled" : "disabled";
    return {
      message: "Notification preset " + action.name + " " + state + ".",
      tone: "success",
    };
  }
  if (action.kind === "notification-preset-deleted") {
    return {
      message: "Notification preset " + action.name + " deleted.",
      tone: "info",
    };
  }
  const state = action.enabled ? "enabled" : "disabled";
  return {
    message: `Extension "${action.name}" ${state} · applied immediately.`,
    tone: "info",
  };
}
