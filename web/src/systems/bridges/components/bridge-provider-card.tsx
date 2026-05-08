import { Pill, type PillTone } from "@agh/ui";

import { cn } from "@/lib/utils";
import { KindChip } from "@/systems/network";
import {
  buildBridgeProviderKey,
  isBridgeProviderSelectable,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeProvider } from "@/systems/bridges/types";

interface BridgeProviderCardProps {
  onSelect?: () => void;
  provider: BridgeProvider;
  selected?: boolean;
}

function healthBadgeTone(health?: string): PillTone {
  switch (health) {
    case "healthy":
      return "success";
    case "unhealthy":
      return "danger";
    default:
      return "neutral";
  }
}

function stateBadgeTone(state?: string): PillTone {
  switch (state) {
    case "active":
      return "success";
    case "error":
      return "danger";
    case "registered":
      return "info";
    case "enabled":
      return "warning";
    default:
      return "neutral";
  }
}

export function BridgeProviderCard({
  onSelect,
  provider,
  selected = false,
}: BridgeProviderCardProps) {
  const selectable = isBridgeProviderSelectable(provider);
  const content = (
    <>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <h3 className="truncate text-small-body font-medium text-(--color-text-primary)">
            {provider.display_name}
          </h3>
          <div className="flex flex-wrap items-center gap-1.5">
            <KindChip kind={provider.platform} />
            <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-label)">
              {provider.extension_name}
            </span>
          </div>
        </div>
        <Pill mono tone={healthBadgeTone(provider.health)}>
          {provider.health}
        </Pill>
      </div>

      <p className="text-xs leading-relaxed text-(--color-text-secondary)">
        {provider.description ?? "Bridge adapter installed and ready for instance configuration."}
      </p>

      <div className="flex flex-wrap items-center gap-1.5">
        <Pill mono tone={stateBadgeTone(provider.state)}>
          {provider.state}
        </Pill>
        {!selectable ? (
          <Pill mono tone="danger">
            UNAVAILABLE
          </Pill>
        ) : null}
      </div>

      <p className="text-eyebrow leading-relaxed text-(--color-text-tertiary)">
        {provider.health_message ||
          (selectable
            ? "This provider can be used to create a bridge instance."
            : "This provider is installed but not available for bridge creation right now.")}
      </p>
    </>
  );

  const className = cn(
    "flex w-full flex-col gap-3 rounded-md border bg-(--color-surface) p-4 text-left transition-colors",
    selected ? "border-accent bg-(--color-surface-elevated)" : "border-(--color-divider)",
    onSelect && selectable && "cursor-pointer hover:border-accent hover:bg-(--color-hover)",
    onSelect && !selectable && "cursor-not-allowed opacity-70"
  );

  if (!onSelect) {
    return (
      <div
        className={className}
        data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
      >
        {content}
      </div>
    );
  }

  return (
    <button
      className={className}
      data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
      disabled={!selectable}
      onClick={onSelect}
      type="button"
    >
      {content}
    </button>
  );
}
