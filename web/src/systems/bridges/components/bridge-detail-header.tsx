import { Pencil, Power, RotateCw } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  PageHead,
  Pill,
  TopbarOverflowIcon,
  type PillTone,
  useTopbarSlot,
} from "@agh/ui";

import { bridgeStatusLabel, bridgeStatusTone } from "../lib/bridge-formatters";
import type { BridgeStatus, BridgeSummary } from "../types";

interface BridgeDetailHeaderProps {
  bridge: BridgeSummary;
  effectiveStatus: BridgeStatus;
  isLifecyclePending: boolean;
  onDisableBridge?: () => void;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
  onRestartBridge?: () => void;
}

function statusToPillTone(status: BridgeStatus): PillTone {
  return status === "disabled" ? "danger" : bridgeStatusTone(status);
}

function BridgeDetailActions({
  enabled,
  isLifecyclePending,
  onEnableBridge,
  onOpenEdit,
}: {
  enabled: boolean;
  isLifecyclePending: boolean;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
}) {
  return (
    <div className="flex items-center gap-2" data-testid="bridge-detail-actions">
      <Button
        data-testid="edit-bridge-btn"
        disabled={isLifecyclePending}
        onClick={onOpenEdit}
        size="sm"
        type="button"
        variant="neutral"
      >
        <Pencil className="size-3" />
        Edit
      </Button>
      {!enabled ? (
        <Button
          data-testid="enable-bridge-btn"
          disabled={isLifecyclePending}
          onClick={onEnableBridge}
          size="sm"
          type="button"
        >
          <Power className="size-3" />
          Enable
        </Button>
      ) : null}
    </div>
  );
}

function BridgeDetailOverflow({
  enabled,
  isLifecyclePending,
  onDisableBridge,
  onRestartBridge,
}: {
  enabled: boolean;
  isLifecyclePending: boolean;
  onDisableBridge?: () => void;
  onRestartBridge?: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="More actions"
        data-testid="bridge-detail-overflow"
        render={<Button type="button" variant="ghost" size="icon-sm" />}
      >
        <TopbarOverflowIcon aria-hidden="true" className="size-3" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="bridge-detail-overflow-menu">
        <DropdownMenuItem
          data-testid="restart-bridge-btn"
          disabled={isLifecyclePending}
          onClick={onRestartBridge}
        >
          <RotateCw className="size-3" />
          Restart
        </DropdownMenuItem>
        {enabled ? (
          <DropdownMenuItem
            data-testid="disable-bridge-btn"
            disabled={isLifecyclePending}
            onClick={onDisableBridge}
          >
            <Power className="size-3" />
            Disable
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function BridgeDetailHeader({
  bridge,
  effectiveStatus,
  isLifecyclePending,
  onDisableBridge,
  onEnableBridge,
  onOpenEdit,
  onRestartBridge,
}: BridgeDetailHeaderProps) {
  const statusTone = statusToPillTone(effectiveStatus);
  const pills = (
    <>
      <span className="flex items-center gap-2">
        <Pill.Dot pulse={effectiveStatus === "starting"} tone={statusTone} />
        <Pill mono tone={statusTone}>
          {bridgeStatusLabel(effectiveStatus)}
        </Pill>
      </span>
      <Pill mono tone={bridge.scope === "workspace" ? "info" : "neutral"}>
        {bridge.scope}
      </Pill>
    </>
  );

  // Route chrome §07: Edit + Enable (accent when disabled) → Restart/Disable in overflow.
  // Named zone components keep DropdownMenu state across Topbar slot republish.
  useTopbarSlot({
    actions: (
      <BridgeDetailActions
        enabled={bridge.enabled}
        isLifecyclePending={isLifecyclePending}
        onEnableBridge={onEnableBridge}
        onOpenEdit={onOpenEdit}
      />
    ),
    crumb: bridge.display_name,
    overflow: (
      <BridgeDetailOverflow
        enabled={bridge.enabled}
        isLifecyclePending={isLifecyclePending}
        onDisableBridge={onDisableBridge}
        onRestartBridge={onRestartBridge}
      />
    ),
  });

  return (
    <div className="pt-5">
      <PageHead
        data-testid="bridge-detail-header"
        title={bridge.display_name}
        variant="detail"
        meta={
          <span data-testid="bridge-detail-meta-platform">
            {bridge.platform} / {bridge.extension_name}
          </span>
        }
        pills={pills}
      />
    </div>
  );
}
