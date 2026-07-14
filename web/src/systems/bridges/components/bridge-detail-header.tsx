import { Pencil, Power, RotateCw } from "lucide-react";

import { Button, DetailHeader, Pill, type PillTone } from "@agh/ui";

import { bridgeStatusLabel, bridgeStatusTone } from "../lib/bridge-formatters";
import type { BridgeStatus, BridgeSummary } from "../types";

interface BridgeDetailHeaderProps {
  bridge: BridgeSummary;
  effectiveStatus: BridgeStatus;
  isLifecyclePending: boolean;
  onBack?: () => void;
  onDisableBridge?: () => void;
  onEnableBridge?: () => void;
  onOpenEdit?: () => void;
  onRestartBridge?: () => void;
}

function statusToPillTone(status: BridgeStatus): PillTone {
  return status === "disabled" ? "danger" : bridgeStatusTone(status);
}

export function BridgeDetailHeader({
  bridge,
  effectiveStatus,
  isLifecyclePending,
  onBack,
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

  const actions = (
    <>
      <Button
        data-testid="edit-bridge-btn"
        disabled={isLifecyclePending}
        onClick={onOpenEdit}
        size="sm"
        type="button"
        variant="outline"
      >
        <Pencil className="size-3" />
        Edit
      </Button>
      <Button
        data-testid="restart-bridge-btn"
        disabled={isLifecyclePending}
        onClick={onRestartBridge}
        size="sm"
        type="button"
        variant="outline"
      >
        <RotateCw className="size-3" />
        Restart
      </Button>
      {bridge.enabled ? (
        <Button
          data-testid="disable-bridge-btn"
          disabled={isLifecyclePending}
          onClick={onDisableBridge}
          size="sm"
          type="button"
          variant="outline"
        >
          <Power className="size-3" />
          Disable
        </Button>
      ) : (
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
      )}
    </>
  );

  return (
    <DetailHeader
      actions={actions}
      back={onBack}
      backLabel="Back to bridges"
      crumbs={onBack ? [{ id: "bridges", label: "Bridges", onSelect: onBack }] : undefined}
      data-testid="bridge-detail-header"
      meta={
        <span data-testid="bridge-detail-meta-platform">
          {bridge.platform} / {bridge.extension_name}
        </span>
      }
      pills={pills}
      title={bridge.display_name}
    />
  );
}
